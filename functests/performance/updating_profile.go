package performance

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	machineconfigv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	testutils "github.com/openshift-kni/performance-addon-operators/functests/utils"
	testclient "github.com/openshift-kni/performance-addon-operators/functests/utils/client"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/nodes"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/profiles"
	performancev1alpha1 "github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v1alpha1"
)

const (
	mcpUpdateTimeout = 20
)

var _ = Describe("[rfe_id:28761] Updating parameters in performance profile", func() {
	var workerRTNodes []corev1.Node
	var profile *performancev1alpha1.PerformanceProfile
	var err error

	chkKernel := []string{"uname", "-a"}
	chkCmdLine := []string{"cat", "/proc/cmdline"}
	chkKubeletConfig := []string{"cat", "/rootfs/etc/kubernetes/kubelet.conf"}

	BeforeEach(func() {
		workerRTNodes, err = nodes.GetByRole(testclient.Client, testutils.RoleWorkerRT)
		Expect(err).ToNot(HaveOccurred())
		Expect(workerRTNodes).ToNot(BeEmpty())
		profile, err = profiles.GetByNodeLabels(
			testclient.Client,
			map[string]string{
				fmt.Sprintf("%s/%s", testutils.LabelRole, testutils.RoleWorkerRT): "",
			},
		)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("Verify that all performance profile parameters can be updated", func() {
		var removedKernelArgs string

		hpSize := performancev1alpha1.HugePageSize("2M")
		isolated := performancev1alpha1.CPUSet("1-2")
		reserved := performancev1alpha1.CPUSet("0,3")
		policy := "best-effort"
		f := false

		// Modify profile and verify that MCO successfully updated the node
		testutils.BeforeAll(func() {
			// timeout should be based on the number of worker nodes
			timeout := time.Duration(len(workerRTNodes) * mcpUpdateTimeout)

			By("Modifying profile")
			profile.Spec.HugePages = &performancev1alpha1.HugePages{
				DefaultHugePagesSize: &hpSize,
				Pages: []performancev1alpha1.HugePage{
					{
						Count: 5,
						Size:  hpSize,
					},
				},
			}
			profile.Spec.CPU = &performancev1alpha1.CPU{
				BalanceIsolated: &f,
				Reserved:        &reserved,
				Isolated:        &isolated,
			}
			profile.Spec.NUMA = &performancev1alpha1.NUMA{
				TopologyPolicy: &policy,
			}
			profile.Spec.RealTimeKernel = &performancev1alpha1.RealTimeKernel{
				Enabled: &f,
			}

			if profile.Spec.AdditionalKernelArgs == nil {
				By("AdditionalKernelArgs is empty. Checking only adding new arguments")
				profile.Spec.AdditionalKernelArgs = append(profile.Spec.AdditionalKernelArgs, "new-argument=test")
			} else {
				removedKernelArgs = profile.Spec.AdditionalKernelArgs[0]
				profile.Spec.AdditionalKernelArgs = append(profile.Spec.AdditionalKernelArgs[1:], "new-argument=test")
			}

			By("Verifying that mcp is ready for update")
			waitForMcpCondition(testutils.RoleWorkerRT, machineconfigv1.MachineConfigPoolUpdated, corev1.ConditionTrue, timeout)

			By("Applying changes in performance profile and waiting until mcp will start updating")
			Expect(testclient.Client.Update(context.TODO(), profile)).ToNot(HaveOccurred())
			waitForMcpCondition(testutils.RoleWorkerRT, machineconfigv1.MachineConfigPoolUpdating, corev1.ConditionTrue, timeout)

			By("Waiting when mcp finishes updates")
			waitForMcpCondition(testutils.RoleWorkerRT, machineconfigv1.MachineConfigPoolUpdated, corev1.ConditionTrue, timeout)
		})

		table.DescribeTable("Verify that profile parameters were updated", func(cmd, parameter []string, shouldContain bool) {
			for _, node := range workerRTNodes {
				for _, param := range parameter {
					if shouldContain {
						Expect(execCommandOnWorker(cmd, &node)).To(ContainSubstring(param))
					} else {
						Expect(execCommandOnWorker(cmd, &node)).NotTo(ContainSubstring(param))
					}
				}
			}
		},
			table.Entry("[test_id:28024] verify that hugepages size and count updated", chkCmdLine, []string{"default_hugepagesz=2M", "hugepagesz=2M", "hugepages=5"}, true),
			table.Entry("[test_id:28070] verify that hugepages updated (NUMA node unspecified)", chkCmdLine, []string{"hugepagesz=2M"}, true),
			table.Entry("[test_id:28025] verify that cpu affinity mask was updated", chkCmdLine, []string{"tuned.non_isolcpus=00000009"}, true),
			table.Entry("[test_id:28071] verify that cpu balancer disabled", chkCmdLine, []string{"isolcpus=1-2"}, true),
			table.Entry("[test_id:28935] verify that reservedSystemCPUs was updated", chkKubeletConfig, []string{`"reservedSystemCPUs":"0,3"`}, true),
			table.Entry("[test_id:28760] verify that topologyManager was updated", chkKubeletConfig, []string{`"topologyManagerPolicy":"best-effort"`}, true),
			table.Entry("[test_id:27738] verify that realTimeKernerl was updated", chkKernel, []string{"PREEMPT RT"}, false),
		)

		It("[test_id:28612]Verify that Kernel arguments can me updated (added, removed) thru performance profile", func() {
			for _, node := range workerRTNodes {
				// Verifying that new argument was added
				Expect(execCommandOnWorker(chkCmdLine, &node)).To(ContainSubstring("new-argument=test"))

				// Verifying that one of old arguments was removed
				if removedKernelArgs != "" {
					Expect(execCommandOnWorker(chkCmdLine, &node)).NotTo(ContainSubstring(removedKernelArgs), fmt.Sprintf("%s should be removed from /proc/cmdline", removedKernelArgs))
				}
			}
		})
	})

	It("[test_id:28440]Verifies that nodeSelector can be updated in performance profile", func() {
		var newWorkerNode *corev1.Node
		newRole := "worker-test"
		newLabel := fmt.Sprintf("%s/%s", testutils.LabelRole, newRole)
		newNodeSelector := map[string]string{newLabel: ""}

		By("Skipping test if cluster does not have another available worker node")
		workerNodes, err := nodes.GetByRole(testclient.Client, "worker")
		Expect(err).ToNot(HaveOccurred())

		for _, node := range workerNodes {
			if _, ok := node.Labels[fmt.Sprintf("%s/%s", testutils.LabelRole, testutils.RoleWorkerRT)]; ok {
				continue
			}
			newWorkerNode = &node
			break
		}
		if newWorkerNode == nil {
			Skip("Skipping test because there are no additional worker nodes")
		}
		newWorkerNode.Labels[newLabel] = ""
		Expect(testclient.Client.Update(context.TODO(), newWorkerNode)).ToNot(HaveOccurred())

		By("Creating new MachineConfigPool")
		mcp := newMCP(newRole, newNodeSelector)
		err = testclient.Client.Create(context.TODO(), mcp)
		Expect(err).ToNot(HaveOccurred())

		By("Updating Node Selector performance profile")
		profile.Spec.NodeSelector = newNodeSelector
		Expect(testclient.Client.Update(context.TODO(), profile)).ToNot(HaveOccurred())
		waitForMcpCondition(newRole, machineconfigv1.MachineConfigPoolUpdating, corev1.ConditionTrue, mcpUpdateTimeout)

		By("Waiting when MCP finishes updates and verifying new node has updated configuration")
		waitForMcpCondition(newRole, machineconfigv1.MachineConfigPoolUpdated, corev1.ConditionTrue, mcpUpdateTimeout)

		Expect(execCommandOnWorker(chkKubeletConfig, newWorkerNode)).To(ContainSubstring("topologyManagerPolicy"))
		Expect(execCommandOnWorker(chkCmdLine, newWorkerNode)).To(ContainSubstring("tuned.non_isolcpus"))
	})
})

func waitForMcpCondition(mcpName string, conditionType machineconfigv1.MachineConfigPoolConditionType, conditionStatus corev1.ConditionStatus, timeout time.Duration) {
	mcp := &machineconfigv1.MachineConfigPool{}
	key := types.NamespacedName{
		Name:      mcpName,
		Namespace: "",
	}
	Eventually(func() corev1.ConditionStatus {
		err := testclient.Client.Get(context.TODO(), key, mcp)
		Expect(err).ToNot(HaveOccurred())
		for i := range mcp.Status.Conditions {
			if mcp.Status.Conditions[i].Type == conditionType {
				return mcp.Status.Conditions[i].Status
			}
		}
		return corev1.ConditionUnknown
	}, timeout*time.Minute, 30*time.Second).Should(Equal(conditionStatus))
}

func newMCP(mcpName string, nodeSelector map[string]string) *machineconfigv1.MachineConfigPool {
	return &machineconfigv1.MachineConfigPool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mcpName,
			Namespace: "",
			Labels:    map[string]string{"machineconfiguration.openshift.io/role": mcpName},
		},
		Spec: machineconfigv1.MachineConfigPoolSpec{
			MachineConfigSelector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      "machineconfiguration.openshift.io/role",
						Operator: "In",
						Values:   []string{"worker", mcpName},
					},
				},
			},
			NodeSelector: &metav1.LabelSelector{
				MatchLabels: nodeSelector,
			},
		},
	}
}
