package __performance

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	kubefeatures "k8s.io/kubernetes/pkg/features"

	testutils "github.com/openshift-kni/performance-addon-operators/functests/utils"
	testlog "github.com/openshift-kni/performance-addon-operators/functests/utils/log"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/nodes"
)

var _ = Describe("Kubelet configuration", func() {
	var workerRTNodes []corev1.Node

	BeforeEach(func() {
		var err error
		workerRTNodes, err = nodes.GetByLabels(testutils.NodeSelectorLabels)
		Expect(err).ToNot(HaveOccurred())
		workerRTNodes, err = nodes.MatchingOptionalSelector(workerRTNodes)
		Expect(err).ToNot(HaveOccurred(), "Error looking for the optional selector: %v", err)
		Expect(workerRTNodes).ToNot(BeEmpty(), "No RT worker node found!")
	})

	It("should have enabled the podresources GetAllocatable API", func() {
		kubeletConfig, err := nodes.GetKubeletConfig(&workerRTNodes[0])
		Expect(err).ToNot(HaveOccurred())

		testlog.Infof("kubelet config for %q: %#v", workerRTNodes[0].Name, kubeletConfig)

		Expect(kubeletConfig.FeatureGates).ToNot(BeNil(), "no feature gates enabled at all!")
		podresourcesEnabled := kubeletConfig.FeatureGates[string(kubefeatures.KubeletPodResourcesGetAllocatable)]
		Expect(podresourcesEnabled).To(BeTrue(), "podresources feature gate not enabled on node %q", workerRTNodes[0].Name)
	})
})
