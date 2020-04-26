// +build !unittests

package __performance_config_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"

	ginkgo_reporters "kubevirt.io/qe-tools/pkg/ginkgo-reporters"

	"github.com/openshift-kni/performance-addon-operators/functests/utils/junit"
)

func TestPerformanceConfig(t *testing.T) {
	RegisterFailHandler(Fail)

	rr := []Reporter{}
	if ginkgo_reporters.Polarion.Run {
		rr = append(rr, &ginkgo_reporters.Polarion)
	}
	rr = append(rr, junit.NewJUnitReporter("performance_config"))
	RunSpecsWithDefaultAndCustomReporters(t, "Performance Addon Operator configuration", rr)
}
