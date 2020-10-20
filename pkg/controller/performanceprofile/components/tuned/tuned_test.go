package tuned

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ghodss/yaml"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	performancev2 "github.com/openshift-kni/performance-addon-operators/api/v2"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"
	testutils "github.com/openshift-kni/performance-addon-operators/pkg/utils/testing"

	"k8s.io/utils/pointer"
)

const testAssetsDir = "../../../../../build/assets"
const expectedMatchSelector = `
  - machineConfigLabels:
      mcKey: mcValue
`

var (
	cmdlineCPUsPartitioning            = regexp.MustCompile(`\s*cmdline_cpu_part=\+\s*nohz=on\s+rcu_nocbs=\${isolated_cores}\s+tuned.non_isolcpus=\${not_isolated_cpumask}\s+intel_pstate=disable\s+nosoftlockup\s*`)
	cmdlineRealtimeWithCPUBalancing    = regexp.MustCompile(`\s*cmdline_realtime=\+\s*tsc=nowatchdog\s+intel_iommu=on\s+iommu=pt\s+isolcpus=managed_irq,\${isolated_cores}\s+systemd.cpu_affinity=\${not_isolated_cores_expanded}\s*`)
	cmdlineRealtimeWithoutCPUBalancing = regexp.MustCompile(`\s*cmdline_realtime=\+\s*tsc=nowatchdog\s+intel_iommu=on\s+iommu=pt\s+isolcpus=domain,managed_irq,\${isolated_cores}\s+systemd.cpu_affinity=\${not_isolated_cores_expanded}\s*`)
	cmdlineHugepages                   = regexp.MustCompile(`\s*cmdline_hugepages=\+\s*default_hugepagesz=1G\s+hugepagesz=1G\s+hugepages=4\s*`)
	cmdlineAdditionalArg               = regexp.MustCompile(`\s*cmdline_additionalArg=\+\s*test1=val1\s+test2=val2\s*`)
	cmdlineDummy2MHugePages            = regexp.MustCompile(`\s*cmdline_hugepages=\+\s*default_hugepagesz=1G\s+hugepagesz=1G\s+hugepages=4\s+hugepagesz=2M\s+hugepages=0\s*`)
	cmdlineMultipleHugePages           = regexp.MustCompile(`\s*cmdline_hugepages=\+\s*default_hugepagesz=1G\s+hugepagesz=1G\s+hugepages=4\s+hugepagesz=2M\s+hugepages=128\s*`)
)

var additionalArgs = []string{"test1=val1", "test2=val2"}

var _ = Describe("Tuned", func() {
	var profile *performancev2.PerformanceProfile

	BeforeEach(func() {
		profile = testutils.NewPerformanceProfile("test")
	})

	getTunedManifest := func(profile *performancev2.PerformanceProfile) string {
		tuned, err := NewNodePerformance(testAssetsDir, profile)
		Expect(err).ToNot(HaveOccurred())
		y, err := yaml.Marshal(tuned)
		Expect(err).ToNot(HaveOccurred())
		return string(y)
	}

	Context("with worker performance profile", func() {
		It("should generate yaml with expected parameters", func() {
			manifest := getTunedManifest(profile)

			Expect(manifest).To(ContainSubstring(expectedMatchSelector))
			Expect(manifest).To(ContainSubstring(fmt.Sprintf("isolated_cores=4-7")))
			By("Populating CPU partitioning cmdline")
			Expect(cmdlineCPUsPartitioning.MatchString(manifest)).To(BeTrue())
			By("Populating realtime cmdline")
			Expect(cmdlineRealtimeWithCPUBalancing.MatchString(manifest)).To(BeTrue())
			By("Populating hugepages cmdline")
			Expect(cmdlineHugepages.MatchString(manifest)).To(BeTrue())
			By("Populating empty additional kernel arguments cmdline")
			Expect(manifest).To(ContainSubstring("cmdline_additionalArg="))

		})

		It("should generate yaml with expected parameters for Isolated balancing disabled", func() {
			profile.Spec.CPU.BalanceIsolated = pointer.BoolPtr(false)
			manifest := getTunedManifest(profile)

			Expect(cmdlineRealtimeWithoutCPUBalancing.MatchString(manifest)).To(BeTrue())
		})

		It("should generate yaml with expected parameters for additional kernel arguments", func() {
			profile.Spec.AdditionalKernelArgs = additionalArgs
			manifest := getTunedManifest(profile)

			Expect(cmdlineAdditionalArg.MatchString(manifest)).To(BeTrue())
		})

		It("should not allocate hugepages on the specific NUMA node via kernel arguments", func() {
			manifest := getTunedManifest(profile)
			Expect(strings.Count(manifest, "hugepagesz=")).Should(BeNumerically("==", 2))
			Expect(strings.Count(manifest, "hugepages=")).Should(BeNumerically("==", 3))

			profile.Spec.HugePages.Pages[0].Node = pointer.Int32Ptr(1)
			manifest = getTunedManifest(profile)
			Expect(strings.Count(manifest, "hugepagesz=")).Should(BeNumerically("==", 1))
			Expect(strings.Count(manifest, "hugepages=")).Should(BeNumerically("==", 2))
		})

		Context("with 1G default huge pages", func() {
			Context("with requested 2M huge pages allocation on the specified node", func() {
				It("should append the dummy 2M huge pages kernel arguments", func() {
					profile.Spec.HugePages.Pages = append(profile.Spec.HugePages.Pages, performancev2.HugePage{
						Size:  components.HugepagesSize2M,
						Count: 128,
						Node:  pointer.Int32Ptr(0),
					})

					manifest := getTunedManifest(profile)
					Expect(cmdlineDummy2MHugePages.MatchString(manifest)).To(BeTrue())
				})
			})

			Context("with requested 2M huge pages allocation via kernel arguments", func() {
				It("should not append the dummy 2M kernel arguments", func() {
					profile.Spec.HugePages.Pages = append(profile.Spec.HugePages.Pages, performancev2.HugePage{
						Size:  components.HugepagesSize2M,
						Count: 128,
					})

					manifest := getTunedManifest(profile)
					Expect(cmdlineDummy2MHugePages.MatchString(manifest)).To(BeFalse())
					Expect(cmdlineMultipleHugePages.MatchString(manifest)).To(BeTrue())
				})
			})

			Context("without requested 2M hugepages", func() {
				It("should not append dummy 2M huge pages kernel arguments", func() {
					manifest := getTunedManifest(profile)
					Expect(cmdlineDummy2MHugePages.MatchString(manifest)).To(BeFalse())
				})
			})

			Context("with requested 2M huge pages allocation on the specified node and via kernel arguments", func() {
				It("should not append the dummy 2M kernel arguments", func() {
					profile.Spec.HugePages.Pages = append(profile.Spec.HugePages.Pages, performancev2.HugePage{
						Size:  components.HugepagesSize2M,
						Count: 128,
						Node:  pointer.Int32Ptr(0),
					})
					profile.Spec.HugePages.Pages = append(profile.Spec.HugePages.Pages, performancev2.HugePage{
						Size:  components.HugepagesSize2M,
						Count: 128,
					})

					manifest := getTunedManifest(profile)
					Expect(cmdlineDummy2MHugePages.MatchString(manifest)).To(BeFalse())
					Expect(cmdlineMultipleHugePages.MatchString(manifest)).To(BeTrue())
				})
			})
		})

		Context("with 2M default huge pages", func() {
			Context("with requested 2M huge pages allocation on the specified node", func() {
				It("should not append the dummy 2M huge pages kernel arguments", func() {
					defaultSize := performancev2.HugePageSize(components.HugepagesSize2M)
					profile.Spec.HugePages.DefaultHugePagesSize = &defaultSize
					profile.Spec.HugePages.Pages = append(profile.Spec.HugePages.Pages, performancev2.HugePage{
						Size:  components.HugepagesSize2M,
						Count: 128,
						Node:  pointer.Int32Ptr(0),
					})

					manifest := getTunedManifest(profile)
					Expect(cmdlineDummy2MHugePages.MatchString(manifest)).To(BeFalse())
					Expect(cmdlineMultipleHugePages.MatchString(manifest)).To(BeFalse())
				})
			})
		})
	})
})
