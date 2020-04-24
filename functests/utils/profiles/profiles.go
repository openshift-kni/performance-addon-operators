package profiles

import (
	"context"
	"fmt"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	performancev1alpha1 "github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetByNodeLabels gets the performance profile that must have node selector equals to passed node labels
func GetByNodeLabels(c client.Client, nodeLabels map[string]string) (*performancev1alpha1.PerformanceProfile, error) {
	profiles := &performancev1alpha1.PerformanceProfileList{}
	if err := c.List(context.TODO(), profiles); err != nil {
		return nil, err
	}

	var result *performancev1alpha1.PerformanceProfile
	for _, profile := range profiles.Items {
		if reflect.DeepEqual(profile.Spec.NodeSelector, nodeLabels) {
			if result != nil {
				return nil, fmt.Errorf("found more than one performance profile with specified node selector %v", nodeLabels)
			}
			result = &profile
		}
	}

	if result == nil {
		return nil, fmt.Errorf("failed to find performance profile with specified node selector %v", nodeLabels)
	}

	return result, nil
}

// NewProfile creates new performance profile based on the Name and Spec parameters
func NewProfile(profileName string, spec performancev1alpha1.PerformanceProfileSpec) *performancev1alpha1.PerformanceProfile {
	return &performancev1alpha1.PerformanceProfile{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "performance.openshift.io/v1alpha1",
			Kind:       "PerformanceProfile",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: profileName,
		},
		Spec: spec,
	}
}
