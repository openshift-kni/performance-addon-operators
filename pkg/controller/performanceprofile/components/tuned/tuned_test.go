package tuned

import (
	"fmt"
	"regexp"

	"github.com/ghodss/yaml"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	testutils "github.com/openshift-kni/performance-addon-operators/pkg/utils/testing"
)

const testAssetsDir = "../../../../../build/assets"
const expectedMatchSelector = `
  - match:
    - label: label1
      match:
      - label: label2
        value: label2
      value: label1
`

var cmdlineCPUsPartitioning = regexp.MustCompile(`\s*cmdline_cpu_part=\+\s*nohz=on\s+rcu_nocbs=\${isolated_cores}\s+tuned.non_isolcpus=\${not_isolated_cpumask}\s+intel_pstate=disable\s+nosoftlockup\s*`)
var cmdlineRealtimeWithCPUBalancing = regexp.MustCompile(`\s*cmdline_realtime=\+\s*tsc=nowatchdog\s+intel_iommu=on\s+iommu=pt\s+systemd.cpu_affinity=\${not_isolated_cores_expanded}\s*`)
var cmdlineRealtimeWithoutCPUBalancing = regexp.MustCompile(`\s*cmdline_realtime=\+\s*tsc=nowatchdog\s+intel_iommu=on\s+iommu=pt\s+isolated_cores=4-7\s+systemd.cpu_affinity=\${not_isolated_cores_expanded}\s*`)
var cmdlineHugepages = regexp.MustCompile(`\s*cmdline_hugepages=\+\s*default_hugepagesz=1G\s+hugepagesz=1G\s+hugepages=4\s*`)
var cmdlineAdditionalArg = regexp.MustCompile(`\s*cmdline_additionalArg=\+\s*test1=val1\s+test2=val2\s*`)
var additionalArgs = []string{"test1=val1", "test2=val2"}

var _ = Describe("Tuned", func() {
	Context("with worker performance profile", func() {
		It("should generate yaml with expected parameters", func() {
			profile := testutils.NewPerformanceProfile("test")
			profile.Spec.NodeSelector = map[string]string{
				"label1": "label1",
				"label2": "label2",
			}
			tuned, err := NewNodePerformance(testAssetsDir, profile)
			Expect(err).ToNot(HaveOccurred())
			y, err := yaml.Marshal(tuned)
			Expect(err).ToNot(HaveOccurred())

			manifest := string(y)
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
			profile := testutils.NewPerformanceProfile("test")
			f := false
			profile.Spec.CPU.BalanceIsolated = &f
			tuned, err := NewNodePerformance(testAssetsDir, profile)
			Expect(err).ToNot(HaveOccurred())
			y, err := yaml.Marshal(tuned)
			Expect(err).ToNot(HaveOccurred())
			manifest := string(y)
			Expect(cmdlineRealtimeWithoutCPUBalancing.MatchString(manifest)).To(BeTrue())
		})

		It("should generate yaml with expected parameters for additional kernel args", func() {
			profile := testutils.NewPerformanceProfile("test")
			profile.Spec.AdditionalKernelArgs = additionalArgs
			tuned, err := NewNodePerformance(testAssetsDir, profile)
			Expect(err).ToNot(HaveOccurred())
			y, err := yaml.Marshal(tuned)
			Expect(err).ToNot(HaveOccurred())
			manifest := string(y)
			Expect(cmdlineAdditionalArg.MatchString(manifest)).To(BeTrue())
		})

	})
})
