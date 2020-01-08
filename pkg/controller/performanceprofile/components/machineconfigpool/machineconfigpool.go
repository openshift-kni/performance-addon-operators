package machineconfigpool

import (
	performancev1alpha1 "github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v1alpha1"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"
	machineconfigv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// New returns new machine config pool for performance sensitive workflows
func New(profile *performancev1alpha1.PerformanceProfile) *machineconfigv1.MachineConfigPool {
	name := components.GetComponentName(profile.Name, components.RoleWorkerPerformance)
	return &machineconfigv1.MachineConfigPool{
		TypeMeta: metav1.TypeMeta{
			APIVersion: machineconfigv1.GroupVersion.String(),
			Kind:       "MachineConfigPool",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				components.LabelMachineConfigPoolRole: name,
			},
		},
		Spec: machineconfigv1.MachineConfigPoolSpec{
			MachineConfigSelector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      components.LabelMachineConfigurationRole,
						Operator: metav1.LabelSelectorOpIn,
						Values:   []string{components.RoleWorker, name},
					},
				},
			},
			NodeSelector: &metav1.LabelSelector{
				MatchLabels: profile.Spec.NodeSelector,
			},
		},
	}
}
