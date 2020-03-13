package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	yaml "github.com/ghodss/yaml"
	csvv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
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
	csvInput      = flag.String("csv-input", "", "path to csv to update")
	operatorImage = flag.String("operator-image", "", "operator container image")
)

func unmarshalCSV(filePath string) *csvv1.ClusterServiceVersion {
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

func processCSV(operatorImage, csvInput string, dst io.Writer) {
	operatorCSV := unmarshalCSV(csvInput)

	strategySpec := unmarshalStrategySpec(operatorCSV)

	// this forces us to update this logic if another deployment is introduced.
	if len(strategySpec.Deployments) != 1 {
		panic(fmt.Errorf("expected 1 deployment, found %d", len(strategySpec.Deployments)))
	}

	strategySpec.Deployments[0].Spec.Template.Spec.Containers[0].Image = operatorImage

	// Re-serialize deployments and permissions into csv strategy.
	updatedStrat, err := json.Marshal(strategySpec)
	if err != nil {
		panic(err)
	}
	operatorCSV.Spec.InstallStrategy.StrategySpecRaw = updatedStrat

	operatorCSV.Annotations["containerImage"] = operatorImage

	marshallObject(operatorCSV, dst)
}

func main() {
	flag.Parse()

	if *csvInput == "" {
		log.Fatal("--csv-input is required")
	} else if *operatorImage == "" {
		log.Fatal("--operator-image is required")
	}

	processCSV(*operatorImage, *csvInput, os.Stdout)
}
