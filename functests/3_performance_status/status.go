package __performance_status

import (
	"context"
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	testutils "github.com/openshift-kni/performance-addon-operators/functests/utils"
	testclient "github.com/openshift-kni/performance-addon-operators/functests/utils/client"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/mcps"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/nodes"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/profiles"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"
	tunedv1 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/tuned/v1"
	v1 "github.com/openshift/custom-resource-status/conditions/v1"
	machineconfigv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Status testing of performance profile", func() {
	var workerCNFNodes []corev1.Node
	var err error

	BeforeEach(func() {
		workerCNFNodes, err = nodes.GetByRole(testutils.RoleWorkerCNF)
		Expect(err).ToNot(HaveOccurred())
		Expect(workerCNFNodes).ToNot(BeEmpty())
	})

	Context("[rfe_id:28881][performance] Performance Addons detailed status", func() {
		var currentConfig string
		nodeAnnotationCurrentConfig := "machineconfiguration.openshift.io/currentConfig"
		nodeAnnotationDesiredConfig := "machineconfiguration.openshift.io/desiredConfig"
		nodeLabel := map[string]string{fmt.Sprintf("%s/%s", testutils.LabelRole, testutils.RoleWorkerCNF): ""}

		It("[test_id:30894] Tuned status field tied to Performance Profile", func() {
			profile, err := profiles.GetByNodeLabels(
				map[string]string{
					fmt.Sprintf("%s/%s", testutils.LabelRole, testutils.RoleWorkerCNF): "",
				},
			)
			Expect(err).ToNot(HaveOccurred())
			key := types.NamespacedName{
				Name:      components.GetComponentName(testutils.PerformanceProfileName, components.ProfileNamePerformance),
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

		It("[test_id:29673] Machine config pools status tied to Performance Profile", func() {
			node := &workerCNFNodes[0]
			annotations := node.GetAnnotations()
			for k, v := range annotations {
				if k == nodeAnnotationCurrentConfig {
					currentConfig = v
				}
			}

			currentConfigAnnotation := map[string]string{
				nodeAnnotationCurrentConfig: currentConfig,
				nodeAnnotationDesiredConfig: currentConfig,
			}
			updateAnnotation := map[string]string{
				nodeAnnotationCurrentConfig: "",
				nodeAnnotationDesiredConfig: currentConfig,
			}

			// Empty the value of "machineconfiguration.openshift.io/currentConfig" for node with worker-cnf label
			By("Apply changes in machineconfiguration currentConfig of CNF worker node")
			annotate, err := json.Marshal(updateAnnotation)
			Expect(err).ToNot(HaveOccurred())
			Expect(testclient.Client.Patch(context.TODO(), node,
				client.ConstantPatch(
					types.JSONPatchType,
					[]byte(fmt.Sprintf(`[{ "op": "replace", "path": "/metadata/annotations", "value": %s }]`, annotate)),
				),
			)).ToNot(HaveOccurred())
			// Wait until worker-cnf MCP is in degraded state and get condition reason
			By("Wait for MCP condition to be Degraded")
			mcps.WaitForCondition(testutils.RoleWorkerCNF, machineconfigv1.MachineConfigPoolDegraded, corev1.ConditionTrue)
			mcpConditionReason := mcps.GetConditionReason(testutils.RoleWorkerCNF, machineconfigv1.MachineConfigPoolDegraded)
			profileConditionMessage := profiles.GetConditionMessage(nodeLabel, v1.ConditionDegraded)
			// Verify the status reason of performance profile
			Expect(profileConditionMessage).To(ContainSubstring(mcpConditionReason))
			// Revert back the currentConfig
			By("Revert changes in machineconfiguration currentConfig of CNF worker node")
			revertAnnotate, er := json.Marshal(currentConfigAnnotation)
			Expect(er).ToNot(HaveOccurred())
			Expect(testclient.Client.Patch(context.TODO(), node,
				client.ConstantPatch(
					types.JSONPatchType,
					[]byte(fmt.Sprintf(`[{ "op": "replace", "path": "/metadata/annotations", "value": %s }]`, revertAnnotate)),
				),
			)).ToNot(HaveOccurred())
			mcps.WaitForCondition(testutils.RoleWorkerCNF, machineconfigv1.MachineConfigPoolUpdated, corev1.ConditionTrue)
		})
	})
})
