package profile

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"
	pinfo "github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components/profileinfo"
	testutils "github.com/openshift-kni/performance-addon-operators/pkg/utils/testing"
)

const (
	NodeSelectorRole = "barRole"
)

var _ = Describe("PerformanceProfile", func() {
	var profile *pinfo.PerformanceProfileInfo

	BeforeEach(func() {
		profile = testutils.NewPerformanceProfileInfo("test")
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

func setValidNodeSelector(profile *pinfo.PerformanceProfileInfo) {
	selector := make(map[string]string)
	selector["fooDomain/"+NodeSelectorRole] = ""
	profile.Spec.NodeSelector = selector
}
