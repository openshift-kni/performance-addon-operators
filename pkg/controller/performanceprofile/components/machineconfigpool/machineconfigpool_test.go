package machineconfigpool

import (
	"fmt"

	"github.com/ghodss/yaml"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"
	testutils "github.com/openshift-kni/performance-addon-operators/pkg/utils/testing"
)

const expectedMachineConfigSelectorValues = `
      values:
      - worker
      - performance-test
`

var _ = Describe("Machine Config Pool", func() {
	It("should generate yaml with expected parameters", func() {
		profile := testutils.NewPerformanceProfile("test")
		profile.Spec.NodeSelector = map[string]string{"test": "test"}
		mcp := New(profile)

		y, err := yaml.Marshal(mcp)
		Expect(err).ToNot(HaveOccurred())

		manifest := string(y)
		Expect(manifest).To(ContainSubstring(fmt.Sprintf("%s: %s", "test", "test")))
		Expect(manifest).To(ContainSubstring(fmt.Sprintf("key: %s", components.LabelMachineConfigurationRole)))
		Expect(manifest).To(ContainSubstring(expectedMachineConfigSelectorValues))
	})
})
