package __performance_update

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	performancev2 "github.com/openshift-kni/performance-addon-operators/api/v2"
	testutils "github.com/openshift-kni/performance-addon-operators/functests/utils"
	testclient "github.com/openshift-kni/performance-addon-operators/functests/utils/client"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/cluster"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/mcps"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/nodes"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/pods"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/profiles"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"
	componentprofile "github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components/profile"
	machineconfigv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	hugepagesResourceName2Mi = "hugepages-2Mi"
	mediumHugepages2Mi       = "HugePages-2Mi"
)

var RunningOnSingleNode bool

var _ = Describe("[ref_id: OCP-43186][pao] Memory Manager tests", func() {
	var profile, initialProfile *performancev2.PerformanceProfile
	var err error

	testutils.BeforeAll(func() {
		isSNO, err := cluster.IsSingleNode()
		Expect(err).ToNot(HaveOccurred())
		RunningOnSingleNode = isSNO
	})
	profile, err = profiles.GetByNodeLabels(testutils.NodeSelectorLabels)
	Expect(err).ToNot(HaveOccurred())
	By("Getting MCP for profile")
	mcpLabel := componentprofile.GetMachineConfigLabel(profile)
	key, value := components.GetFirstKeyAndValue(mcpLabel)
	mcpsByLabel, err := mcps.GetByLabel(key, value)
	Expect(err).ToNot(HaveOccurred(), "Failed getting MCP by label key %v value %v", key, value)
	Expect(len(mcpsByLabel)).To(Equal(1), fmt.Sprintf("Unexpected number of MCPs found: %v", len(mcpsByLabel)))
	performanceMCP := &mcpsByLabel[0]

	Context("Use case: Create a group containing both NUMA nodes", func() {
		var workerRTNodes []corev1.Node
		var err error
		workerRTNodes, err = nodes.GetByLabels(testutils.NodeSelectorLabels)
		workerRTNodes, err = nodes.MatchingOptionalSelector(workerRTNodes)
		Expect(err).ToNot(HaveOccurred())
		hpSize2M := performancev2.HugePageSize("2M")
		hpSize1G := performancev2.HugePageSize("1G")
		policy := "restricted"
		//Modifying profile and verify that MCO successfully updated the node
		testutils.BeforeAll(func() {
			By("Modifying profile")
			initialProfile = profile.DeepCopy()
			profile.Spec.HugePages = &performancev2.HugePages{
				DefaultHugePagesSize: &hpSize1G,
				Pages: []performancev2.HugePage{
					{
						Size:  hpSize2M,
						Count: 20,
					},
				},
			}
			profile.Spec.NUMA = &performancev2.NUMA{
				TopologyPolicy: &policy,
			}
			By("Verifying that mcp is ready for update")
			mcps.WaitForCondition(performanceMCP.Name, machineconfigv1.MachineConfigPoolUpdated, corev1.ConditionTrue)

			By("Applying changes in performance profile and waiting until mcp will start updating")
			profiles.UpdateWithRetry(profile)
			mcps.WaitForCondition(performanceMCP.Name, machineconfigv1.MachineConfigPoolUpdating, corev1.ConditionTrue)

			By("Waiting when mcp finishes updates")
			mcps.WaitForCondition(performanceMCP.Name, machineconfigv1.MachineConfigPoolUpdated, corev1.ConditionTrue)
		})
		It("[test_id: OCP-44245] Deploy Pod1", func() {

			testpod := &pods.MMPod{}
			testpod.DefaultPod()
			mmPod := pods.MMPodTemplate(testpod, &workerRTNodes[0])
			err = testclient.Client.Create(context.TODO(), mmPod)
			Expect(err).ToNot(HaveOccurred())
			err = pods.WaitForCondition(mmPod, corev1.PodReady, corev1.ConditionTrue, 10*time.Minute)
			Expect(err).ToNot(HaveOccurred())
			By("Getting the container cgroup")
			containerID, err := pods.GetContainerIDByName(mmPod, testpod.CtnName)
			Expect(err).ToNot(HaveOccurred())
			containerCgroup := ""
			Eventually(func() string {
				cmd := []string{"/bin/bash", "-c", fmt.Sprintf("find /rootfs/sys/fs/cgroup/cpuset/ -name *%s*", containerID)}
				containerCgroup, err = nodes.ExecCommandOnNode(cmd, &workerRTNodes[0])
				Expect(err).ToNot(HaveOccurred())
				return containerCgroup
			}, cluster.ComputeTestTimeout(30*time.Second, RunningOnSingleNode), 5*time.Second).ShouldNot(BeEmpty(),
				fmt.Sprintf("cannot find cgroup for container %q", containerID))
			By("Checking what memory the pod is using")
			cmd := []string{"/bin/bash", "-c", fmt.Sprintf("cat %s/cpuset.mems", containerCgroup)}
			output, err := nodes.ExecCommandOnNode(cmd, &workerRTNodes[0])
			Expect(err).ToNot(HaveOccurred())
			Expect(output).To(Equal("0-0"))
		})
		It("Reverts back all profile configuration", func() {
			spec, err := json.Marshal(initialProfile.Spec)
			Expect(err).ToNot(HaveOccurred())
			Expect(testclient.Client.Patch(context.TODO(), profile,
				client.RawPatch(
					types.JSONPatchType,
					[]byte(fmt.Sprintf(`[{"op": "replace", "path": "/spec", "value": %s }]`, spec)),
				),
			)).ToNot(HaveOccurred())
			mcps.WaitForCondition(performanceMCP.Name, machineconfigv1.MachineConfigPoolUpdating, corev1.ConditionTrue)
			mcps.WaitForCondition(performanceMCP.Name, machineconfigv1.MachineConfigPoolUpdated, corev1.ConditionTrue)
		})
	})
})
