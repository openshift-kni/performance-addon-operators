package performance

import (
	"fmt"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	testutils "github.com/openshift-kni/performance-addon-operators/functests/utils"
	testclient "github.com/openshift-kni/performance-addon-operators/functests/utils/client"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/nodes"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/pods"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"
)

var _ = Describe("[performance]Topology Manager", func() {
	var workerRTNodes []corev1.Node

	BeforeEach(func() {
		var err error
		workerRTNodes, err = nodes.GetByRole(testclient.Client, testutils.RoleWorkerRT)
		Expect(err).ToNot(HaveOccurred())
		Expect(workerRTNodes).ToNot(BeEmpty())
	})

	It("should be enabled with the best-effort policy", func() {
		kubeletConfig, err := nodes.GetKubeletConfig(testclient.Client, &workerRTNodes[0])
		Expect(err).ToNot(HaveOccurred())

		// verify topology manager feature gate
		enabled, ok := kubeletConfig.FeatureGates[testutils.FeatureGateTopologyManager]
		Expect(ok).To(BeTrue())
		Expect(enabled).To(BeTrue())

		// verify topology manager poicy
		Expect(kubeletConfig.TopologyManagerPolicy).To(Equal(kubeletconfigv1beta1.BestEffortTopologyManagerPolicy))
	})

	Context("with the SR-IOV devices and static CPU's", func() {
		var testpod *corev1.Pod
		var sriovNode *corev1.Node

		BeforeEach(func() {
			sriovNodes := nodes.FilterByResource(testclient.Client, workerRTNodes, testutils.ResourceSRIOV)
			// TODO: once we will have different CI job for SR-IOV test cases, this skip should be removed
			// and replaced by ginkgo CLI --focus parameter
			if len(sriovNodes) < 1 {
				Skip(
					fmt.Sprintf(
						"The environment does not have nodes with role %q and available %q resources",
						testutils.RoleWorkerRT,
						string(testutils.ResourceSRIOV),
					),
				)
			}
			sriovNode = &sriovNodes[0]

			var err error
			if testpod != nil {
				err = testclient.Client.Pods(testutils.NamespaceTesting).Delete(testpod.Name, &metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())

				err = pods.WaitForDeletion(testclient.Client, testpod, 60*time.Second)
				Expect(err).ToNot(HaveOccurred())
			}
			testpod = pods.GetBusybox()
			testpod.Spec.Containers[0].Resources.Requests = map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceCPU:      resource.MustParse("1"),
				corev1.ResourceMemory:   resource.MustParse("64Mi"),
				testutils.ResourceSRIOV: resource.MustParse("1"),
			}
			testpod.Spec.Containers[0].Resources.Limits = map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceCPU:      resource.MustParse("1"),
				corev1.ResourceMemory:   resource.MustParse("64Mi"),
				testutils.ResourceSRIOV: resource.MustParse("1"),
			}
			testpod.Spec.NodeSelector = map[string]string{
				testutils.LabelHostname: sriovNode.Name,
			}
			testpod, err = testclient.Client.Pods(testutils.NamespaceTesting).Create(testpod)
			Expect(err).ToNot(HaveOccurred())

			err = pods.WaitForCondition(testclient.Client, testpod, corev1.PodReady, corev1.ConditionTrue, 60*time.Second)
			Expect(err).ToNot(HaveOccurred())

			// Get updated testpod
			testpod, err = testclient.Client.Pods(testpod.Namespace).Get(testpod.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("should allocate resources from the same NUMA node", func() {
			sriovPciDevice, err := getSriovPciDeviceFromPod(testpod)
			Expect(err).ToNot(HaveOccurred())

			sriovDeviceNumaNode, err := getSriovPciDeviceNumaNode(sriovNode, sriovPciDevice)
			Expect(err).ToNot(HaveOccurred())

			cpuSet, err := getContainerCPUSet(sriovNode, testpod)
			Expect(err).ToNot(HaveOccurred())

			cpuSetNumaNodes, err := getCPUSetNumaNodes(sriovNode, cpuSet)
			Expect(err).ToNot(HaveOccurred())

			for _, cpuNumaNode := range cpuSetNumaNodes {
				Expect(sriovDeviceNumaNode).To(Equal(cpuNumaNode))
			}
		})
	})
})

func getSriovPciDeviceFromPod(pod *corev1.Pod) (string, error) {
	envBytes, err := exec.Command(
		"oc", "rsh", "-n", pod.Namespace, pod.Name, "env",
	).CombinedOutput()
	if err != nil {
		return "", err
	}

	re := regexp.MustCompile(fmt.Sprintf("%s=(.*)", testutils.EnvPciSriovDevice))
	results := re.FindSubmatch(envBytes)
	if len(results) < 2 {
		return "", fmt.Errorf("failed to find ENV variable %q under the pod %q", testutils.EnvPciSriovDevice, pod.Name)
	}

	return string(results[1]), nil
}

func getSriovPciDeviceNumaNode(sriovNode *corev1.Node, sriovPciDevice string) (string, error) {
	// we will use machine-config-daemon to get all information from the node, because it has
	// mounted node filesystem under /rootfs
	command := []string{"cat", path.Join("/rootfs", testutils.FilePathSRIOVDevice, sriovPciDevice, "numa_node")}
	numaNode, err := nodes.ExecCommandOnMachineConfigDaemon(testclient.Client, sriovNode, command)
	if err != nil {
		return "", err
	}
	return strings.Trim(string(numaNode), "\n"), nil
}

func getContainerCPUSet(sriovNode *corev1.Node, pod *corev1.Pod) ([]string, error) {
	podDir := fmt.Sprintf("kubepods-pod%s.slice", strings.ReplaceAll(string(pod.UID), "-", "_"))

	containerID := strings.Trim(pod.Status.ContainerStatuses[0].ContainerID, "cri-o://")
	containerDir := fmt.Sprintf("crio-%s.scope", containerID)

	// we will use machine-config-daemon to get all information from the node, because it has
	// mounted node filesystem under /rootfs
	command := []string{"cat", path.Join("/rootfs", testutils.FilePathKubePodsSlice, podDir, containerDir, "cpuset.cpus")}
	cpuSet, err := nodes.ExecCommandOnMachineConfigDaemon(testclient.Client, sriovNode, command)
	if err != nil {
		return nil, err
	}

	results := []string{}
	for _, cpuRange := range strings.Split(string(cpuSet), ",") {
		if strings.Contains(cpuRange, "-") {
			seq := strings.Split(cpuRange, "-")
			if len(seq) != 2 {
				return nil, fmt.Errorf("incorrect CPU range: %q", cpuRange)
			}
			// we will iterate over runes, so we should specify [0] to get it from string
			for i := seq[0][0]; i <= seq[1][0]; i++ {
				results = append(results, strings.Trim(string(i), "\n"))
			}
			continue
		}
		results = append(results, strings.Trim(cpuRange, "\n"))
	}
	return results, nil
}

func getCPUSetNumaNodes(sriovNode *corev1.Node, cpuSet []string) ([]string, error) {
	numaNodes := []string{}
	for _, cpuID := range cpuSet {
		cpuPath := path.Join("/rootfs", testutils.FilePathSysCPU, "cpu"+cpuID)
		cpuDirContent, err := nodes.ExecCommandOnMachineConfigDaemon(testclient.Client, sriovNode, []string{"ls", cpuPath})
		if err != nil {
			return nil, err
		}
		re := regexp.MustCompile(`node(\d+)`)
		match := re.FindStringSubmatch(string(cpuDirContent))
		if len(match) != 2 {
			return nil, fmt.Errorf("incorrect match for 'ls' command: %v", match)
		}
		numaNodes = append(numaNodes, match[1])
	}
	return numaNodes, nil
}
