package components

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type listToMask struct {
	cpuList string
	cpuMask string
}

var cpuListToMask = []listToMask{
	{"0", "01"}, {"2-3", "0c"}, {"3,4,53-55,61-63", "e0e0000000000018"},
}
var cpuListToInvertMask = []listToMask{
	{"0", "3ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe"}, {"2-3", "3ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff3"}, {"3,4,53-55,61-63", "3fffffffffffffffffffffffffffffffffffffffffffffff1f1fffffffffffe7"},
}

var _ = Describe("Components utils", func() {
	Context("Convert CPU list to CPU mask", func() {
		It("should generate a valid CPU mask from CPU list ", func() {
			for _, cpuEntry := range cpuListToMask {
				cpuMask, err := CPUListToHexMask(cpuEntry.cpuList)
				Expect(err).ToNot(HaveOccurred())
				Expect(cpuMask).Should(Equal(cpuEntry.cpuMask))
			}
		})
		It("should generate a valid CPU inverted mask from CPU list ", func() {
			for _, cpuEntry := range cpuListToInvertMask {
				cpuMask, err := CPUListToInvertedMask(cpuEntry.cpuList)
				Expect(err).ToNot(HaveOccurred())
				Expect(cpuMask).Should(Equal(cpuEntry.cpuMask))
			}
		})
	})
})
