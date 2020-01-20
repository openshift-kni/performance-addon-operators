package performanceprofile

import (
	"context"
	"time"

	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components/featuregate"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components/kubeletconfig"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components/machineconfig"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components/tuned"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	performancev1alpha1 "github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v1alpha1"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"
	testutils "github.com/openshift-kni/performance-addon-operators/pkg/utils/testing"
	configv1 "github.com/openshift/api/config/v1"
	tunedv1 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/tuned/v1"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	mcov1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const assetsDir = "../../../build/assets"

var _ = Describe("Controller", func() {
	var request reconcile.Request
	var profile *performancev1alpha1.PerformanceProfile

	BeforeEach(func() {
		profile = testutils.NewPerformanceProfile("test")
		request = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: metav1.NamespaceNone,
				Name:      profile.Name,
			},
		}
	})

	It("should add finalizer to the performance profile", func() {
		r := newFakeReconciler(profile)

		result, err := r.Reconcile(request)
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))

		updatedProfile := &performancev1alpha1.PerformanceProfile{}
		key := types.NamespacedName{
			Name:      profile.Name,
			Namespace: metav1.NamespaceNone,
		}
		Expect(r.client.Get(context.TODO(), key, updatedProfile)).ToNot(HaveOccurred())
		Expect(hasFinalizer(updatedProfile, finalizer)).To(Equal(true))
	})

	Context("with profile with finalizer", func() {
		BeforeEach(func() {
			profile.Finalizers = append(profile.Finalizers, finalizer)
		})

		It("should validate scripts required parameters", func() {
			profile.Spec.CPU.Isolated = nil
			r := newFakeReconciler(profile)

			// we do not return error, because we do not want to reconcile again, and just print error under the log,
			// once we will have validation webhook, this test will not be relevant anymore
			result, err := r.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))

			updatedProfile := &performancev1alpha1.PerformanceProfile{}
			key := types.NamespacedName{
				Name:      profile.Name,
				Namespace: metav1.NamespaceNone,
			}
			Expect(r.client.Get(context.TODO(), key, updatedProfile)).ToNot(HaveOccurred())

			// verify profile conditions
			degradeCondition := conditionsv1.FindStatusCondition(updatedProfile.Status.Conditions, conditionsv1.ConditionDegraded)
			Expect(degradeCondition).ToNot(BeNil())
			Expect(degradeCondition.Status).To(Equal(corev1.ConditionTrue))
			Expect(degradeCondition.Reason).To(Equal(conditionReasonValidationFailed))
			Expect(degradeCondition.Message).To(Equal("you should provide isolated CPU set"))

			// verify validation event
			fakeRecorder, ok := r.recorder.(*record.FakeRecorder)
			Expect(ok).To(BeTrue())
			event := <-fakeRecorder.Events
			Expect(event).To(ContainSubstring("Validation failed"))

			// verify that no components created by the controller
			mcp := &mcov1.MachineConfigPool{}
			key.Name = components.GetComponentName(profile.Name, components.ComponentNamePrefix)
			err = r.client.Get(context.TODO(), key, mcp)
			Expect(errors.IsNotFound(err)).To(Equal(true))
		})

		It("should create all resources except KubeletConfig on first reconcile loop", func() {
			r := newFakeReconciler(profile)

			result, err := r.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{RequeueAfter: 10 * time.Second}))

			name := components.GetComponentName(profile.Name, components.ComponentNamePrefix)
			key := types.NamespacedName{
				Name:      name,
				Namespace: metav1.NamespaceNone,
			}

			// verify MachineConfig creation
			mc := &mcov1.MachineConfig{}
			err = r.client.Get(context.TODO(), key, mc)
			Expect(err).ToNot(HaveOccurred())

			// verify that KubeletConfig wasn't created
			kc := &mcov1.KubeletConfig{}
			err = r.client.Get(context.TODO(), key, kc)
			Expect(errors.IsNotFound(err)).To(BeTrue())

			// verify FeatureGate creation
			fg := &configv1.FeatureGate{}
			key.Name = components.FeatureGateLatencySensetiveName
			err = r.client.Get(context.TODO(), key, fg)
			Expect(err).ToNot(HaveOccurred())

			// verify tuned LatencySensitive creation
			tunedLatency := &tunedv1.Tuned{}
			key.Name = components.ProfileNameNetworkLatency
			key.Namespace = components.NamespaceNodeTuningOperator
			err = r.client.Get(context.TODO(), key, tunedLatency)
			Expect(err).ToNot(HaveOccurred())

			// verify tuned tuned real-time kernel creation
			tunedRTKernel := &tunedv1.Tuned{}
			key.Name = components.GetComponentName(profile.Name, components.ProfileNameWorkerRT)
			err = r.client.Get(context.TODO(), key, tunedRTKernel)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should create KubeletConfig on second reconcile loop", func() {
			r := newFakeReconciler(profile)

			result, err := r.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{RequeueAfter: 10 * time.Second}))

			result, err = r.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))

			name := components.GetComponentName(profile.Name, components.ComponentNamePrefix)
			key := types.NamespacedName{
				Name:      name,
				Namespace: metav1.NamespaceNone,
			}

			// verify KubeletConfig creation
			kc := &mcov1.KubeletConfig{}
			err = r.client.Get(context.TODO(), key, kc)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should do nothing on third reconcile loop", func() {
			r := newFakeReconciler(profile)

			result, err := r.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{RequeueAfter: 10 * time.Second}))

			result, err = r.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))

			result, err = r.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))

			// verify creation event
			fakeRecorder, ok := r.recorder.(*record.FakeRecorder)
			Expect(ok).To(BeTrue())
			event := <-fakeRecorder.Events
			Expect(event).To(ContainSubstring("Creation succeeded"))
		})

		It("should update the profile status", func() {

			r := newFakeReconciler(profile)
			result, err := r.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{RequeueAfter: 10 * time.Second}))

			updatedProfile := &performancev1alpha1.PerformanceProfile{}
			key := types.NamespacedName{
				Name:      profile.Name,
				Namespace: metav1.NamespaceNone,
			}
			Expect(r.client.Get(context.TODO(), key, updatedProfile)).ToNot(HaveOccurred())

			// verify performance profile status
			Expect(len(updatedProfile.Status.Conditions)).To(Equal(4))

			// verify profile conditions
			progressingCondition := conditionsv1.FindStatusCondition(updatedProfile.Status.Conditions, conditionsv1.ConditionProgressing)
			Expect(progressingCondition).ToNot(BeNil())
			Expect(progressingCondition.Status).To(Equal(corev1.ConditionFalse))
			availableCondition := conditionsv1.FindStatusCondition(updatedProfile.Status.Conditions, conditionsv1.ConditionAvailable)
			Expect(availableCondition).ToNot(BeNil())
			Expect(availableCondition.Status).To(Equal(corev1.ConditionTrue))

		})

		It("should create nothing when pause annotation is set", func() {
			profile.Annotations = map[string]string{performancev1alpha1.PerformanceProfilePauseAnnotation: "true"}
			r := newFakeReconciler(profile)

			result, err := r.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))

			name := components.GetComponentName(profile.Name, components.ComponentNamePrefix)
			key := types.NamespacedName{
				Name:      name,
				Namespace: metav1.NamespaceNone,
			}

			// verify MachineConfig wasn't created
			mc := &mcov1.MachineConfig{}
			err = r.client.Get(context.TODO(), key, mc)
			Expect(errors.IsNotFound(err)).To(BeTrue())

			// verify that KubeletConfig wasn't created
			kc := &mcov1.KubeletConfig{}
			err = r.client.Get(context.TODO(), key, kc)
			Expect(errors.IsNotFound(err)).To(BeTrue())

			// verify no machine config pool was created
			mcp := &mcov1.MachineConfigPool{}
			err = r.client.Get(context.TODO(), key, mcp)
			Expect(errors.IsNotFound(err)).To(BeTrue())

			// verify FeatureGate wasn't created
			fg := &configv1.FeatureGate{}
			key.Name = components.FeatureGateLatencySensetiveName
			err = r.client.Get(context.TODO(), key, fg)
			Expect(errors.IsNotFound(err)).To(BeTrue())

			// verify tuned LatencySensitive wasn't created
			tunedLatency := &tunedv1.Tuned{}
			key.Name = components.ProfileNameNetworkLatency
			key.Namespace = components.NamespaceNodeTuningOperator
			err = r.client.Get(context.TODO(), key, tunedLatency)
			Expect(errors.IsNotFound(err)).To(BeTrue())

			// verify tuned tuned real-time kernel wasn't created
			tunedRTKernel := &tunedv1.Tuned{}
			key.Name = components.GetComponentName(profile.Name, components.ProfileNameWorkerRT)
			err = r.client.Get(context.TODO(), key, tunedRTKernel)
			Expect(errors.IsNotFound(err)).To(BeTrue())
		})

		Context("when all components exist", func() {
			var mc *mcov1.MachineConfig
			var kc *mcov1.KubeletConfig
			var fg *configv1.FeatureGate
			var tunedRTKernel *tunedv1.Tuned
			var tunedNetworkLatency *tunedv1.Tuned

			BeforeEach(func() {
				var err error
				mc, err = machineconfig.New(assetsDir, profile)
				Expect(err).ToNot(HaveOccurred())

				kc, err = kubeletconfig.New(profile)
				Expect(err).ToNot(HaveOccurred())

				fg = featuregate.NewLatencySensitive()

				tunedRTKernel, err = tuned.NewWorkerRealTimeKernel(assetsDir, profile)
				Expect(err).ToNot(HaveOccurred())

				tunedNetworkLatency, err = tuned.NewNetworkLatency(assetsDir)
				Expect(err).ToNot(HaveOccurred())

			})

			It("should not record new create event", func() {
				r := newFakeReconciler(profile, mc, kc, fg, tunedNetworkLatency, tunedRTKernel)
				result, err := r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(reconcile.Result{}))

				// verify that no creation event created
				fakeRecorder, ok := r.recorder.(*record.FakeRecorder)
				Expect(ok).To(BeTrue())

				select {
				case _ = <-fakeRecorder.Events:
					Fail("the recorder should not have new events")
				default:
				}
			})
		})

	})

	Context("with profile with deletion timestamp", func() {
		BeforeEach(func() {
			profile.DeletionTimestamp = &metav1.Time{
				Time: time.Now(),
			}
			profile.Finalizers = append(profile.Finalizers, finalizer)
		})

		It("should remove all components and remove the finalizer on first reconcile loop", func() {

			mc, err := machineconfig.New(assetsDir, profile)
			Expect(err).ToNot(HaveOccurred())

			kc, err := kubeletconfig.New(profile)
			Expect(err).ToNot(HaveOccurred())

			fg := featuregate.NewLatencySensitive()

			tunedLatency, err := tuned.NewNetworkLatency(assetsDir)
			Expect(err).ToNot(HaveOccurred())

			tunedRTKernel, err := tuned.NewWorkerRealTimeKernel(assetsDir, profile)
			Expect(err).ToNot(HaveOccurred())

			r := newFakeReconciler(profile, mc, kc, fg, tunedLatency, tunedRTKernel)
			result, err := r.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))

			// verify that controller deleted all components
			name := components.GetComponentName(profile.Name, components.ComponentNamePrefix)
			key := types.NamespacedName{
				Name:      name,
				Namespace: metav1.NamespaceNone,
			}

			// verify MachineConfig deletion
			err = r.client.Get(context.TODO(), key, mc)
			Expect(errors.IsNotFound(err)).To(Equal(true))

			// verify KubeletConfig deletion
			err = r.client.Get(context.TODO(), key, kc)
			Expect(errors.IsNotFound(err)).To(Equal(true))

			// verify feature gate deletion
			// TOOD: uncomment once https://bugzilla.redhat.com/show_bug.cgi?id=1788061 fixed
			// key.Name = components.FeatureGateLatencySensetiveName
			// err = r.client.Get(context.TODO(), key, fg)
			// Expect(errors.IsNotFound(err)).To(Equal(true))

			// verify tuned latency deletion
			key.Name = components.ProfileNameNetworkLatency
			key.Namespace = components.NamespaceNodeTuningOperator
			err = r.client.Get(context.TODO(), key, tunedLatency)
			Expect(errors.IsNotFound(err)).To(Equal(true))

			// verify tuned real-time kernel deletion
			key.Name = components.GetComponentName(profile.Name, components.ProfileNameWorkerRT)
			key.Namespace = components.NamespaceNodeTuningOperator
			err = r.client.Get(context.TODO(), key, tunedRTKernel)
			Expect(errors.IsNotFound(err)).To(Equal(true))

			// verify finalizer deletion
			key.Name = profile.Name
			key.Namespace = metav1.NamespaceNone
			updatedProfile := &performancev1alpha1.PerformanceProfile{}
			Expect(r.client.Get(context.TODO(), key, updatedProfile)).ToNot(HaveOccurred())
			Expect(hasFinalizer(updatedProfile, finalizer)).To(Equal(false))
		})
	})
})

// newFakeReconciler returns a new reconcile.Reconciler with a fake client
func newFakeReconciler(initObjects ...runtime.Object) *ReconcilePerformanceProfile {
	fakeClient := fake.NewFakeClient(initObjects...)
	fakeRecorder := record.NewFakeRecorder(10)
	return &ReconcilePerformanceProfile{
		client:    fakeClient,
		scheme:    scheme.Scheme,
		recorder:  fakeRecorder,
		assetsDir: assetsDir,
	}
}

func createMachineConfigPoolCondition(
	conditionType mcov1.MachineConfigPoolConditionType,
	conditionStatus corev1.ConditionStatus,
	timestamp *time.Time,
) *mcov1.MachineConfigPoolCondition {
	return &mcov1.MachineConfigPoolCondition{
		LastTransitionTime: metav1.Time{Time: *timestamp},
		Message:            "test",
		Reason:             "test",
		Status:             conditionStatus,
		Type:               conditionType,
	}
}
