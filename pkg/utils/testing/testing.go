package testing

import (
	performancev1alpha1 "github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
)

const (
	// HugePageSize defines the huge page size used for tests
	HugePageSize = performancev1alpha1.HugePageSize("1G")
	// IsolatedCPUs defines the isolated CPU set used for tests
	IsolatedCPUs = performancev1alpha1.CPUSet("4-7")
	// NonIsolateCPUs defines the non-isolated CPU set used for tests
	NonIsolateCPUs = performancev1alpha1.CPUSet("2-3")
	// ReservedCPUs defines the reserved CPU set used for tests
	ReservedCPUs = performancev1alpha1.CPUSet("0-3")
)

// NewPerformanceProfile returns new performance profile object that used for tests
func NewPerformanceProfile(name string) *performancev1alpha1.PerformanceProfile {
	size := HugePageSize
	isolatedCPUs := IsolatedCPUs
	nonIsolateCPUs := NonIsolateCPUs
	reservedCPUs := ReservedCPUs

	return &performancev1alpha1.PerformanceProfile{
		TypeMeta: metav1.TypeMeta{Kind: "PerformanceProfile"},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			UID:  types.UID("11111111-1111-1111-1111-1111111111111"),
		},
		Spec: performancev1alpha1.PerformanceProfileSpec{
			CPU: &performancev1alpha1.CPU{
				Isolated:    &isolatedCPUs,
				NonIsolated: &nonIsolateCPUs,
				Reserved:    &reservedCPUs,
			},
			HugePages: &performancev1alpha1.HugePages{
				DefaultHugePagesSize: &size,
				Pages: []performancev1alpha1.HugePage{
					{
						Count: 4,
						Size:  size,
					},
				},
			},
			RealTimeKernel: &performancev1alpha1.RealTimeKernel{
				Enabled: pointer.BoolPtr(true),
			},
			MachineConfigLabels: map[string]string{
				"mclKey": "mclValue",
			},
			MachineConfigPoolSelector: map[string]string{
				"mcpkey": "mcpValue",
			},
			NodeSelector: map[string]string{
				"nodekey": "nodeValue",
			},
		},
	}
}

// GetFirstKeyAndValue return the first key / value pair of a map
func GetFirstKeyAndValue(m map[string]string) (key, value string) {
	for k, v := range m {
		key = k
		value = v
		break
	}
	return
}
