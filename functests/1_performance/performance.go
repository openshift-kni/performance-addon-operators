package __performance

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"regexp"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	testutils "github.com/openshift-kni/performance-addon-operators/functests/utils"
	testclient "github.com/openshift-kni/performance-addon-operators/functests/utils/client"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/nodes"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/profiles"
	performancev1alpha1 "github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v1alpha1"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"
	ocv1 "github.com/openshift/api/config/v1"
)

const (
	testTimeout      = 480
	testPollInterval = 2
)

var _ = Describe("[rfe_id:27368][performance]", func() {

	var workerRTNodes []corev1.Node
	var profile *performancev1alpha1.PerformanceProfile

	BeforeEach(func() {
		var err error
		workerRTNodes, err = nodes.GetByRole(testutils.RoleWorkerCNF)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("error looking for node with role %q: %v", testutils.RoleWorkerCNF, err))
		Expect(workerRTNodes).ToNot(BeEmpty(), fmt.Sprintf("no nodes with role %q found", testutils.RoleWorkerCNF))
		profile, err = profiles.GetByNodeLabels(
			map[string]string{
				fmt.Sprintf("%s/%s", testutils.LabelRole, testutils.RoleWorkerCNF): "",
			},
		)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("Pre boot tuning adjusted by the Machine Config Operator ", func() {

		It("[test_id:27081][crit:high][vendor:cnf-qe@redhat.com][level:acceptance] Should set workqueue CPU mask", func() {
			for _, node := range workerRTNodes {
				By("Getting tuned.non_isolcpus kernel argument")
				cmdline, err := nodes.ExecCommandOnMachineConfigDaemon(&node, []string{"cat", "/proc/cmdline"})
				re := regexp.MustCompile(`tuned.non_isolcpus=\S+`)
				nonIsolcpusFullArgument := re.FindString(string(cmdline))
				Expect(nonIsolcpusFullArgument).To(ContainSubstring("tuned.non_isolcpus="))
				nonIsolcpusMask := strings.Split(string(nonIsolcpusFullArgument), "=")[1]
				nonIsolcpusMaskNoDelimiters := strings.Replace(nonIsolcpusMask, ",", "", -1)
				Expect(err).ToNot(HaveOccurred())
				By("executing the command \"cat /sys/devices/virtual/workqueue/cpumask\"")
				workqueueMask, err := nodes.ExecCommandOnMachineConfigDaemon(&node, []string{"cat", "/sys/devices/virtual/workqueue/cpumask"})
				Expect(err).ToNot(HaveOccurred())
				workqueueMaskTrimmed := strings.TrimSpace(string(workqueueMask))
				workqueueMaskTrimmedNoDelimiters := strings.Replace(workqueueMaskTrimmed, ",", "", -1)
				Expect(strings.TrimLeft(nonIsolcpusMaskNoDelimiters, "0")).Should(Equal(strings.TrimLeft(workqueueMaskTrimmedNoDelimiters, "0")), "workqueueMask is not set to "+workqueueMaskTrimmed)
				By("executing the command \"cat /sys/bus/workqueue/devices/writeback/cpumask\"")
				workqueueWritebackMask, err := nodes.ExecCommandOnMachineConfigDaemon(&node, []string{"cat", "/sys/bus/workqueue/devices/writeback/cpumask"})
				Expect(err).ToNot(HaveOccurred())
				workqueueWritebackMaskTrimmed := strings.TrimSpace(string(workqueueWritebackMask))
				workqueueWritebackMaskTrimmedNoDelimiters := strings.Replace(workqueueWritebackMaskTrimmed, ",", "", -1)
				Expect(strings.TrimLeft(nonIsolcpusMaskNoDelimiters, "0")).Should(Equal(strings.TrimLeft(workqueueWritebackMaskTrimmedNoDelimiters, "0")), "workqueueMask is not set to "+workqueueWritebackMaskTrimmed)
			}
		})

		// Check /usr/local/bin/pre-boot-tuning.sh existence under worker's rootfs
		It("[test_id:28523][crit:high][vendor:cnf-qe@redhat.com][level:acceptance] /usr/local/bin/pre-boot-tuning.sh should exist on the nodes", func() {
			checkFileExistence(workerRTNodes, testutils.PerfRtKernelPrebootTuningScript)
		})

		It("[test_id:28525][crit:high][vendor:cnf-qe@redhat.com][level:acceptance] Should inject systemd configuration files into initramfs", func() {
			for _, node := range workerRTNodes {
				initramfsImagesPath, err := nodes.ExecCommandOnMachineConfigDaemon(&node, []string{"find", "/rootfs/boot/ostree/", "-name", "*.img"})
				Expect(err).ToNot(HaveOccurred())
				found := false
				imagesPath := strings.Split(string(initramfsImagesPath), "\n")
				for _, imagePath := range imagesPath[:2] {
					initrd, err := nodes.ExecCommandOnMachineConfigDaemon(&node,
						[]string{"lsinitrd", strings.TrimSpace(imagePath)})
					Expect(err).ToNot(HaveOccurred())
					initrdString := string(initrd)
					if strings.Contains(initrdString, "'/etc/systemd/system.conf /etc/systemd/system.conf.d/setAffinity.conf'") {
						found = true
						break
					}
				}
				Expect(found).Should(BeTrue())
			}
		})
	})

	Context("FeatureGate - FeatureSet configuration", func() {
		It("[test_id:28529][crit:high][vendor:cnf-qe@redhat.com][level:acceptance] FeatureGates with LatencySensitive should exist", func() {
			key := types.NamespacedName{
				Name:      components.FeatureGateLatencySensetiveName,
				Namespace: metav1.NamespaceNone,
			}
			fg := &ocv1.FeatureGate{}
			err := testclient.Client.Get(context.TODO(), key, fg)
			Expect(err).ToNot(HaveOccurred())

			lsStr := string(ocv1.LatencySensitive)
			By("Checking whether FeatureSet is configured as " + lsStr)
			Expect(string(fg.Spec.FeatureSet)).Should(Equal(lsStr), "FeauterSet is not set to "+lsStr)
		})
	})

	Context("Additional kernel arguments added from perfomance profile", func() {
		It("[test_id:28611][crit:high][vendor:cnf-qe@redhat.com][level:acceptance] Should set additional kernel arguments on the machine", func() {
			if profile.Spec.AdditionalKernelArgs != nil {
				additionalArgs := strings.Join(profile.Spec.AdditionalKernelArgs, " ")
				for _, node := range workerRTNodes {
					cmdline, err := nodes.ExecCommandOnMachineConfigDaemon(&node, []string{"cat", "/proc/cmdline"})
					Expect(err).ToNot(HaveOccurred())
					Expect(cmdline).To(ContainSubstring(additionalArgs))
				}
			}
		})
	})
})

// Check whether appropriate file exists on the system
func checkFileExistence(workerNodes []corev1.Node, file string) {
	for _, node := range workerNodes {
		By(fmt.Sprintf("Searching for the file %s.Executing the command \"ls /rootfs/%s\"", file, file))
		_, err := nodes.ExecCommandOnMachineConfigDaemon(&node, []string{"ls", "/rootfs/" + file})
		Expect(err).To(BeNil(), "cannot find the file "+file)
	}
}
