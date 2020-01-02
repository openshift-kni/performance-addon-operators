// +build !unittests

package test_test

import (
	"flag"
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	_ "github.com/openshift-kni/performance-addon-operators/functests/performance" // this is needed otherwise the performance test won't be executed
	testutils "github.com/openshift-kni/performance-addon-operators/functests/utils"
	testclient "github.com/openshift-kni/performance-addon-operators/functests/utils/client"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/machineconfigpool"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/namespaces"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/nodes"
	mcov1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TODO: we should refactor tests to use client from controller-runtime package
// see - https://github.com/openshift/cluster-api-actuator-pkg/blob/master/pkg/e2e/framework/framework.go

var junitPath *string

func init() {
	junitPath = flag.String("junit", "junit.xml", "the path for the junit format report")
}

func TestTest(t *testing.T) {
	RegisterFailHandler(Fail)

	rr := []Reporter{}
	if junitPath != nil {
		rr = append(rr, reporters.NewJUnitReporter(*junitPath))
	}
	RunSpecsWithDefaultAndCustomReporters(t, "Performance Addon Operators e2e tests", rr)
}

var _ = BeforeSuite(func() {
	// create test namespace
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testutils.NamespaceTesting,
		},
	}
	_, err := testclient.Client.Namespaces().Create(ns)
	Expect(err).ToNot(HaveOccurred())

	// lable one of workers nodes with worker RT label
	workerNodes, err := nodes.GetByRole(testclient.Client, testutils.RoleWorker)
	Expect(err).ToNot(HaveOccurred())
	Expect(workerNodes).ToNot(BeEmpty())

	worker := &workerNodes[0]
	workerRTLabel := fmt.Sprintf("%s/%s", testutils.LabelRole, testutils.RoleWorkerRT)
	worker.Labels[workerRTLabel] = ""
	_, err = testclient.Client.Nodes().Update(worker)
	Expect(err).ToNot(HaveOccurred())

	// wait for all machine config pools to finish update
	mcps, err := testclient.Client.MachineConfigPools().List(metav1.ListOptions{})
	Expect(err).ToNot(HaveOccurred())

	for _, mcp := range mcps.Items {
		err := machineconfigpool.WaitForCondition(
			testclient.Client,
			&mcp,
			mcov1.MachineConfigPoolUpdated,
			corev1.ConditionTrue,
			25*time.Minute,
		)
		Expect(err).ToNot(HaveOccurred())
	}
})

var _ = AfterSuite(func() {
	err := testclient.Client.Namespaces().Delete(testutils.NamespaceTesting, &metav1.DeleteOptions{})
	Expect(err).ToNot(HaveOccurred())
	err = namespaces.WaitForDeletion(testclient.Client, testutils.NamespaceTesting, 5*time.Minute)
})
