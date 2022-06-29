package main

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/ghodss/yaml"

	machineconfigv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	performancev2 "github.com/openshift-kni/performance-addon-operators/api/v2"
	"github.com/openshift-kni/performance-addon-operators/pkg/utils/hugepages"
)

var (
	nodeRole   = flag.String("n", "worker", "node role of the machine config object")
	inputFile  = flag.String("i", "", "performance profile yaml file")
	outputFile = flag.String("o", "", "performance profile yaml file")
)

func main() {
	flag.Parse()

	var err error
	var bytes []byte
	if inputFile == nil || *inputFile == "" {
		// reads the full content of stdin - possibly a large block of data
		bytes, err = ioutil.ReadAll(os.Stdin)
	} else {
		//reads the full content of the input file
		bytes, err = ioutil.ReadFile(*inputFile)
	}
	if err != nil {
		log.Fatalf("failed to read the input: %v", err)
	}
	profile := &performancev2.PerformanceProfile{}
	err = yaml.Unmarshal(bytes, profile)
	if err != nil {
		log.Fatalf("failed to unmarshal the input into performance profile: %v", err)
	}

	mc, err := createMachineConfig(profile, nodeRole)
	if err != nil {
		log.Fatalf("failed to generate a machine config with hugepages settings: %v", err)
	}

	y, err := yaml.Marshal(mc)
	if err != nil {
		log.Fatalf("failed to get the machine config as yaml file: %v", err)
	}

	manifest := string(y)
	var sink io.Writer = os.Stdout
	if outputFile != nil && *outputFile != "" {
		f, err := os.Create(*outputFile)
		if err != nil {
			log.Fatalf("error opening %s: %v\n", *outputFile, err)
		}
		defer f.Close()
		sink = f
	}
	// writes all the content to the destination file in one go
	_, err = io.WriteString(sink, manifest)
	if err != nil {
		log.Fatalf("unable to write the output: %v", err)
	}

}

func createMachineConfig(profile *performancev2.PerformanceProfile, noderole *string) (*machineconfigv1.MachineConfig, error) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Fatalf("missing page.Node: %v", rec)
		}
	}()
	return hugepages.MakeMachineConfig(profile.Spec.HugePages, *nodeRole)
}
