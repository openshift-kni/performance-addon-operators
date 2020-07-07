package discovery

import (
	"context"
	"os"
	"strconv"

	testclient "github.com/openshift-kni/performance-addon-operators/functests/utils/client"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/profiles"
	performancev1alpha1 "github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Enabled indicates whether test discovery mode is enabled.
func Enabled() bool {
	discoveryMode, _ := strconv.ParseBool(os.Getenv("DISCOVERY_MODE"))
	return discoveryMode
}

// GetDiscoveryPerformanceProfile returns an existing profile in the cluster with the most nodes using it.
// In case no profile exists - return nil
func GetDiscoveryPerformanceProfile(exclude ...string) (*performancev1alpha1.PerformanceProfile, error) {
	performanceProfiles, err := profiles.GetAllProfiles()
	if err != nil {
		return nil, err
	}

	var currentProfile *performancev1alpha1.PerformanceProfile = nil
	maxNodesNumber := 0
	for _, profile := range performanceProfiles.Items {
		if isExcluded(profile.GetName(), exclude) {
			continue
		}
		selector := labels.SelectorFromSet(profile.Spec.NodeSelector)

		profileNodes := &corev1.NodeList{}
		if err := testclient.Client.List(context.TODO(), profileNodes, &client.ListOptions{LabelSelector: selector}); err != nil {
			return nil, err
		}

		if len(profileNodes.Items) > maxNodesNumber {
			currentProfile = &profile
			maxNodesNumber = len(profileNodes.Items)
		}
	}
	return currentProfile, nil
}

func isExcluded(item string, exclude []string) bool {
	for _, excluded := range exclude {
		if item == excluded {
			return true
		}
	}
	return false
}
