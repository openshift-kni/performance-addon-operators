package __performance_status

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	ign2types "github.com/coreos/ignition/config/v2_2/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	tunedv1 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/tuned/v1"
	v1 "github.com/openshift/custom-resource-status/conditions/v1"
	machineconfigv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	"k8s.io/apimachinery/pkg/runtime"

	performancev1 "github.com/openshift-kni/performance-addon-operators/api/v1"
	testutils "github.com/openshift-kni/performance-addon-operators/functests/utils"
	testclient "github.com/openshift-kni/performance-addon-operators/functests/utils/client"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/discovery"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/mcps"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/nodes"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/profiles"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"

	corev1 "k8s.io/api/core/v1"
	nodev1beta1 "k8s.io/api/node/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/pointer"
)

var _ = Describe("Status testing of performance profile", func() {
	var workerCNFNodes []corev1.Node
	var err error

	BeforeEach(func() {
		if discovery.Enabled() && testutils.ProfileNotFound {
			Skip("Discovery mode enabled, performance profile not found")
		}
		workerCNFNodes, err = nodes.GetByLabels(testutils.NodeSelectorLabels)
		Expect(err).ToNot(HaveOccurred())
		workerCNFNodes, err = nodes.MatchingOptionalSelector(workerCNFNodes)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("error looking for the optional selector: %v", err))
		Expect(workerCNFNodes).ToNot(BeEmpty())
	})

	Context("[rfe_id:28881][performance] Performance Addons detailed status", func() {

		It("[test_id:30894] Tuned status field tied to Performance Profile", func() {
			profile, err := profiles.GetByNodeLabels(testutils.NodeSelectorLabels)
			Expect(err).ToNot(HaveOccurred())
			key := types.NamespacedName{
				Name:      components.GetComponentName(profile.Name, components.ProfileNamePerformance),
				Namespace: components.NamespaceNodeTuningOperator,
			}
			tuned := &tunedv1.Tuned{}
			err = testclient.GetWithRetry(context.TODO(), key, tuned)
			Expect(err).ToNot(HaveOccurred(), "cannot find the Cluster Node Tuning Operator Tuned object "+key.String())
			tunedNamespacedname := types.NamespacedName{
				Name:      components.GetComponentName(profile.Name, components.ProfileNamePerformance),
				Namespace: components.NamespaceNodeTuningOperator,
			}
			tunedStatus := tunedNamespacedname.String()
			Expect(profile.Status.Tuned).NotTo(BeNil())
			Expect(*profile.Status.Tuned).To(Equal(tunedStatus))
		})

		It("[test_id:33791] Should include the generated runtime class name", func() {
			profile, err := profiles.GetByNodeLabels(testutils.NodeSelectorLabels)
			Expect(err).ToNot(HaveOccurred())

			key := types.NamespacedName{
				Name:      components.GetComponentName(profile.Name, components.ComponentNamePrefix),
				Namespace: metav1.NamespaceAll,
			}
			runtimeClass := &nodev1beta1.RuntimeClass{}
			err = testclient.GetWithRetry(context.TODO(), key, runtimeClass)
			Expect(err).ToNot(HaveOccurred(), "cannot find the RuntimeClass object "+key.String())

			Expect(profile.Status.RuntimeClass).NotTo(BeNil())
			Expect(*profile.Status.RuntimeClass).To(Equal(runtimeClass.Name))
		})

		It("[test_id:29673] Machine config pools status tied to Performance Profile", func() {
			// Creating bad MC that leads to degraded state
			By("Creating bad MachineConfig")
			badMC := createBadMachineConfig("bad-mc")
			err = testclient.Client.Create(context.TODO(), badMC)
			Expect(err).ToNot(HaveOccurred())

			By("Wait for MCP condition to be Degraded")
			profile, err := profiles.GetByNodeLabels(testutils.NodeSelectorLabels)
			Expect(err).ToNot(HaveOccurred())
			performanceMCP, err := mcps.GetByProfile(profile)
			Expect(err).ToNot(HaveOccurred())
			mcps.WaitForCondition(performanceMCP, machineconfigv1.MachineConfigPoolDegraded, corev1.ConditionTrue)
			mcpConditionReason := mcps.GetConditionReason(performanceMCP, machineconfigv1.MachineConfigPoolDegraded)
			profileConditionMessage := profiles.GetConditionMessage(testutils.NodeSelectorLabels, v1.ConditionDegraded)
			// Verify the status reason of performance profile
			Expect(profileConditionMessage).To(ContainSubstring(mcpConditionReason))

			By("Deleting bad MachineConfig and waiting when Degraded state is removed")
			err = testclient.Client.Delete(context.TODO(), badMC)
			Expect(err).ToNot(HaveOccurred())

			mcps.WaitForCondition(performanceMCP, machineconfigv1.MachineConfigPoolUpdated, corev1.ConditionTrue)
		})
	})

	Context("Status reports degraded condition", func() {
		It("Should report Degraded status if overlapping cpus are configured", func() {
			if discovery.Enabled() {
				Skip("Discovery mode enabled, test skipped because it creates incorrect profiles")
			}

			newRole := "worker-overlapping"
			newLabel := fmt.Sprintf("%s/%s", testutils.LabelRole, newRole)

			reserved := performancev1.CPUSet("0-3")
			isolated := performancev1.CPUSet("0-7")

			overlappingProfile := &performancev1.PerformanceProfile{
				TypeMeta: metav1.TypeMeta{
					Kind:       "PerformanceProfile",
					APIVersion: performancev1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "profile-overlapping-cpus",
				},
				Spec: performancev1.PerformanceProfileSpec{
					CPU: &performancev1.CPU{
						Reserved: &reserved,
						Isolated: &isolated,
					},
					NodeSelector: map[string]string{newLabel: ""},
					RealTimeKernel: &performancev1.RealTimeKernel{
						Enabled: pointer.BoolPtr(true),
					},
					NUMA: &performancev1.NUMA{
						TopologyPolicy: pointer.StringPtr("restricted"),
					},
				},
			}
			err := testclient.Client.Create(context.TODO(), overlappingProfile)
			Expect(err).ToNot(HaveOccurred(), "error creating overlappingProfile: %v", err)
			defer func() {
				Expect(testclient.Client.Delete(context.TODO(), overlappingProfile)).ToNot(HaveOccurred())
				Expect(profiles.WaitForDeletion(overlappingProfile, 60*time.Second)).ToNot(HaveOccurred())

				Consistently(func() corev1.ConditionStatus {
					return mcps.GetConditionStatus(testutils.RoleWorkerCNF, machineconfigv1.MachineConfigPoolUpdating)
				}, 30, 5).Should(Equal(corev1.ConditionFalse), "Machine Config Pool is updating, and it should not")

			}()

			nodeLabels := map[string]string{
				newLabel: "",
			}

			cond := profiles.GetConditionWithStatus(nodeLabels, v1.ConditionDegraded)
			Expect(cond.Message).To(ContainSubstring("reserved and isolated cpus overlap"), "Profile condition degraded unexpected status: %q")
		})
	})
})

func createBadMachineConfig(name string) *machineconfigv1.MachineConfig {
	rawIgnition, _ := json.Marshal(
		&ign2types.Config{
			Ignition: ign2types.Ignition{
				Version: ign2types.MaxVersion.String(),
			},
			Storage: ign2types.Storage{
				Disks: []ign2types.Disk{
					{
						Device: "/one",
					},
				},
			},
		},
	)

	return &machineconfigv1.MachineConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: machineconfigv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{"machineconfiguration.openshift.io/role": testutils.RoleWorkerCNF},
			UID:    types.UID(utilrand.String(5)),
		},
		Spec: machineconfigv1.MachineConfigSpec{
			OSImageURL: "",
			Config: runtime.RawExtension{
				Raw: rawIgnition,
			},
		},
	}
}
