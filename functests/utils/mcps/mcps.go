package mcps

import (
	"context"
	"time"

	. "github.com/onsi/gomega"
	machineconfigv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	testclient "github.com/openshift-kni/performance-addon-operators/functests/utils/client"
)

// Wait for specific MCP condition
func WaitForCondition(mcpName string, conditionType machineconfigv1.MachineConfigPoolConditionType, conditionStatus corev1.ConditionStatus, timeout time.Duration) {
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
