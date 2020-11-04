package profile

import (
	"fmt"

	performancev2 "github.com/openshift-kni/performance-addon-operators/api/v2"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"
	"k8s.io/utils/pointer"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	testutils "github.com/openshift-kni/performance-addon-operators/pkg/utils/testing"
)

const (
	NodeSelectorRole = "barRole"
)

var _ = Describe("PerformanceProfile", func() {

	var profile *performancev2.PerformanceProfile

	BeforeEach(func() {
		profile = testutils.NewPerformanceProfile("test")
	})

	Describe("Validation", func() {

		It("should have CPU fields populated", func() {
			Expect(ValidateParameters(profile)).ShouldNot(HaveOccurred(), "should pass with populated CPU fields")
			profile.Spec.CPU.Isolated = nil
			Expect(ValidateParameters(profile)).Should(HaveOccurred(), "should fail with missing CPU Isolated field")
			profile.Spec.CPU = nil
			Expect(ValidateParameters(profile)).Should(HaveOccurred(), "should fail with missing CPU")
		})

		It("should have 0 or 1 MachineConfigLabels", func() {
			Expect(ValidateParameters(profile)).ShouldNot(HaveOccurred(), "should pass with 1 MachineConfigLabel")

			profile.Spec.MachineConfigLabel["foo"] = "bar"
			Expect(ValidateParameters(profile)).Should(HaveOccurred(), "should fail with 2 MachineConfigLabels")

			profile.Spec.MachineConfigLabel = nil
			setValidNodeSelector(profile)

			Expect(ValidateParameters(profile)).ShouldNot(HaveOccurred(), "should pass with nil MachineConfigLabels")
		})

		It("should should have 0 or 1 MachineConfigPoolSelector labels", func() {
			Expect(ValidateParameters(profile)).ShouldNot(HaveOccurred(), "should pass with 1 MachineConfigPoolSelector label")

			profile.Spec.MachineConfigPoolSelector["foo"] = "bar"
			Expect(ValidateParameters(profile)).Should(HaveOccurred(), "should fail with 2 MachineConfigPoolSelector labels")

			profile.Spec.MachineConfigPoolSelector = nil
			setValidNodeSelector(profile)

			Expect(ValidateParameters(profile)).ShouldNot(HaveOccurred(), "should pass with nil MachineConfigPoolSelector")
		})

		It("should have sensible NodeSelector in case MachineConfigLabel or MachineConfigPoolSelector is empty", func() {
			profile.Spec.MachineConfigLabel = nil
			Expect(ValidateParameters(profile)).Should(HaveOccurred(), "should fail with invalid NodeSelector")

			setValidNodeSelector(profile)
			Expect(ValidateParameters(profile)).ShouldNot(HaveOccurred(), "should pass with valid NodeSelector")

		})

		It("should reject on incorrect default hugepages size", func() {
			incorrectDefaultSize := performancev2.HugePageSize("!#@")
			profile.Spec.HugePages.DefaultHugePagesSize = &incorrectDefaultSize

			err := ValidateParameters(profile)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("hugepages default size should be equal"))
		})

		It("should reject hugepages allocation with unexpected page size", func() {
			profile.Spec.HugePages.Pages = append(profile.Spec.HugePages.Pages, performancev2.HugePage{
				Count: 128,
				Node:  pointer.Int32Ptr(0),
				Size:  performancev2.HugePageSize("14M"),
			})
			err := ValidateParameters(profile)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("the page size should be equal to %q or %q", hugepagesSize1G, hugepagesSize2M)))
		})

		It("should allow cpus allocation with no reserved CPUs", func() {
			reservedCPUs := performancev2.CPUSet("")
			isolatedCPUs := performancev2.CPUSet("0-7")
			profile.Spec.CPU.Reserved = &reservedCPUs
			profile.Spec.CPU.Isolated = &isolatedCPUs
			err := ValidateParameters(profile)
			Expect(err).Should(Not(HaveOccurred()))
		})

		It("should reject cpus allocation with no isolated CPUs", func() {
			reservedCPUs := performancev2.CPUSet("0-3")
			isolatedCPUs := performancev2.CPUSet("")
			profile.Spec.CPU.Reserved = &reservedCPUs
			profile.Spec.CPU.Isolated = &isolatedCPUs
			err := ValidateParameters(profile)
			Expect(err.Error()).To(ContainSubstring("must contain isolated cpus"))
		})

		It("should allow cpus allocation with not overlapping sets", func() {
			reservedCPUs := performancev2.CPUSet("0-3")
			isolatedCPUs := performancev2.CPUSet("4-15")
			profile.Spec.CPU.Reserved = &reservedCPUs
			profile.Spec.CPU.Isolated = &isolatedCPUs
			err := ValidateParameters(profile)
			Expect(err).Should(Not(HaveOccurred()))
		})

		It("should reject cpus allocation with overlapping sets", func() {
			reservedCPUs := performancev2.CPUSet("0-7")
			isolatedCPUs := performancev2.CPUSet("0-15")
			profile.Spec.CPU.Reserved = &reservedCPUs
			profile.Spec.CPU.Isolated = &isolatedCPUs
			err := ValidateParameters(profile)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("reserved and isolated cpus overlap"))
		})

		When("pages have duplication", func() {
			Context("with specified NUMA node", func() {
				It("should raise the validation error", func() {
					profile.Spec.HugePages.Pages = append(profile.Spec.HugePages.Pages, performancev2.HugePage{
						Count: 128,
						Size:  hugepagesSize1G,
						Node:  pointer.Int32Ptr(0),
					})
					profile.Spec.HugePages.Pages = append(profile.Spec.HugePages.Pages, performancev2.HugePage{
						Count: 64,
						Size:  hugepagesSize1G,
						Node:  pointer.Int32Ptr(0),
					})
					err := ValidateParameters(profile)
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("the page with the size %q and with specified NUMA node 0, has duplication", hugepagesSize1G)))
				})
			})

			Context("without specified NUMA node", func() {
				It("should raise the validation error", func() {
					profile.Spec.HugePages.Pages = append(profile.Spec.HugePages.Pages, performancev2.HugePage{
						Count: 128,
						Size:  hugepagesSize1G,
					})
					err := ValidateParameters(profile)
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("the page with the size %q and without the specified NUMA node, has duplication", hugepagesSize1G)))
				})
			})

			Context("with not sequentially duplication blocks", func() {
				It("should raise the validation error", func() {
					profile.Spec.HugePages.Pages = append(profile.Spec.HugePages.Pages, performancev2.HugePage{
						Count: 128,
						Size:  hugepagesSize2M,
					})
					profile.Spec.HugePages.Pages = append(profile.Spec.HugePages.Pages, performancev2.HugePage{
						Count: 128,
						Size:  hugepagesSize1G,
					})
					err := ValidateParameters(profile)
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("the page with the size %q and without the specified NUMA node, has duplication", hugepagesSize1G)))
				})
			})
		})
	})

	Describe("Defaulting", func() {

		It("should return given MachineConfigLabel", func() {

			labels := GetMachineConfigLabel(profile)
			k, v := components.GetFirstKeyAndValue(labels)
			Expect(k).To(Equal(testutils.MachineConfigLabelKey))
			Expect(v).To(Equal(testutils.MachineConfigLabelValue))

		})

		It("should return given MachineConfigPoolSelector", func() {

			labels := GetMachineConfigPoolSelector(profile)
			k, v := components.GetFirstKeyAndValue(labels)
			Expect(k).To(Equal(testutils.MachineConfigPoolLabelKey))
			Expect(v).To(Equal(testutils.MachineConfigPoolLabelValue))

		})

		It("should return default MachineConfigLabels", func() {

			profile.Spec.MachineConfigLabel = nil

			setValidNodeSelector(profile)

			labels := GetMachineConfigLabel(profile)
			k, v := components.GetFirstKeyAndValue(labels)
			Expect(k).To(Equal(components.MachineConfigRoleLabelKey))
			Expect(v).To(Equal(NodeSelectorRole))

		})

		It("should return default MachineConfigPoolSelector", func() {

			profile.Spec.MachineConfigPoolSelector = nil

			setValidNodeSelector(profile)

			labels := GetMachineConfigPoolSelector(profile)
			k, v := components.GetFirstKeyAndValue(labels)
			Expect(k).To(Equal(components.MachineConfigRoleLabelKey))
			Expect(v).To(Equal(NodeSelectorRole))

		})
	})
})

func setValidNodeSelector(profile *performancev2.PerformanceProfile) {
	selector := make(map[string]string)
	selector["fooDomain/"+NodeSelectorRole] = ""
	profile.Spec.NodeSelector = selector
}
