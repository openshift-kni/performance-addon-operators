package featuregate

import (
	"github.com/ghodss/yaml"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Feature Gate", func() {
	It("should generate yaml with 'LatencySensitive' feature set", func() {
		fg := NewLatencySensitive()
		y, err := yaml.Marshal(fg)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(y)).To(ContainSubstring("featureSet: LatencySensitive"))
	})
})
