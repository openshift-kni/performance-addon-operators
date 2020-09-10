package performanceprofile

import (
	"context"
	"reflect"
	"time"

	performancev2 "github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v2"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components/kubeletconfig"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components/machineconfig"
	profileutil "github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components/profile"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components/runtimeclass"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components/tuned"
	tunedv1 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/tuned/v1"
	mcov1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	corev1 "k8s.io/api/core/v1"
	nodev1beta1 "k8s.io/api/node/v1beta1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const finalizer = "foreground-deletion"

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new PerformanceProfile Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) *ReconcilePerformanceProfile {
	return &ReconcilePerformanceProfile{
		client:    mgr.GetClient(),
		scheme:    mgr.GetScheme(),
		recorder:  mgr.GetEventRecorderFor("performance-profile-controller"),
		assetsDir: components.AssetsDir,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r *ReconcilePerformanceProfile) error {
	// Create a new controller
	c, err := controller.New("performanceprofile-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource PerformanceProfile
	err = c.Watch(&source.Kind{Type: &performancev2.PerformanceProfile{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// we want to initate reconcile loop only on change under labels or spec of the object
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			if e.MetaOld == nil {
				klog.Error("Update event has no old metadata")
				return false
			}
			if e.ObjectOld == nil {
				klog.Error("Update event has no old runtime object to update")
				return false
			}
			if e.ObjectNew == nil {
				klog.Error("Update event has no new runtime object for update")
				return false
			}
			if e.MetaNew == nil {
				klog.Error("Update event has no new metadata")
				return false
			}

			return e.MetaNew.GetGeneration() != e.MetaOld.GetGeneration() ||
				!apiequality.Semantic.DeepEqual(e.MetaNew.GetLabels(), e.MetaOld.GetLabels())
		},
	}

	// Watch for changes to machine configs owned by our controller
	err = c.Watch(&source.Kind{Type: &mcov1.MachineConfig{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &performancev2.PerformanceProfile{},
	}, p)
	if err != nil {
		return err
	}

	// Watch for changes to kubelet configs owned by our controller
	err = c.Watch(&source.Kind{Type: &mcov1.KubeletConfig{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &performancev2.PerformanceProfile{},
	}, p)
	if err != nil {
		return err
	}

	// Watch for changes to tuned owned by our controller
	err = c.Watch(&source.Kind{Type: &tunedv1.Tuned{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &performancev2.PerformanceProfile{},
	}, p)
	if err != nil {
		return err
	}

	// Watch for changes for the RuntimeClass owned by our resource
	err = c.Watch(&source.Kind{Type: &nodev1beta1.RuntimeClass{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &performancev2.PerformanceProfile{},
	}, p)
	if err != nil {
		return err
	}

	mcpPredicates := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			if e.MetaOld == nil {
				klog.Error("Update event has no old metadata")
				return false
			}
			if e.ObjectOld == nil {
				klog.Error("Update event has no old runtime object to update")
				return false
			}
			if e.ObjectNew == nil {
				klog.Error("Update event has no new runtime object for update")
				return false
			}
			if e.MetaNew == nil {
				klog.Error("Update event has no new metadata")
				return false
			}

			mcpOld := e.ObjectOld.(*mcov1.MachineConfigPool)
			mcpNew := e.ObjectNew.(*mcov1.MachineConfigPool)
			mcpNewCopy := mcpNew.DeepCopy()

			mcpNewCopy.Spec.Paused = mcpOld.Spec.Paused
			mcpNewCopy.Spec.Configuration = mcpOld.Spec.Configuration

			return !reflect.DeepEqual(mcpOld.Status.Conditions, mcpNewCopy.Status.Conditions)
		},
	}

	err = c.Watch(&source.Kind{Type: &mcov1.MachineConfigPool{}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: (handler.ToRequestsFunc)(r.ppRequestsFromMCP)}, mcpPredicates)
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcilePerformanceProfile implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcilePerformanceProfile{}

// ReconcilePerformanceProfile reconciles a PerformanceProfile object
type ReconcilePerformanceProfile struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client    client.Client
	scheme    *runtime.Scheme
	recorder  record.EventRecorder
	assetsDir string
}

// Reconcile reads that state of the cluster for a PerformanceProfile object and makes changes based on the state read
// and what is in the PerformanceProfile.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcilePerformanceProfile) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	klog.Info("Reconciling PerformanceProfile")

	// Fetch the PerformanceProfile instance
	instance := &performancev2.PerformanceProfile{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
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
			r.recorder.Eventf(instance, corev1.EventTypeWarning, "Deletion failed", "Failed to delete components: %v", err)
			return reconcile.Result{}, err
		}
		r.recorder.Eventf(instance, corev1.EventTypeNormal, "Deletion succeeded", "Succeeded to delete all components")

		if r.isComponentsExist(instance) {
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}

		// remove finalizer
		if hasFinalizer(instance, finalizer) {
			removeFinalizer(instance, finalizer)
			if err := r.client.Update(context.TODO(), instance); err != nil {
				return reconcile.Result{}, err
			}

			return reconcile.Result{}, nil
		}
	}

	// add finalizer
	if !hasFinalizer(instance, finalizer) {
		instance.Finalizers = append(instance.Finalizers, finalizer)
		instance.Status.Conditions = r.getProgressingConditions("DeploymentStarting", "Deployment is starting")
		if err := r.client.Update(context.TODO(), instance); err != nil {
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
		r.recorder.Eventf(instance, corev1.EventTypeWarning, "Validation failed", "Profile validation failed: %v", err)
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
		r.recorder.Eventf(instance, corev1.EventTypeWarning, "Creation failed", "Failed to create all components: %v", err)
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

	return reconcile.Result{}, nil
}

func (r *ReconcilePerformanceProfile) ppRequestsFromMCP(o handler.MapObject) []reconcile.Request {

	mcp := &mcov1.MachineConfigPool{}

	if err := r.client.Get(context.TODO(),
		types.NamespacedName{
			Namespace: o.Meta.GetNamespace(),
			Name:      o.Meta.GetName(),
		},
		mcp,
	); err != nil {
		klog.Errorf("Unable to retrieve mcp %q from store: %v", namespacedName(o.Meta).String(), err)
		return nil
	}

	ppList := &performancev2.PerformanceProfileList{}
	if err := r.client.List(context.TODO(), ppList); err != nil {
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

func (r *ReconcilePerformanceProfile) applyComponents(profile *performancev2.PerformanceProfile) (*reconcile.Result, error) {

	if profileutil.IsPaused(profile) {
		klog.Infof("Ignoring reconcile loop for pause performance profile %s", profile.Name)
		return nil, nil
	}

	// get mutated machine config
	mc, err := machineconfig.New(r.assetsDir, profile)
	if err != nil {
		return nil, err
	}
	if err := controllerutil.SetControllerReference(profile, mc, r.scheme); err != nil {
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
	if err := controllerutil.SetControllerReference(profile, kc, r.scheme); err != nil {
		return nil, err
	}
	kcMutated, err := r.getMutatedKubeletConfig(kc)
	if err != nil {
		return nil, err
	}

	// get mutated performance tuned
	performanceTuned, err := tuned.NewNodePerformance(r.assetsDir, profile)
	if err != nil {
		return nil, err
	}

	if err := controllerutil.SetControllerReference(profile, performanceTuned, r.scheme); err != nil {
		return nil, err
	}
	performanceTunedMutated, err := r.getMutatedTuned(performanceTuned)
	if err != nil {
		return nil, err
	}

	// get mutated RuntimeClass
	runtimeClass := runtimeclass.New(profile, machineconfig.HighPerformanceRuntime)
	if err := controllerutil.SetControllerReference(profile, runtimeClass, r.scheme); err != nil {
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

	r.recorder.Eventf(profile, corev1.EventTypeNormal, "Creation succeeded", "Succeeded to create all components")
	return &reconcile.Result{}, nil
}

func (r *ReconcilePerformanceProfile) deleteComponents(profile *performancev2.PerformanceProfile) error {
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

func (r *ReconcilePerformanceProfile) isComponentsExist(profile *performancev2.PerformanceProfile) bool {
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

func hasFinalizer(profile *performancev2.PerformanceProfile, finalizer string) bool {
	for _, f := range profile.Finalizers {
		if f == finalizer {
			return true
		}
	}
	return false
}

func removeFinalizer(profile *performancev2.PerformanceProfile, finalizer string) {
	finalizers := []string{}
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

func hasMatchingLabels(performanceprofile *performancev2.PerformanceProfile, mcp *mcov1.MachineConfigPool) bool {

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
