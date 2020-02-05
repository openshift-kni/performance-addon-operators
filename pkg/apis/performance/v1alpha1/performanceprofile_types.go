package v1alpha1

import (
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PerformanceProfilePauseAnnotation allows an admin to suspend the operator's
// reconcile loop in order to perform manual changes to performance profile owned
// objects.
const PerformanceProfilePauseAnnotation = "performance.openshift.io/pause-reconcile"

// PerformanceProfileSpec defines the desired state of PerformanceProfile.
type PerformanceProfileSpec struct {
	// CPU defines set of CPU related parameters.
	CPU *CPU `json:"cpu,omitempty"`
	// HugePages defines set of huge pages related parameters.
	HugePages *HugePages `json:"hugepages,omitempty"`
	// MachineConfigLabel defines the label to add to the MachineConfigs the operator creates. It has to be
	// used in the MachineConfigSelector of the MachineConfigPool which targets this performance profile.
	// Defaults to "machineconfiguration.openshift.io/role=<same role as in NodeSelector label key>"
	// +optional
	MachineConfigLabel map[string]string `json:"machineConfigLabel,omitempty"`
	// MachineConfigPoolSelector defines the MachineConfigPool label to use in the MachineConfigPoolSelector
	// of resources like KubeletConfigs created by the operator.
	// Defaults to "machineconfiguration.openshift.io/role=<same role as in NodeSelector label key>"
	// +optional
	MachineConfigPoolSelector map[string]string `json:"machineConfigPoolSelector,omitempty"`
	// NodeSelector defines the Node label to use in the NodeSelectors of resources like Tuned created by the operator.
	// It most likely should, but does not have to match the node label in the NodeSelector of the MachineConfigPool
	// which targets this performance profile.
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
	// +optional
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
	Size HugePageSize `json:"size,omitempty"`
	// Count defines amount of huge pages, maps to the 'hugepages' kernel boot parameter.
	Count int32 `json:"count,omitempty"`
	// Node defines the NUMA node where hugepages will be allocated,
	// if not specified, pages will be allocated equally between NUMA nodes
	// +optional
	Node *int32 `json:"node,omitempty"`
}

// RealTimeKernel defines the set of parameters relevant for the real time kernel.
type RealTimeKernel struct {
	// Enabled defines if the real time kernel packages should be installed
	Enabled *bool `json:"enabled,omitempty"`
}

// PerformanceProfileStatus defines the observed state of PerformanceProfile.
type PerformanceProfileStatus struct {
	// conditions represents the latest available observations of current state.
	// +optional
	Conditions []conditionsv1.Condition `json:"conditions,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PerformanceProfile is the Schema for the performanceprofiles API.
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=performanceprofiles,scope=Cluster
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
