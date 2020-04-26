package machineconfig

import (
	"fmt"

	"k8s.io/utils/pointer"

	"github.com/ghodss/yaml"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	performancev1alpha1 "github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v1alpha1"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"
	testutils "github.com/openshift-kni/performance-addon-operators/pkg/utils/testing"
)

const testAssetsDir = "../../../../../build/assets"
const expectedSystemdUnits = `
      - contents: |
          [Unit]
          Description=Preboot tuning patch
          Before=kubelet.service
          Before=reboot.service

          [Service]
          Environment=RESERVED_CPUS=0-3
          Environment=RESERVED_CPU_MASK_INVERT=ffffffff,fffffff0
          Type=oneshot
          RemainAfterExit=true
          ExecStart=/usr/local/bin/pre-boot-tuning.sh

          [Install]
          WantedBy=multi-user.target
        enabled: true
        name: pre-boot-tuning.service
      - contents: |
          [Unit]
          Description=Reboot initiated by pre-boot-tuning
          Wants=network-online.target
          After=network-online.target
          Before=kubelet.service

          [Service]
          Type=oneshot
          RemainAfterExit=true
          ExecStart=/usr/local/bin/reboot.sh

          [Install]
          WantedBy=multi-user.target
        enabled: true
        name: reboot.service
`
const hugepagesAllocationService = `
      - contents: |
          [Unit]
          Description=Hugepages-1048576kB allocation on the node 0
          Before=kubelet.service

          [Service]
          Environment=HUGEPAGES_COUNT=4
          Environment=HUGEPAGES_SIZE=1048576
          Environment=NUMA_NODE=0
          Type=oneshot
          RemainAfterExit=true
          ExecStart=/usr/local/bin/hugepages-allocation.sh

          [Install]
          WantedBy=multi-user.target
        enabled: true
        name: hugepages-allocation-1048576kB-NUMA0.service
`

var _ = Describe("Machine Config", func() {
	It("should generate yaml with expected parameters", func() {
		profile := testutils.NewPerformanceProfile("test")
		profile.Spec.HugePages.Pages = append(profile.Spec.HugePages.Pages, performancev1alpha1.HugePage{
			Count: 1024,
			Size:  "2M",
		})
		f := false
		profile.Spec.CPU.BalanceIsolated = &f
		mc, err := New(testAssetsDir, profile)
		Expect(err).ToNot(HaveOccurred())

		Expect(mc.Spec.KernelType).To(Equal(MCKernelRT))

		y, err := yaml.Marshal(mc)
		Expect(err).ToNot(HaveOccurred())

		manifest := string(y)

		labelKey, labelValue := components.GetFirstKeyAndValue(profile.Spec.MachineConfigLabel)
		Expect(manifest).To(ContainSubstring(fmt.Sprintf("%s: %s", labelKey, labelValue)))
		Expect(manifest).To(ContainSubstring(expectedSystemdUnits))
	})

	It("should generate yaml with expected parameters when balanced isolated defaults to true", func() {
		profile := testutils.NewPerformanceProfile("test")
		profile.Spec.HugePages.Pages = append(profile.Spec.HugePages.Pages, performancev1alpha1.HugePage{
			Count: 1024,
			Size:  "2M",
		})
		mc, err := New(testAssetsDir, profile)
		Expect(err).ToNot(HaveOccurred())

		Expect(mc.Spec.KernelType).To(Equal(MCKernelRT))

		y, err := yaml.Marshal(mc)
		Expect(err).ToNot(HaveOccurred())

		manifest := string(y)

		labelKey, labelValue := components.GetFirstKeyAndValue(profile.Spec.MachineConfigLabel)
		Expect(manifest).To(ContainSubstring(fmt.Sprintf("%s: %s", labelKey, labelValue)))
		Expect(manifest).To(ContainSubstring(expectedSystemdUnits))
	})

	It("should generate yaml with expected parameters and additional kernel arguments", func() {
		profile := testutils.NewPerformanceProfile("test")
		profile.Spec.AdditionalKernelArgs = append(profile.Spec.AdditionalKernelArgs,
			"nmi_watchdog=0", "audit=0",
			"mce=off",
			"processor.max_cstate=1",
			"idle=poll",
			"intel_idle.max_cstate=0")
		mc, err := New(testAssetsDir, profile)
		Expect(err).ToNot(HaveOccurred())

		y, err := yaml.Marshal(mc)
		Expect(err).ToNot(HaveOccurred())

		manifest := string(y)

		labelKey, labelValue := components.GetFirstKeyAndValue(profile.Spec.MachineConfigLabel)
		Expect(manifest).To(ContainSubstring(fmt.Sprintf("%s: %s", labelKey, labelValue)))
		Expect(manifest).To(ContainSubstring(expectedSystemdUnits))
	})

	Context("with hugepages with specified NUMA node", func() {
		var manifest string

		BeforeEach(func() {
			profile := testutils.NewPerformanceProfile("test")
			profile.Spec.HugePages.Pages[0].Node = pointer.Int32Ptr(0)

			mc, err := New(testAssetsDir, profile)
			Expect(err).ToNot(HaveOccurred())
			Expect(mc.Spec.KernelType).To(Equal(MCKernelRT))

			y, err := yaml.Marshal(mc)
			Expect(err).ToNot(HaveOccurred())

			manifest = string(y)
			labelKey, labelValue := components.GetFirstKeyAndValue(profile.Spec.MachineConfigLabel)
			Expect(manifest).To(ContainSubstring(fmt.Sprintf("%s: %s", labelKey, labelValue)))
		})

		It("should not add hugepages kernel boot parameters", func() {
			Expect(manifest).ToNot(ContainSubstring("- hugepagesz=1G"))
			Expect(manifest).ToNot(ContainSubstring("- hugepages=4"))
		})

		It("should add systemd unit to allocate hugepages", func() {
			Expect(manifest).To(ContainSubstring(hugepagesAllocationService))
		})
	})
})
