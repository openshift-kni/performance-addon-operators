package performance

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testutils "github.com/openshift-kni/performance-addon-operators/functests/utils"
	testclient "github.com/openshift-kni/performance-addon-operators/functests/utils/client"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/nodes"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/pods"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/profiles"
	performancev1alpha1 "github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v1alpha1"
)

const (
	stressPod = "vish/stress"
)

var _ = Describe("[performance] CPU Management", func() {
	var workerRTNode *corev1.Node
	var profile *performancev1alpha1.PerformanceProfile
	var balanceIsolated bool
	var reservedCPU, isolatedCPU string
	var listReservedCPU []string
	var listIsolatedCPU []string

	BeforeEach(func() {
		workerRTNodes, err := nodes.GetByRole(testclient.Client, testutils.RoleWorkerRT)
		Expect(err).ToNot(HaveOccurred())
		Expect(workerRTNodes).ToNot(BeEmpty())
		workerRTNode = &workerRTNodes[0]

		profile, err = profiles.GetByNodeLabels(
			testclient.Client,
			map[string]string{
				fmt.Sprintf("%s/%s", testutils.LabelRole, testutils.RoleWorkerRT): "",
			},
		)
		Expect(err).ToNot(HaveOccurred())
		Expect(profile.Spec.HugePages).ToNot(BeNil())

		balanceIsolated = true
		if profile.Spec.CPU.BalanceIsolated != nil {
			balanceIsolated = *profile.Spec.CPU.BalanceIsolated
		}

		isolatedCPU = string(*profile.Spec.CPU.Isolated)
		Expect(isolatedCPU).ToNot(BeEmpty())
		listIsolatedCPU, err = CPUListConverter(isolatedCPU)
		Expect(err).ToNot(HaveOccurred())

		reservedCPU = string(*profile.Spec.CPU.Reserved)
		Expect(reservedCPU).ToNot(BeEmpty())
		listReservedCPU, err = CPUListConverter(reservedCPU)
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("Verification of configuration on the worker node", func() {
		It("Verify CPU reservation on the node", func() {
			By(fmt.Sprintf("Allocatable CPU should be less then capacity by %d", len(listReservedCPU)))
			capacityCPU, _ := workerRTNode.Status.Capacity.Cpu().AsInt64()
			allocatableCPU, _ := workerRTNode.Status.Allocatable.Cpu().AsInt64()
			Expect(capacityCPU - allocatableCPU).To(Equal(int64(len(listReservedCPU))))
		})

		It("Verify CPU affinity mask, CPU reservation and CPU isolation on worker node", func() {
			By("checking isolated CPU")
			cmd := []string{"cat", "/sys/devices/system/cpu/isolated"}
			sysIsolatedCpus := execCommandOnWorker(cmd, workerRTNode)
			if balanceIsolated {
				Expect(sysIsolatedCpus).To(BeEmpty())
			} else {
				Expect(sysIsolatedCpus).To(Equal(isolatedCPU))
			}

			By("checking reserved CPU in kubelet config file")
			cmd = []string{"cat", "/rootfs/etc/kubernetes/kubelet.conf"}
			Expect(execCommandOnWorker(cmd, workerRTNode)).To(ContainSubstring(fmt.Sprintf(`"reservedSystemCPUs":"%s"`, reservedCPU)))

			By("checking CPU affinity mask for kernel scheduler")
			cmd = []string{"/bin/bash", "-c", "taskset -pc $(pgrep rcu_sched)"}
			Expect(execCommandOnWorker(cmd, workerRTNode)).To(ContainSubstring(fmt.Sprintf("current affinity list: %s", reservedCPU)))
		})

	})

	Describe("Verification of cpu manager functionality", func() {
		var testpod *corev1.Pod

		AfterEach(func() {
			err := testclient.Client.Delete(context.TODO(), testpod)
			Expect(err).ToNot(HaveOccurred())

			err = pods.WaitForDeletion(testclient.Client, testpod, 60*time.Second)
			Expect(err).ToNot(HaveOccurred())
		})

		table.DescribeTable("Verify CPU usage by stress PODs", func(guaranteed bool) {
			var listCPU []string

			testpod = getStressPod(workerRTNode.Name)
			testpod.Namespace = testutils.NamespaceTesting

			if guaranteed {
				listCPU = listIsolatedCPU
				testpod.Spec.Containers[0].Resources.Limits = map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("1Gi"),
				}
			} else if balanceIsolated {
				// when balanceIsolated is True - non-guaranteed pod can take ANY cpu
				cmd := []string{"/bin/bash", "-c", "lscpu | grep On-line | awk '{print $4}'"}
				listCPU, _ = CPUListConverter(execCommandOnWorker(cmd, workerRTNode))
			} else {
				// when balanceIsolated is False - non-guaranteed pod should run on reserved cpu
				listCPU = listReservedCPU
			}

			err := testclient.Client.Create(context.TODO(), testpod)
			Expect(err).ToNot(HaveOccurred())

			err = pods.WaitForCondition(testclient.Client, testpod, corev1.PodReady, corev1.ConditionTrue, 60*time.Second)
			Expect(err).ToNot(HaveOccurred())

			cmd := []string{"/bin/bash", "-c", "ps -o psr $(pgrep -n stress) | tail -1"}
			Expect(strings.Trim(execCommandOnWorker(cmd, workerRTNode), " ")).To(BeElementOf(listCPU))
		},
			table.Entry("Non-guaranteed POD can work on any CPU", false),
			table.Entry("Guaranteed POD should work on isolated cpu", true),
		)
	})
})

func execCommandOnWorker(cmd []string, workerRTNode *corev1.Node) string {
	out, err := nodes.ExecCommandOnMachineConfigDaemon(testclient.Client, workerRTNode, cmd)
	Expect(err).ToNot(HaveOccurred())
	return strings.Trim(string(out), "\n")
}

func CPUListConverter(cpusetLine string) (cpusList []string, err error) {
	elements := strings.Split(cpusetLine, ",")
	for _, item := range elements {
		cpuRange := strings.Split(item, "-")
		if len(cpuRange) > 1 {
			start, err := strconv.Atoi(cpuRange[0])
			if err != nil {
				return nil, err
			}
			end, err := strconv.Atoi(cpuRange[1])
			if err != nil {
				return nil, err
			}
			// Add cpus to the list. Assuming it's a valid range.
			for cpuNum := start; cpuNum <= end; cpuNum++ {
				cpusList = append(cpusList, strconv.Itoa(cpuNum))
			}
		} else {
			cpuNum, err := strconv.Atoi(cpuRange[0])
			if err != nil {
				return nil, err
			}
			cpusList = append(cpusList, strconv.Itoa(cpuNum))
		}
	}
	return
}

func getStressPod(nodeName string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-cpu-",
			Labels: map[string]string{
				"test": "",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "stress-test",
					Image: stressPod,
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("1"),
							corev1.ResourceMemory: resource.MustParse("1Gi"),
						},
					},
					Args: []string{"-cpus", "1"},
				},
			},
			NodeSelector: map[string]string{
				testutils.LabelHostname: nodeName,
			},
		},
	}
}
