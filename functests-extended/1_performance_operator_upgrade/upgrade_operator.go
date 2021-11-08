package __performance_operator_upgrade

import (
	"context"
	"flag"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	testclient "github.com/openshift-kni/performance-addon-operators/functests/utils/client"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/namespaces"
)

var fromVersion string
var toVersion string

var subscription *olmv1alpha1.Subscription

func init() {
	flag.StringVar(&fromVersion, "fromVersion", "", "the version to start with")
	flag.StringVar(&toVersion, "toVersion", "", "the version to update to")
}

var _ = Describe("[rfe_id:28567][performance] Performance Addon Operator Upgrades", func() {

	BeforeEach(func() {
		subscriptionsList := &olmv1alpha1.SubscriptionList{}
		err := testclient.Client.List(context.TODO(), subscriptionsList, &client.ListOptions{Namespace: namespaces.PerformanceOperator})
		ExpectWithOffset(1, err).ToNot(HaveOccurred(), "Failed getting Subscriptions")
		Expect(len(subscriptionsList.Items)).To(Equal(1), fmt.Sprintf("Unexpected number of Subscriptions found: %v", len(subscriptionsList.Items)))
		subscription = &subscriptionsList.Items[0]
	})

	It("[test_id:30876] upgrades performance profile operator", func() {

		Expect(fromVersion).ToNot(BeEmpty(), "fromVersion not set")
		Expect(toVersion).ToNot(BeEmpty(), "toVersion not set")

		By(fmt.Sprintf("Upgrading from %s to %s", fromVersion, toVersion))

		By(fmt.Sprintf("Verifying that %s channel is active", fromVersion))
		subscription = getSubscription(subscription.Name, namespaces.PerformanceOperator)
		Expect(subscription.Spec.Channel).To(Equal(fromVersion))
		Expect(subscription.Status.CurrentCSV).To(ContainSubstring(fromVersion))

		csv := getCSV(subscription.Status.CurrentCSV, namespaces.PerformanceOperator)
		fromImage := csv.ObjectMeta.Annotations["containerImage"]

		By(fmt.Sprintf("Change subscription installPlanApproval to Automatic"))
		Expect(testclient.Client.Patch(context.TODO(), subscription,
			client.RawPatch(
				types.JSONPatchType,
				[]byte(fmt.Sprintf(`[{ "op": "replace", "path": "/spec/installPlanApproval", "value": "%s" }]`, olmv1alpha1.ApprovalAutomatic)),
			),
		)).ToNot(HaveOccurred())

		By(fmt.Sprintf("Switch subscription channel to %s version", toVersion))
		Expect(testclient.Client.Patch(context.TODO(), subscription,
			client.ConstantPatch(
				types.JSONPatchType,
				[]byte(fmt.Sprintf(`[{ "op": "replace", "path": "/spec/channel", "value": "%s" }]`, toVersion)),
			),
		)).ToNot(HaveOccurred())

		// TODO: remove the hack once https://github.com/openshift/operator-framework-olm/pull/196 merged
		// the W/A deletes OLM related pods to initiate OLM pods restart
		By("Getting OLM pods")
		for _, labelValue := range []string{"olm-operator", "catalog-operator"} {
			olmPods := &v1.PodList{}
			Expect(testclient.Client.List(
				context.TODO(),
				olmPods,
				client.InNamespace("openshift-operator-lifecycle-manager"),
				client.MatchingLabels{"app": labelValue}),
			).To(BeNil())

			By("Deleting OLM pods")
			for i := range olmPods.Items {
				Expect(testclient.Client.Delete(context.TODO(), &olmPods.Items[i])).To(BeNil())
			}
		}

		By(fmt.Sprintf("Verifying that channel was updated to %s", toVersion))
		subscriptionWaitForUpdate(subscription.Name, namespaces.PerformanceOperator, toVersion)

		// CSV is updated and image tag was changed
		subscription = getSubscription(subscription.Name, namespaces.PerformanceOperator)
		csv = getCSV(subscription.Status.CurrentCSV, namespaces.PerformanceOperator)
		csvWaitForPhaseWithConditionReason(csv.Name, namespaces.PerformanceOperator, olmv1alpha1.CSVPhaseSucceeded, olmv1alpha1.CSVReasonInstallSuccessful)
		Expect(csv.ObjectMeta.Annotations["containerImage"]).NotTo(Equal(fromImage))
	})
})

func getSubscription(subsName, namespace string) *olmv1alpha1.Subscription {
	subs := &olmv1alpha1.Subscription{}
	key := types.NamespacedName{
		Name:      subsName,
		Namespace: namespace,
	}
	err := testclient.GetWithRetry(context.TODO(), key, subs)
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "Failed getting subscription")
	return subs
}

func getCSV(name, namespace string) *olmv1alpha1.ClusterServiceVersion {
	csv := &olmv1alpha1.ClusterServiceVersion{}
	key := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	err := testclient.GetWithRetry(context.TODO(), key, csv)
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "Failed getting CSV")
	return csv
}

func subscriptionWaitForUpdate(subsName, namespace, channel string) {
	EventuallyWithOffset(1, func() string {
		subs := getSubscription(subsName, namespace)
		return subs.Status.CurrentCSV
	}, 5*time.Minute, 15*time.Second).Should(ContainSubstring(channel))
}

func csvWaitForPhaseWithConditionReason(csvName, namespace string, phase olmv1alpha1.ClusterServiceVersionPhase, reason olmv1alpha1.ConditionReason) {
	EventuallyWithOffset(1, func() olmv1alpha1.ClusterServiceVersionPhase {
		csv := getCSV(csvName, namespace)
		if csv.Status.Reason == reason {
			return csv.Status.Phase
		}
		return olmv1alpha1.CSVPhaseNone
	}, 5*time.Minute, 15*time.Second).Should(Equal(phase))
}
