package kubeletconfig

import (
	"encoding/json"
	"time"

	performancev1alpha1 "github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v1alpha1"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"
	profile2 "github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components/profile"
	machineconfigv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"
)

const (
	cpuManagerPolicyStatic          = "static"
	defaultKubeReservedCPU          = "1000m"
	defaultKubeReservedMemory       = "500Mi"
	defaultSystemReservedCPU        = "1000m"
	defaultSystemReservedMemory     = "500Mi"
	topologyManagerPolicyBestEffort = "best-effort"
)

// New returns new KubeletConfig object for performance sensetive workflows
func New(profile *performancev1alpha1.PerformanceProfile) (*machineconfigv1.KubeletConfig, error) {
	name := components.GetComponentName(profile.Name, components.ComponentNamePrefix)
	kubeletConfig := &kubeletconfigv1beta1.KubeletConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubelet.config.k8s.io/v1beta1",
			Kind:       "KubeletConfiguration",
		},
		CPUManagerPolicy:          cpuManagerPolicyStatic,
		CPUManagerReconcilePeriod: metav1.Duration{Duration: 5 * time.Second},
		TopologyManagerPolicy:     topologyManagerPolicyBestEffort,
		KubeReserved: map[string]string{
			"cpu":    defaultKubeReservedCPU,
			"memory": defaultKubeReservedMemory,
		},
		SystemReserved: map[string]string{
			"cpu":    defaultSystemReservedCPU,
			"memory": defaultSystemReservedMemory,
		},
	}

	if profile.Spec.CPU != nil && profile.Spec.CPU.Reserved != nil {
		kubeletConfig.ReservedSystemCPUs = string(*profile.Spec.CPU.Reserved)
	}

	raw, err := json.Marshal(kubeletConfig)
	if err != nil {
		return nil, err
	}

	return &machineconfigv1.KubeletConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: machineconfigv1.GroupVersion.String(),
			Kind:       "KubeletConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: machineconfigv1.KubeletConfigSpec{
			MachineConfigPoolSelector: &metav1.LabelSelector{
				MatchLabels: profile2.GetMachineConfigPoolSelector(profile),
			},
			KubeletConfig: &runtime.RawExtension{
				Raw: raw,
			},
		},
	}, nil
}
