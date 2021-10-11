package machineconfig

import (
	"encoding/json"
	"fmt"

	"k8s.io/utils/pointer"

	"github.com/ghodss/yaml"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	gm "github.com/onsi/gomega/types"

	igntypes "github.com/coreos/ignition/v2/config/v3_2/types"
	performancev2 "github.com/openshift-kni/performance-addon-operators/api/v2"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"
	testutils "github.com/openshift-kni/performance-addon-operators/pkg/utils/testing"
)

const testAssetsDir = "../../../../../build/assets"
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

	Context("machine config creation ", func() {
		It("should create machine config with valid assests", func() {
			profile := testutils.NewPerformanceProfile("test")
			profile.Spec.HugePages.Pages[0].Node = pointer.Int32Ptr(0)

			_, err := New(testAssetsDir, profile)
			Expect(err).ToNot(HaveOccurred())
			_, err = New("../../../../../build/invalid/assets", profile)
			Expect(err).Should(HaveOccurred(), "should fail with missing CPU")
		})
	})

	Context("with hugepages with specified NUMA node", func() {
		var manifest string

		BeforeEach(func() {
			profile := testutils.NewPerformanceProfile("test")
			profile.Spec.HugePages.Pages[0].Node = pointer.Int32Ptr(0)

			labelKey, labelValue := components.GetFirstKeyAndValue(profile.Spec.MachineConfigLabel)
			mc, err := New(testAssetsDir, profile)
			Expect(err).ToNot(HaveOccurred())
			Expect(mc.Spec.KernelType).To(Equal(MCKernelRT))

			y, err := yaml.Marshal(mc)
			Expect(err).ToNot(HaveOccurred())

			manifest = string(y)
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

	Context("with CPU spec", func() {
		var profile *performancev2.PerformanceProfile
		var ignition igntypes.Config

		BeforeEach(func() {
			profile = testutils.NewPerformanceProfile("test")
			isolated := performancev2.CPUSet("0-3")
			reserved := performancev2.CPUSet("4-15")
			profile.Spec.CPU = &performancev2.CPU{
				Isolated: &isolated,
				Reserved: &reserved,
			}
		})
		JustBeforeEach(func() {
			mc, err := New(testAssetsDir, profile)
			Expect(err).ToNot(HaveOccurred())

			err = json.Unmarshal(mc.Spec.Config.Raw, &ignition)
			Expect(err).ToNot(HaveOccurred())

			_, err = yaml.Marshal(mc)
			Expect(err).ToNot(HaveOccurred())
		})

		containAcceleratedStartupUnits := func() gm.GomegaMatcher {
			return ContainElements(
				MatchFields(IgnoreExtras, Fields{
					"Name":     Equal("accelerated-container-startup.service"),
					"Contents": PointTo(ContainSubstring("ExecStart=/usr/local/bin/accelerated-container-startup.sh\n")),
				}),
				MatchFields(IgnoreExtras, Fields{
					"Name":     Equal("accelerated-container-shutdown.service"),
					"Contents": PointTo(ContainSubstring("ExecStart=/usr/local/bin/accelerated-container-startup.sh\n")),
				}),
			)
		}
		containAcceleratedStartupFiles := func() gm.GomegaMatcher {
			executableFilemode := func() gm.GomegaMatcher {
				return WithTransform(func(mode int) bool {
					return mode&0700 == 0700
				}, BeTrue())
			}

			return ContainElements(
				MatchFields(IgnoreExtras, Fields{
					"Node": MatchFields(IgnoreExtras, Fields{
						"Path": Equal("/usr/local/bin/accelerated-container-startup.sh"),
					}),
					"FileEmbedded1": MatchFields(IgnoreExtras, Fields{
						"Mode": PointTo(executableFilemode()),
					}),
				}),
			)
		}

		Context("without AcceleratedStartup", func() {
			It("should not add systemd unit for accelerated startup", func() {
				Expect(ignition.Systemd.Units).NotTo(containAcceleratedStartupUnits())
				Expect(ignition.Storage.Files).NotTo(containAcceleratedStartupFiles())
			})
		})

		Context("with acceleratedStartup=false", func() {
			BeforeEach(func() {
				profile.Spec.CPU.AcceleratedStartup = pointer.Bool(false)
			})
			It("should not add systemd unit for accelerated startup", func() {
				Expect(ignition.Systemd.Units).NotTo(containAcceleratedStartupUnits())
				Expect(ignition.Storage.Files).NotTo(containAcceleratedStartupFiles())
			})
		})

		Context("with acceleratedStartup=true", func() {
			BeforeEach(func() {
				profile.Spec.CPU.AcceleratedStartup = pointer.Bool(true)
			})
			It("should add systemd unit for accelerated startup", func() {
				Expect(ignition.Systemd.Units).To(containAcceleratedStartupUnits())
				Expect(ignition.Storage.Files).To(containAcceleratedStartupFiles())
			})
		})

	})

})
