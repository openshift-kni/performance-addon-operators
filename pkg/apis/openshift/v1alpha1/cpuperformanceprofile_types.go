package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CpuPerformanceProfileSpec defines the desired state of CpuPerformanceProfile
// +k8s:openapi-gen=true
type CpuPerformanceProfileSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// CpuPerformanceProfileStatus defines the observed state of CpuPerformanceProfile
// +k8s:openapi-gen=true
type CpuPerformanceProfileStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CpuPerformanceProfile is the Schema for the cpuperformanceprofiles API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=cpuperformanceprofiles,scope=Namespaced
type CpuPerformanceProfile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CpuPerformanceProfileSpec   `json:"spec,omitempty"`
	Status CpuPerformanceProfileStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CpuPerformanceProfileList contains a list of CpuPerformanceProfile
type CpuPerformanceProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CpuPerformanceProfile `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CpuPerformanceProfile{}, &CpuPerformanceProfileList{})
}
