package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PerformanceProfileSpec defines the desired state of PerformanceProfile.
type PerformanceProfileSpec struct {
	// CPU defines set of CPU related parameters.
	CPU *CPU `json:"cpu,omitempty"`
	// HugePages defines set of huge pages related parameters.
	HugePages *HugePages `json:"hugepages,omitempty"`
	// NodeSelector is a selector which must be true for the performance profile to fit on a node.
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// RealTimeKernel defines set of real time kernel related parameters.
	RealTimeKernel *RealTimeKernel `json:"realTimeKernel,omitempty"`
}

// CPUSet defines the set of CPU's(0-3,8-11).
type CPUSet string

// CPU defines set of CPU related features.
type CPU struct {
	// Reserved defines set of CPU's that will not be used for any container workloads initiated by kubelet.
	Reserved *CPUSet `json:"reserved,omitempty"`
	// Isolated defines set of CPU's that will used to give to application threads the most execution time possible,
	// which means removing as many extraneous tasks off a CPU as possible.
	Isolated *CPUSet `json:"isolated,omitempty"`
	// NonIsolated defines set of CPU's that will be used for OS tasks, like serving interupts or workqueues.
	NonIsolated *CPUSet `json:"nonIsolated,omitempty"`
}

// HugePageSize defines size of huge pages, can be 2M or 1G.
type HugePageSize string

// HugePages defines set of huge pages that we want to allocate on the boot.
type HugePages struct {
	// DefaultHugePagesSize defines huge pages default size under kernel boot parameters.
	DefaultHugePagesSize *HugePageSize `json:"defaultHugepagesSize,omitempty"`
	// Pages defines huge pages that we want to allocate on the boot time.
	Pages []HugePage `json:"pages,omitempty"`
}

// HugePage defines the number of allocated huge pages of the specific size.
type HugePage struct {
	// Size defines huge page size, maps to the 'hugepagesz' kernel boot parameter.
	Size *HugePageSize `json:"size,omitempty"`
	// Count defines amount of huge pages, maps to the 'hugepages' kernel boot parameter.
	Count *int32 `json:"count,omitempty"`
}

// RealTimeKernel defines the set of parameters relevant for the real time kernel.
type RealTimeKernel struct {
	// Enabled enables real time kernel on relevant nodes.
	Enabled *bool `json:"enabled,omitempty"`
}

// PerformanceProfileStatus defines the observed state of PerformanceProfile.
type PerformanceProfileStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PerformanceProfile is the Schema for the performanceprofiles API.
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=performanceprofiles,scope=Namespaced
type PerformanceProfile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PerformanceProfileSpec   `json:"spec,omitempty"`
	Status PerformanceProfileStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PerformanceProfileList contains a list of PerformanceProfile.
type PerformanceProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PerformanceProfile `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PerformanceProfile{}, &PerformanceProfileList{})
}
