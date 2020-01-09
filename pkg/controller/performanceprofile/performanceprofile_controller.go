package performanceprofile

import (
	"context"
	"fmt"
	"reflect"
	"time"

	performancev1alpha1 "github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v1alpha1"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components/featuregate"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components/kubeletconfig"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components/machineconfig"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components/machineconfigpool"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components/tuned"
	configv1 "github.com/openshift/api/config/v1"
	tunedv1 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/tuned/v1"
	mcov1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
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
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcilePerformanceProfile{
		client:    mgr.GetClient(),
		scheme:    mgr.GetScheme(),
		assetsDir: components.AssetsDir,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("performanceprofile-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource PerformanceProfile
	err = c.Watch(&source.Kind{Type: &performancev1alpha1.PerformanceProfile{}}, &handler.EnqueueRequestForObject{})
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
		OwnerType:    &performancev1alpha1.PerformanceProfile{},
	}, p)
	if err != nil {
		return err
	}

	// Watch for changes to kubelet configs owned by our controller
	err = c.Watch(&source.Kind{Type: &mcov1.KubeletConfig{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &performancev1alpha1.PerformanceProfile{},
	}, p)
	if err != nil {
		return err
	}

	// Watch for changes to feature gates owned by our controller
	err = c.Watch(&source.Kind{Type: &configv1.FeatureGate{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &performancev1alpha1.PerformanceProfile{},
	}, p)
	if err != nil {
		return err
	}

	// Watch for changes to tuned owned by our controller
	err = c.Watch(&source.Kind{Type: &tunedv1.Tuned{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &performancev1alpha1.PerformanceProfile{},
	}, p)
	if err != nil {
		return err
	}

	// we do not want initiate reconcile loop on the configuration or pause fields update
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

			return !reflect.DeepEqual(mcpOld.Spec, mcpNewCopy.Spec) ||
				!apiequality.Semantic.DeepEqual(e.MetaNew.GetLabels(), e.MetaOld.GetLabels())
		},
	}

	// Watch for changes to machine config pools owned by our controller
	err = c.Watch(&source.Kind{Type: &mcov1.MachineConfigPool{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &performancev1alpha1.PerformanceProfile{},
	}, mcpPredicates)
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
	assetsDir string
}

// Reconcile reads that state of the cluster for a PerformanceProfile object and makes changes based on the state read
// and what is in the PerformanceProfile.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcilePerformanceProfile) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	klog.Info("Reconciling PerformanceProfile")

	// Fetch the PerformanceProfile instance
	instance := &performancev1alpha1.PerformanceProfile{}
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
		name := components.GetComponentName(instance.Name, components.RoleWorkerPerformance)
		mcp, err := r.getMachineConfigPool(name)
		if err != nil {
			if !errors.IsNotFound(err) {
				return reconcile.Result{}, err
			}
			klog.Warning("does not pause, the machine config pool does not exist, probably it was deleted")
		} else {
			// pause machine config pool
			updated, err := r.pauseMachineConfigPool(mcp, true)
			if err != nil {
				return reconcile.Result{}, err
			}

			// we want to give time to the machine-config controllers to get updated values
			if updated {
				return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
			}
		}

		// delete components
		if err := r.deleteComponents(instance); err != nil {
			klog.Errorf("failed to delete components: %v", err)
			return reconcile.Result{}, err
		}

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

	// TODO: we need to check if all under performance profiles values != nil
	// first we need to decide if each of values required and we should move the check into validation webhook
	// for now let's assume that all parameters needed for assets scrips are required
	if err := r.validatePerformanceProfileParameters(instance); err != nil {
		// we do not want to reconcile again in case of error, because a user will need to update the PerformanceProfile anyway
		klog.Errorf("failed to reconcile: %v", err)
		return reconcile.Result{}, nil
	}

	// add finalizer
	if !hasFinalizer(instance, finalizer) {
		instance.Finalizers = append(instance.Finalizers, finalizer)
		if err := r.client.Update(context.TODO(), instance); err != nil {
			return reconcile.Result{}, err
		}

		// we exit reconcile loop because we will have additional update reconcile
		return reconcile.Result{}, nil
	}

	// apply components
	result, err := r.applyComponents(instance)
	if err != nil {
		klog.Errorf("failed to deploy components: %v", err)
		return reconcile.Result{}, err
	}

	// TODO: we need to update the status

	if result != nil {
		return *result, nil
	}

	return reconcile.Result{}, nil
}

func (r *ReconcilePerformanceProfile) applyComponents(profile *performancev1alpha1.PerformanceProfile) (*reconcile.Result, error) {
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

	// get mutated feature gate
	fg := featuregate.NewLatencySensitive()
	// TOOD: uncomment once https://bugzilla.redhat.com/show_bug.cgi?id=1788061 fixed
	// if err := controllerutil.SetControllerReference(profile, fg, r.scheme); err != nil {
	// 	return err
	// }
	fgMutated, err := r.getMutatedFeatureGate(fg)
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

	// get mutated network latency tuned
	networkLatencyTuned, err := tuned.NewNetworkLatency(r.assetsDir)
	if err != nil {
		return nil, err
	}
	if err := controllerutil.SetControllerReference(profile, networkLatencyTuned, r.scheme); err != nil {
		return nil, err
	}
	networkLatencyTunedMutated, err := r.getMutatedTuned(networkLatencyTuned)
	if err != nil {
		return nil, err
	}

	// get mutated real time kernel tuned
	realTimeKernelTuned, err := tuned.NewWorkerRealTimeKernel(r.assetsDir, profile)
	if err != nil {
		return nil, err
	}
	if err := controllerutil.SetControllerReference(profile, realTimeKernelTuned, r.scheme); err != nil {
		return nil, err
	}
	realTimeKernelTunedMutated, err := r.getMutatedTuned(realTimeKernelTuned)
	if err != nil {
		return nil, err
	}

	updated := (mcMutated != nil ||
		kcMutated != nil ||
		fgMutated != nil ||
		networkLatencyTunedMutated != nil ||
		realTimeKernelTunedMutated != nil)

	// get mutated machine config pool
	mcp := machineconfigpool.New(profile)
	// we set MCP paused to updated, so if we need to update any resources it will be equal true,
	// otherwise false
	mcp.Spec.Paused = updated

	if err := controllerutil.SetControllerReference(profile, mcp, r.scheme); err != nil {
		return nil, err
	}
	mcpMutated, err := r.getMutatedMachineConfigPool(mcp)
	if err != nil {
		return nil, err
	}

	// does not update any resources, if it no changes to relevant objects and just continue to the status update
	if mcpMutated == nil && !updated {
		return nil, nil
	}

	// create or update machine config pool and pause it
	if mcpMutated != nil {
		if err := r.createOrUpdateMachineConfigPool(mcpMutated); err != nil {
			return nil, err
		}
		return &reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	if mcMutated != nil {
		if err := r.createOrUpdateMachineConfig(mcMutated); err != nil {
			return nil, err
		}
	}

	if networkLatencyTunedMutated != nil {
		if err := r.createOrUpdateTuned(networkLatencyTunedMutated); err != nil {
			return nil, err
		}
	}

	if realTimeKernelTunedMutated != nil {
		if err := r.createOrUpdateTuned(realTimeKernelTunedMutated); err != nil {
			return nil, err
		}
	}

	if fgMutated != nil {
		if err := r.createOrUpdateFeatureGate(fgMutated); err != nil {
			return nil, err
		}

		// feature gate resource should be updated before KubeletConfig creation, otherwise
		// we will lack TopologyManager feature gate under the kubelet configuration
		// see - https://bugzilla.redhat.com/show_bug.cgi?id=1788061#c3
		// we want to give time to the kubelet-config controllers to get updated feature gate resource
		return &reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	if kcMutated != nil {
		if err := r.createOrUpdateKubeletConfig(kcMutated); err != nil {
			return nil, err
		}
	}

	return &reconcile.Result{RequeueAfter: 10 * time.Second}, nil
}

func (r *ReconcilePerformanceProfile) pauseMachineConfigPool(mcp *mcov1.MachineConfigPool, pause bool) (bool, error) {
	if mcp.Spec.Paused == pause {
		return false, nil
	}

	mcp.Spec.Paused = pause
	klog.Infof("Set machine-config-pool %q pause to %t", mcp.Name, pause)
	return true, r.client.Update(context.TODO(), mcp)
}

func (r *ReconcilePerformanceProfile) validatePerformanceProfileParameters(performanceProfile *performancev1alpha1.PerformanceProfile) error {
	if performanceProfile.Spec.CPU == nil {
		return fmt.Errorf("you should provide CPU section")
	}

	if performanceProfile.Spec.CPU.Isolated == nil {
		return fmt.Errorf("you should provide isolated CPU set")
	}

	if performanceProfile.Spec.CPU.NonIsolated == nil {
		return fmt.Errorf("you should provide non isolated CPU set")
	}

	return nil
}

func (r *ReconcilePerformanceProfile) deleteComponents(profile *performancev1alpha1.PerformanceProfile) error {
	tunedName := components.GetComponentName(profile.Name, components.ProfileNameWorkerRT)
	if err := r.deleteTuned(tunedName, components.NamespaceNodeTuningOperator); err != nil {
		return err
	}

	if err := r.deleteTuned(components.ProfileNameNetworkLatency, components.NamespaceNodeTuningOperator); err != nil {
		return err
	}

	// TOOD: uncomment once https://bugzilla.redhat.com/show_bug.cgi?id=1788061 fixed
	// if err := r.deleteFeatureGate(components.FeatureGateLatencySensetiveName); err != nil {
	// 	return err
	// }

	name := components.GetComponentName(profile.Name, components.RoleWorkerPerformance)
	if err := r.deleteKubeletConfig(name); err != nil {
		return err
	}

	if err := r.deleteMachineConfig(name); err != nil {
		return err
	}

	return r.deleteMachineConfigPool(name)
}

func (r *ReconcilePerformanceProfile) isComponentsExist(profile *performancev1alpha1.PerformanceProfile) bool {
	tunedName := components.GetComponentName(profile.Name, components.ProfileNameWorkerRT)
	if _, err := r.getTuned(tunedName, components.NamespaceNodeTuningOperator); !errors.IsNotFound(err) {
		klog.Infof("Tuned %q custom resource is still exists under the namespace %q", tunedName, components.NamespaceNodeTuningOperator)
		return true
	}

	if _, err := r.getTuned(components.ProfileNameNetworkLatency, components.NamespaceNodeTuningOperator); !errors.IsNotFound(err) {
		klog.Infof("Tuned %q custom resource is still exists under the namespace %q", components.ProfileNameNetworkLatency, components.NamespaceNodeTuningOperator)
		return true
	}

	name := components.GetComponentName(profile.Name, components.RoleWorkerPerformance)
	if _, err := r.getKubeletConfig(name); !errors.IsNotFound(err) {
		klog.Infof("Kubelet Config %q custom resource is still exists under the cluster", name)
		return true
	}

	if _, err := r.getMachineConfig(name); !errors.IsNotFound(err) {
		klog.Infof("Machine Config %q custom resource is still exists under the cluster", name)
		return true
	}

	if _, err := r.getMachineConfigPool(name); !errors.IsNotFound(err) {
		klog.Infof("Machine Config Pool %q custom resource is still exists under the cluster", name)
		return true
	}
	return false
}

func hasFinalizer(profile *performancev1alpha1.PerformanceProfile, finalizer string) bool {
	for _, f := range profile.Finalizers {
		if f == finalizer {
			return true
		}
	}
	return false
}

func removeFinalizer(profile *performancev1alpha1.PerformanceProfile, finalizer string) {
	finalizers := []string{}
	for _, f := range profile.Finalizers {
		if f == finalizer {
			continue
		}
		finalizers = append(finalizers, f)
	}
	profile.Finalizers = finalizers
}
