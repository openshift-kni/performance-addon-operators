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
	{"0", "00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000001"},
	{"2-3", "00000000,00000000,00000000,00000000,00000000,00000000,00000000,0000000c"},
	{"3,4,53-55,61-63", "00000000,00000000,00000000,00000000,00000000,00000000,e0e00000,00000018"},
	{"0-127", "00000000,00000000,00000000,00000000,ffffffff,ffffffff,ffffffff,ffffffff"},
	{"0-255", "ffffffff,ffffffff,ffffffff,ffffffff,ffffffff,ffffffff,ffffffff,ffffffff"},
}
var cpuListToInvertMask = []listToMask{
	{"0", "ffffffff,fffffffe"}, {"2-3", "ffffffff,fffffff3"}, {"3,4,53-55,61-63", "1f1fffff,ffffffe7"},
	{"80", "ffffffff,ffffffff"},
}

var _ = Describe("Components utils", func() {
	Context("Convert CPU list to CPU mask", func() {
		It("should generate a valid CPU mask from CPU list ", func() {
			for _, cpuEntry := range cpuListToMask {
				cpuMask, err := CPUListTo256BitsMaskList(cpuEntry.cpuList)
				Expect(err).ToNot(HaveOccurred())
				Expect(cpuMask).Should(Equal(cpuEntry.cpuMask))
			}
		})
		It("should generate a valid CPU inverted mask from CPU list ", func() {
			for _, cpuEntry := range cpuListToInvertMask {
				cpuMask, err := CPUListTo64BitsMaskList(cpuEntry.cpuList)
				Expect(err).ToNot(HaveOccurred())
				Expect(cpuMask).Should(Equal(cpuEntry.cpuMask))
			}
		})
	})
})
