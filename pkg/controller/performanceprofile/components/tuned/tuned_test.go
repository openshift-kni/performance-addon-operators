package tuned

import (
	"fmt"

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
			Expect(manifest).To(ContainSubstring(fmt.Sprintf("isolated_cores=4-7")))
			Expect(manifest).To(ContainSubstring(expectedMatchSelector))
		})
	})
})
