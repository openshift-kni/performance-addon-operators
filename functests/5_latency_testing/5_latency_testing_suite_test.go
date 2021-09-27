package __latency_testing_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openshift-kni/performance-addon-operators/functests/utils/junit"
	ginkgo_reporters "kubevirt.io/qe-tools/pkg/ginkgo-reporters"
)

func Test5LatencyTesting(t *testing.T) {
	RegisterFailHandler(Fail)

	rr := []Reporter{}
	if ginkgo_reporters.Polarion.Run {
		rr = append(rr, &ginkgo_reporters.Polarion)
	}
	rr = append(rr, junit.NewJUnitReporter("latency_testing"))
	RunSpecsWithDefaultAndCustomReporters(t, "Performance Addon Operator latency tools testing", rr)
}
