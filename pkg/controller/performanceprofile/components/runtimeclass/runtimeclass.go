package runtimeclass

import (
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"
	pinfo "github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components/profileinfo"

	nodev1beta1 "k8s.io/api/node/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// New returns a new RuntimeClass object
func New(profile *pinfo.PerformanceProfileInfo, handler string) *nodev1beta1.RuntimeClass {
	name := components.GetComponentName(profile.Name, components.ComponentNamePrefix)
	return &nodev1beta1.RuntimeClass{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RuntimeClass",
			APIVersion: "node.k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Handler: handler,
		Scheduling: &nodev1beta1.Scheduling{
			NodeSelector: profile.Spec.NodeSelector,
		},
	}
}
