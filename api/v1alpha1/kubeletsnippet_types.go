/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubelet/config/v1beta1"
)

// KubeletSnippetSpec defines the desired state of KubeletSnippet.
type KubeletSnippetSpec struct {
	// AdditionalKubeletArguments gives you a way to provide a KubeletConfig snippet with additional
	// configurations you want to apply on top of the machine.
	// To find the specific argument see https://kubernetes.io/docs/reference/config-api/kubelet-config.v1beta1/.
	// By default, the performance-addon-operator will override:
	// 1. CPU manager policy
	// 2. CPU manager reconcile period
	// 3. Topology manager policy
	// 4. Reserved CPUs
	// 5. Memory manager policy
	// 6. Reserved Memory
	// Please avoid specifying them and use the relevant API to configure these parameters.
	AdditionalKubeletArguments *v1beta1.KubeletConfiguration `json:"additionalKubeletArguments,omitempty"`

	// PerformanceProfileName defines the performance profile name that should apply the additional kubelet arguments
	// specified under the KubeletSnippet.
	PerformanceProfileName string `json:"performanceProfileName"`
}

// KubeletSnippetStatus defines the observed state of KubeletSnippet.
type KubeletSnippetStatus struct {
	// Conditions represents the latest available observations of current state.
	// +optional
	Conditions []conditionsv1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=kubeletsnippets,scope=Cluster
// +kubebuilder:printcolumn:JSONPath=".spec.performanceProfileName",name=Profile,type=string

// KubeletSnippet is the Schema for the kubelet snippet API
type KubeletSnippet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KubeletSnippetSpec   `json:"spec,omitempty"`
	Status KubeletSnippetStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KubeletSnippetList contains a list of KubeletSnippets
type KubeletSnippetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KubeletSnippet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KubeletSnippet{}, &KubeletSnippetList{})
}
