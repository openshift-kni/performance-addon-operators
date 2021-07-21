package kubeletconfig

import (
	"encoding/json"
	"time"

	performancev2 "github.com/openshift-kni/performance-addon-operators/api/v2"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"
	profile2 "github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components/profile"
	machineconfigv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"
	kubefeatures "k8s.io/kubernetes/pkg/features"
)

const (
	cpuManagerPolicyStatic              = "static"
	cpuManagerPolicyOptionFullPCPUsOnly = "full-pcpus-only"

	defaultKubeReservedCPU      = "1000m"
	defaultKubeReservedMemory   = "500Mi"
	defaultSystemReservedCPU    = "1000m"
	defaultSystemReservedMemory = "500Mi"
)

// New returns new KubeletConfig object for performance sensetive workflows
func New(profile *performancev2.PerformanceProfile) (*machineconfigv1.KubeletConfig, error) {
	name := components.GetComponentName(profile.Name, components.ComponentNamePrefix)
	kubeletConfig := &kubeletconfigv1beta1.KubeletConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubelet.config.k8s.io/v1beta1",
			Kind:       "KubeletConfiguration",
		},
		CPUManagerPolicy:          cpuManagerPolicyStatic,
		CPUManagerReconcilePeriod: metav1.Duration{Duration: 5 * time.Second},
		TopologyManagerPolicy:     kubeletconfigv1beta1.BestEffortTopologyManagerPolicy,
		KubeReserved: map[string]string{
			"cpu":    defaultKubeReservedCPU,
			"memory": defaultKubeReservedMemory,
		},
		SystemReserved: map[string]string{
			"cpu":    defaultSystemReservedCPU,
			"memory": defaultSystemReservedMemory,
		},
		FeatureGates: map[string]bool{
			kubefeatures.CPUManagerPolicyOptions: true,
		},
	}

	if profile.Spec.CPU != nil && profile.Spec.CPU.Reserved != nil {
		kubeletConfig.ReservedSystemCPUs = string(*profile.Spec.CPU.Reserved)
	}

	if isPCPUIsolationEnabled(profile) {
		kubeletConfig.CPUManagerPolicyOptions = map[string]string{
			cpuManagerPolicyOptionFullPCPUsOnly: "true",
		}
	}

	if profile.Spec.NUMA != nil {
		if profile.Spec.NUMA.TopologyPolicy != nil {
			kubeletConfig.TopologyManagerPolicy = string(*profile.Spec.NUMA.TopologyPolicy)
		}
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

func isPCPUIsolationEnabled(profile *performancev2.PerformanceProfile) bool {
	if profile.Spec.CPU == nil || profile.Spec.CPU.DisablePCPUIsolation == nil {
		// default if not specified
		return true
	}
	if *profile.Spec.CPU.DisablePCPUIsolation {
		// explicitely disabled per user request
		return false
	}
	return true
}
