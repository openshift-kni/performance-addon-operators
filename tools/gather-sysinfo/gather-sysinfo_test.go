package main

import (
	"io/ioutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/debug-tools/pkg/knit/cmd"
	"github.com/spf13/cobra"
)

var kniEntries = []string{
	"/host/proc/cmdline",
	"/host/proc/interrupts",
	"/host/proc/irq/default_smp_affinity",
	"/host/proc/irq/*/*affinity_list",
	"/host/proc/irq/*/node",
	"/host/proc/softirqs",
	"/host/sys/devices/system/cpu/smt/active",
	"/host/proc/sys/kernel/sched_domain/cpu*/domain*/flags",
	"/host/sys/devices/system/cpu/offline",
	"/host/sys/class/dmi/id/bios*",
	"/host/sys/class/dmi/id/product_family",
	"/host/sys/class/dmi/id/product_name",
	"/host/sys/class/dmi/id/product_sku",
	"/host/sys/class/dmi/id/product_version",
}

var snapshotEntries = []string{"/host/proc/cmdline", "/host/sys/devices/system/node/online"}

var expectedEntries = []string{
	"/host/proc/cmdline",
	"/host/proc/interrupts",
	"/host/proc/irq/default_smp_affinity",
	"/host/proc/irq/*/*affinity_list",
	"/host/proc/irq/*/node",
	"/host/proc/softirqs",
	"/host/sys/devices/system/cpu/smt/active",
	"/host/proc/sys/kernel/sched_domain/cpu*/domain*/flags",
	"/host/sys/devices/system/cpu/offline",
	"/host/sys/class/dmi/id/bios*",
	"/host/sys/class/dmi/id/product_family",
	"/host/sys/class/dmi/id/product_name",
	"/host/sys/class/dmi/id/product_sku",
	"/host/sys/class/dmi/id/product_version",
	"/host/sys/devices/system/node/online",
}

var _ = Describe("Components utils", func() {
	Context("gather sysinfo", func() {
		It("collect machine info test", func() {
			//Check if collect machine info file is created correctly
			knitOpts := &cmd.KnitOptions{}
			knitOpts.SysFSRoot = "/host/sys"

			err := collectMachineinfo(knitOpts, "./output")
			Expect(err).ToNot(HaveOccurred())

			content, err := ioutil.ReadFile("./output")
			Expect(err).ToNot(HaveOccurred())

			output := string(content)
			Expect(output).To(ContainSubstring("timestamp"))
		})

		It("chroot file spec test", func() {
			entries := chrootFileSpecs(kniExpectedCloneContent(), "/host")
			Expect(entries).To(Equal(kniEntries))
		})

		It("no duplicates entries test", func() {
			resultEntries := dedupExpectedContent(kniEntries, snapshotEntries)
			Expect(len(expectedEntries)).To(Equal(len(resultEntries)))
		})

		It("makeSnapshot test", func() {
			knitOpts := &cmd.KnitOptions{}

			opts := &snapshotOptions{}
			cmd := &cobra.Command{}
			args := []string{}

			err := makeSnapshot(cmd, knitOpts, opts, args)
			Expect(err).To(HaveOccurred(), "--output is requidred")

			opts.output = "testSnapshot.tgz"
			err = makeSnapshot(cmd, knitOpts, opts, args)
			Expect(err).ToNot(HaveOccurred())

			Expect(opts.output).To(BeAnExistingFile())
		})
	})
})
