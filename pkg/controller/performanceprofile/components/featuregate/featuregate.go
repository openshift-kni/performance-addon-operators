package featuregate

import (
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"
	configv1 "github.com/openshift/api/config/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewLatencySensitive returns new latency sensetive feature gate object
func NewLatencySensitive() *configv1.FeatureGate {
	return &configv1.FeatureGate{
		TypeMeta: metav1.TypeMeta{
			APIVersion: configv1.GroupVersion.String(),
			Kind:       "FeatureGate",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: components.FeatureGateLatencySensetiveName,
		},
		Spec: configv1.FeatureGateSpec{
			FeatureGateSelection: configv1.FeatureGateSelection{
				FeatureSet: configv1.LatencySensitive,
			},
		},
	}
}
