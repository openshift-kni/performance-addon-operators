package performance

import (
	"fmt"
	"github.com/RHsyseng/operator-utils/pkg/validation"
	"github.com/ghodss/yaml"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v1alpha1"
	"io/ioutil"
	"strings"
)

const (
	deployCrdPath      = "../../../deploy/crds"
	crFilename         = "performance.openshift.io_v1alpha1_performanceprofile_cr.yaml"
	crdFilename        = "performance.openshift.io_performanceprofiles_crd.yaml"
	lastHeartbeatPath  = "/status/conditions/lastHeartbeatTime"
	lastTransitionPath = "/status/conditions/lastTransitionTime"
)

var _ = Describe("PerformanceProfile CR(D) Schema", func() {
	var schema validation.Schema

	BeforeEach(func() {
		var err error
		schema, err = getSchema(fmt.Sprintf("%s/%s", deployCrdPath, crdFilename))
		Expect(err).ToNot(HaveOccurred())
		Expect(schema).ToNot(BeNil())
	})

	It("should validate PerformanceProfile struct fields are represented recursively in the CRD", func() {
		// add any CRD paths to omit from validation check [deeply nested properties, generated timestamps, etc.]
		pathOmissions := []string{
			lastHeartbeatPath,
			lastTransitionPath,
		}
		missingEntries := getMissingEntries(schema, &v1.PerformanceProfile{}, pathOmissions...)
		Expect(missingEntries).To(BeEmpty())
	})

	It("should validate CR contents & formatting against provided CRD schema", func() {
		cr, err := getCR(fmt.Sprintf("%s/%s", deployCrdPath, crFilename))
		Expect(err).ToNot(HaveOccurred())
		Expect(cr).ToNot(BeNil())

		// schema.Validate wraps a number of custom validator triggers for slice/string formatting, schema layout, etc.
		// reference operator-utils/validate/schema:NewSchemaValidator for inclusive list
		err = schema.Validate(cr)
		Expect(err).ToNot(HaveOccurred())
	})
})

// getSchema reads in & returns CRD schema file as openAPIV3Schema{} for validation usage.
// See references operator-utils/validation/schema & go-openapi/spec/schema
func getSchema(crdPath string) (validation.Schema, error) {
	bytes, err := ioutil.ReadFile(crdPath)
	if err != nil {
		return nil, err
	}
	schema, err := validation.New(bytes)
	if err != nil {
		return nil, err
	}
	return schema, nil
}

// getCR unmarshals a *_cr.yaml file and returns the representing struct
func getCR(crPath string) (map[string]interface{}, error) {
	bytes, err := ioutil.ReadFile(crPath)
	if err != nil {
		return nil, err
	}
	var input map[string]interface{}
	if err = yaml.Unmarshal(bytes, &input); err != nil {
		return nil, err
	}
	return input, nil
}

// getMissingEntries recursively walks schemaInstance fields (PerformanceProfile), checking that each (and its fields
// recursively) are represented in CRD (schema); returns list of missing fields with specified omissions filtered out
func getMissingEntries(schema validation.Schema, schemaInstance interface{}, omissions ...string) []validation.SchemaEntry {
	missingEntries := schema.GetMissingEntries(schemaInstance)
	var filtered bool
	var filteredMissing []validation.SchemaEntry
	for _, missing := range missingEntries {
		filtered = false
		for _, omit := range omissions {
			if strings.HasPrefix(missing.Path, omit) {
				filtered = true
				break
			}
		}
		if !filtered {
			filteredMissing = append(filteredMissing, missing)
		}
	}
	return filteredMissing
}
