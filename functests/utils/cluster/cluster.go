package cluster

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"

	testclient "github.com/openshift-kni/performance-addon-operators/functests/utils/client"
)

// IsSingleNode validates if the environment is single node cluster
func IsSingleNode() (bool, error) {
	nodes := &corev1.NodeList{}
	if err := testclient.Client.List(context.TODO(), nodes, &client.ListOptions{}); err != nil {
		return false, err
	}
	return len(nodes.Items) == 1, nil
}

// ComputeTestTimeout returns the desired timeout for a test based on a given base timeout.
// If the tested cluster is Single-Node it needs more time to react (due to being highly loaded) so we double the given timeout.
func ComputeTestTimeout(baseTimeout time.Duration, isSno bool) time.Duration {
	testTimeout := baseTimeout
	if isSno {
		testTimeout += baseTimeout
	}

	return testTimeout
}

// EnforceRequirements indicates if tests with specific cluster preconditions should fail rather than skip
// if the cluster we are running against doesn't meet the requirements of the tests
func EnforceRequirements() bool {
	enforceReqs, _ := strconv.ParseBool(os.Getenv("ENFORCE_REQUIREMENTS"))
	return enforceReqs
}

func HaveEnoughCores(nodes []corev1.Node, cpus int) (bool, error) {
	requestCpu := resource.MustParse(fmt.Sprintf("%dm", cpus*1000))
	for _, node := range nodes {
		availCpu, ok := node.Status.Allocatable[corev1.ResourceCPU]
		if !ok {
			return false, fmt.Errorf("cpu resource not in allocatable on node %q", node.Name)
		}
		if availCpu.IsZero() {
			return false, fmt.Errorf("zero available cpu on node %q", node.Name)
		}

		if availCpu.Cmp(requestCpu) >= 1 {
			return true, nil
		}
	}

	return false, nil
}
