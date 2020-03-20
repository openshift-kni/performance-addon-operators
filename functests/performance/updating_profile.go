package performance

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
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

	It("Verifies that all performance profile parameters can be updated", func() {
		// timeout should be based on the number of worker nodes
		timeout := time.Duration(len(workerRTNodes) * mcpUpdateTimeout)
		newCmdLineArgs := map[string]string{
			"default_hugepagesz": "2M",
			"hugepagesz":         "2M",
			"hugepages":          "5",
			"tuned.non_isolcpus": "00000009",
			"isolcpus":           "1-2",
			"new-argument":       "test",
		}

		By("Verifying that mcp is updated")
		waitForMcpCondition(testutils.RoleWorkerRT, machineconfigv1.MachineConfigPoolUpdated, corev1.ConditionTrue, timeout)

		By("Making changes in profile")
		hpSize := performancev1alpha1.HugePageSize("2M")
		isolated := performancev1alpha1.CPUSet("1-2")
		reserved := performancev1alpha1.CPUSet("0,3")
		policy := "best-effort"
		f := false

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

		var removedKernelArgs string
		if profile.Spec.AdditionalKernelArgs == nil {
			By("AdditionalKernelArgs is empty. Checking only adding new arguments")
			profile.Spec.AdditionalKernelArgs = append(profile.Spec.AdditionalKernelArgs, "new-argument=test")
		} else {
			removedKernelArgs = profile.Spec.AdditionalKernelArgs[0]
			profile.Spec.AdditionalKernelArgs = append(profile.Spec.AdditionalKernelArgs[1:], "new-argument=test")
		}

		By("Applying changes in performance profile and waiting until mcp will start updating")
		Expect(testclient.Client.Update(context.TODO(), profile)).ToNot(HaveOccurred())
		waitForMcpCondition(testutils.RoleWorkerRT, machineconfigv1.MachineConfigPoolUpdating, corev1.ConditionTrue, timeout)

		By("Waiting when mcp finishes updates and verifying new parameters applied")
		waitForMcpCondition(testutils.RoleWorkerRT, machineconfigv1.MachineConfigPoolUpdated, corev1.ConditionTrue, timeout)

		for _, node := range workerRTNodes {
			checkCmdLineArgs(node, newCmdLineArgs)

			cmd := []string{"cat", "/rootfs/etc/kubernetes/kubelet.conf"}
			Expect(execCommandOnWorker(cmd, &node)).To(ContainSubstring(fmt.Sprintf(`"reservedSystemCPUs":"%s"`, reserved)))
			Expect(execCommandOnWorker(cmd, &node)).To(ContainSubstring(fmt.Sprintf(`"topologyManagerPolicy":"%s"`, policy)))

			cmd = []string{"uname", "-a"}
			Expect(execCommandOnWorker(cmd, &node)).NotTo(ContainSubstring("PREEMPT RT"), "Node should have non-RT kernel")

			if removedKernelArgs != "" {
				cmd = []string{"cat", "/proc/cmdline"}
				Expect(execCommandOnWorker(cmd, &node)).NotTo(ContainSubstring(removedKernelArgs), fmt.Sprintf("%s should be removed from /proc/cmdline", removedKernelArgs))
			}
		}
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

		cmd := []string{"cat", "/rootfs/etc/kubernetes/kubelet.conf"}
		Expect(execCommandOnWorker(cmd, newWorkerNode)).To(ContainSubstring("topologyManagerPolicy"))

		cmd = []string{"cat", "/proc/cmdline"}
		Expect(execCommandOnWorker(cmd, newWorkerNode)).To(ContainSubstring("tuned.non_isolcpus"))
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

func checkCmdLineArgs(node corev1.Node, cmdLineArgs map[string]string) {
	out, err := nodes.ExecCommandOnMachineConfigDaemon(testclient.Client, &node, []string{"cat", "/proc/cmdline"})
	Expect(err).ToNot(HaveOccurred())
	allArgs := strings.Split(strings.TrimSpace(string(out)), " ")
	argsMap := make(map[string]string)
	for _, arg := range allArgs {
		if strings.Contains(arg, "=") {
			parts := strings.Split(arg, "=")
			argsMap[parts[0]] = parts[1]
		} else {
			argsMap[arg] = ""
		}
	}
	for param, expected := range cmdLineArgs {
		Expect(argsMap[param]).To(Equal(expected), fmt.Sprintf("parameter %s value is not %s.", param, expected))
	}
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
