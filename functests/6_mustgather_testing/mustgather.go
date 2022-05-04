package pao_mustgather

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"

	performancev2 "github.com/openshift-kni/performance-addon-operators/api/v2"
	testutils "github.com/openshift-kni/performance-addon-operators/functests/utils"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/nodes"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/pods"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/profiles"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var mustGatherPath = os.Getenv("MUSTGATHER_DIR")
var mustGatherFailed bool
var mustGatherContentDir string

var _ = Describe("[rfe_id: 50649]Performance Addon Operator Must Gather", func() {
	var profile *performancev2.PerformanceProfile
	var workerRTNodes []corev1.Node
	var err error
	if mustGatherPath == "" {
		mustGatherFailed = true
	}

	testutils.BeforeAll(func() {
		if mustGatherFailed {
			Skip("No mustgather directory provided, check MUSTGATHER_DIR environment variable")
		} else {
			mustgatherPathContent, err := ioutil.ReadDir(mustGatherPath)
			Expect(err).To(BeNil(), "failed to read the Mustgather Directory %s: %v", mustGatherPath, err)
			for _, file := range mustgatherPathContent {
				if strings.Contains(file.Name(), "registry") {
					mustGatherContentDir = filepath.Join(mustGatherPath, file.Name())
				}
			}
		}

	})
	Context("PAO Mustgather Tests", func() {
		It("[test_id: 50650][crit:high][vendor:cnf-qe@redhat.com][level:acceptance] Verify Generic cluster resource definitions are captured", func() {
			var genericFiles = []string{
				"version",
				"cluster-scoped-resources/config.openshift.io/featuregates/cluster.yaml",
				"cluster-scoped-resources/machineconfiguration.openshift.io/kubeletconfigs/performance-performance.yaml",
				"cluster-scoped-resources/machineconfiguration.openshift.io/machineconfigpools/master.yaml",
				"cluster-scoped-resources/machineconfiguration.openshift.io/machineconfigpools/worker.yaml",
				"cluster-scoped-resources/performance.openshift.io/performanceprofiles/performance.yaml",
				"namespaces/openshift-cluster-node-tuning-operator/tuned.openshift.io/tuneds/default.yaml",
				"namespaces/openshift-cluster-node-tuning-operator/tuned.openshift.io/tuneds/rendered.yaml",
				"namespaces/openshift-performance-addon-operator/openshift-performance-addon-operator.yaml",
				"namespaces/openshift-performance-addon-operator/apps.openshift.io/deploymentconfigs.yaml",
				"namespaces/openshift-performance-addon-operator/apps/daemonsets.yaml",
				"namespaces/openshift-performance-addon-operator/apps/deployments.yaml",
				"namespaces/openshift-performance-addon-operator/apps/replicasets.yaml",
				"namespaces/openshift-performance-addon-operator/apps/statefulsets.yaml",
				"namespaces/openshift-performance-addon-operator/autoscaling/horizontalpodautoscalers.yaml",
				"namespaces/openshift-performance-addon-operator/batch/cronjobs.yaml",
				"namespaces/openshift-performance-addon-operator/batch/jobs.yaml",
				"namespaces/openshift-performance-addon-operator/build.openshift.io/buildconfigs.yaml",
				"namespaces/openshift-performance-addon-operator/build.openshift.io/builds.yaml",
				"namespaces/openshift-performance-addon-operator/core/configmaps.yaml",
				"namespaces/openshift-performance-addon-operator/core/endpoints.yaml",
				"namespaces/openshift-performance-addon-operator/core/events.yaml",
				"namespaces/openshift-performance-addon-operator/core/persistentvolumeclaims.yaml",
				"namespaces/openshift-performance-addon-operator/core/pods.yaml",
				"namespaces/openshift-performance-addon-operator/core/replicationcontrollers.yaml",
				"namespaces/openshift-performance-addon-operator/core/secrets.yaml",
				"namespaces/openshift-performance-addon-operator/core/services.yaml",
				"namespaces/openshift-performance-addon-operator/image.openshift.io/imagestreams.yaml",
				"namespaces/openshift-performance-addon-operator/route.openshift.io/routes.yaml",
			}
			err := checkfilesExist(genericFiles)
			Expect(err).To(BeNil())
		})

		It("[test_id: 50651][crit:high][vendor:cnf-qe@redhat.com][level:acceptance] Verify PAO cluster resources are captured", func() {
			profile, _ = profiles.GetByNodeLabels(testutils.NodeSelectorLabels)
			pod, err := pods.GetPerformanceOperatorPod()
			Expect(err).ToNot(HaveOccurred(), "Failed to find the Performance Addon Operator pod")
			clusterSpecificFiles := []string{
				fmt.Sprintf("cluster-scoped-resources/machineconfiguration.openshift.io/machineconfigpools/%s.yaml", testutils.RoleWorker),
				fmt.Sprintf("namespaces/openshift-cluster-node-tuning-operator/tuned.openshift.io/tuneds/openshift-node-performance-%s.yaml", profile.Name),
				fmt.Sprintf("namespaces/openshift-performance-addon-operator/pods/%s/%s.yaml", pod.Name, pod.Name),
				fmt.Sprintf("namespaces/openshift-performance-addon-operator/pods/%s/performance-operator/performance-operator/logs/current.log", pod.Name),
			}
			err = checkfilesExist(clusterSpecificFiles)
			Expect(err).To(BeNil())
		})
		It("[test_id:50652][crit:high][vendor:cnf-qe@redhat.com][level:acceptance] Verify hardware related information are captured", func() {
			workerRTNodes, err = nodes.GetByLabels(testutils.NodeSelectorLabels)
			Expect(err).ToNot(HaveOccurred())
			workerRTNodes, err = nodes.MatchingOptionalSelector(workerRTNodes)
			Expect(err).ToNot(HaveOccurred())
			for _, node := range workerRTNodes {
				nodeSpecificFiles := []string{
					fmt.Sprintf("nodes/%s/%s_logs_kubelet.gz", node.Name, node.Name),
					fmt.Sprintf("nodes/%s/lscpu", node.Name),
					fmt.Sprintf("nodes/%s/lspci", node.Name),
					fmt.Sprintf("nodes/%s/proc_cmdline", node.Name),
				}
				err := checkfilesExist(nodeSpecificFiles)
				Expect(err).To(BeNil())
			}

		})
		It("[test_id:50653][crit:high][vendor:cnf-qe@redhat.com][level:acceptance] Verify machineconfig resources are captured", func() {
			mcpList := []string{"master", "worker", testutils.RoleWorkerCNF}
			mcpFiles := make([]string, len(mcpList))
			for _, mcp := range mcpList {
				mcpFiles = append(mcpFiles, fmt.Sprintf("cluster-scoped-resources/machineconfiguration.openshift.io/machineconfigpools/%s.yaml", mcp))
			}
			err := checkfilesExist(mcpFiles)
			Expect(err).To(BeNil())
		})

	})
})

func checkfilesExist(listOfFiles []string) error {
	for _, f := range listOfFiles {
		file := filepath.Join(mustGatherContentDir, f)
		info, err := os.Stat(file)
		if err != nil {
			return err
		}
		Expect(info.Size()).ToNot(BeZero())
	}
	return nil
}
