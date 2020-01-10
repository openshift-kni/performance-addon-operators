package performance

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	testutils "github.com/openshift-kni/performance-addon-operators/functests/utils"
	testclient "github.com/openshift-kni/performance-addon-operators/functests/utils/client"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/pods"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("[performance]RT Kernel", func() {
	var testpod *corev1.Pod

	AfterEach(func() {
		if err := testclient.Client.Pods(testutils.NamespaceTesting).Delete(testpod.Name, &metav1.DeleteOptions{}); err == nil {
			pods.WaitForDeletion(testclient.Client, testpod, 60*time.Second)
		}
	})

	It("should have RT kernel enabled", func() {

		Eventually(func() string {

			// run uname -a in a busybox pod and get logs
			testpod = pods.GetBusybox()
			testpod.Spec.Containers[0].Command = []string{"uname", "-a"}
			testpod.Spec.RestartPolicy = corev1.RestartPolicyNever
			testpod.Spec.NodeSelector = map[string]string{
				fmt.Sprintf("%s/%s", testutils.LabelRole, testutils.RoleWorkerRT): "",
			}

			var err error
			testpod, err = testclient.Client.Pods(testutils.NamespaceTesting).Create(testpod)
			if err != nil {
				return ""
			}

			err = pods.WaitForPhase(testclient.Client, testpod, corev1.PodSucceeded, 60*time.Second)
			if err != nil {
				return ""
			}

			logs, err := pods.GetLogs(testclient.Client, testpod)
			if err != nil {
				return ""
			}

			return logs

		}, 15*time.Minute, 30*time.Second).Should(ContainSubstring("PREEMPT RT"))

	})

})
