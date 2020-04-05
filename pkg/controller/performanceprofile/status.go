package performanceprofile

import (
	"bytes"
	"context"
	"time"

	performancev1alpha1 "github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v1alpha1"
	profileutil "github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components/profile"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	mcov1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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

func (r *ReconcilePerformanceProfile) getMCPConditionsByProfile(profile *performancev1alpha1.PerformanceProfile) ([]conditionsv1.Condition, error) {

	mcpList := &mcov1.MachineConfigPoolList{}

	if err := r.client.List(context.TODO(), mcpList); err != nil {
		klog.Errorf("Cannot list Machine config pools to match with profile %q : %v", profile.Name, err)
		return nil, err
	}

	mcpItems := removeMCPDuplicateEntries(mcpList.Items)

	performanceProfileLabels := labels.Set(profileutil.GetMachineConfigPoolSelector(profile))

	reason := bytes.Buffer{}
	reason.WriteString("Matching Machine Config Pools are in a degraded state")
	message := bytes.Buffer{}

	for _, mcp := range mcpItems {
		selector, err := metav1.LabelSelectorAsSelector(mcp.Spec.MachineConfigSelector)
		if err != nil {
			klog.Errorf("Cannot create label selector for %q : %v ", mcp.Name, err)
			return nil, err
		}
		if selector.Matches(performanceProfileLabels) {
			for _, condition := range mcp.Status.Conditions {
				if (condition.Type == mcov1.MachineConfigPoolNodeDegraded || condition.Type == mcov1.MachineConfigPoolRenderDegraded) && condition.Status == corev1.ConditionTrue {
					if len(condition.Reason) > 0 {
						message.WriteString("Machine config pool " + mcp.GetName() + " Degraded Reason: " + condition.Reason + ".\n")
					}
					if len(condition.Message) > 0 {
						message.WriteString("Machine config pool " + mcp.GetName() + " Degraded Message: " + condition.Message + ".\n")
					}
				}
			}
		}
	}

	messageString := message.String()
	if len(messageString) == 0 {
		return nil, nil
	}

	reasonString := reason.String()
	return r.getDegradedConditions(reasonString, messageString), nil
}

func removeMCPDuplicateEntries(mcps []mcov1.MachineConfigPool) []mcov1.MachineConfigPool {

	items := map[string]mcov1.MachineConfigPool{}
	for _, mcp := range mcps {
		if _, exists := items[mcp.Name]; !exists {
			items[mcp.Name] = mcp
		}
	}

	filtered := make([]mcov1.MachineConfigPool, 0, len(items))

	for _, value := range items {
		filtered = append(filtered, value)
	}

	return filtered
}
