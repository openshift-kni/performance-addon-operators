package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/openshift-kni/performance-addon-operators/pkg/utils/csvtools"
)

var (
	csvInput      = flag.String("csv-input", "", "path to csv to update")
	operatorImage = flag.String("operator-image", "", "operator container image")
)

func processCSV(operatorImage, csvInput string, dst io.Writer) {
	operatorCSV := csvtools.UnmarshalCSV(csvInput)

	strategySpec := operatorCSV.Spec.InstallStrategy.StrategySpec

	// this forces us to update this logic if another deployment is introduced.
	if len(strategySpec.DeploymentSpecs) != 1 {
		panic(fmt.Errorf("expected 1 deployment, found %d", len(strategySpec.DeploymentSpecs)))
	}

	strategySpec.DeploymentSpecs[0].Spec.Template.Spec.Containers[0].Image = operatorImage

	operatorCSV.Annotations["containerImage"] = operatorImage

	if err := csvtools.MarshallObject(operatorCSV, dst); err != nil {
		panic(fmt.Errorf("could not marshall CSV, err: %s", err))
	}
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
