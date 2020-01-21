package profile

import (
	"github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v1alpha1"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	testutils "github.com/openshift-kni/performance-addon-operators/pkg/utils/testing"
)

const (
	NodeSelectorRole = "barRole"
)

var _ = Describe("PerformanceProfile", func() {

	var profile v1alpha1.PerformanceProfile

	BeforeEach(func() {
		profile = *testutils.NewPerformanceProfile("test")
	})

	Describe("Validation", func() {

		It("Should have 0 or 1 MachineConfigLabels", func() {

			Expect(ValidateParameters(profile)).ShouldNot(HaveOccurred(), "should pass with 1 MachineConfigLabel")

			profile.Spec.MachineConfigLabel["foo"] = "bar"
			Expect(ValidateParameters(profile)).Should(HaveOccurred(), "should fail with 2 MachineConfigLabels")

			profile.Spec.MachineConfigLabel = nil
			setValidNodeSelector(&profile)

			Expect(ValidateParameters(profile)).ShouldNot(HaveOccurred(), "should pass with nil MachineConfigLabels")

		})

		It("Should should have 0 or 1 MachineConfigPoolSelector labels", func() {

			Expect(ValidateParameters(profile)).ShouldNot(HaveOccurred(), "should pass with 1 MachineConfigPoolSelector label")

			profile.Spec.MachineConfigPoolSelector["foo"] = "bar"
			Expect(ValidateParameters(profile)).Should(HaveOccurred(), "should fail with 2 MachineConfigPoolSelector labels")

			profile.Spec.MachineConfigPoolSelector = nil
			setValidNodeSelector(&profile)

			Expect(ValidateParameters(profile)).ShouldNot(HaveOccurred(), "should pass with nil MachineConfigPoolSelector")

		})

		It("Should have sensible NodeSelector in case MachineConfigLabel or MachineConfigPoolSelector is empty", func() {

			profile.Spec.MachineConfigLabel = nil
			Expect(ValidateParameters(profile)).Should(HaveOccurred(), "should fail with invalid NodeSelector")

			setValidNodeSelector(&profile)
			Expect(ValidateParameters(profile)).ShouldNot(HaveOccurred(), "should pass with valid NodeSelector")

		})

	})

	Describe("Defaulting", func() {

		It("Should return given MachineConfigLabel", func() {

			labels := GetMachineConfigLabel(profile)
			k, v := components.GetFirstKeyAndValue(labels)
			Expect(k).To(Equal(testutils.MachineConfigLabelKey))
			Expect(v).To(Equal(testutils.MachineConfigLabelValue))

		})

		It("Should return given MachineConfigPoolSelector", func() {

			labels := GetMachineConfigPoolSelector(profile)
			k, v := components.GetFirstKeyAndValue(labels)
			Expect(k).To(Equal(testutils.MachineConfigPoolLabelKey))
			Expect(v).To(Equal(testutils.MachineConfigPoolLabelValue))

		})

		It("Should return default MachineConfigLabels", func() {

			profile.Spec.MachineConfigLabel = nil

			setValidNodeSelector(&profile)

			labels := GetMachineConfigLabel(profile)
			k, v := components.GetFirstKeyAndValue(labels)
			Expect(k).To(Equal(components.MachineConfigRoleLabelKey))
			Expect(v).To(Equal(NodeSelectorRole))

		})

		It("Should return default MachineConfigPoolSelector", func() {

			profile.Spec.MachineConfigPoolSelector = nil

			setValidNodeSelector(&profile)

			labels := GetMachineConfigPoolSelector(profile)
			k, v := components.GetFirstKeyAndValue(labels)
			Expect(k).To(Equal(components.MachineConfigRoleLabelKey))
			Expect(v).To(Equal(NodeSelectorRole))

		})

	})

})

func setValidNodeSelector(profile *v1alpha1.PerformanceProfile) {
	selector := make(map[string]string)
	selector["fooDomain/"+NodeSelectorRole] = ""
	profile.Spec.NodeSelector = selector
}
