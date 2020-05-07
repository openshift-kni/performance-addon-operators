package __performance_operator_upgrade

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"

	testclient "github.com/openshift-kni/performance-addon-operators/functests/utils/client"
)

var _ = Describe("[rfe_id:28567][performance] Performance Addon Operator Upgrades", func() {

	It("[test_id:29811] upgrades performance profile operator - 4.4.0 to 4.5.0", func() {
		operatorNamespace := "openshift-performance-addon"
		subscriptionName := "performance-addon-operator"
		previousVersion := "4.4.0"
		currentVersion := "4.5.0"

		By(fmt.Sprintf("Verifying that %s channel is active", previousVersion))
		subscription := getSubscription(subscriptionName, operatorNamespace)
		Expect(subscription.Spec.Channel).To(Equal(previousVersion))
		Expect(subscription.Status.CurrentCSV).To(ContainSubstring(previousVersion))

		// CSV is pointed to previous image tag and CRD is an old version
		csv := getCSV(subscription.Status.CurrentCSV, operatorNamespace)
		Expect(csv.ObjectMeta.Annotations["containerImage"]).To(ContainSubstring(previousVersion))

		crd := getCRD(csv.Spec.CustomResourceDefinitions.Owned[0].Name)
		Expect(crd.Spec.Validation.OpenAPIV3Schema).NotTo(ContainSubstring("topologyPolicy"))

		By(fmt.Sprintf("Switch subscription channel to %s version", currentVersion))
		Expect(testclient.Client.Patch(context.TODO(), subscription,
			client.ConstantPatch(
				types.JSONPatchType,
				[]byte(fmt.Sprintf(`[{ "op": "replace", "path": "/spec/channel", "value": "%s" }]`, currentVersion)),
			),
		)).ToNot(HaveOccurred())

		By(fmt.Sprintf("Verifying that channel was updated to %s", currentVersion))
		subscriptionWaitForUpdate(subscription.Name, operatorNamespace, currentVersion)

		// CSV is updated and pointed to right image tag
		subscription = getSubscription(subscriptionName, operatorNamespace)
		csv = getCSV(subscription.Status.CurrentCSV, operatorNamespace)
		csvWaitForPhaseWithConditionReason(csv.Name, operatorNamespace, olmv1alpha1.CSVPhaseSucceeded, olmv1alpha1.CSVReasonInstallSuccessful)

		// CRD should be current version with all new features
		crd = getCRD(csv.Spec.CustomResourceDefinitions.Owned[0].Name)
		Expect(crd.Spec.Validation.OpenAPIV3Schema).To(ContainSubstring("topologyPolicy"))
	})
})

func getSubscription(name, namespace string) *olmv1alpha1.Subscription {
	subs := &olmv1alpha1.Subscription{}
	key := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	err := testclient.GetWithRetry(context.TODO(), key, subs)
	Expect(err).ToNot(HaveOccurred(), "Failed getting subscription")
	return subs
}

func getCSV(name, namespace string) *olmv1alpha1.ClusterServiceVersion {
	csv := &olmv1alpha1.ClusterServiceVersion{}
	key := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	err := testclient.GetWithRetry(context.TODO(), key, csv)
	Expect(err).ToNot(HaveOccurred(), "Failed getting CSV")
	return csv
}

func getCRD(name string) *v1beta1.CustomResourceDefinition {
	crd := &v1beta1.CustomResourceDefinition{}
	key := types.NamespacedName{
		Name:      name,
		Namespace: metav1.NamespaceNone,
	}
	err := testclient.GetWithRetry(context.TODO(), key, crd)
	Expect(err).ToNot(HaveOccurred(), "Failed getting CRD")
	return crd
}

func subscriptionWaitForUpdate(subsName, namespace, channel string) {
	Eventually(func() string {
		subs := getSubscription(subsName, namespace)
		return subs.Status.CurrentCSV
	}, 5*time.Minute, 15*time.Second).Should(ContainSubstring(channel))
}

func csvWaitForPhaseWithConditionReason(csvName, namespace string, phase olmv1alpha1.ClusterServiceVersionPhase, reason olmv1alpha1.ConditionReason) {
	Eventually(func() olmv1alpha1.ClusterServiceVersionPhase {
		csv := getCSV(csvName, namespace)
		if csv.Status.Reason == reason {
			return csv.Status.Phase
		}
		return olmv1alpha1.CSVPhaseNone
	}, 5*time.Minute, 15*time.Second).Should(Equal(phase))
}
