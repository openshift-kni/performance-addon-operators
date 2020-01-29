package performance

import (
	"fmt"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	testutils "github.com/openshift-kni/performance-addon-operators/functests/utils"
	testclient "github.com/openshift-kni/performance-addon-operators/functests/utils/client"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/nodes"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/profiles"
	performancev1alpha1 "github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v1alpha1"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components/machineconfig"

	corev1 "k8s.io/api/core/v1"
)

const (
	pathHugepages1048576kBNumaNode0 = "/sys/devices/system/node/node0/hugepages/hugepages-1048576kB/nr_hugepages"
	pathHugepages2048kB             = "/sys/kernel/mm/hugepages/hugepages-2048kB/nr_hugepages"
)

var _ = Describe("[performance]Hugepages", func() {
	var workerRTNode *corev1.Node
	var profile *performancev1alpha1.PerformanceProfile

	BeforeEach(func() {
		var err error
		workerRTNodes, err := nodes.GetByRole(testclient.Client, testutils.RoleWorkerRT)
		Expect(err).ToNot(HaveOccurred())
		Expect(workerRTNodes).ToNot(BeEmpty())
		workerRTNode = &workerRTNodes[0]

		profile, err = profiles.GetByNodeLabels(
			testclient.Client,
			map[string]string{
				fmt.Sprintf("%s/%s", testutils.LabelRole, testutils.RoleWorkerRT): "",
			},
		)
		Expect(err).ToNot(HaveOccurred())
		Expect(profile.Spec.HugePages).ToNot(BeNil())
	})

	Context("when NUMA node specified", func() {
		It("should be allocated on the specifed NUMA node ", func() {
			for _, page := range profile.Spec.HugePages.Pages {
				if page.Node == nil {
					continue
				}

				hugepagesSize, err := machineconfig.GetHugepagesSizeKilobytes(page.Size)
				Expect(err).ToNot(HaveOccurred())

				hugepagesFile := fmt.Sprintf("/sys/devices/system/node/node%d/hugepages/hugepages-%skB/nr_hugepages", *page.Node, hugepagesSize)
				command := []string{"cat", hugepagesFile}
				output, err := nodes.ExecCommandOnMachineConfigDaemon(testclient.Client, workerRTNode, command)
				Expect(err).ToNot(HaveOccurred())

				nrHugepages, err := strconv.Atoi(strings.Trim(string(output), "\n"))
				Expect(err).ToNot(HaveOccurred())

				Expect(int32(nrHugepages)).To(Equal(page.Count))
			}
		})
	})

	// TODO: enable it once https://github.com/kubernetes/kubernetes/pull/84051
	// is available under the openshift
	// Context("when NUMA node unspecified", func() {
	// 	It("should be allocated equally among NUMA nodes", func() {
	// 		command := []string{"cat", pathHugepages2048kB}
	// 		nrHugepages, err := nodes.ExecCommandOnMachineConfigDaemon(testclient.Client, workerRTNode, command)
	// 		Expect(err).ToNot(HaveOccurred())
	// 		Expect(string(nrHugepages)).To(Equal("128"))
	// 	})
	// })
})
