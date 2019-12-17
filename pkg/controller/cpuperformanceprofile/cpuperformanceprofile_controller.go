package cpuperformanceprofile

import (
	"context"

	performancev1alpha1 "github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v1alpha1"

	mcov1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_cpuperformanceprofile")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new CPUPerformanceProfile Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileCPUPerformanceProfile{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("cpuperformanceprofile-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource CPUPerformanceProfile
	err = c.Watch(&source.Kind{Type: &performancev1alpha1.CPUPerformanceProfile{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resources and requeue the owner CPUPerformanceProfile
	err = c.Watch(&source.Kind{Type: &mcov1.MachineConfig{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &performancev1alpha1.CPUPerformanceProfile{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileCPUPerformanceProfile implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileCPUPerformanceProfile{}

// ReconcileCPUPerformanceProfile reconciles a CPUPerformanceProfile object
type ReconcileCPUPerformanceProfile struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a CPUPerformanceProfile object and makes changes based on the state read
// and what is in the CPUPerformanceProfile.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileCPUPerformanceProfile) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling CPUPerformanceProfile")

	// Fetch the CPUPerformanceProfile instance
	instance := &performancev1alpha1.CPUPerformanceProfile{}
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

	// Define a new MC object
	// TODO this is just an example for adding the MCO dependency for now!
	mc := newMachineConfigForCR(instance)

	// Set CPUPerformanceProfile instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, mc, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this MC already exists
	found := &mcov1.MachineConfig{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: mc.Name, Namespace: ""}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new MachineConfig", "MachineConfig.Name", mc.Name)
		err = r.client.Create(context.TODO(), mc)
		if err != nil {
			return reconcile.Result{}, err
		}

		// MC created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// MC already exists - don't requeue
	reqLogger.Info("Skip reconcile: MachineConfig already exists", "MachineConfig.Name", found.Name)
	return reconcile.Result{}, nil
}

func newMachineConfigForCR(cr *performancev1alpha1.CPUPerformanceProfile) *mcov1.MachineConfig {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &mcov1.MachineConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:   cr.Name + "-performance-machine-config",
			Labels: labels,
		},
		Spec: mcov1.MachineConfigSpec{
			//OSImageURL:      "",
			//Config:          types2.Config{},
			//KernelArguments: nil,
			//FIPS:            false,
		},
	}
}
