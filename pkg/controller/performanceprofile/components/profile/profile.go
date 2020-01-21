package profile

import (
	"fmt"

	"github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v1alpha1"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"
)

// ValidateParameters validates parameters of the given profile
func ValidateParameters(profile v1alpha1.PerformanceProfile) error {

	if profile.Spec.CPU == nil {
		return fmt.Errorf("you should provide CPU section")
	}

	if profile.Spec.CPU.Isolated == nil {
		return fmt.Errorf("you should provide isolated CPU set")
	}

	if profile.Spec.CPU.NonIsolated == nil {
		return fmt.Errorf("you should provide non isolated CPU set")
	}

	if profile.Spec.MachineConfigLabel != nil && len(profile.Spec.MachineConfigLabel) > 1 {
		return fmt.Errorf("you should provide only 1 MachineConfigLabel")
	}

	if profile.Spec.MachineConfigPoolSelector != nil && len(profile.Spec.MachineConfigPoolSelector) > 1 {
		return fmt.Errorf("you should provide onlyt 1 MachineConfigPoolSelector")
	}

	if profile.Spec.NodeSelector == nil {
		return fmt.Errorf("you should provide NodeSelector")
	}
	if len(profile.Spec.NodeSelector) > 1 {
		return fmt.Errorf("you should provide ony 1 NodeSelector")
	}

	// in case MachineConfigLabels or MachineConfigPoolSelector are not set, we expect a certain format (domain/role)
	// on the NodeSelector in order to be able to calculate the default values for the former metioned fields.
	if profile.Spec.MachineConfigLabel == nil || profile.Spec.MachineConfigPoolSelector == nil {
		k, _ := components.GetFirstKeyAndValue(profile.Spec.NodeSelector)
		if _, _, err := components.SplitLabelKey(k); err != nil {
			return fmt.Errorf("invalid NodeSelector label key, can't be split into domain/role")
		}
	}

	// TODO add validation for MachineConfigLabels and MachineConfigPoolSelector if they are not set
	// by checking if a MCP with our default values exists

	return nil
}

// GetMachineConfigPoolSelector returns the MachineConfigPoolSelector from the CR or a default value calculated based on NodeSelector
func GetMachineConfigPoolSelector(profile v1alpha1.PerformanceProfile) map[string]string {
	if profile.Spec.MachineConfigPoolSelector != nil {
		return profile.Spec.MachineConfigPoolSelector
	}

	return getDefaultLabel(profile)
}

// GetMachineConfigLabel returns the MachineConfigLabels from the CR or a default value calculated based on NodeSelector
func GetMachineConfigLabel(profile v1alpha1.PerformanceProfile) map[string]string {
	if profile.Spec.MachineConfigLabel != nil {
		return profile.Spec.MachineConfigLabel
	}

	return getDefaultLabel(profile)
}

func getDefaultLabel(profile v1alpha1.PerformanceProfile) map[string]string {
	nodeSelectorKey, _ := components.GetFirstKeyAndValue(profile.Spec.NodeSelector)
	// no error handling needed, it's validated already
	_, nodeRole, _ := components.SplitLabelKey(nodeSelectorKey)

	labels := make(map[string]string)
	labels[components.MachineConfigRoleLabelKey] = nodeRole

	return labels
}

// IsPaused returns whether or not a performance profile's reconcile loop is paused
func IsPaused(profile *v1alpha1.PerformanceProfile) bool {

	if profile.Annotations == nil {
		return false
	}

	isPaused, ok := profile.Annotations[v1alpha1.PerformanceProfilePauseAnnotation]
	if ok && isPaused == "true" {
		return true
	}

	return false
}
