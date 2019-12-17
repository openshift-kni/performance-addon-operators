package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CPUPerformanceProfileSpec defines the desired state of CPUPerformanceProfile
type CPUPerformanceProfileSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// CPUPerformanceProfileStatus defines the observed state of CPUPerformanceProfile
type CPUPerformanceProfileStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CPUPerformanceProfile is the Schema for the cpuperformanceprofiles API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=cpuperformanceprofiles,scope=Namespaced
type CPUPerformanceProfile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CPUPerformanceProfileSpec   `json:"spec,omitempty"`
	Status CPUPerformanceProfileStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CPUPerformanceProfileList contains a list of CPUPerformanceProfile
type CPUPerformanceProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CPUPerformanceProfile `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CPUPerformanceProfile{}, &CPUPerformanceProfileList{})
}
