package __performance_profile_creator

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/ghodss/yaml"

	performancev2 "github.com/openshift-kni/performance-addon-operators/api/v2"
	"github.com/openshift-kni/performance-addon-operators/cmd/performance-profile-creator/cmd"
	testutils "github.com/openshift-kni/performance-addon-operators/functests/utils"
)

const (
	mustGatherPath       = "../../testdata/must-gather"
	expectedProfilesPath = "../../testdata/ppc-expected-profiles"
	ppcPath              = "../../build/_output/bin/performance-profile-creator"
)

var _ = Describe("[rfe_id:OCP-38968][ppc] Performance Profile Creator", func() {
	It("[test_id:OCP-40940] performance profile creator regression tests", func() {
		Expect(ppcPath).To(BeAnExistingFile())

		// directory base name => full path
		mustGatherDirs := getMustGatherDirs(mustGatherPath)
		// full profile path => arguments the profile was created with
		expectedProfiles := getExpectedProfiles(expectedProfilesPath, mustGatherDirs)

		for expectedProfilePath, args := range expectedProfiles {
			cmdArgs := []string{
				fmt.Sprintf("--disable-ht=%v", args.DisableHT),
				fmt.Sprintf("--mcp-name=%s", args.MCPName),
				fmt.Sprintf("--must-gather-dir-path=%s", args.MustGatherDirPath),
				fmt.Sprintf("--reserved-cpu-count=%d", args.ReservedCPUCount),
				fmt.Sprintf("--rt-kernel=%v", args.RTKernel),
				fmt.Sprintf("--split-reserved-cpus-across-numa=%v", args.SplitReservedCPUsAcrossNUMA),
				fmt.Sprintf("--user-level-networking=%v", args.UserLevelNetworking),
			}

			// do not pass empty strings for optional args
			if len(args.ProfileName) > 0 {
				cmdArgs = append(cmdArgs, fmt.Sprintf("--profile-name=%s", args.ProfileName))
			}
			if len(args.PowerConsumptionMode) > 0 {
				cmdArgs = append(cmdArgs, fmt.Sprintf("--power-consumption-mode=%s", args.PowerConsumptionMode))
			}
			if len(args.TMPolicy) > 0 {
				cmdArgs = append(cmdArgs, fmt.Sprintf("--topology-manager-policy=%s", args.TMPolicy))
			}

			out, err := testutils.ExecAndLogCommand(ppcPath, cmdArgs...)
			Expect(err).To(BeNil(), "failed to run ppc for '%s': %v", expectedProfilePath, err)

			profile := &performancev2.PerformanceProfile{}
			err = yaml.Unmarshal(out, profile)
			Expect(err).To(BeNil(), "failed to unmarshal the output yaml for '%s': %v", expectedProfilePath, err)

			bytes, err := ioutil.ReadFile(expectedProfilePath)
			Expect(err).To(BeNil(), "failed to read the expected yaml for '%s': %v", expectedProfilePath, err)

			expectedProfile := &performancev2.PerformanceProfile{}
			err = yaml.Unmarshal(bytes, expectedProfile)
			Expect(err).To(BeNil(), "failed to unmarshal the expected yaml for '%s': %v", expectedProfilePath, err)

			Expect(profile).To(BeEquivalentTo(expectedProfile), "regression test failed for '%s' case", expectedProfilePath)
		}
	})
})

func getMustGatherDirs(mustGatherPath string) map[string]string {
	Expect(mustGatherPath).To(BeADirectory())

	mustGatherDirs := make(map[string]string)
	mustGatherPathContent, err := ioutil.ReadDir(mustGatherPath)
	Expect(err).To(BeNil(), fmt.Errorf("can't list '%s' files: %v", mustGatherPath, err))

	for _, file := range mustGatherPathContent {
		fullFilePath := filepath.Join(mustGatherPath, file.Name())
		Expect(fullFilePath).To(BeADirectory())

		mustGatherDirs[file.Name()] = fullFilePath
	}

	return mustGatherDirs
}

func getExpectedProfiles(expectedProfilesPath string, mustGatherDirs map[string]string) map[string]cmd.ProfileCreatorArgs {
	Expect(expectedProfilesPath).To(BeADirectory())

	expectedProfilesPathContent, err := ioutil.ReadDir(expectedProfilesPath)
	Expect(err).To(BeNil(), fmt.Errorf("can't list '%s' files: %v", expectedProfilesPath, err))

	// read ppc params files
	ppcParams := make(map[string]cmd.ProfileCreatorArgs)
	for _, file := range expectedProfilesPathContent {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		fullFilePath := filepath.Join(expectedProfilesPath, file.Name())
		bytes, err := ioutil.ReadFile(fullFilePath)
		Expect(err).To(BeNil(), "failed to read the ppc params file for '%s': %v", fullFilePath, err)

		var ppcArgs cmd.ProfileCreatorArgs
		err = json.Unmarshal(bytes, &ppcArgs)
		Expect(err).To(BeNil(), "failed to decode the ppc params file for '%s': %v", fullFilePath, err)

		Expect(ppcArgs.MustGatherDirPath).ToNot(BeEmpty(), "must-gather arg missing for '%s'", fullFilePath)
		ppcArgs.MustGatherDirPath = path.Join(mustGatherPath, ppcArgs.MustGatherDirPath)
		Expect(ppcArgs.MustGatherDirPath).To(BeADirectory())

		profileKey := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
		ppcParams[profileKey] = ppcArgs
	}

	// pickup profile files
	expectedProfiles := make(map[string]cmd.ProfileCreatorArgs)
	for _, file := range expectedProfilesPathContent {
		if filepath.Ext(file.Name()) != ".yaml" {
			continue
		}

		profileKey := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
		ppcArgs, ok := ppcParams[profileKey]
		Expect(ok).To(BeTrue(), "can't find ppc params for the expected profile: '%s'", file.Name())

		fullFilePath := filepath.Join(expectedProfilesPath, file.Name())
		expectedProfiles[fullFilePath] = ppcArgs
	}

	return expectedProfiles
}
