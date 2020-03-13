package main

import (
	"encoding/json"
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

	strategySpec := csvtools.UnmarshalStrategySpec(operatorCSV)

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

	csvtools.MarshallObject(operatorCSV, dst)
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
