package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/blang/semver"
	yaml "github.com/ghodss/yaml"
	csvv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/version"
	appsv1 "k8s.io/api/apps/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type csvClusterPermissions struct {
	ServiceAccountName string              `json:"serviceAccountName"`
	Rules              []rbacv1.PolicyRule `json:"rules"`
}

type csvPermissions struct {
	ServiceAccountName string              `json:"serviceAccountName"`
	Rules              []rbacv1.PolicyRule `json:"rules"`
}

type csvDeployments struct {
	Name string                `json:"name"`
	Spec appsv1.DeploymentSpec `json:"spec,omitempty"`
}

type csvStrategySpec struct {
	ClusterPermissions []csvClusterPermissions `json:"clusterPermissions"`
	Permissions        []csvPermissions        `json:"permissions"`
	Deployments        []csvDeployments        `json:"deployments"`
}

var (
	csvVersion          = flag.String("csv-version", "", "the unified CSV version")
	replacesCsvVersion  = flag.String("replaces-csv-version", "", "the unified CSV version this new CSV will replace")
	skipRange           = flag.String("skip-range", "", "the CSV version skip range")
	operatorCSVTemplate = flag.String("operator-csv-template-file", "", "path to csv template example")

	operatorImage = flag.String("operator-image", "", "operator container image")

	inputManifestsDir = flag.String("manifests-directory", "", "The directory containing the extra manifests to be included in the registry bundle")

	outputDir = flag.String("olm-bundle-directory", "", "The directory to output the unified CSV and CRDs to")
)

func finalizedCsvFilename() string {
	return "performance-addon-operator.v" + *csvVersion + ".clusterserviceversion.yaml"
}

func copyFile(src string, dst string) {
	srcFile, err := os.Open(src)
	if err != nil {
		panic(err)
	}
	defer srcFile.Close()

	outFile, err := os.Create(dst)
	if err != nil {
		panic(err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, srcFile)
	if err != nil {
		panic(err)
	}
}

func unmarshalCSV(filePath string) *csvv1.ClusterServiceVersion {

	fmt.Printf("reading in csv at %s\n", filePath)
	bytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		panic(err)
	}

	csvStruct := &csvv1.ClusterServiceVersion{}
	err = yaml.Unmarshal(bytes, csvStruct)
	if err != nil {
		panic(err)
	}

	return csvStruct
}

func unmarshalStrategySpec(csv *csvv1.ClusterServiceVersion) *csvStrategySpec {

	templateStrategySpec := &csvStrategySpec{}
	err := json.Unmarshal(csv.Spec.InstallStrategy.StrategySpecRaw, templateStrategySpec)
	if err != nil {
		panic(err)
	}

	return templateStrategySpec
}

func marshallObject(obj interface{}, writer io.Writer) error {
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	var r unstructured.Unstructured
	if err := json.Unmarshal(jsonBytes, &r.Object); err != nil {
		return err
	}

	// remove status and metadata.creationTimestamp
	unstructured.RemoveNestedField(r.Object, "metadata", "creationTimestamp")
	unstructured.RemoveNestedField(r.Object, "template", "metadata", "creationTimestamp")
	unstructured.RemoveNestedField(r.Object, "spec", "template", "metadata", "creationTimestamp")
	unstructured.RemoveNestedField(r.Object, "status")

	deployments, exists, err := unstructured.NestedSlice(r.Object, "spec", "install", "spec", "deployments")
	if exists {
		for _, obj := range deployments {
			deployment := obj.(map[string]interface{})
			unstructured.RemoveNestedField(deployment, "metadata", "creationTimestamp")
			unstructured.RemoveNestedField(deployment, "spec", "template", "metadata", "creationTimestamp")
			unstructured.RemoveNestedField(deployment, "status")
		}
		unstructured.SetNestedSlice(r.Object, deployments, "spec", "install", "spec", "deployments")
	}

	jsonBytes, err = json.Marshal(r.Object)
	if err != nil {
		return err
	}

	yamlBytes, err := yaml.JSONToYAML(jsonBytes)
	if err != nil {
		return err
	}

	// fix double quoted strings by removing unneeded single quotes...
	s := string(yamlBytes)
	s = strings.Replace(s, " '\"", " \"", -1)
	s = strings.Replace(s, "\"'\n", "\"\n", -1)

	yamlBytes = []byte(s)

	_, err = writer.Write([]byte("---\n"))
	if err != nil {
		return err
	}

	_, err = writer.Write(yamlBytes)
	if err != nil {
		return err
	}

	return nil
}

func generateUnifiedCSV() {

	operatorCSV := unmarshalCSV(*operatorCSVTemplate)

	strategySpec := unmarshalStrategySpec(operatorCSV)

	// this forces us to update this logic if another deployment is introduced.
	if len(strategySpec.Deployments) != 1 {
		panic(fmt.Errorf("expected 1 deployment, found %d", len(strategySpec.Deployments)))
	}

	strategySpec.Deployments[0].Spec.Template.Spec.Containers[0].Image = *operatorImage

	// Inject deplay names and descriptions for our crds
	for i, definition := range operatorCSV.Spec.CustomResourceDefinitions.Owned {
		switch definition.Name {
		case "performanceprofiles.performance.openshift.io":
			operatorCSV.Spec.CustomResourceDefinitions.Owned[i].DisplayName = "Performance Profile"
		}
	}

	// Re-serialize deployments and permissions into csv strategy.
	updatedStrat, err := json.Marshal(strategySpec)
	if err != nil {
		panic(err)
	}
	operatorCSV.Spec.InstallStrategy.StrategySpecRaw = updatedStrat

	// Set correct csv versions and name
	semverVersion, err := semver.New(*csvVersion)
	if err != nil {
		panic(err)
	}
	v := version.OperatorVersion{Version: *semverVersion}
	operatorCSV.Spec.Version = v
	operatorCSV.Name = "performance-addon-operator.v" + *csvVersion
	if *replacesCsvVersion != "" {
		operatorCSV.Spec.Replaces = "performance-addon-operator.v" + *replacesCsvVersion
	}

	// Set api maturity
	operatorCSV.Spec.Maturity = "alpha"

	// Set links
	operatorCSV.Spec.Links = []csvv1.AppLink{
		{
			Name: "Source Code",
			URL:  "https://github.com/openshift-kni/performance-addon-operator",
		},
	}

	// Set Keywords
	operatorCSV.Spec.Keywords = []string{
		"numa",
		"realtime",
		"cpu pinning",
		"hugepages",
	}

	// Set Provider
	operatorCSV.Spec.Provider = csvv1.AppLink{
		Name: "Red Hat",
	}

	// Set Description
	operatorCSV.Spec.Description = `
Performance Addon Operator provides the ability to enable advanced node performance tunings on a set of nodes.`

	operatorCSV.Spec.DisplayName = "Performance Addon Operator"

	// Set Annotations
	if *skipRange != "" {
		operatorCSV.Annotations["olm.skipRange"] = *skipRange
	}

	// write CSV to out dir
	writer := strings.Builder{}
	marshallObject(operatorCSV, &writer)
	err = ioutil.WriteFile(filepath.Join(*outputDir, finalizedCsvFilename()), []byte(writer.String()), 0644)
	if err != nil {
		panic(err)
	}

	fmt.Printf("CSV written to %s\n", filepath.Join(*outputDir, finalizedCsvFilename()))
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

	// start with a fresh output directory if it already exists
	os.RemoveAll(*outputDir)

	// create output directory
	os.MkdirAll(*outputDir, os.FileMode(0755))

	generateUnifiedCSV()
}
