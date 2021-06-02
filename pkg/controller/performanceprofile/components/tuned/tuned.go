package tuned

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"text/template"

	"k8s.io/utils/pointer"

	"github.com/hashicorp/go-version"
	performancev2 "github.com/openshift-kni/performance-addon-operators/api/v2"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"
	componentsprofile "github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components/profile"
	tunedv1 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/tuned/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	cmdlineDelimiter                        = " "
	minimalStalldClusterVersion             = "4.7.7"
	templateIsolatedCpus                    = "IsolatedCpus"
	templateStaticIsolation                 = "StaticIsolation"
	templateDefaultHugepagesSize            = "DefaultHugepagesSize"
	templateHugepages                       = "Hugepages"
	templateAdditionalArgs                  = "AdditionalArgs"
	templateGloballyDisableIrqLoadBalancing = "GloballyDisableIrqLoadBalancing"
	templateEnabledStalld                   = "EnableStalld"
)

func new(name string, profiles []tunedv1.TunedProfile, recommends []tunedv1.TunedRecommend) *tunedv1.Tuned {
	return &tunedv1.Tuned{
		TypeMeta: metav1.TypeMeta{
			APIVersion: tunedv1.SchemeGroupVersion.String(),
			Kind:       "Tuned",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: components.NamespaceNodeTuningOperator,
		},
		Spec: tunedv1.TunedSpec{
			Profile:   profiles,
			Recommend: recommends,
		},
	}
}

// NewNodePerformance returns tuned profile for performance sensitive workflows
func NewNodePerformance(assetsDir string, profile *performancev2.PerformanceProfile, clusterVersion string) (*tunedv1.Tuned, error) {

	templateArgs := make(map[string]string)

	if profile.Spec.CPU.Isolated != nil {
		templateArgs[templateIsolatedCpus] = string(*profile.Spec.CPU.Isolated)
		if profile.Spec.CPU.BalanceIsolated != nil && *profile.Spec.CPU.BalanceIsolated == false {
			templateArgs[templateStaticIsolation] = strconv.FormatBool(true)
		}
	}

	if profile.Spec.HugePages != nil {
		var defaultHugepageSize performancev2.HugePageSize
		if profile.Spec.HugePages.DefaultHugePagesSize != nil {
			defaultHugepageSize = *profile.Spec.HugePages.DefaultHugePagesSize
			templateArgs[templateDefaultHugepagesSize] = string(defaultHugepageSize)
		}

		var is2MHugepagesRequested *bool
		var hugepages []string
		for _, page := range profile.Spec.HugePages.Pages {
			// we can not allocate huge pages on the specific NUMA node via kernel boot arguments
			if page.Node != nil {
				// a user requested to allocate 2M huge pages on the specific NUMA node,
				// append dummy kernel arguments
				if page.Size == components.HugepagesSize2M && is2MHugepagesRequested == nil {
					is2MHugepagesRequested = pointer.BoolPtr(true)
				}
				continue
			}

			// a user requested to allocated 2M huge pages without specifying the node
			// we need to append 2M hugepages kernel arguments anyway, no need to add dummy
			// kernel arguments
			if page.Size == components.HugepagesSize2M {
				is2MHugepagesRequested = pointer.BoolPtr(false)
			}

			hugepages = append(hugepages, fmt.Sprintf("hugepagesz=%s", string(page.Size)))
			hugepages = append(hugepages, fmt.Sprintf("hugepages=%d", page.Count))
		}

		// append dummy 2M huge pages kernel arguments to guarantee that the kernel will create 2M related files
		// and directories under the filesystem
		if is2MHugepagesRequested != nil && *is2MHugepagesRequested {
			if defaultHugepageSize == components.HugepagesSize1G {
				hugepages = append(hugepages, fmt.Sprintf("hugepagesz=%s", components.HugepagesSize2M))
				hugepages = append(hugepages, fmt.Sprintf("hugepages=%d", 0))
			}
		}

		hugepagesArgs := strings.Join(hugepages, cmdlineDelimiter)
		templateArgs[templateHugepages] = hugepagesArgs
	}

	if profile.Spec.AdditionalKernelArgs != nil {
		templateArgs[templateAdditionalArgs] = strings.Join(profile.Spec.AdditionalKernelArgs, cmdlineDelimiter)
	}

	if profile.Spec.GloballyDisableIrqLoadBalancing != nil &&
		*profile.Spec.GloballyDisableIrqLoadBalancing == true {
		templateArgs[templateGloballyDisableIrqLoadBalancing] = strconv.FormatBool(true)
	}

	currentClusterVersion, err := version.NewVersion(clusterVersion)
	if err != nil {
		return nil, err
	}
	requiredStalldClusterVersion, err := version.NewVersion(minimalStalldClusterVersion)
	if err != nil {
		return nil, err
	}

	if isNonStableRelease(clusterVersion) || currentClusterVersion.GreaterThanOrEqual(requiredStalldClusterVersion) {
		templateArgs[templateEnabledStalld] = strconv.FormatBool(true)
	}

	profileData, err := getProfileData(getProfilePath(components.ProfileNamePerformance, assetsDir), templateArgs)

	if err != nil {
		return nil, err
	}

	name := components.GetComponentName(profile.Name, components.ProfileNamePerformance)
	profiles := []tunedv1.TunedProfile{
		{
			Name: &name,
			Data: &profileData,
		},
	}

	priority := uint64(20)
	recommends := []tunedv1.TunedRecommend{
		{
			Profile:             &name,
			Priority:            &priority,
			MachineConfigLabels: componentsprofile.GetMachineConfigLabel(profile),
		},
	}
	return new(name, profiles, recommends), nil
}

func getProfilePath(name string, assetsDir string) string {
	return fmt.Sprintf("%s/tuned/%s", assetsDir, name)
}

func getProfileData(profileOperatorlPath string, data interface{}) (string, error) {
	profileContent, err := ioutil.ReadFile(profileOperatorlPath)
	if err != nil {
		return "", err
	}

	profile := &bytes.Buffer{}
	profileTemplate := template.Must(template.New("profile").Parse(string(profileContent)))
	if err := profileTemplate.Execute(profile, data); err != nil {
		return "", err
	}
	return profile.String(), nil
}

func isNonStableRelease(clusterVersion string) bool {
	if strings.Contains(clusterVersion, "ci") || strings.Contains(clusterVersion, "nightly") {
		return true
	}
	return false
}
