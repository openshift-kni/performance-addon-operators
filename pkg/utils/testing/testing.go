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
	// HugePagesCount defines the huge page count used for tests
	HugePagesCount = 4
	// IsolatedCPUs defines the isolated CPU set used for tests
	IsolatedCPUs = performancev1alpha1.CPUSet("4-7")
	// ReservedCPUs defines the reserved CPU set used for tests
	ReservedCPUs = performancev1alpha1.CPUSet("0-3")
	// SingleNUMAPolicy defines the topologyManager policy used for tests
	SingleNUMAPolicy = "single-numa-node"

	//MachineConfigLabelKey defines the MachineConfig label key of the test profile
	MachineConfigLabelKey = "mcKey"
	//MachineConfigLabelValue defines the MachineConfig label vlue of the test profile
	MachineConfigLabelValue = "mcValue"
	//MachineConfigPoolLabelKey defines the MachineConfigPool label key of the test profile
	MachineConfigPoolLabelKey = "mcpKey"
	//MachineConfigPoolLabelValue defines the MachineConfigPool label value of the test profile
	MachineConfigPoolLabelValue = "mcpValue"
)

// NewPerformanceProfile returns new performance profile object that used for tests
func NewPerformanceProfile(name string) *performancev1alpha1.PerformanceProfile {
	size := HugePageSize
	isolatedCPUs := IsolatedCPUs
	reservedCPUs := ReservedCPUs
	numaPolicy := SingleNUMAPolicy

	return &performancev1alpha1.PerformanceProfile{
		TypeMeta: metav1.TypeMeta{Kind: "PerformanceProfile"},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			UID:  types.UID("11111111-1111-1111-1111-1111111111111"),
		},
		Spec: performancev1alpha1.PerformanceProfileSpec{
			CPU: &performancev1alpha1.CPU{
				Isolated: &isolatedCPUs,
				Reserved: &reservedCPUs,
			},
			HugePages: &performancev1alpha1.HugePages{
				DefaultHugePagesSize: &size,
				Pages: []performancev1alpha1.HugePage{
					{
						Count: HugePagesCount,
						Size:  size,
					},
				},
			},
			RealTimeKernel: &performancev1alpha1.RealTimeKernel{
				Enabled: pointer.BoolPtr(true),
			},
			NUMA: &performancev1alpha1.NUMA{
				TopologyPolicy: &numaPolicy,
			},
			MachineConfigLabel: map[string]string{
				MachineConfigLabelKey: MachineConfigLabelValue,
			},
			MachineConfigPoolSelector: map[string]string{
				MachineConfigPoolLabelKey: MachineConfigPoolLabelValue,
			},
			NodeSelector: map[string]string{
				"nodekey": "nodeValue",
			},
		},
	}
}
