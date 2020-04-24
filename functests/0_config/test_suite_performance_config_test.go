// +build !unittests

package __performance_config_test

import (
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	machineconfigv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	corev1 "k8s.io/api/core/v1"

	ginkgo_reporters "kubevirt.io/qe-tools/pkg/ginkgo-reporters"

	testutils "github.com/openshift-kni/performance-addon-operators/functests/utils"
	testclient "github.com/openshift-kni/performance-addon-operators/functests/utils/client"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/junit"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/mcps"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/profiles"
	performancev1alpha1 "github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v1alpha1"
)

var _ = BeforeSuite(func() {
	// Creating Performance Profile
	profile := createPerformanceProfile()
	Expect(testclient.Client.Create(context.TODO(), profile)).ToNot(HaveOccurred())

	// Waiting when MCP finished updates
	mcps.WaitForCondition(testutils.RoleWorkerRT, machineconfigv1.MachineConfigPoolUpdating, corev1.ConditionTrue, testutils.McpUpdateTimeout)
	mcps.WaitForCondition(testutils.RoleWorkerRT, machineconfigv1.MachineConfigPoolUpdated, corev1.ConditionTrue, testutils.McpUpdateTimeout)
})

var _ = AfterSuite(func() {

})

func TestPerformanceConfig(t *testing.T) {
	RegisterFailHandler(Fail)

	rr := []Reporter{}
	if ginkgo_reporters.Polarion.Run {
		rr = append(rr, &ginkgo_reporters.Polarion)
	}
	rr = append(rr, junit.NewJUnitReporter("performance_config"))
	RunSpecsWithDefaultAndCustomReporters(t, "Performance Addon Operator - profile configuration", rr)
}

func createPerformanceProfile() *performancev1alpha1.PerformanceProfile {
	hpSize := performancev1alpha1.HugePageSize("1G")
	hpCount := int32(1)
	hpNode := int32(0)

	isolated := performancev1alpha1.CPUSet("1-3")
	reserved := performancev1alpha1.CPUSet("0")
	policy := "single-numa-node"
	t := true

	kernelArgs := []string{
		"nmi_watchdog=0",
		"audit=0",
		"mce=off",
		"processor.max_cstate=1",
		"idle=poll",
		"intel_idle.max_cstate=0",
	}

	spec := performancev1alpha1.PerformanceProfileSpec{
		CPU: &performancev1alpha1.CPU{
			Reserved: &reserved,
			Isolated: &isolated,
		},
		HugePages: &performancev1alpha1.HugePages{
			DefaultHugePagesSize: &hpSize,
			Pages: []performancev1alpha1.HugePage{
				{
					Count: hpCount,
					Node:  &hpNode,
					Size:  hpSize,
				},
			},
		},
		RealTimeKernel: &performancev1alpha1.RealTimeKernel{
			Enabled: &t,
		},
		NUMA: &performancev1alpha1.NUMA{
			TopologyPolicy: &policy,
		},
		NodeSelector:         map[string]string{fmt.Sprintf("%s/%s", testutils.LabelRole, testutils.RoleWorkerRT): ""},
		AdditionalKernelArgs: kernelArgs,
	}
	return profiles.NewProfile(testutils.ProfileName, spec)
}
