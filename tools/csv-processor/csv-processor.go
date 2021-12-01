package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	semver "github.com/blang/semver/v4"
	"github.com/openshift-kni/performance-addon-operators/pkg/utils/csvtools"
	"github.com/operator-framework/api/pkg/lib/version"
	csvv1 "github.com/operator-framework/api/pkg/operators/v1alpha1"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

var (
	minKubeVersion      = flag.String("min-kube-version", "", "minimum required kubernetes version")
	csvVersion          = flag.String("csv-version", "", "the unified CSV version")
	replacesCsvVersion  = flag.String("replaces-csv-version", "", "the unified CSV version this new CSV will replace")
	skipRange           = flag.String("skip-range", "", "the CSV version skip range")
	operatorCSVTemplate = flag.String("operator-csv-template-file", "", "path to csv template example")

	operatorImage = flag.String("operator-image", "", "operator container image")

	inputManifestsDir = flag.String("manifests-directory", "", "The directory containing the extra manifests to be included in the registry bundle")

	outputDir = flag.String("olm-bundle-directory", "", "The directory to output the unified CSV and CRDs to")

	annotationsFile = flag.String("annotations-from", "", "add metadata annotations from given file")
	maintainersFile = flag.String("maintainers-from", "", "add maintainers list from given file")
	descriptionFile = flag.String("description-from", "", "replace the description with the content of the given file")
	iconFile        = flag.String("icon-from", "", "inject icon from given file (must be svg)")
)

func finalizedCsvFilename() string {
	return "performance-addon-operator.v" + *csvVersion + ".clusterserviceversion.yaml"
}

type csvUserData struct {
	Description      string
	ExtraAnnotations map[string]string
	Maintainers      map[string]string

	CSVVersion     *semver.Version
	MinKubeVersion *semver.Version
}

func generateUnifiedCSV(userData csvUserData) {

	operatorCSV := csvtools.UnmarshalCSV(*operatorCSVTemplate)

	strategySpec := operatorCSV.Spec.InstallStrategy.StrategySpec

	// this forces us to update this logic if another deployment is introduced.
	if len(strategySpec.DeploymentSpecs) != 1 {
		panic(fmt.Errorf("expected 1 deployment, found %d", len(strategySpec.DeploymentSpecs)))
	}

	strategySpec.DeploymentSpecs[0].Spec.Template.Spec.Containers[0].Image = *operatorImage

	// set operator resource requests
	cpuRequest := resource.MustParse("10m")
	memoryRequest := resource.MustParse("25Mi")
	strategySpec.DeploymentSpecs[0].Spec.Template.Spec.Containers[0].Resources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    cpuRequest,
			corev1.ResourceMemory: memoryRequest,
		},
	}

	// set operator toleration
	strategySpec.DeploymentSpecs[0].Spec.Template.Spec.Tolerations = []corev1.Toleration{
		{
			// Just tolerate NoSchedule taint on master node. If there are other conditions like disk-pressure etc, let's not schedule the control-plane pods onto that node.
			Key:      "node-role.kubernetes.io/master",
			Operator: corev1.TolerationOpExists,
			Effect:   corev1.TaintEffectNoSchedule,
		},
		{
			Key:               "node.kubernetes.io/unreachable",
			Operator:          corev1.TolerationOpExists,
			Effect:            corev1.TaintEffectNoExecute,
			TolerationSeconds: pointer.Int64Ptr(120), // Evict pods within 2 minutes.
		},
		{
			Key:               "node.kubernetes.io/not-ready",
			Operator:          corev1.TolerationOpExists,
			Effect:            corev1.TaintEffectNoExecute,
			TolerationSeconds: pointer.Int64Ptr(120), // Evict pods within 2 minutes.
		},
	}

	// Inject display names and descriptions for our crds
	for i, definition := range operatorCSV.Spec.CustomResourceDefinitions.Owned {
		switch definition.Name {
		case "performanceprofiles.performance.openshift.io":
			operatorCSV.Spec.CustomResourceDefinitions.Owned[i].DisplayName = fmt.Sprintf("Performance Profile %s", operatorCSV.Spec.CustomResourceDefinitions.Owned[i].Version)
			operatorCSV.Spec.CustomResourceDefinitions.Owned[i].Description =
				fmt.Sprintf("PerformanceProfile is the Schema for the performanceprofiles API %v.", operatorCSV.Spec.CustomResourceDefinitions.Owned[i].Version)
		}
	}

	operatorCSV.Annotations["containerImage"] = *operatorImage
	for key, value := range userData.ExtraAnnotations {
		operatorCSV.Annotations[key] = value
	}

	// Set correct csv versions and name
	v := version.OperatorVersion{Version: *userData.CSVVersion}
	operatorCSV.Spec.Version = v
	operatorCSV.Name = "performance-addon-operator.v" + *csvVersion
	if *replacesCsvVersion != "" {
		operatorCSV.Spec.Replaces = "performance-addon-operator.v" + *replacesCsvVersion
	} else {
		operatorCSV.Spec.Replaces = ""
	}

	// Set api maturity
	operatorCSV.Spec.Maturity = "alpha"

	// Set links
	operatorCSV.Spec.Links = []csvv1.AppLink{
		{
			Name: "Source Code",
			URL:  "https://github.com/openshift-kni/performance-addon-operators",
		},
	}

	// Set Keywords
	operatorCSV.Spec.Keywords = []string{
		"numa",
		"realtime",
		"cpu pinning",
		"hugepages",
		"pao",
	}

	// Set Provider
	operatorCSV.Spec.Provider = csvv1.AppLink{
		Name: "Red Hat",
	}

	// Set Description
	operatorCSV.Spec.Description = `
Performance Addon Operator provides the ability to enable advanced node performance tunings on a set of nodes.`
	if userData.Description != "" {
		operatorCSV.Spec.Description = userData.Description
	}

	operatorCSV.Spec.DisplayName = "Performance Addon Operator"

	if userData.Maintainers != nil {
		for name, email := range userData.Maintainers {
			operatorCSV.Spec.Maintainers = append(operatorCSV.Spec.Maintainers, csvv1.Maintainer{
				Name:  name,
				Email: email,
			})
		}
		// Override generator default values
		if len(userData.Maintainers) == 0 {
			operatorCSV.Spec.Maintainers = nil
		}
	}

	if *iconFile != "" {
		operatorCSV.Spec.Icon = []csvv1.Icon{
			{
				Data:      base64.StdEncoding.EncodeToString(readFileOrPanic(*iconFile)),
				MediaType: "image/svg+xml",
			},
		}
	} else {
		operatorCSV.Spec.Icon = nil
	}

	if *skipRange != "" {
		operatorCSV.Annotations["olm.skipRange"] = *skipRange
	}

	// Setup the Conversion Webhook
	targetPort := intstr.FromInt(4343)
	sideEffects := admissionregistrationv1.SideEffectClassNone
	conversionWebhookPath := "/convert"
	validationWebhookPath := "/validate-performance-openshift-io-v2-performanceprofile"
	failurePolicy := admissionregistrationv1.Fail

	operatorCSV.Spec.WebhookDefinitions = []csvv1.WebhookDescription{
		{
			GenerateName:            "cwb.performance.openshift.io",
			Type:                    csvv1.ConversionWebhook,
			DeploymentName:          strategySpec.DeploymentSpecs[0].Name,
			ContainerPort:           443,
			TargetPort:              &targetPort,
			SideEffects:             &sideEffects,
			AdmissionReviewVersions: []string{"v1", "v1alpha1"},
			WebhookPath:             &conversionWebhookPath,
			ConversionCRDs:          []string{"performanceprofiles.performance.openshift.io"},
		},
		{
			GenerateName:            "vwb.performance.openshift.io",
			Type:                    csvv1.ValidatingAdmissionWebhook,
			DeploymentName:          strategySpec.DeploymentSpecs[0].Name,
			ContainerPort:           443,
			TargetPort:              &targetPort,
			SideEffects:             &sideEffects,
			AdmissionReviewVersions: []string{"v1beta1"},
			WebhookPath:             &validationWebhookPath,
			FailurePolicy:           &failurePolicy,
			Rules: []admissionregistrationv1.RuleWithOperations{
				{
					Operations: []admissionregistrationv1.OperationType{
						admissionregistrationv1.Create,
						admissionregistrationv1.Update,
					},
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{"performance.openshift.io"},
						APIVersions: []string{"v2"},
						Resources:   []string{"performanceprofiles"},
					},
				},
			},
		},
	}

	// Required by OLM for Conversion Webhooks
	operatorCSV.Spec.InstallModes = []csvv1.InstallMode{
		{Type: csvv1.InstallModeTypeOwnNamespace, Supported: false},
		{Type: csvv1.InstallModeTypeSingleNamespace, Supported: false},
		{Type: csvv1.InstallModeTypeMultiNamespace, Supported: false},
		{Type: csvv1.InstallModeTypeAllNamespaces, Supported: true},
	}

	if userData.MinKubeVersion != nil {
		operatorCSV.Spec.MinKubeVersion = userData.MinKubeVersion.String()
	}

	// write CSV to out dir
	writer := strings.Builder{}
	csvtools.MarshallObject(operatorCSV, &writer)
	outputFilename := filepath.Join(*outputDir, finalizedCsvFilename())
	err := ioutil.WriteFile(outputFilename, []byte(writer.String()), 0644)
	if err != nil {
		panic(err)
	}

	fmt.Printf("CSV written to %s\n", outputFilename)
}

func readFileOrPanic(filename string) []byte {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	return data
}

func readKeyValueMapFromFileOrPanic(filename string) map[string]string {
	kvMap := make(map[string]string)
	if err := json.Unmarshal(readFileOrPanic(filename), &kvMap); err != nil {
		panic(err)
	}
	return kvMap
}

func main() {
	flag.Parse()

	if *csvVersion == "" {
		log.Fatal("--csv-version is required")
	} else if *operatorCSVTemplate == "" {
		log.Fatal("--operator-csv-template-file is required")
	} else if *operatorImage == "" {
		log.Fatal("--operator-image is required")
	} else if *outputDir == "" {
		log.Fatal("--olm-bundle-directory is required")
	}

	userData := csvUserData{
		Description: `
Performance Addon Operator provides the ability to enable advanced node performance tunings on a set of nodes.`,
		ExtraAnnotations: make(map[string]string),
		Maintainers:      make(map[string]string),
	}

	var err error
	// Set correct csv versions and name
	userData.CSVVersion, err = semver.New(*csvVersion)
	if err != nil {
		panic(err)
	}

	if *minKubeVersion != "" {
		// per https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/design/building-your-csv.md
		// The Kubernetes version must be in "Major.Minor.Patch" format
		userData.MinKubeVersion, err = semver.New(*minKubeVersion)
		if err != nil {
			panic(err)
		}
	}

	if *annotationsFile != "" {
		userData.ExtraAnnotations = readKeyValueMapFromFileOrPanic(*annotationsFile)
	}
	if *maintainersFile != "" {
		userData.Maintainers = readKeyValueMapFromFileOrPanic(*maintainersFile)
	}
	if *descriptionFile != "" {
		userData.Description = string(readFileOrPanic(*descriptionFile))
	}

	// start with a fresh output directory if it already exists
	os.RemoveAll(*outputDir)

	// create output directory
	os.MkdirAll(*outputDir, os.FileMode(0775))

	generateUnifiedCSV(userData)
}
