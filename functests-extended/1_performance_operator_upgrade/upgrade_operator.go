package __performance_operator_upgrade

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/openshift-kni/performance-addon-operators/functests/utils/tuned"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	machineconfigv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	performancev2 "github.com/openshift-kni/performance-addon-operators/api/v2"
	testutils "github.com/openshift-kni/performance-addon-operators/functests/utils"
	testclient "github.com/openshift-kni/performance-addon-operators/functests/utils/client"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/mcps"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/namespaces"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/nodes"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/profiles"
)

var fromVersion string
var toVersion string

var subscription *olmv1alpha1.Subscription

func init() {
	flag.StringVar(&fromVersion, "fromVersion", "", "the version to start with")
	flag.StringVar(&toVersion, "toVersion", "", "the version to update to")
}

var _ = Describe("[rfe_id:28567][performance] Performance Addon Operator Upgrades", func() {
	var performanceProfile *performancev2.PerformanceProfile
	var performanceMCP string
	var workerRTNodes []corev1.Node

	testutils.BeforeAll(func() {
		subscriptionsList := &olmv1alpha1.SubscriptionList{}
		err := testclient.Client.List(context.TODO(), subscriptionsList, &client.ListOptions{Namespace: namespaces.PerformanceOperator})
		ExpectWithOffset(1, err).ToNot(HaveOccurred(), "Failed getting Subscriptions")
		Expect(len(subscriptionsList.Items)).To(Equal(1), fmt.Sprintf("Unexpected number of Subscriptions found: %v", len(subscriptionsList.Items)))
		subscription = &subscriptionsList.Items[0]

		workerRTNodes, err = nodes.GetByLabels(testutils.NodeSelectorLabels)
		Expect(err).ToNot(HaveOccurred())
		workerRTNodes, err = nodes.MatchingOptionalSelector(workerRTNodes)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("error looking for the optional selector: %v", err))
		Expect(workerRTNodes).ToNot(BeEmpty(), "cannot find RT enabled worker nodes")

		nodeLabel := testutils.NodeSelectorLabels
		performanceProfile, err = profiles.GetByNodeLabels(nodeLabel)
		Expect(err).ToNot(HaveOccurred())
		performanceMCP, err = mcps.GetByProfile(performanceProfile)
		Expect(err).ToNot(HaveOccurred())
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
			client.RawPatch(
				types.JSONPatchType,
				[]byte(fmt.Sprintf(`[{ "op": "replace", "path": "/spec/channel", "value": "%s" }]`, toVersion)),
			),
		)).ToNot(HaveOccurred())

		By(fmt.Sprintf("Verifying that channel was updated to %s", toVersion))
		subscriptionWaitForUpdate(subscription.Name, namespaces.PerformanceOperator, toVersion)

		// CSV is updated and image tag was changed
		subscription = getSubscription(subscription.Name, namespaces.PerformanceOperator)
		csv = getCSV(subscription.Status.CurrentCSV, namespaces.PerformanceOperator)
		csvWaitForPhaseWithConditionReason(csv.Name, namespaces.PerformanceOperator, olmv1alpha1.CSVPhaseSucceeded, olmv1alpha1.CSVReasonInstallSuccessful)
		Expect(csv.ObjectMeta.Annotations["containerImage"]).NotTo(Equal(fromImage))

		// it is impossible to predict if it was some changes under generated by PAO KubeletConfig or Tuned
		// during the PAO upgrade, so the best we can do here is to wait some time,
		// to give other controllers time to notify changes
		time.Sleep(2 * time.Minute)
		mcps.WaitForCondition(performanceMCP, machineconfigv1.MachineConfigPoolUpdated, corev1.ConditionTrue)

		var workerRTNodesNames []string
		for _, workerRTNode := range workerRTNodes {
			workerRTNodesNames = append(workerRTNodesNames, workerRTNode.Name)
		}
		Expect(tuned.WaitForAppliedCondition(workerRTNodesNames, corev1.ConditionTrue, 5*time.Minute))
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
