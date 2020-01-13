package performance

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/openshift-kni/performance-addon-operators/functests/utils/nodes"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	testutils "github.com/openshift-kni/performance-addon-operators/functests/utils"
	testclient "github.com/openshift-kni/performance-addon-operators/functests/utils/client"

	ocv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	testTimeout      = 480
	testPollInterval = 2
)

var _ = Describe("performance", func() {

	var workerRTNodes []corev1.Node

	BeforeEach(func() {
		var err error
		workerRTNodes, err = nodes.GetByRole(testclient.Client, testutils.RoleWorkerRT)
		Expect(err).ToNot(HaveOccurred())
		Expect(workerRTNodes).ToNot(BeEmpty())
	})

	Context("Pre boot tuning adjusted by the Machine Config Operator ", func() {

		It("Should contain a custom initrd image in the boot loader", func() {
			for _, node := range workerRTNodes {
				By("executing the command \"grep -R  initrd /rootfs/boot/loader/entries/\"")
				bootLoaderEntries, err := nodes.ExecCommandOnMachineConfigDaemon(testclient.Client, &node, []string{"grep", "-R", "initrd", "/rootfs/boot/loader/entries/"})
				Expect(err).ToNot(HaveOccurred())
				Expect(strings.Contains(string(bootLoaderEntries), "iso_initrd.img")).To(BeTrue(),
					"cannot find iso_initrd.img entry among the bootloader entries")
			}
		})

		// Check /usr/local/bin/pre-boot-tuning.sh existence under worker's rootfs
		const perfRtKernelPrebootTuningScript = "/usr/local/bin/pre-boot-tuning.sh"
		It(perfRtKernelPrebootTuningScript+" should exist on the nodes", func() {
			checkFileExistence(workerRTNodes, perfRtKernelPrebootTuningScript)
		})

		// Check /usr/local/bin/rt-kernel.sh existence under worker's rootfs
		const perfRTKernelPatchScript = "/usr/local/bin/rt-kernel.sh"
		It(perfRTKernelPatchScript+" should exist on the nodes", func() {
			checkFileExistence(workerRTNodes, perfRTKernelPatchScript)
		})
	})

	Context("FeatureGate - FeatureSet configuration", func() {
		It("FeatureGates with LatencySensitive should exist", func() {
			fg, err := testclient.Client.FeatureGates().Get(components.FeatureGateLatencySensetiveName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			lsStr := string(ocv1.LatencySensitive)
			By("Checking whether FetureSet is configured as " + lsStr)
			Expect(string(fg.Spec.FeatureSet)).Should(Equal(lsStr), "FeauterSet is not set to "+lsStr)
		})
	})

	// openshift node real time kernel verification
	// (performance-addon-operators/build/assets/tuned/openshift-node-real-time-kernel)
	Context("Tuned kernel parameters", func() {
		It("Should contain configuration injected through openshift-node-real-time-kernel profile", func() {
			sysctlMap := map[string]string{
				"kernel.hung_task_timeout_secs": "600",
				"kernel.nmi_watchdog":           "0",
				"kernel.sched_rt_runtime_us":    "-1",
				"vm.stat_interval":              "10",
				"kernel.timer_migration":        "0",
			}

			tunedName := components.GetComponentName("ci", components.ProfileNameWorkerRT)
			_, err := testclient.Client.Tuneds(components.NamespaceNodeTuningOperator).Get(tunedName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred(), "cannot find the Cluster Node Tuning Operator object "+tunedName)
			validatTunedActiveProfile(workerRTNodes)
			execSysctlOnWorkers(workerRTNodes, sysctlMap)
		})
	})

	// openshift node network latency profile verification
	// (performance-addon-operators/build/assets/tuned/openshift-node-network-latency)
	Context("Network latency parameters adjusted by the Node Tuning Operator", func() {
		It("Should contain configuration injected through the openshift-node-network-latency profile", func() {
			sysctlMap := map[string]string{
				"net.core.busy_read":              "50",
				"net.core.busy_poll":              "50",
				"net.ipv4.tcp_fastopen":           "3",
				"kernel.numa_balancing":           "0",
				"kernel.sched_min_granularity_ns": "10000000",
				"vm.dirty_ratio":                  "10",
				"vm.dirty_background_ratio":       "3",
				"vm.swappiness":                   "10",
				"kernel.sched_migration_cost_ns":  "5000000",
			}
			_, err := testclient.Client.Tuneds(components.NamespaceNodeTuningOperator).Get(components.ProfileNameNetworkLatency, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred(), "cannot find the Cluster Node Tuning Operator object "+components.ProfileNameNetworkLatency)
			validatTunedActiveProfile(workerRTNodes)
			execSysctlOnWorkers(workerRTNodes, sysctlMap)
		})
	})
})

func execSysctlOnWorkers(workerNodes []corev1.Node, sysctlMap map[string]string) {
	var err error
	var out []byte
	for _, node := range workerNodes {
		for param, expected := range sysctlMap {
			By(fmt.Sprintf("executing the command \"sysctl -n %s\"", param))
			out, err = nodes.ExecCommandOnMachineConfigDaemon(testclient.Client, &node, []string{"sysctl", "-n", param})
			Expect(err).ToNot(HaveOccurred())
			Expect(strings.TrimSpace(string(out))).Should(Equal(expected), fmt.Sprintf("parameter %s value is not %s.", param, expected))
		}
	}
}

// execute sysctl command inside container in a tuned pod
func validatTunedActiveProfile(nodes []corev1.Node) {
	var err error
	var out []byte
	activeProfileName := components.GetComponentName("ci", components.ProfileNameWorkerRT)
	for _, node := range nodes {
		tuned := tunedForNode(&node)
		tunedName := tuned.ObjectMeta.Name
		By(fmt.Sprintf("executing the command cat /etc/tuned/active_profile inside the pod %s", tunedName))
		Eventually(func() string {
			out, err = exec.Command("oc", "rsh", "-n", tuned.ObjectMeta.Namespace,
				tunedName, "cat", "/etc/tuned/active_profile").CombinedOutput()
			return strings.TrimSpace(string(out))
		}, testTimeout*time.Second, testPollInterval*time.Second).Should(Equal(activeProfileName),
			fmt.Sprintf("active_profile is not set to %s. %v", activeProfileName, err))
	}
}

// find tuned pod for appropriate node
func tunedForNode(node *corev1.Node) *corev1.Pod {
	listOptions := metav1.ListOptions{
		FieldSelector: fields.SelectorFromSet(fields.Set{"spec.nodeName": node.Name}).String(),
	}
	listOptions.LabelSelector = labels.SelectorFromSet(labels.Set{"openshift-app": "tuned"}).String()

	var tunedList *corev1.PodList
	var err error

	Eventually(func() bool {
		tunedList, err = testclient.Client.Pods(components.NamespaceNodeTuningOperator).List(listOptions)
		if err != nil {
			return false
		}
		if len(tunedList.Items) == 0 {
			return false
		}
		for _, s := range tunedList.Items[0].Status.ContainerStatuses {
			if s.Ready == false {
				return false
			}
		}
		return true

	}, testTimeout*time.Second, testPollInterval*time.Second).Should(BeTrue(),
		"there should be one tuned daemon per node")

	return &tunedList.Items[0]
}

// Check whether appropriate file exists on the system
func checkFileExistence(workerNodes []corev1.Node, file string) {
	for _, node := range workerNodes {
		By(fmt.Sprintf("Searching for the file %s.Executing the command \"ls /rootfs/%s\"", file, file))
		_, err := nodes.ExecCommandOnMachineConfigDaemon(testclient.Client, &node, []string{"ls", "/rootfs/" + file})
		Expect(err).To(BeNil(), "cannot find the file "+file)
	}
}
