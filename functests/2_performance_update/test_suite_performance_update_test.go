// +build !unittests

package __performance_update_test

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	ginkgo_reporters "kubevirt.io/qe-tools/pkg/ginkgo-reporters"

	testutils "github.com/openshift-kni/performance-addon-operators/functests/utils"
	testclient "github.com/openshift-kni/performance-addon-operators/functests/utils/client"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/junit"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/namespaces"
)

var _ = BeforeSuite(func() {
	// create test namespace
	err := testclient.Client.Create(context.TODO(), namespaces.TestingNamespace)
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	err := testclient.Client.Delete(context.TODO(), namespaces.TestingNamespace)
	Expect(err).ToNot(HaveOccurred())
	err = namespaces.WaitForDeletion(testclient.Client, testutils.NamespaceTesting, 5*time.Minute)
})

func TestPerformanceUpdate(t *testing.T) {
	RegisterFailHandler(Fail)

	rr := []Reporter{}
	if ginkgo_reporters.Polarion.Run {
		rr = append(rr, &ginkgo_reporters.Polarion)
	}
	rr = append(rr, junit.NewJUnitReporter("performance_update"))
	RunSpecsWithDefaultAndCustomReporters(t, "Performance Addon Operator Update e2e tests", rr)
}
