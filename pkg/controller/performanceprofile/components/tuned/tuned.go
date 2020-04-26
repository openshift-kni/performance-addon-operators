package tuned

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"sort"
	"strconv"
	"strings"
	"text/template"

	performancev1alpha1 "github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v1alpha1"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"
	tunedv1 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/tuned/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

const (
	cmdlineDelimiter             = " "
	templateIsolatedCpus         = "IsolatedCpus"
	templateStaticIsolation      = "StaticIsolation"
	templateDefaultHugepagesSize = "DefaultHugepagesSize"
	templateHugepages            = "Hugepages"
	templateAdditionalArgs       = "AdditionalArgs"
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
func NewNodePerformance(assetsDir string, profile *performancev1alpha1.PerformanceProfile) (*tunedv1.Tuned, error) {

	templateArgs := make(map[string]string)

	if profile.Spec.CPU.Isolated != nil {
		templateArgs[templateIsolatedCpus] = string(*profile.Spec.CPU.Isolated)
		if profile.Spec.CPU.BalanceIsolated != nil && *profile.Spec.CPU.BalanceIsolated == false {
			templateArgs[templateStaticIsolation] = strconv.FormatBool(true)
		}
	}

	if profile.Spec.HugePages != nil {
		if profile.Spec.HugePages.DefaultHugePagesSize != nil {
			templateArgs[templateDefaultHugepagesSize] = string(*profile.Spec.HugePages.DefaultHugePagesSize)
		}

		hugepages := []string{}
		for _, page := range profile.Spec.HugePages.Pages {
			// we can not allocate hugepages on the specific NUMA node via kernel boot arguments
			if page.Node != nil {
				continue
			}
			hugepages = append(hugepages, fmt.Sprintf("hugepagesz=%s", string(page.Size)))
			hugepages = append(hugepages, fmt.Sprintf("hugepages=%d", page.Count))
		}
		hugepagesArgs := strings.Join(hugepages, cmdlineDelimiter)
		templateArgs[templateHugepages] = hugepagesArgs
	}

	if profile.Spec.AdditionalKernelArgs != nil {
		templateArgs[templateAdditionalArgs] = strings.Join(profile.Spec.AdditionalKernelArgs, cmdlineDelimiter)
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

	// we should sort our matches, otherwise we can not predict the order of nested matches
	sortedKeys := []string{}
	for k := range profile.Spec.NodeSelector {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	priority := uint64(30)
	copyNodeSelector := map[string]string{}
	for k, v := range profile.Spec.NodeSelector {
		copyNodeSelector[k] = v
	}
	recommends := []tunedv1.TunedRecommend{
		{
			Profile:  &name,
			Priority: &priority,
			Match:    getProfileMatches(sortedKeys, copyNodeSelector),
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

func getProfileMatches(sortedKeys []string, matchNodeLabels map[string]string) []tunedv1.TunedMatch {
	matches := []tunedv1.TunedMatch{}
	for _, label := range sortedKeys {
		value, ok := matchNodeLabels[label]
		if !ok {
			continue
		}

		delete(matchNodeLabels, label)
		matches = append(matches, tunedv1.TunedMatch{
			Label: pointer.StringPtr(label),
			Value: pointer.StringPtr(value),
			Match: getProfileMatches(sortedKeys, matchNodeLabels),
		})
	}
	return matches
}
