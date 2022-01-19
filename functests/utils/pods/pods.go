package pods

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client/config"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	testclient "github.com/openshift-kni/performance-addon-operators/functests/utils/client"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/images"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/namespaces"
)

//Container Resources
type CtnResources struct {
	cpu, memory, hpgSize, medium, noOfhpgs string
}

//Memory Manager Container structure
type MMContainer struct {
	CtnName, CtnImage string
	command           []string
	CtnResources
}

//Memory Manager Volume structure
type MMVolumes struct {
	volumenName, medium string
}

//Memory Manager Pod Definition
type MMPod struct {
	podV1Struct        *corev1.Pod
	podName, nameSpace string
	labels             map[string]string
	MMContainer
	MMVolumes
}

// DefaultDeletionTimeout contains the default pod deletion timeout in seconds
const DefaultDeletionTimeout = 120

const (
	hugepagesResourceName2Mi = "hugepages-2Mi"
	mediumHugepages2Mi       = "HugePages-2Mi"
)

// GetTestPod returns pod with the busybox image
func GetTestPod() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
			Labels: map[string]string{
				"test": "",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "test",
					Image:   images.Test(),
					Command: []string{"sleep", "10h"},
				},
			},
		},
	}
}

// WaitForDeletion waits until the pod will be removed from the cluster
func WaitForDeletion(pod *corev1.Pod, timeout time.Duration) error {
	key := types.NamespacedName{
		Name:      pod.Name,
		Namespace: pod.Namespace,
	}
	return wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		pod := &corev1.Pod{}
		if err := testclient.Client.Get(context.TODO(), key, pod); errors.IsNotFound(err) {
			return true, nil
		}
		return false, nil
	})
}

// WaitForCondition waits until the pod will have specified condition type with the expected status
func WaitForCondition(pod *corev1.Pod, conditionType corev1.PodConditionType, conditionStatus corev1.ConditionStatus, timeout time.Duration) error {
	key := types.NamespacedName{
		Name:      pod.Name,
		Namespace: pod.Namespace,
	}
	return wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		updatedPod := &corev1.Pod{}
		if err := testclient.Client.Get(context.TODO(), key, updatedPod); err != nil {
			return false, nil
		}

		for _, c := range updatedPod.Status.Conditions {
			if c.Type == conditionType && c.Status == conditionStatus {
				return true, nil
			}
		}
		return false, nil
	})
}

// WaitForPhase waits until the pod will have specified phase
func WaitForPhase(pod *corev1.Pod, phase corev1.PodPhase, timeout time.Duration) error {
	key := types.NamespacedName{
		Name:      pod.Name,
		Namespace: pod.Namespace,
	}
	return wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		updatedPod := &corev1.Pod{}
		if err := testclient.Client.Get(context.TODO(), key, updatedPod); err != nil {
			return false, nil
		}

		if updatedPod.Status.Phase == phase {
			return true, nil
		}

		return false, nil
	})
}

// GetLogs returns logs of the specified pod
func GetLogs(c *kubernetes.Clientset, pod *corev1.Pod) (string, error) {
	logStream, err := c.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{}).Stream(context.TODO())
	if err != nil {
		return "", err
	}
	defer logStream.Close()

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, logStream); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// ExecCommandOnPod runs command in the pod and returns buffer output
func ExecCommandOnPod(c *kubernetes.Clientset, pod *corev1.Pod, command []string) ([]byte, error) {
	var outputBuf bytes.Buffer
	var errorBuf bytes.Buffer

	req := c.CoreV1().RESTClient().
		Post().
		Namespace(pod.Namespace).
		Resource("pods").
		Name(pod.Name).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: pod.Spec.Containers[0].Name,
			Command:   command,
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec)

	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	exec, err := remotecommand.NewSPDYExecutor(cfg, "POST", req.URL())
	if err != nil {
		return nil, err
	}

	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  os.Stdin,
		Stdout: &outputBuf,
		Stderr: &errorBuf,
		Tty:    true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to run command %v: output %s; error %s", command, outputBuf.String(), errorBuf.String())
	}

	if errorBuf.Len() != 0 {
		return nil, fmt.Errorf("failed to run command %v: output %s; error %s", command, outputBuf.String(), errorBuf.String())
	}

	return outputBuf.Bytes(), nil
}

func WaitForPodOutput(c *kubernetes.Clientset, pod *corev1.Pod, command []string) ([]byte, error) {
	var out []byte
	if err := wait.PollImmediate(15*time.Second, time.Minute, func() (done bool, err error) {
		out, err = ExecCommandOnPod(c, pod, command)
		if err != nil {
			return false, err
		}

		return len(out) != 0, nil
	}); err != nil {
		return nil, err
	}

	return out, nil
}

// GetContainerIDByName returns container ID under the pod by the container name
func GetContainerIDByName(pod *corev1.Pod, containerName string) (string, error) {
	updatedPod := &corev1.Pod{}
	key := types.NamespacedName{
		Name:      pod.Name,
		Namespace: pod.Namespace,
	}
	if err := testclient.Client.Get(context.TODO(), key, updatedPod); err != nil {
		return "", err
	}
	for _, containerStatus := range updatedPod.Status.ContainerStatuses {
		if containerStatus.Name == containerName {
			return strings.Trim(containerStatus.ContainerID, "cri-o://"), nil
		}
	}
	return "", fmt.Errorf("failed to find the container ID for the container %q under the pod %q", containerName, pod.Name)
}

// GetPerformanceOperatorPod returns the pod running the Performance Profile Operator
func GetPerformanceOperatorPod() (*corev1.Pod, error) {
	selector, err := labels.Parse(fmt.Sprintf("%s=%s", "name", "performance-operator"))
	if err != nil {
		return nil, err
	}

	pods := &corev1.PodList{}

	opts := &client.ListOptions{LabelSelector: selector, Namespace: namespaces.PerformanceOperator}
	if err := testclient.Client.List(context.TODO(), pods, opts); err != nil {
		return nil, err
	}
	if len(pods.Items) != 1 {
		return nil, fmt.Errorf("incorrect performance operator pods count: %d", len(pods.Items))
	}

	return &pods.Items[0], nil
}

/*//Container Resources
type CtnResources struct {
	cpu, memory, hpgSize, medium, noOfhpgs string
}

//Memory Manager Container structure
type MMContainer struct {
	name, image string
	command     []string
	CtnResources
}

//Memory Manager Volume structure
type MMVolumes struct {
	volumenName, medium string
}

//Memory Manager Pod Definition
type MMPod struct {
	podV1Struct        *corev1.Pod
	podName, nameSpace string
	labels             map[string]string
	MMContainer
	MMVolumes
    }*/

//Create Container resources with cpu, memory and huagepages
func CreateCtnResources(ctn *MMContainer) *corev1.ResourceRequirements {

	ctnLimits := v1.ResourceList{
		v1.ResourceCPU:    resource.MustParse(ctn.cpu),
		v1.ResourceMemory: resource.MustParse(ctn.memory),
	}
	if ctn.hpgSize == "hugepages-2mi" {
		ctnLimits[hugepagesResourceName2Mi] = resource.MustParse(ctn.noOfhpgs)
	}
	return &v1.ResourceRequirements{
		Limits: ctnLimits,
	}
}

//Create Memory Manager Container
func CreateContainer(ctnData *MMContainer) *corev1.Container {
	mmctn := v1.Container{
		Name:      ctnData.CtnName,
		Image:     ctnData.CtnImage,
		Command:   ctnData.command,
		Resources: *CreateCtnResources(ctnData),
		VolumeMounts: []corev1.VolumeMount{
			*CreateVolumeMounts(ctnData),
		},
	}
	return &mmctn
}

//Create Volume Mounts
func CreateVolumeMounts(ctnData *MMContainer) *corev1.VolumeMount {
	if ctnData.hpgSize == "hugepages-2mi" {
		return &corev1.VolumeMount{
			Name:      "hugepages-2mi",
			MountPath: "/hugepages-2Mi",
		}
	} else {
		return &corev1.VolumeMount{
			Name:      "hugepage-1gi",
			MountPath: "/hugepages-1Gi",
		}
	}
}

//Pod Template to create pods with hugepages for memory manager tests
func MMPodTemplate(testPod *MMPod, targetNode *corev1.Node) *corev1.Pod {
	testNode := make(map[string]string)
	testNode["kubernetes.io/hostname"] = targetNode.Name
	testPod.podV1Struct = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-pod", testPod.podName),
			Namespace: testPod.nameSpace,
			Labels: map[string]string{
				"name": testPod.podName,
			},
		},
		Spec: v1.PodSpec{
			Containers: []corev1.Container{
				*CreateContainer(&testPod.MMContainer),
			},
			Volumes: []corev1.Volume{
				{
					Name: testPod.hpgSize,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{
							Medium: corev1.StorageMedium(testPod.medium),
						},
					},
				},
			},
			NodeSelector: testNode,
		},
	}
	return testPod.podV1Struct
}

func (m *MMPod) DefaultPod() {
	m.nameSpace = "default"
	m.CtnImage = "fedora:latest"
	m.cpu = "2"
	m.memory = "200Mi"
	m.command = []string{"sleep", "inf"}
	m.hpgSize = "hugepages-2mi"
	m.noOfhpgs = "24Mi"
	m.podName = "example1"
	m.CtnName = fmt.Sprintf("%s-container", m.podName)
}
