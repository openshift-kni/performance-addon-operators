package clean

import (
	"context"
	"fmt"
	"os"

	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components/profile"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/performance-addon-operators/functests/utils"
	testclient "github.com/openshift-kni/performance-addon-operators/functests/utils/client"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/mcps"
	performancev2 "github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v2"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
)

var cleanPerformance bool

func init() {
	clean, found := os.LookupEnv("CLEAN_PERFORMANCE_PROFILE")
	if !found || clean != "false" {
		cleanPerformance = true
	}
}

// All deletes any leftovers created when running the performance tests.
func All() {
	if !cleanPerformance {
		klog.Info("Performance cleaning disabled, skipping")
		return
	}

	perfProfile := performancev2.PerformanceProfile{}
	err := testclient.Client.Get(context.TODO(), types.NamespacedName{Name: utils.PerformanceProfileName}, &perfProfile)
	if errors.IsNotFound(err) {
		return
	}
	Expect(err).ToNot(HaveOccurred(), "Failed to find perf profile")
	err = testclient.Client.Delete(context.TODO(), &perfProfile)
	Expect(err).ToNot(HaveOccurred(), "Failed to delete perf profile")

	mcpLabel := profile.GetMachineConfigLabel(&perfProfile)
	key, value := components.GetFirstKeyAndValue(mcpLabel)
	mcpsByLabel, err := mcps.GetByLabel(key, value)
	Expect(err).ToNot(HaveOccurred(), "Failed getting MCP")
	Expect(len(mcpsByLabel)).To(Equal(1), fmt.Sprintf("Unexpected number of MCPs found: %v", len(mcpsByLabel)))

	performanceMCP := &mcpsByLabel[0]
	By("Waiting for MCP starting to update")
	mcps.WaitForCondition(performanceMCP.Name, mcv1.MachineConfigPoolUpdating, corev1.ConditionTrue)

	By("Waiting for MCP being updated")
	mcps.WaitForCondition(performanceMCP.Name, mcv1.MachineConfigPoolUpdated, corev1.ConditionTrue)
}
