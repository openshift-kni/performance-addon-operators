package performanceprofile

import (
	"context"
	"reflect"
	"time"

	performancev1alpha1 "github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v1alpha1"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	mcov1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

const (
	conditionReasonValidationFailed         = "validation failed"
	conditionReasonComponentsCreationFailed = "failed to create components"
)

func (r *ReconcilePerformanceProfile) updateStatus(profile *performancev1alpha1.PerformanceProfile, conditions []conditionsv1.Condition) error {
	// TODO: once we will have tuned resource status we will need to merge output from the machine-config-pool status
	// and tuned status
	profileCopy := profile.DeepCopy()

	name := components.GetComponentName(profile.Name, components.RoleWorkerPerformance)
	mcp, err := r.getMachineConfigPool(name)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	if errors.IsNotFound(err) {
		profileCopy.Status.MachineCount = 0
		profileCopy.Status.UpdatedMachineCount = 0
		profileCopy.Status.UnavailableMachineCount = 0
		profileCopy.Status.Conditions = conditions
	} else {
		profileCopy.Status.MachineCount = mcp.Status.MachineCount
		profileCopy.Status.UpdatedMachineCount = mcp.Status.UpdatedMachineCount
		profileCopy.Status.UnavailableMachineCount = mcp.Status.UnavailableMachineCount
		if conditions != nil {
			profileCopy.Status.Conditions = conditions
		} else {
			if mcov1.IsMachineConfigPoolConditionTrue(mcp.Status.Conditions, mcov1.MachineConfigPoolUpdated) {
				profileCopy.Status.Conditions = r.getAvailableConditions()
			}

			if mcov1.IsMachineConfigPoolConditionTrue(mcp.Status.Conditions, mcov1.MachineConfigPoolUpdating) {
				updatingCondition := mcov1.GetMachineConfigPoolCondition(mcp.Status, mcov1.MachineConfigPoolUpdating)
				profileCopy.Status.Conditions = r.getProgressingConditions(updatingCondition.Reason, updatingCondition.Message)
			}

			if mcov1.IsMachineConfigPoolConditionTrue(mcp.Status.Conditions, mcov1.MachineConfigPoolDegraded) {
				degradedCondition := mcov1.GetMachineConfigPoolCondition(mcp.Status, mcov1.MachineConfigPoolUpdating)
				profileCopy.Status.Conditions = r.getDegradedConditions(degradedCondition.Reason, degradedCondition.Message)
			}
		}
	}

	if reflect.DeepEqual(profile.Status, profileCopy.Status) {
		return nil
	}

	klog.Infof("Updating the performance profile %q status", profile.Name)
	return r.client.Status().Update(context.TODO(), profileCopy)
}

func (r *ReconcilePerformanceProfile) getAvailableConditions() []conditionsv1.Condition {
	now := time.Now()
	return []conditionsv1.Condition{
		{
			Type:               conditionsv1.ConditionAvailable,
			Status:             corev1.ConditionTrue,
			LastTransitionTime: metav1.Time{Time: now},
		},
		{
			Type:               conditionsv1.ConditionUpgradeable,
			Status:             corev1.ConditionTrue,
			LastTransitionTime: metav1.Time{Time: now},
		},
		{
			Type:               conditionsv1.ConditionProgressing,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
		},
		{
			Type:               conditionsv1.ConditionDegraded,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
		},
	}
}

func (r *ReconcilePerformanceProfile) getDegradedConditions(reason string, message string) []conditionsv1.Condition {
	now := time.Now()
	return []conditionsv1.Condition{
		{
			Type:               conditionsv1.ConditionAvailable,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
		},
		{
			Type:               conditionsv1.ConditionUpgradeable,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
		},
		{
			Type:               conditionsv1.ConditionProgressing,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
		},
		{
			Type:               conditionsv1.ConditionDegraded,
			Status:             corev1.ConditionTrue,
			LastTransitionTime: metav1.Time{Time: now},
			Reason:             reason,
			Message:            message,
		},
	}
}

func (r *ReconcilePerformanceProfile) getProgressingConditions(reason string, message string) []conditionsv1.Condition {
	now := time.Now()

	return []conditionsv1.Condition{
		{
			Type:               conditionsv1.ConditionAvailable,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
		},
		{
			Type:               conditionsv1.ConditionUpgradeable,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
		},
		{
			Type:               conditionsv1.ConditionProgressing,
			Status:             corev1.ConditionTrue,
			LastTransitionTime: metav1.Time{Time: now},
			Reason:             reason,
			Message:            message,
		},
		{
			Type:               conditionsv1.ConditionDegraded,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
		},
	}
}
