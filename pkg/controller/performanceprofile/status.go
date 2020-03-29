package performanceprofile

import (
	"bytes"
	"context"
	"reflect"
	"time"

	performancev1alpha1 "github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v1alpha1"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	mcov1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

const (
	conditionReasonValidationFailed         = "validation failed"
	conditionReasonComponentsCreationFailed = "failed to create components"
)

func (r *ReconcilePerformanceProfile) updateStatus(profile *performancev1alpha1.PerformanceProfile, conditions []conditionsv1.Condition) error {
	profileCopy := profile.DeepCopy()

	if conditions != nil {
		profileCopy.Status.Conditions = conditions
	}

	// check if we need to update the status
	modified := false

	// since we always set the same four conditions, we don't need to check if we need to remove old conditions
	for _, newCondition := range profileCopy.Status.Conditions {
		oldCondition := conditionsv1.FindStatusCondition(profile.Status.Conditions, newCondition.Type)
		if oldCondition == nil {
			modified = true
			break
		}

		// ignore timestamps to avoid infinite reconcile loops
		if oldCondition.Status != newCondition.Status ||
			oldCondition.Reason != newCondition.Reason ||
			oldCondition.Message != newCondition.Message {

			modified = true
			break
		}
	}

	// Note: add checks for new status fields when added

	if !modified {
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
			LastHeartbeatTime:  metav1.Time{Time: now},
		},
		{
			Type:               conditionsv1.ConditionUpgradeable,
			Status:             corev1.ConditionTrue,
			LastTransitionTime: metav1.Time{Time: now},
			LastHeartbeatTime:  metav1.Time{Time: now},
		},
		{
			Type:               conditionsv1.ConditionProgressing,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
			LastHeartbeatTime:  metav1.Time{Time: now},
		},
		{
			Type:               conditionsv1.ConditionDegraded,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
			LastHeartbeatTime:  metav1.Time{Time: now},
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
			LastHeartbeatTime:  metav1.Time{Time: now},
		},
		{
			Type:               conditionsv1.ConditionUpgradeable,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
			LastHeartbeatTime:  metav1.Time{Time: now},
		},
		{
			Type:               conditionsv1.ConditionProgressing,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
			LastHeartbeatTime:  metav1.Time{Time: now},
		},
		{
			Type:               conditionsv1.ConditionDegraded,
			Status:             corev1.ConditionTrue,
			LastTransitionTime: metav1.Time{Time: now},
			LastHeartbeatTime:  metav1.Time{Time: now},
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

func (r *ReconcilePerformanceProfile) getConditionsByMCPStatus(profile *performancev1alpha1.PerformanceProfile) []conditionsv1.Condition {

	mcpList := &mcov1.MachineConfigPoolList{}
	err := r.client.List(context.TODO(), mcpList)
	if err != nil {
		return nil
	}

	reason := bytes.Buffer{}
	reason.WriteString("Matching Machine Config Pools: ")

	message := bytes.Buffer{}

	for _, mcp := range mcpList.Items {
		if reflect.DeepEqual(profile.Spec.MachineConfigPoolSelector, mcp.Spec.MachineConfigSelector.MatchLabels) {
			for _, condition := range mcp.Status.Conditions {
				if condition.Type == mcov1.MachineConfigPoolDegraded && condition.Status == corev1.ConditionTrue {
					reason.WriteString(mcp.GetName() + " ")
					message.WriteString(mcp.GetName() + " Reason: " + condition.Reason + " \n")
					message.WriteString(mcp.GetName() + " Message: " + condition.Message + " \n")

				}
			}
		}
	}
	reason.WriteString("are in a Degraded state.")
	messageString := message.String()

	if len(messageString) == 0 {
		return nil
	}

	reasonString := reason.String()
	return r.getDegradedConditions(reasonString, messageString)
}
