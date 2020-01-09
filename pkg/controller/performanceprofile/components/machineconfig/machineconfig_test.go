package machineconfig

import (
	"fmt"

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
          Description=RT kernel patch
          Wants=network-online.target
          After=network-online.target
          Before=kubelet.service
          Before=pre-boot-tuning.service

          [Service]
          Type=oneshot
          RemainAfterExit=true
          ExecStart=/usr/local/bin/rt-kernel.sh

          [Install]
          WantedBy=multi-user.target
        enabled: true
        name: rt-kernel.service
      - contents: |
          [Unit]
          Description=Preboot tuning patch
          Wants=rt-kernel.service
          After=rt-kernel.service
          Before=kubelet.service
          Before=reboot.service

          [Service]
          Environment=NON_ISOLATED_CPUS=2-3
          Type=oneshot
          RemainAfterExit=true
          ExecStart=/usr/local/bin/pre-boot-tuning.sh

          [Install]
          WantedBy=multi-user.target
        enabled: true
        name: pre-boot-tuning.service
      - contents: |
          [Unit]
          Description=Reboot initiated by rt-kernel and pre-boot-tuning
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
const expectedBootArguments = `
  kernelArguments:
  - nohz=on
  - nosoftlockup
  - nmi_watchdog=0
  - audit=0
  - mce=off
  - irqaffinity=0
  - skew_tick=1
  - processor.max_cstate=1
  - idle=poll
  - intel_pstate=disable
  - intel_idle.max_cstate=0
  - intel_iommu=on
  - iommu=pt
  - isolcpus=4-7
  - default_hugepagesz=1G
  - hugepagesz=1G
  - hugepages=4
  - hugepagesz=2M
  - hugepages=1024
`

var _ = Describe("Machine Config", func() {
	It("should generate yaml with expected parameters", func() {
		profile := testutils.NewPerformanceProfile("test")
		profile.Spec.HugePages.Pages = append(profile.Spec.HugePages.Pages, performancev1alpha1.HugePage{
			Count: 1024,
			Size:  "2M",
		})
		mc, err := New(testAssetsDir, profile)
		Expect(err).ToNot(HaveOccurred())

		y, err := yaml.Marshal(mc)
		Expect(err).ToNot(HaveOccurred())

		manifest := string(y)
		Expect(manifest).To(ContainSubstring(fmt.Sprintf("%s: %s", components.LabelMachineConfigurationRole, components.RoleWorkerPerformance)))
		Expect(manifest).To(ContainSubstring(expectedSystemdUnits))
		Expect(manifest).To(ContainSubstring(expectedBootArguments))
	})
})
