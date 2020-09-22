/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	performancev1 "github.com/openshift-kni/performance-addon-operators/api/v1"
	"github.com/openshift-kni/performance-addon-operators/controllers/components"
	"github.com/openshift-kni/performance-addon-operators/controllers/components/kubeletconfig"
	"github.com/openshift-kni/performance-addon-operators/controllers/components/machineconfig"
	profileutil "github.com/openshift-kni/performance-addon-operators/controllers/components/profile"
	"github.com/openshift-kni/performance-addon-operators/controllers/components/runtimeclass"
	"github.com/openshift-kni/performance-addon-operators/controllers/components/tuned"
	tunedv1 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/tuned/v1"
	mcov1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	corev1 "k8s.io/api/core/v1"
	nodev1beta1 "k8s.io/api/node/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

const finalizer = "foreground-deletion"

// PerformanceProfileReconciler reconciles a PerformanceProfile object
type PerformanceProfileReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Recorder  record.EventRecorder
	AssetsDir string
}

// +kubebuilder:rbac:groups=performance.openshift.io,resources=performanceprofiles;performanceprofiles/status;performanceprofiles/finalizers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=machineconfiguration.openshift.io,resources=machineconfigs;machineconfigpools;kubeletconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tuned.openshift.io,resources=tuneds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=node.k8s.io,resources=runtimeclasses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:namespace=Role,groups=core,resources=events;pods;services;services/finalizers;configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:namespace=Role,groups=apps,resources=deployments;daemonsets;replicasets;statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:namespace=Role,groups=apps,resources=deployments/finalizers,verbs=update
// +kubebuilder:rbac:namespace=Role,groups=monitoring.coreos.com,resources=servicemonitors,verbs=get;list;watch;create;update;patch;delete

func (r *PerformanceProfileReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	klog.Info("Reconciling PerformanceProfile")

	// Fetch the PerformanceProfile instance
	instance := &performancev1.PerformanceProfile{}
	err := r.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if instance.DeletionTimestamp != nil {
		// delete components
		if err := r.deleteComponents(instance); err != nil {
			klog.Errorf("failed to delete components: %v", err)
			r.Recorder.Eventf(instance, corev1.EventTypeWarning, "Deletion failed", "Failed to delete components: %v", err)
			return reconcile.Result{}, err
		}
		r.Recorder.Eventf(instance, corev1.EventTypeNormal, "Deletion succeeded", "Succeeded to delete all components")

		if r.isComponentsExist(instance) {
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}

		// remove finalizer
		if hasFinalizer(instance, finalizer) {
			removeFinalizer(instance, finalizer)
			if err := r.Update(context.TODO(), instance); err != nil {
				return reconcile.Result{}, err
			}

			return reconcile.Result{}, nil
		}
	}

	// add finalizer
	if !hasFinalizer(instance, finalizer) {
		instance.Finalizers = append(instance.Finalizers, finalizer)
		instance.Status.Conditions = r.getProgressingConditions("DeploymentStarting", "Deployment is starting")
		if err := r.Update(context.TODO(), instance); err != nil {
			return reconcile.Result{}, err
		}

		// we exit reconcile loop because we will have additional update reconcile
		return reconcile.Result{}, nil
	}

	// TODO: we need to check if all under performance profiles values != nil
	// first we need to decide if each of values required and we should move the check into validation webhook
	// for now let's assume that all parameters needed for assets scrips are required
	if err := profileutil.ValidateParameters(instance); err != nil {
		klog.Errorf("failed to reconcile: %v", err)
		r.Recorder.Eventf(instance, corev1.EventTypeWarning, "Validation failed", "Profile validation failed: %v", err)
		conditions := r.getDegradedConditions(conditionReasonValidationFailed, err.Error())
		if err := r.updateStatus(instance, conditions); err != nil {
			klog.Errorf("failed to update performance profile %q status: %v", instance.Name, err)
			return reconcile.Result{}, err
		}
		// we do not want to reconcile again in case of error, because a user will need to update the PerformanceProfile anyway
		return reconcile.Result{}, nil
	}

	// apply components
	result, err := r.applyComponents(instance)
	if err != nil {
		klog.Errorf("failed to deploy performance profile %q components: %v", instance.Name, err)
		r.Recorder.Eventf(instance, corev1.EventTypeWarning, "Creation failed", "Failed to create all components: %v", err)
		conditions := r.getDegradedConditions(conditionReasonComponentsCreationFailed, err.Error())
		if err := r.updateStatus(instance, conditions); err != nil {
			klog.Errorf("failed to update performance profile %q status: %v", instance.Name, err)
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, err
	}

	conditions, err := r.getMCPConditionsByProfile(instance)
	if err != nil {
		conditions := r.getDegradedConditions(conditionFailedGettingMCPStatus, err.Error())
		if err := r.updateStatus(instance, conditions); err != nil {
			klog.Errorf("failed to update performance profile %q status: %v", instance.Name, err)
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, err
	}

	// if conditions were not added due to machine config pool status change then set as availble
	if conditions == nil {
		conditions = r.getAvailableConditions()
	}
	if err := r.updateStatus(instance, conditions); err != nil {
		klog.Errorf("failed to update performance profile %q status: %v", instance.Name, err)
		// we still want to requeue after some, also in case of error, to avoid chance of multiple reboots
		if result != nil {
			return *result, nil
		}

		return reconcile.Result{}, err
	}

	if result != nil {
		return *result, nil
	}

	return ctrl.Result{}, nil
}

func (r *PerformanceProfileReconciler) ppRequestsFromMCP(o handler.MapObject) []reconcile.Request {
	mcp := &mcov1.MachineConfigPool{}

	if err := r.Get(context.TODO(),
		types.NamespacedName{
			Namespace: o.Meta.GetNamespace(),
			Name:      o.Meta.GetName(),
		},
		mcp,
	); err != nil {
		klog.Errorf("Unable to retrieve mcp %q from store: %v", namespacedName(o.Meta).String(), err)
		return nil
	}

	ppList := &performancev1.PerformanceProfileList{}
	if err := r.List(context.TODO(), ppList); err != nil {
		klog.Errorf("Unable to list performance profiles: %v", err)
		return nil
	}

	var requests []reconcile.Request
	for k := range ppList.Items {
		if hasMatchingLabels(&ppList.Items[k], mcp) {
			requests = append(requests, reconcile.Request{NamespacedName: namespacedName(&ppList.Items[k])})
		}
	}

	return requests
}

func (r *PerformanceProfileReconciler) applyComponents(profile *performancev1.PerformanceProfile) (*reconcile.Result, error) {

	if profileutil.IsPaused(profile) {
		klog.Infof("Ignoring reconcile loop for pause performance profile %s", profile.Name)
		return nil, nil
	}

	// get mutated machine config
	mc, err := machineconfig.New(r.AssetsDir, profile)
	if err != nil {
		return nil, err
	}
	if err := controllerutil.SetControllerReference(profile, mc, r.Scheme); err != nil {
		return nil, err
	}
	mcMutated, err := r.getMutatedMachineConfig(mc)
	if err != nil {
		return nil, err
	}

	// get mutated kubelet config
	kc, err := kubeletconfig.New(profile)
	if err != nil {
		return nil, err
	}
	if err := controllerutil.SetControllerReference(profile, kc, r.Scheme); err != nil {
		return nil, err
	}
	kcMutated, err := r.getMutatedKubeletConfig(kc)
	if err != nil {
		return nil, err
	}

	// get mutated performance tuned
	performanceTuned, err := tuned.NewNodePerformance(r.AssetsDir, profile)
	if err != nil {
		return nil, err
	}

	if err := controllerutil.SetControllerReference(profile, performanceTuned, r.Scheme); err != nil {
		return nil, err
	}
	performanceTunedMutated, err := r.getMutatedTuned(performanceTuned)
	if err != nil {
		return nil, err
	}

	// get mutated RuntimeClass
	runtimeClass := runtimeclass.New(profile, machineconfig.HighPerformanceRuntime)
	if err := controllerutil.SetControllerReference(profile, runtimeClass, r.Scheme); err != nil {
		return nil, err
	}
	runtimeClassMutated, err := r.getMutatedRuntimeClass(runtimeClass)
	if err != nil {
		return nil, err
	}

	updated := mcMutated != nil ||
		kcMutated != nil ||
		performanceTunedMutated != nil ||
		runtimeClassMutated != nil

	// does not update any resources, if it no changes to relevant objects and just continue to the status update
	if !updated {
		return nil, nil
	}

	if mcMutated != nil {
		if err := r.createOrUpdateMachineConfig(mcMutated); err != nil {
			return nil, err
		}
	}

	if performanceTunedMutated != nil {
		if err := r.createOrUpdateTuned(performanceTunedMutated, profile.Name); err != nil {
			return nil, err
		}
	}

	if kcMutated != nil {
		if err := r.createOrUpdateKubeletConfig(kcMutated); err != nil {
			return nil, err
		}
	}

	if runtimeClassMutated != nil {
		if err := r.createOrUpdateRuntimeClass(runtimeClassMutated); err != nil {
			return nil, err
		}
	}

	r.Recorder.Eventf(profile, corev1.EventTypeNormal, "Creation succeeded", "Succeeded to create all components")
	return &reconcile.Result{}, nil
}

func (r *PerformanceProfileReconciler) deleteComponents(profile *performancev1.PerformanceProfile) error {
	tunedName := components.GetComponentName(profile.Name, components.ProfileNamePerformance)
	if err := r.deleteTuned(tunedName, components.NamespaceNodeTuningOperator); err != nil {
		return err
	}

	name := components.GetComponentName(profile.Name, components.ComponentNamePrefix)
	if err := r.deleteKubeletConfig(name); err != nil {
		return err
	}

	if err := r.deleteMachineConfig(name); err != nil {
		return err
	}

	if err := r.deleteRuntimeClass(name); err != nil {
		return err
	}

	return nil

}

func (r *PerformanceProfileReconciler) isComponentsExist(profile *performancev1.PerformanceProfile) bool {
	tunedName := components.GetComponentName(profile.Name, components.ProfileNamePerformance)
	if _, err := r.getTuned(tunedName, components.NamespaceNodeTuningOperator); !errors.IsNotFound(err) {
		klog.Infof("Tuned %q custom resource is still exists under the namespace %q", tunedName, components.NamespaceNodeTuningOperator)
		return true
	}

	name := components.GetComponentName(profile.Name, components.ComponentNamePrefix)
	if _, err := r.getKubeletConfig(name); !errors.IsNotFound(err) {
		klog.Infof("Kubelet Config %q custom resource is still exists under the cluster", name)
		return true
	}

	if _, err := r.getMachineConfig(name); !errors.IsNotFound(err) {
		klog.Infof("Machine Config %q custom resource is still exists under the cluster", name)
		return true
	}

	return false
}

func hasFinalizer(profile *performancev1.PerformanceProfile, finalizer string) bool {
	for _, f := range profile.Finalizers {
		if f == finalizer {
			return true
		}
	}
	return false
}

func removeFinalizer(profile *performancev1.PerformanceProfile, finalizer string) {
	var finalizers []string
	for _, f := range profile.Finalizers {
		if f == finalizer {
			continue
		}
		finalizers = append(finalizers, f)
	}
	profile.Finalizers = finalizers
}

func namespacedName(obj metav1.Object) types.NamespacedName {
	return types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}
}

func hasMatchingLabels(performanceprofile *performancev1.PerformanceProfile, mcp *mcov1.MachineConfigPool) bool {

	selector, err := metav1.LabelSelectorAsSelector(mcp.Spec.MachineConfigSelector)
	if err != nil {
		return false
	}
	// If a deployment with a nil or empty selector creeps in, it should match nothing, not everything.
	if selector.Empty() {
		return false
	}

	if !selector.Matches(labels.Set(profileutil.GetMachineConfigPoolSelector(performanceprofile))) {
		return false
	}
	return true
}

func (r *PerformanceProfileReconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := ctrl.NewControllerManagedBy(mgr).
		For(&performancev1.PerformanceProfile{}).
		Owns(&mcov1.MachineConfig{}).
		Owns(&tunedv1.Tuned{}).
		Owns(&nodev1beta1.RuntimeClass{}).
		Owns(&mcov1.MachineConfigPool{}).
		Complete(r)
	if err != nil {
		return err
	}
	return nil
}
