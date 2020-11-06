package __performance_update

import (
	"context"
	"encoding/json"
	"fmt"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	machineconfigv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	performancev2 "github.com/openshift-kni/performance-addon-operators/api/v2"
	testutils "github.com/openshift-kni/performance-addon-operators/functests/utils"
	testclient "github.com/openshift-kni/performance-addon-operators/functests/utils/client"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/discovery"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/mcps"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/nodes"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/profiles"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"
)

var _ = Describe("[rfe_id:28761][performance] Updating parameters in performance profile", func() {
	var workerRTNodes []corev1.Node
	var profile, initialProfile *performancev2.PerformanceProfile
	var performanceMCP string
	var err error

	chkCmdLine := []string{"cat", "/proc/cmdline"}
	chkKubeletConfig := []string{"cat", "/rootfs/etc/kubernetes/kubelet.conf"}
	chkIrqbalance := []string{"cat", "/rootfs/etc/sysconfig/irqbalance"}
	chkHp2M := []string{"cat", "/sys/devices/system/node/node0/hugepages/hugepages-2048kB/nr_hugepages"}
	chkHp1G := []string{"cat", "/sys/devices/system/node/node0/hugepages/hugepages-1048576kB/nr_hugepages"}

	nodeLabel := testutils.NodeSelectorLabels

	BeforeEach(func() {
		if discovery.Enabled() && testutils.ProfileNotFound {
			Skip("Discovery mode enabled, performance profile not found")
		}

		workerRTNodes, err = nodes.GetByLabels(testutils.NodeSelectorLabels)
		Expect(err).ToNot(HaveOccurred())
		workerRTNodes, err = nodes.MatchingOptionalSelector(workerRTNodes)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("error looking for the optional selector: %v", err))
		Expect(workerRTNodes).ToNot(BeEmpty())
		profile, err = profiles.GetByNodeLabels(nodeLabel)
		Expect(err).ToNot(HaveOccurred())
		performanceMCP, err = mcps.GetByProfile(profile)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("Verify GloballyDisableIrqLoadBalancing Spec field", func() {
		It("[test_id:36150] Verify that IRQ load balancing is enabled/disabled correctly", func() {
			irqLoadBalancingDisabled := profile.Spec.GloballyDisableIrqLoadBalancing != nil && *profile.Spec.GloballyDisableIrqLoadBalancing

			Expect(profile.Spec.CPU.Isolated).NotTo(BeNil())
			isolatedCPUSet, err := cpuset.Parse(string(*profile.Spec.CPU.Isolated))
			Expect(err).ToNot(HaveOccurred())

			var expectedBannedCPUs cpuset.CPUSet
			if irqLoadBalancingDisabled {
				expectedBannedCPUs = isolatedCPUSet
			} else {
				expectedBannedCPUs = cpuset.NewCPUSet()
			}

			for _, node := range workerRTNodes {
				bannedCPUs, err := nodes.BannedCPUs(node)
				Expect(err).ToNot(HaveOccurred(), fmt.Errorf("failed to extract the banned CPUs from node %s: %v", node.Name, err))
				Expect(bannedCPUs.Equals(expectedBannedCPUs)).To(BeTrue(),
					fmt.Errorf("banned CPUs %v do not match the expected mask %v on node %s",
						bannedCPUs, expectedBannedCPUs, node.Name))
			}

			By("Modifying profile")
			initialProfile = profile.DeepCopy()

			irqLoadBalancingDisabled = !irqLoadBalancingDisabled
			if irqLoadBalancingDisabled {
				expectedBannedCPUs = isolatedCPUSet
			} else {
				expectedBannedCPUs = cpuset.NewCPUSet()
			}
			profile.Spec.GloballyDisableIrqLoadBalancing = &irqLoadBalancingDisabled

			By("Updating the performance profile")
			Expect(testclient.Client.Update(context.TODO(), profile)).ToNot(HaveOccurred())
			defer func() { // return initial configuration
				spec, err := json.Marshal(initialProfile.Spec)
				Expect(err).ToNot(HaveOccurred())
				Expect(testclient.Client.Patch(context.TODO(), profile,
					client.RawPatch(
						types.JSONPatchType,
						[]byte(fmt.Sprintf(`[{ "op": "replace", "path": "/spec", "value": %s }]`, spec)),
					),
				)).ToNot(HaveOccurred())
			}()

			Eventually(func() error {
				for _, node := range workerRTNodes {
					bannedCPUs, err := nodes.BannedCPUs(node)
					if err != nil {
						return fmt.Errorf("failed to extract the banned CPUs from node %s: %v", node.Name, err)
					}
					if !bannedCPUs.Equals(expectedBannedCPUs) {
						return fmt.Errorf("banned CPUs %v do not match the expected mask %v on node %s",
							bannedCPUs, expectedBannedCPUs, node.Name)
					}
				}
				return nil
			}, 1*time.Minute, 10*time.Second).ShouldNot(HaveOccurred())

		})
	})

	Context("Verify that all performance profile parameters can be updated", func() {
		var removedKernelArgs string

		hpSize2M := performancev2.HugePageSize("2M")
		hpSize1G := performancev2.HugePageSize("1G")
		isolated := performancev2.CPUSet("1-2")
		reserved := performancev2.CPUSet("0,3")
		policy := "best-effort"
		f := false

		// Modify profile and verify that MCO successfully updated the node
		testutils.BeforeAll(func() {
			By("Modifying profile")
			initialProfile = profile.DeepCopy()

			profile.Spec.HugePages = &performancev2.HugePages{
				DefaultHugePagesSize: &hpSize2M,
				Pages: []performancev2.HugePage{
					{
						Count: 256,
						Size:  hpSize2M,
					},
					{
						Count: 3,
						Size:  hpSize1G,
					},
				},
			}
			profile.Spec.CPU = &performancev2.CPU{
				BalanceIsolated: &f,
				Reserved:        &reserved,
				Isolated:        &isolated,
			}
			profile.Spec.NUMA = &performancev2.NUMA{
				TopologyPolicy: &policy,
			}
			profile.Spec.RealTimeKernel = &performancev2.RealTimeKernel{
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
			mcps.WaitForCondition(performanceMCP, machineconfigv1.MachineConfigPoolUpdated, corev1.ConditionTrue)

			By("Applying changes in performance profile and waiting until mcp will start updating")
			Expect(testclient.Client.Update(context.TODO(), profile)).ToNot(HaveOccurred())
			mcps.WaitForCondition(performanceMCP, machineconfigv1.MachineConfigPoolUpdating, corev1.ConditionTrue)

			By("Waiting when mcp finishes updates")
			mcps.WaitForCondition(performanceMCP, machineconfigv1.MachineConfigPoolUpdated, corev1.ConditionTrue)
		})

		table.DescribeTable("Verify that profile parameters were updated", func(cmd, parameter []string, shouldContain bool, useRegex bool) {
			for _, node := range workerRTNodes {
				for _, param := range parameter {
					result, err := nodes.ExecCommandOnNode(cmd, &node)
					Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to execute %s", cmd))

					matcher := ContainSubstring(param)
					if useRegex {
						matcher = MatchRegexp(param)
					}

					if shouldContain {
						Expect(result).To(matcher)
					} else {
						Expect(result).NotTo(matcher)
					}
				}
			}
		},
			table.Entry("[test_id:34081] verify that hugepages size and count updated", chkCmdLine, []string{"default_hugepagesz=2M", "hugepagesz=1G", "hugepages=3"}, true, false),
			table.Entry("[test_id:28070] verify that hugepages updated (NUMA node unspecified)", chkCmdLine, []string{"hugepagesz=2M"}, true, false),
			table.Entry("verify that the right number of hugepages 1G is available on the system", chkHp1G, []string{"3"}, true, false),
			table.Entry("verify that the right number of hugepages 2M is available on the system", chkHp2M, []string{"256"}, true, false),
			table.Entry("[test_id:28025] verify that cpu affinity mask was updated", chkCmdLine, []string{"tuned.non_isolcpus=.*9"}, true, true),
			table.Entry("[test_id:28071] verify that cpu balancer disabled", chkCmdLine, []string{"isolcpus=domain,managed_irq,1-2"}, true, false),
			table.Entry("[test_id:28071] verify that cpu balancer disabled", chkCmdLine, []string{"systemd.cpu_affinity=0,3"}, true, false),
			// kubelet.conf changed formatting, there is a space after colons atm. Let's deal with both cases with a regex
			table.Entry("[test_id:28935] verify that reservedSystemCPUs was updated", chkKubeletConfig, []string{`"reservedSystemCPUs": ?"0,3"`}, true, true),
			table.Entry("[test_id:28760] verify that topologyManager was updated", chkKubeletConfig, []string{`"topologyManagerPolicy": ?"best-effort"`}, true, true),
		)

		It("[test_id:27738] should succeed to disable the RT kernel", func() {
			for _, node := range workerRTNodes {
				err := nodes.HasPreemptRTKernel(&node)
				Expect(err).To(HaveOccurred())
			}
		})

		It("[test_id:28612]Verify that Kernel arguments can me updated (added, removed) thru performance profile", func() {
			for _, node := range workerRTNodes {
				cmdline, err := nodes.ExecCommandOnNode(chkCmdLine, &node)
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to execute %s", chkCmdLine))

				// Verifying that new argument was added
				Expect(cmdline).To(ContainSubstring("new-argument=test"))

				// Verifying that one of old arguments was removed
				if removedKernelArgs != "" {
					Expect(cmdline).NotTo(ContainSubstring(removedKernelArgs), fmt.Sprintf("%s should be removed from /proc/cmdline", removedKernelArgs))
				}
			}
		})

		It("[test_id:22764] verify that by default RT kernel is disabled", func() {
			conditionUpdating := machineconfigv1.MachineConfigPoolUpdating

			if profile.Spec.RealTimeKernel == nil || *profile.Spec.RealTimeKernel.Enabled == true {
				Skip("Skipping test - This test expects RT Kernel to be disabled. Found it to be enabled or nil.")
			}

			By("Applying changes in performance profile")
			profile.Spec.RealTimeKernel = nil
			Expect(testclient.Client.Update(context.TODO(), profile)).ToNot(HaveOccurred())

			Expect(profile.Spec.RealTimeKernel).To(BeNil())
			By("Checking that the updating MCP status will consistently stay false")
			Consistently(func() corev1.ConditionStatus {
				return mcps.GetConditionStatus(performanceMCP, conditionUpdating)
			}, 30, 5).Should(Equal(corev1.ConditionFalse))

			for _, node := range workerRTNodes {
				err := nodes.HasPreemptRTKernel(&node)
				Expect(err).To(HaveOccurred())
			}
		})

		It("Reverts back all profile configuration", func() {
			// return initial configuration
			spec, err := json.Marshal(initialProfile.Spec)
			Expect(err).ToNot(HaveOccurred())
			Expect(testclient.Client.Patch(context.TODO(), profile,
				client.ConstantPatch(
					types.JSONPatchType,
					[]byte(fmt.Sprintf(`[{ "op": "replace", "path": "/spec", "value": %s }]`, spec)),
				),
			)).ToNot(HaveOccurred())
			mcps.WaitForCondition(performanceMCP, machineconfigv1.MachineConfigPoolUpdating, corev1.ConditionTrue)
			mcps.WaitForCondition(performanceMCP, machineconfigv1.MachineConfigPoolUpdated, corev1.ConditionTrue)
		})
	})

	Context("Updating of nodeSelector parameter and node labels", func() {
		var mcp *machineconfigv1.MachineConfigPool
		var newCnfNode *corev1.Node

		newRole := "worker-test"
		newLabel := fmt.Sprintf("%s/%s", testutils.LabelRole, newRole)
		newNodeSelector := map[string]string{newLabel: ""}

		testutils.BeforeAll(func() {
			nonPerformancesWorkers, err := nodes.GetNonPerformancesWorkers(profile.Spec.NodeSelector)
			Expect(err).ToNot(HaveOccurred())
			if len(nonPerformancesWorkers) != 0 {
				newCnfNode = &nonPerformancesWorkers[0]
			}
		})

		JustBeforeEach(func() {
			if newCnfNode == nil {
				Skip("Skipping the test - cluster does not have another available worker node ")
			}
		})

		It("[test_id:28440]Verifies that nodeSelector can be updated in performance profile", func() {
			nodeLabel = newNodeSelector
			newCnfNode.Labels[newLabel] = ""
			Expect(testclient.Client.Update(context.TODO(), newCnfNode)).ToNot(HaveOccurred())

			By("Creating new MachineConfigPool")
			mcp = mcps.New(newRole, newNodeSelector)
			err = testclient.Client.Create(context.TODO(), mcp)
			Expect(err).ToNot(HaveOccurred())

			By("Updating Node Selector performance profile")
			profile.Spec.NodeSelector = newNodeSelector
			Expect(testclient.Client.Update(context.TODO(), profile)).ToNot(HaveOccurred())
			mcps.WaitForCondition(newRole, machineconfigv1.MachineConfigPoolUpdating, corev1.ConditionTrue)

			By("Waiting when MCP finishes updates and verifying new node has updated configuration")
			mcps.WaitForCondition(newRole, machineconfigv1.MachineConfigPoolUpdated, corev1.ConditionTrue)

			kblcfg, err := nodes.ExecCommandOnNode(chkKubeletConfig, newCnfNode)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to execute %s", chkKubeletConfig))
			Expect(kblcfg).To(ContainSubstring("topologyManagerPolicy"))

			cmdline, err := nodes.ExecCommandOnNode(chkCmdLine, newCnfNode)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to execute %s", chkCmdLine))
			Expect(cmdline).To(ContainSubstring("tuned.non_isolcpus"))
		})

		It("[test_id:27484]Verifies that node is reverted to plain worker when the extra labels are removed", func() {
			By("Deleting cnf labels from the node")
			for l := range profile.Spec.NodeSelector {
				delete(newCnfNode.Labels, l)
			}
			label, err := json.Marshal(newCnfNode.Labels)
			Expect(err).ToNot(HaveOccurred())
			Expect(testclient.Client.Patch(context.TODO(), newCnfNode,
				client.ConstantPatch(
					types.JSONPatchType,
					[]byte(fmt.Sprintf(`[{ "op": "replace", "path": "/metadata/labels", "value": %s }]`, label)),
				),
			)).ToNot(HaveOccurred())
			mcps.WaitForCondition(testutils.RoleWorker, machineconfigv1.MachineConfigPoolUpdating, corev1.ConditionTrue)

			By("Waiting when MCP Worker complete updates and verifying that node reverted back configuration")
			mcps.WaitForCondition(testutils.RoleWorker, machineconfigv1.MachineConfigPoolUpdated, corev1.ConditionTrue)

			// Check if node is Ready
			for i := range newCnfNode.Status.Conditions {
				if newCnfNode.Status.Conditions[i].Type == corev1.NodeReady {
					Expect(newCnfNode.Status.Conditions[i].Status).To(Equal(corev1.ConditionTrue))
				}
			}

			// check that the configs reverted
			err = nodes.HasPreemptRTKernel(newCnfNode)
			Expect(err).To(HaveOccurred())

			cmdline, err := nodes.ExecCommandOnNode(chkCmdLine, newCnfNode)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to execute %s", chkCmdLine))
			Expect(cmdline).NotTo(ContainSubstring("tuned.non_isolcpus"))

			kblcfg, err := nodes.ExecCommandOnNode(chkKubeletConfig, newCnfNode)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to execute %s", chkKubeletConfig))
			Expect(kblcfg).NotTo(ContainSubstring("reservedSystemCPUs"))

			Expect(profile.Spec.CPU.Reserved).NotTo(BeNil())
			reservedCPU := string(*profile.Spec.CPU.Reserved)
			cpuMask, err := components.CPUListToHexMask(reservedCPU)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to list in Hex %s", reservedCPU))
			irqBal, err := nodes.ExecCommandOnNode(chkIrqbalance, newCnfNode)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to execute %s", chkIrqbalance))
			Expect(irqBal).NotTo(ContainSubstring(cpuMask))
		})

		It("Reverts back nodeSelector and cleaning up leftovers", func() {
			selectorLabels := []string{}
			for k, v := range testutils.NodeSelectorLabels {
				selectorLabels = append(selectorLabels, fmt.Sprintf(`"%s":"%s"`, k, v))
			}
			nodeSelector := strings.Join(selectorLabels, ",")
			Expect(testclient.Client.Patch(context.TODO(), profile,
				client.ConstantPatch(
					types.JSONPatchType,
					[]byte(fmt.Sprintf(`[{ "op": "replace", "path": "/spec/nodeSelector", "value": {%s} }]`, nodeSelector)),
				),
			)).ToNot(HaveOccurred())

			updatedProfile := &performancev2.PerformanceProfile{}
			Eventually(func() string {
				key := types.NamespacedName{
					Name:      profile.Name,
					Namespace: profile.Namespace,
				}
				Expect(testclient.Client.Get(context.TODO(), key, updatedProfile)).ToNot(HaveOccurred())
				updatedSelectorLabels := []string{}
				for k, v := range updatedProfile.Spec.NodeSelector {
					updatedSelectorLabels = append(updatedSelectorLabels, fmt.Sprintf(`"%s":"%s"`, k, v))
				}
				updatedNodeSelector := strings.Join(updatedSelectorLabels, ",")
				return updatedNodeSelector
			}, 2*time.Minute, 15*time.Second).Should(Equal(nodeSelector))

			performanceMCP, err = mcps.GetByProfile(updatedProfile)
			Expect(err).ToNot(HaveOccurred())
			Expect(testclient.Client.Delete(context.TODO(), mcp)).ToNot(HaveOccurred())
			mcps.WaitForCondition(performanceMCP, machineconfigv1.MachineConfigPoolUpdated, corev1.ConditionTrue)
		})
	})
})
