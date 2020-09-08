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

	"github.com/openshift-kni/performance-addon-operators/pkg/utils/csvtools"
)

const (
	containerImageAnnotationKey = "containerImage"
	csvSuffix                   = "clusterserviceversion.yaml"
	csvTmpFilePrefix            = "tmp_csv"
)

var (
	catalogRoot   = flag.String("catalog-root", "", "path to the catalog root")
	csvInput      = flag.String("csv-input", "", "path to csv to update")
	operatorImage = flag.String("operator-image", "", "operator container image")
)

func processCSV(operatorImage, csvInput string, dst io.Writer) error {
	operatorCSV := csvtools.UnmarshalCSV(csvInput)

	strategySpec := csvtools.UnmarshalStrategySpec(operatorCSV)

	// this forces us to update this logic if another deployment is introduced.
	if len(strategySpec.Deployments) != 1 {
		return fmt.Errorf("expected 1 deployment, found %d", len(strategySpec.Deployments))
	}

	strategySpec.Deployments[0].Spec.Template.Spec.Containers[0].Image = operatorImage

	// Re-serialize deployments and permissions into csv strategy.
	updatedStrat, err := json.Marshal(strategySpec)
	if err != nil {
		return err
	}
	operatorCSV.Spec.InstallStrategy.StrategySpecRaw = updatedStrat

	operatorCSV.Annotations[containerImageAnnotationKey] = operatorImage

	csvtools.MarshallObject(operatorCSV, dst)

	return nil
}

func main() {
	flag.Parse()

	if (*csvInput == "" && *catalogRoot == "") || (*csvInput != "" && *catalogRoot != "") {
		log.Fatal("either --csv-input or --catalog-root is required")
	}
	if *operatorImage == "" {
		log.Fatal("--operator-image is required")
	}

	var err error
	if *catalogRoot == "" {
		err = processCSV(*operatorImage, *csvInput, os.Stdout)
	} else {
		err = filepath.Walk(*catalogRoot, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !strings.HasSuffix(path, csvSuffix) {
				return nil
			}
			tmpCsv, err := ioutil.TempFile(".", csvTmpFilePrefix)
			if err != nil {
				return err
			}

			tmpCsvName := tmpCsv.Name()
			log.Printf("fixing %q with %q (to %q)", path, *operatorImage, tmpCsvName)
			processCSV(*operatorImage, path, tmpCsv)
			err = tmpCsv.Close()
			if err != nil {
				return err
			}

			err = os.Rename(tmpCsvName, path)
			if err != nil {
				log.Printf("failed renaming %q to %q (err=%v)", tmpCsvName, path, err)
			}
			return err
		})
	}
	if err != nil {
		log.Fatalf("failed: %v", err)
	}
}
