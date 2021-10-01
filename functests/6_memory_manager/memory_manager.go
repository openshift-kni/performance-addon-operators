package memory_manager

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	testutils "github.com/openshift-kni/performance-addon-operators/functests/utils"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/profiles"
)

var _ = Describe("[ref_id:OCP-43186][pao] Memory Manager tests", func() {
	profile, _ := profiles.GetByNodeLabels(testutils.NodeSelectorLabels)

	Context("Use Case: Create a group containing both NUMA nodes", func() {
		It("[test_id: 12345] Deploy Pod1", func() {
			if profile.Spec.HugePages != nil {
				fmt.Println("Huge pages enabled")
			} else {
				fmt.Println("Huge pages not enabled")
			}
		})
	})
})
