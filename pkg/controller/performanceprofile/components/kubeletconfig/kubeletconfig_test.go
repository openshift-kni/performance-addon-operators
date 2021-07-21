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
		selectorKey, selectorValue := components.GetFirstKeyAndValue(profile.Spec.MachineConfigPoolSelector)
		kc, err := New(profile, map[string]string{selectorKey: selectorValue})
		Expect(err).ToNot(HaveOccurred())

		y, err := yaml.Marshal(kc)
		Expect(err).ToNot(HaveOccurred())

		manifest := string(y)

		Expect(manifest).To(ContainSubstring(fmt.Sprintf("%s: %s", selectorKey, selectorValue)))
		Expect(manifest).To(ContainSubstring("reservedSystemCPUs: 0-3"))
		Expect(manifest).To(ContainSubstring("topologyManagerPolicy: single-numa-node"))
		Expect(manifest).To(ContainSubstring("cpuManagerPolicy: static"))
	})
})
