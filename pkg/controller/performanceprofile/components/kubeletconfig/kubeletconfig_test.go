package kubeletconfig

import (
	"fmt"
	"time"

	"github.com/openshift-kni/performance-addon-operators/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ghodss/yaml"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"
	testutils "github.com/openshift-kni/performance-addon-operators/pkg/utils/testing"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"
	"k8s.io/utils/pointer"
)

const testReservedMemory = `reservedMemory:
    - limits:
        memory: 1100Mi
      numaNode: 0`

var _ = Describe("Kubelet Config", func() {
	It("should generate yaml with expected parameters", func() {
		profile := testutils.NewPerformanceProfile("test")
		selectorKey, selectorValue := components.GetFirstKeyAndValue(profile.Spec.MachineConfigPoolSelector)
		kc, err := New(profile, map[string]string{selectorKey: selectorValue}, nil)
		Expect(err).ToNot(HaveOccurred())

		y, err := yaml.Marshal(kc)
		Expect(err).ToNot(HaveOccurred())

		manifest := string(y)

		Expect(manifest).To(ContainSubstring(fmt.Sprintf("%s: %s", selectorKey, selectorValue)))
		Expect(manifest).To(ContainSubstring("reservedSystemCPUs: 0-3"))
		Expect(manifest).To(ContainSubstring("topologyManagerPolicy: single-numa-node"))
		Expect(manifest).To(ContainSubstring("cpuManagerPolicy: static"))
		Expect(manifest).To(ContainSubstring("memoryManagerPolicy: Static"))
		Expect(manifest).To(ContainSubstring(testReservedMemory))
	})

	Context("with topology manager restricted policy", func() {
		It("should have the memory manager related parameters", func() {
			profile := testutils.NewPerformanceProfile("test")
			profile.Spec.NUMA.TopologyPolicy = pointer.String(kubeletconfigv1beta1.RestrictedTopologyManagerPolicy)
			selectorKey, selectorValue := components.GetFirstKeyAndValue(profile.Spec.MachineConfigPoolSelector)
			kc, err := New(profile, map[string]string{selectorKey: selectorValue}, nil)
			Expect(err).ToNot(HaveOccurred())

			y, err := yaml.Marshal(kc)
			Expect(err).ToNot(HaveOccurred())

			manifest := string(y)
			Expect(manifest).To(ContainSubstring("memoryManagerPolicy: Static"))
			Expect(manifest).To(ContainSubstring(testReservedMemory))
		})
	})

	Context("with topology manager best-effort policy", func() {
		It("should not have the memory manager related parameters", func() {
			profile := testutils.NewPerformanceProfile("test")
			profile.Spec.NUMA.TopologyPolicy = pointer.String(kubeletconfigv1beta1.BestEffortTopologyManagerPolicy)
			selectorKey, selectorValue := components.GetFirstKeyAndValue(profile.Spec.MachineConfigPoolSelector)
			kc, err := New(profile, map[string]string{selectorKey: selectorValue}, nil)
			Expect(err).ToNot(HaveOccurred())

			y, err := yaml.Marshal(kc)
			Expect(err).ToNot(HaveOccurred())

			manifest := string(y)
			Expect(manifest).ToNot(ContainSubstring("memoryManagerPolicy: Static"))
			Expect(manifest).ToNot(ContainSubstring(testReservedMemory))
		})
	})

	Context("with additional kubelet arguments", func() {
		It("should not override CPU manager parameters", func() {
			profile := testutils.NewPerformanceProfile("test")
			selectorKey, selectorValue := components.GetFirstKeyAndValue(profile.Spec.MachineConfigPoolSelector)
			kc, err := New(profile, map[string]string{selectorKey: selectorValue}, &v1alpha1.KubeletSnippet{
				Spec: v1alpha1.KubeletSnippetSpec{
					AdditionalKubeletArguments: &kubeletconfigv1beta1.KubeletConfiguration{
						CPUManagerPolicy:          "none",
						CPUManagerReconcilePeriod: metav1.Duration{Duration: 10 * time.Second},
						ReservedSystemCPUs:        "4,5",
					},
					PerformanceProfileName: profile.Name,
				},
			})
			y, err := yaml.Marshal(kc)
			Expect(err).ToNot(HaveOccurred())

			manifest := string(y)
			Expect(manifest).ToNot(ContainSubstring("cpuManagerPolicy: none"))
			Expect(manifest).ToNot(ContainSubstring("cpuManagerReconcilePeriod: 10s"))
			Expect(manifest).ToNot(ContainSubstring("reservedSystemCPUs: 4-5"))
		})

		It("should not override topology manager parameters", func() {
			profile := testutils.NewPerformanceProfile("test")
			selectorKey, selectorValue := components.GetFirstKeyAndValue(profile.Spec.MachineConfigPoolSelector)
			kc, err := New(profile, map[string]string{selectorKey: selectorValue}, &v1alpha1.KubeletSnippet{
				Spec: v1alpha1.KubeletSnippetSpec{
					AdditionalKubeletArguments: &kubeletconfigv1beta1.KubeletConfiguration{
						TopologyManagerPolicy: "none",
					},
					PerformanceProfileName: profile.Name,
				},
			})
			y, err := yaml.Marshal(kc)
			Expect(err).ToNot(HaveOccurred())

			manifest := string(y)
			Expect(manifest).ToNot(ContainSubstring("topologyManagerPolicy: none"))
		})

		It("should not override memory manager policy", func() {
			profile := testutils.NewPerformanceProfile("test")
			memoryQuantity := resource.NewQuantity(1024, resource.DecimalSI)

			selectorKey, selectorValue := components.GetFirstKeyAndValue(profile.Spec.MachineConfigPoolSelector)
			kc, err := New(profile, map[string]string{selectorKey: selectorValue}, &v1alpha1.KubeletSnippet{
				Spec: v1alpha1.KubeletSnippetSpec{
					AdditionalKubeletArguments: &kubeletconfigv1beta1.KubeletConfiguration{
						MemoryManagerPolicy: "None",
						ReservedMemory: []kubeletconfigv1beta1.MemoryReservation{
							{
								NumaNode: 10,
								Limits: corev1.ResourceList{
									"test": *memoryQuantity,
								},
							},
						},
					},
					PerformanceProfileName: profile.Name,
				},
			})
			y, err := yaml.Marshal(kc)
			Expect(err).ToNot(HaveOccurred())

			manifest := string(y)
			Expect(manifest).ToNot(ContainSubstring("memoryManagerPolicy: None"))
			Expect(manifest).ToNot(ContainSubstring("numaNode: 10"))
		})

		It("should set the kubelet config accordingly", func() {
			profile := testutils.NewPerformanceProfile("test")
			selectorKey, selectorValue := components.GetFirstKeyAndValue(profile.Spec.MachineConfigPoolSelector)
			kc, err := New(profile, map[string]string{selectorKey: selectorValue}, &v1alpha1.KubeletSnippet{
				Spec: v1alpha1.KubeletSnippetSpec{
					AdditionalKubeletArguments: &kubeletconfigv1beta1.KubeletConfiguration{
						AllowedUnsafeSysctls: []string{"net.core.somaxconn"},
						EvictionHard:         map[string]string{evictionHardMemoryAvailable: "200Mi"},
					},
					PerformanceProfileName: profile.Name,
				},
			})
			y, err := yaml.Marshal(kc)
			Expect(err).ToNot(HaveOccurred())

			manifest := string(y)
			Expect(manifest).To(ContainSubstring("net.core.somaxconn"))
			Expect(manifest).To(ContainSubstring("memory.available: 200Mi"))
		})
	})
})
