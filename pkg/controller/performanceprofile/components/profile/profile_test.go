package profile

import (
	"github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v1"
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

	var profile *v1.PerformanceProfile

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
			incorrectDefaultSize := v1.HugePageSize("!#@")
			profile.Spec.HugePages.DefaultHugePagesSize = &incorrectDefaultSize

			err := ValidateParameters(profile)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("hugepages default size should be equal"))
		})

		It("should reject hugepages allocation with different sizes", func() {
			profile.Spec.HugePages.Pages = append(profile.Spec.HugePages.Pages, v1.HugePage{
				Count: 128,
				Node:  pointer.Int32Ptr(0),
				Size:  v1.HugePageSize("2M"),
			})
			err := ValidateParameters(profile)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("allocation of hugepages with different sizes not supported"))
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

func setValidNodeSelector(profile *v1.PerformanceProfile) {
	selector := make(map[string]string)
	selector["fooDomain/"+NodeSelectorRole] = ""
	profile.Spec.NodeSelector = selector
}
