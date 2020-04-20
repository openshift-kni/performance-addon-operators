package namespaces

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	"sigs.k8s.io/controller-runtime/pkg/client"

	testutils "github.com/openshift-kni/performance-addon-operators/functests/utils"
)

// TestingNamespace is the namespace the tests will use for running test pods
var TestingNamespace = &corev1.Namespace{
	ObjectMeta: metav1.ObjectMeta{
		Name: testutils.NamespaceTesting,
	},
}

// WaitForDeletion waits until the namespace will be removed from the cluster
func WaitForDeletion(c client.Client, name string, timeout time.Duration) error {
	key := types.NamespacedName{
		Name:      name,
		Namespace: metav1.NamespaceNone,
	}
	return wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		ns := &corev1.Namespace{}
		if err := c.Get(context.TODO(), key, ns); errors.IsNotFound(err) {
			return true, nil
		}
		return false, nil
	})
}
