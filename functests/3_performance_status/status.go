package __performance_status

import (
	"context"
	"encoding/json"
	"fmt"

	ign2types "github.com/coreos/ignition/config/v2_2/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	tunedv1 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/tuned/v1"
	v1 "github.com/openshift/custom-resource-status/conditions/v1"
	machineconfigv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	"k8s.io/apimachinery/pkg/runtime"

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
