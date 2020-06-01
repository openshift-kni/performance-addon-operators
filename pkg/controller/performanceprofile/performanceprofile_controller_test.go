package performanceprofile

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components/kubeletconfig"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components/machineconfig"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components/tuned"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	performancev1alpha1 "github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v1alpha1"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"
	testutils "github.com/openshift-kni/performance-addon-operators/pkg/utils/testing"
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
	"k8s.io/utils/pointer"

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

		Expect(reconcileTimes(r, request, 1)).To(Equal(reconcile.Result{}))

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
			profile.Spec.MachineConfigPoolSelector = map[string]string{"fake1": "val1", "fake2": "val2"}
			r := newFakeReconciler(profile)

			// we do not return error, because we do not want to reconcile again, and just print error under the log,
			// once we will have validation webhook, this test will not be relevant anymore
			Expect(reconcileTimes(r, request, 1)).To(Equal(reconcile.Result{}))

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
			Expect(degradeCondition.Message).To(Equal("validation error: you should provide only 1 MachineConfigPoolSelector"))

			// verify validation event
			fakeRecorder, ok := r.recorder.(*record.FakeRecorder)
			Expect(ok).To(BeTrue())
			event := <-fakeRecorder.Events
			Expect(event).To(ContainSubstring("Validation failed"))

			// verify that no components created by the controller
			mcp := &mcov1.MachineConfigPool{}
			key.Name = components.GetComponentName(profile.Name, components.ComponentNamePrefix)
			err := r.client.Get(context.TODO(), key, mcp)
			Expect(errors.IsNotFound(err)).To(Equal(true))
		})

		It("should create all resources on first reconcile loop", func() {
			r := newFakeReconciler(profile)

			Expect(reconcileTimes(r, request, 1)).To(Equal(reconcile.Result{}))

			name := components.GetComponentName(profile.Name, components.ComponentNamePrefix)
			key := types.NamespacedName{
				Name:      name,
				Namespace: metav1.NamespaceNone,
			}

			// verify MachineConfig creation
			mc := &mcov1.MachineConfig{}
			err := r.client.Get(context.TODO(), key, mc)
			Expect(err).ToNot(HaveOccurred())

			// verify KubeletConfig creation
			kc := &mcov1.KubeletConfig{}
			err = r.client.Get(context.TODO(), key, kc)
			Expect(err).ToNot(HaveOccurred())

			// verify tuned performance creation
			tunedPerformance := &tunedv1.Tuned{}
			key.Name = components.GetComponentName(profile.Name, components.ProfileNamePerformance)
			key.Namespace = components.NamespaceNodeTuningOperator
			err = r.client.Get(context.TODO(), key, tunedPerformance)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should create event on the second reconcile loop", func() {
			r := newFakeReconciler(profile)

			Expect(reconcileTimes(r, request, 2)).To(Equal(reconcile.Result{}))

			// verify creation event
			fakeRecorder, ok := r.recorder.(*record.FakeRecorder)
			Expect(ok).To(BeTrue())
			event := <-fakeRecorder.Events
			Expect(event).To(ContainSubstring("Creation succeeded"))
		})

		It("should update the profile status", func() {
			r := newFakeReconciler(profile)

			Expect(reconcileTimes(r, request, 1)).To(Equal(reconcile.Result{}))

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

			Expect(reconcileTimes(r, request, 1)).To(Equal(reconcile.Result{}))

			name := components.GetComponentName(profile.Name, components.ComponentNamePrefix)
			key := types.NamespacedName{
				Name:      name,
				Namespace: metav1.NamespaceNone,
			}

			// verify MachineConfig wasn't created
			mc := &mcov1.MachineConfig{}
			err := r.client.Get(context.TODO(), key, mc)
			Expect(errors.IsNotFound(err)).To(BeTrue())

			// verify that KubeletConfig wasn't created
			kc := &mcov1.KubeletConfig{}
			err = r.client.Get(context.TODO(), key, kc)
			Expect(errors.IsNotFound(err)).To(BeTrue())

			// verify no machine config pool was created
			mcp := &mcov1.MachineConfigPool{}
			err = r.client.Get(context.TODO(), key, mcp)
			Expect(errors.IsNotFound(err)).To(BeTrue())

			// verify tuned Performance wasn't created
			tunedPerformance := &tunedv1.Tuned{}
			key.Name = components.ProfileNamePerformance
			key.Namespace = components.NamespaceNodeTuningOperator
			err = r.client.Get(context.TODO(), key, tunedPerformance)
			Expect(errors.IsNotFound(err)).To(BeTrue())
		})

		Context("when all components exist", func() {
			var mc *mcov1.MachineConfig
			var kc *mcov1.KubeletConfig
			var tunedPerformance *tunedv1.Tuned

			BeforeEach(func() {
				var err error
				mc, err = machineconfig.New(assetsDir, profile)
				Expect(err).ToNot(HaveOccurred())

				kc, err = kubeletconfig.New(profile)
				Expect(err).ToNot(HaveOccurred())

				tunedPerformance, err = tuned.NewNodePerformance(assetsDir, profile)
				Expect(err).ToNot(HaveOccurred())

			})

			It("should not record new create event", func() {
				r := newFakeReconciler(profile, mc, kc, tunedPerformance)

				Expect(reconcileTimes(r, request, 1)).To(Equal(reconcile.Result{}))

				// verify that no creation event created
				fakeRecorder, ok := r.recorder.(*record.FakeRecorder)
				Expect(ok).To(BeTrue())

				select {
				case _ = <-fakeRecorder.Events:
					Fail("the recorder should not have new events")
				default:
				}
			})

			It("should update MC when RT kernel gets disabled", func() {
				profile.Spec.RealTimeKernel.Enabled = pointer.BoolPtr(false)
				r := newFakeReconciler(profile, mc, kc, tunedPerformance)

				Expect(reconcileTimes(r, request, 1)).To(Equal(reconcile.Result{}))

				name := components.GetComponentName(profile.Name, components.ComponentNamePrefix)
				key := types.NamespacedName{
					Name:      name,
					Namespace: metav1.NamespaceNone,
				}

				// verify MachineConfig update
				mc := &mcov1.MachineConfig{}
				err := r.client.Get(context.TODO(), key, mc)
				Expect(err).ToNot(HaveOccurred())

				Expect(mc.Spec.KernelType).To(Equal(machineconfig.MCKernelDefault))
			})

			It("should update MC, KC and Tuned when CPU params change", func() {
				reserved := performancev1alpha1.CPUSet("0-1")
				isolated := performancev1alpha1.CPUSet("2-3")
				profile.Spec.CPU = &performancev1alpha1.CPU{
					Reserved: &reserved,
					Isolated: &isolated,
				}

				r := newFakeReconciler(profile, mc, kc, tunedPerformance)

				Expect(reconcileTimes(r, request, 1)).To(Equal(reconcile.Result{}))

				key := types.NamespacedName{
					Name:      components.GetComponentName(profile.Name, components.ComponentNamePrefix),
					Namespace: metav1.NamespaceNone,
				}

				By("Verifying KC update for reserved")
				kc := &mcov1.KubeletConfig{}
				err := r.client.Get(context.TODO(), key, kc)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(kc.Spec.KubeletConfig.Raw)).To(ContainSubstring(fmt.Sprintf(`"reservedSystemCPUs":"%s"`, string(*profile.Spec.CPU.Reserved))))

				By("Verifying Tuned update for isolated")
				key = types.NamespacedName{
					Name:      components.GetComponentName(profile.Name, components.ProfileNamePerformance),
					Namespace: components.NamespaceNodeTuningOperator,
				}
				t := &tunedv1.Tuned{}
				err = r.client.Get(context.TODO(), key, t)
				Expect(err).ToNot(HaveOccurred())
				Expect(*t.Spec.Profile[0].Data).To(ContainSubstring("isolated_cores=" + string(*profile.Spec.CPU.Isolated)))
			})

			It("should add isolcpus to tuned profile when balanced set to false", func() {
				reserved := performancev1alpha1.CPUSet("0-1")
				isolated := performancev1alpha1.CPUSet("2-3")
				profile.Spec.CPU = &performancev1alpha1.CPU{
					Reserved:        &reserved,
					Isolated:        &isolated,
					BalanceIsolated: pointer.BoolPtr(false),
				}

				r := newFakeReconciler(profile, mc, kc, tunedPerformance)

				Expect(reconcileTimes(r, request, 1)).To(Equal(reconcile.Result{}))

				key := types.NamespacedName{
					Name:      components.GetComponentName(profile.Name, components.ProfileNamePerformance),
					Namespace: components.NamespaceNodeTuningOperator,
				}
				t := &tunedv1.Tuned{}
				err := r.client.Get(context.TODO(), key, t)
				Expect(err).ToNot(HaveOccurred())
				cmdlineRealtimeWithoutCPUBalancing := regexp.MustCompile(`\s*cmdline_realtime=\+\s*tsc=nowatchdog\s+intel_iommu=on\s+iommu=pt\s+isolcpus=\s*`)
				Expect(cmdlineRealtimeWithoutCPUBalancing.MatchString(*t.Spec.Profile[0].Data)).To(BeTrue())
			})

			It("should update MC when Hugepages params change without node added", func() {
				size := performancev1alpha1.HugePageSize("2M")
				profile.Spec.HugePages = &performancev1alpha1.HugePages{
					DefaultHugePagesSize: &size,
					Pages: []performancev1alpha1.HugePage{
						{
							Count: 8,
							Size:  size,
						},
					},
				}

				r := newFakeReconciler(profile, mc, kc, tunedPerformance)

				Expect(reconcileTimes(r, request, 1)).To(Equal(reconcile.Result{}))

				By("Verifying Tuned profile update")
				key := types.NamespacedName{
					Name:      components.GetComponentName(profile.Name, components.ProfileNamePerformance),
					Namespace: components.NamespaceNodeTuningOperator,
				}
				t := &tunedv1.Tuned{}
				err := r.client.Get(context.TODO(), key, t)
				Expect(err).ToNot(HaveOccurred())
				cmdlineHugepages := regexp.MustCompile(`\s*cmdline_hugepages=\+\s*default_hugepagesz=2M\s+hugepagesz=2M\s+hugepages=8\s*`)
				Expect(cmdlineHugepages.MatchString(*t.Spec.Profile[0].Data)).To(BeTrue())
			})

			It("should update Tuned when Hugepages params change with node added", func() {
				size := performancev1alpha1.HugePageSize("2M")
				profile.Spec.HugePages = &performancev1alpha1.HugePages{
					DefaultHugePagesSize: &size,
					Pages: []performancev1alpha1.HugePage{
						{
							Count: 8,
							Size:  size,
							Node:  pointer.Int32Ptr(0),
						},
					},
				}

				r := newFakeReconciler(profile, mc, kc, tunedPerformance)

				Expect(reconcileTimes(r, request, 1)).To(Equal(reconcile.Result{}))

				By("Verifying Tuned update")
				key := types.NamespacedName{
					Name:      components.GetComponentName(profile.Name, components.ProfileNamePerformance),
					Namespace: components.NamespaceNodeTuningOperator,
				}
				t := &tunedv1.Tuned{}
				err := r.client.Get(context.TODO(), key, t)
				Expect(err).ToNot(HaveOccurred())
				cmdlineHugepages := regexp.MustCompile(`\s*cmdline_hugepages=\+\s*`)
				Expect(cmdlineHugepages.MatchString(*t.Spec.Profile[0].Data)).To(BeTrue())

				By("Verifying MC update")
				key = types.NamespacedName{
					Name:      components.GetComponentName(profile.Name, components.ComponentNamePrefix),
					Namespace: metav1.NamespaceNone,
				}
				mc := &mcov1.MachineConfig{}
				err = r.client.Get(context.TODO(), key, mc)
				Expect(err).ToNot(HaveOccurred())
				Expect(mc.Spec.Config.Systemd.Units).To(ContainElement(MatchFields(IgnoreMissing|IgnoreExtras, Fields{
					"Contents": And(
						ContainSubstring("Description=Hugepages"),
						ContainSubstring("Environment=HUGEPAGES_COUNT=8"),
						ContainSubstring("Environment=HUGEPAGES_SIZE=2048"),
						ContainSubstring("Environment=NUMA_NODE=0"),
					),
				})))

			})

			It("should update status with generated tuned", func() {
				r := newFakeReconciler(profile, mc, kc, tunedPerformance)
				Expect(reconcileTimes(r, request, 1)).To(Equal(reconcile.Result{}))
				key := types.NamespacedName{
					Name:      components.GetComponentName(profile.Name, components.ProfileNamePerformance),
					Namespace: components.NamespaceNodeTuningOperator,
				}
				t := &tunedv1.Tuned{}
				err := r.client.Get(context.TODO(), key, t)
				Expect(err).ToNot(HaveOccurred())
				tunedNamespacedName := namespacedName(t).String()
				updatedProfile := &performancev1alpha1.PerformanceProfile{}
				key = types.NamespacedName{
					Name:      profile.Name,
					Namespace: metav1.NamespaceNone,
				}
				Expect(r.client.Get(context.TODO(), key, updatedProfile)).ToNot(HaveOccurred())
				Expect(updatedProfile.Status.Tuned).NotTo(BeNil())
				Expect(*updatedProfile.Status.Tuned).To(Equal(tunedNamespacedName))
			})

			It("should update status when MCP is degraded", func() {
				mcpReason := "mcpReason"
				mcpMessage := "MCP message"

				mcp := &mcov1.MachineConfigPool{
					TypeMeta: metav1.TypeMeta{
						APIVersion: mcov1.GroupVersion.String(),
						Kind:       "MachineConfigPool",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "mcp-test",
					},
					Spec: mcov1.MachineConfigPoolSpec{
						MachineConfigSelector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      testutils.MachineConfigPoolLabelKey,
									Operator: metav1.LabelSelectorOpIn,
									Values:   []string{testutils.MachineConfigPoolLabelValue},
								},
							},
						},
					},
					Status: mcov1.MachineConfigPoolStatus{
						Conditions: []mcov1.MachineConfigPoolCondition{
							{
								Type:    mcov1.MachineConfigPoolNodeDegraded,
								Status:  corev1.ConditionTrue,
								Reason:  mcpReason,
								Message: mcpMessage,
							},
						},
					},
				}

				r := newFakeReconciler(profile, mc, kc, tunedPerformance, mcp)

				Expect(reconcileTimes(r, request, 1)).To(Equal(reconcile.Result{}))

				updatedProfile := &performancev1alpha1.PerformanceProfile{}
				key := types.NamespacedName{
					Name:      profile.Name,
					Namespace: metav1.NamespaceNone,
				}
				Expect(r.client.Get(context.TODO(), key, updatedProfile)).ToNot(HaveOccurred())

				// verify performance profile status
				Expect(len(updatedProfile.Status.Conditions)).To(Equal(4))

				// verify profile conditions
				degradedCondition := conditionsv1.FindStatusCondition(updatedProfile.Status.Conditions, conditionsv1.ConditionDegraded)
				Expect(degradedCondition).ToNot(BeNil())
				Expect(degradedCondition.Status).To(Equal(corev1.ConditionTrue))
				Expect(degradedCondition.Reason).To(Equal(conditionReasonMCPDegraded))
				Expect(degradedCondition.Message).To(ContainSubstring(mcpMessage))
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

			tunedPerformance, err := tuned.NewNodePerformance(assetsDir, profile)
			Expect(err).ToNot(HaveOccurred())

			r := newFakeReconciler(profile, mc, kc, tunedPerformance)
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

			// verify tuned real-time kernel deletion
			key.Name = components.GetComponentName(profile.Name, components.ProfileNamePerformance)
			key.Namespace = components.NamespaceNodeTuningOperator
			err = r.client.Get(context.TODO(), key, tunedPerformance)
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

func reconcileTimes(reconciler *ReconcilePerformanceProfile, request reconcile.Request, times int) reconcile.Result {
	var result reconcile.Result
	var err error
	for i := 0; i < times; i++ {
		result, err = reconciler.Reconcile(request)
		Expect(err).ToNot(HaveOccurred())
	}
	return result
}

// newFakeReconciler returns a new reconcile.Reconciler with a fake client
func newFakeReconciler(initObjects ...runtime.Object) *ReconcilePerformanceProfile {
	fakeClient := fake.NewFakeClientWithScheme(scheme.Scheme, initObjects...)
	fakeRecorder := record.NewFakeRecorder(10)
	return &ReconcilePerformanceProfile{
		client:    fakeClient,
		scheme:    scheme.Scheme,
		recorder:  fakeRecorder,
		assetsDir: assetsDir,
	}
}
