package kubeletconfig

import (
	"fmt"

	"github.com/ghodss/yaml"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"
	testutils "github.com/openshift-kni/performance-addon-operators/pkg/utils/testing"
)

var _ = Describe("Kubelet Config", func() {
	It("should generate yaml with expected parameters", func() {
		profile := testutils.NewPerformanceProfile("test")
		kc, err := New(profile)
		Expect(err).ToNot(HaveOccurred())

		y, err := yaml.Marshal(kc)
		Expect(err).ToNot(HaveOccurred())

		manifest := string(y)

		selectorKey, selectorValue := components.GetFirstKeyAndValue(profile.Spec.MachineConfigPoolSelector)
		Expect(manifest).To(ContainSubstring(fmt.Sprintf("%s: %s", selectorKey, selectorValue)))
		Expect(manifest).To(ContainSubstring("reservedSystemCPUs: 0-3"))
		Expect(manifest).To(ContainSubstring("topologyManagerPolicy: single-numa-node"))
		Expect(manifest).To(ContainSubstring("cpuManagerPolicy: static"))
	})

	Context("with additional kubelet arguments", func() {
		It("should not override CPU manager parameters", func() {
			profile := testutils.NewPerformanceProfile("test")
			profile.Annotations = map[string]string{
				experimentalKubeletSnippetAnnotation: `{"cpuManagerPolicy": "none", "cpuManagerReconcilePeriod": "10s", "reservedSystemCPUs": "4,5"}`,
			}
			kc, err := New(profile)
			y, err := yaml.Marshal(kc)
			Expect(err).ToNot(HaveOccurred())

			manifest := string(y)
			Expect(manifest).ToNot(ContainSubstring("cpuManagerPolicy: none"))
			Expect(manifest).ToNot(ContainSubstring("cpuManagerReconcilePeriod: 10s"))
			Expect(manifest).ToNot(ContainSubstring("reservedSystemCPUs: 4-5"))
		})

		It("should not override topology manager parameters", func() {
			profile := testutils.NewPerformanceProfile("test")
			profile.Annotations = map[string]string{
				experimentalKubeletSnippetAnnotation: `{"topologyManagerPolicy": "none"}`,
			}
			kc, err := New(profile)
			y, err := yaml.Marshal(kc)
			Expect(err).ToNot(HaveOccurred())

			manifest := string(y)
			Expect(manifest).ToNot(ContainSubstring("topologyManagerPolicy: none"))
		})

		It("should set the kubelet config accordingly", func() {
			profile := testutils.NewPerformanceProfile("test")
			profile.Annotations = map[string]string{
				experimentalKubeletSnippetAnnotation: `{"allowedUnsafeSysctls": ["net.core.somaxconn"], "evictionHard": {"memory.available": "200Mi"}}`,
			}
			kc, err := New(profile)
			y, err := yaml.Marshal(kc)
			Expect(err).ToNot(HaveOccurred())

			manifest := string(y)
			Expect(manifest).To(ContainSubstring("net.core.somaxconn"))
			Expect(manifest).To(ContainSubstring("memory.available: 200Mi"))
		})
	})
})
