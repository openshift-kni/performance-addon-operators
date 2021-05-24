package __performance_profile_creator

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os/exec"
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
	expectedInfoPath     = "../../testdata/ppc-expected-info"
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
			}

			if args.UserLevelNetworking != nil {
				cmdArgs = append(cmdArgs, fmt.Sprintf("--user-level-networking=%v", *args.UserLevelNetworking))
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
<<<<<<< HEAD
=======
<<<<<<< HEAD

	It("should describe the cluster from must-gather data in info mode", func() {
		Expect(ppcPath).To(BeAnExistingFile())

		// directory base name => full path
		mustGatherDirs := getMustGatherDirs(mustGatherPath)

		for name, path := range mustGatherDirs {
			cmdArgs := []string{
				"--info=json",
				fmt.Sprintf("--must-gather-dir-path=%s", path),
			}

			out, err := testutils.ExecAndLogCommand(ppcPath, cmdArgs...)
			Expect(err).To(BeNil(), "failed to run ppc for %q: %v", path, err)

			var cInfo cmd.ClusterInfo
			err = json.Unmarshal(out, &cInfo)
			Expect(err).To(BeNil(), "failed to unmarshal the output json for %q: %v", path, err)
			expectedClusterInfoPath := filepath.Join(expectedInfoPath, fmt.Sprintf("%s.json", name))
			bytes, err := ioutil.ReadFile(expectedClusterInfoPath)
			Expect(err).To(BeNil(), "failed to read the expected json for %q: %v", expectedClusterInfoPath, err)

			var expectedInfo cmd.ClusterInfo
			err = json.Unmarshal(bytes, &expectedInfo)
			Expect(err).To(BeNil(), "failed to unmarshal the expected json for '%s': %v", expectedClusterInfoPath, err)

			expectedInfo.Sort()

			Expect(cInfo).To(BeEquivalentTo(expectedInfo), "regression test failed for '%s' case", expectedClusterInfoPath)
		}
	})

=======
>>>>>>> openshift-kni-master
	//TestCase4
	It("[test_id:OCP-40941] Verify PPC script fails when the splitting of reserved cpus and single numa-node policy is specified", func() {
		Expect(ppcPath).To(BeAnExistingFile())
		mustGatherFullPath := path.Join(mustGatherPath, "must-gather.bare-metal")
		Expect(mustGatherFullPath).To(BeADirectory())
		cmdArgs := []string {
			fmt.Sprintf("--disable-ht=%t", false),
			fmt.Sprintf("--mcp-name=%s", "worker-cnf"),
			fmt.Sprintf("--must-gather-dir-path=%s", mustGatherFullPath),
			fmt.Sprintf("--reserved-cpu-count=%d", 2),
			fmt.Sprintf("--rt-kernel=%t", true),
			fmt.Sprintf("--split-reserved-cpus-across-numa=%t", true),
			fmt.Sprintf("--user-level-networking=%t", false),
			fmt.Sprintf("--profile-name=%s", "Performance"),
			fmt.Sprintf("--topology-manager-policy=%s", "single-numa-node"),
		}
		_, err := testutils.ExecAndLogCommand(ppcPath, cmdArgs...)
		ppcErrorString := errorStringParser(err)
		Expect(ppcErrorString).To(ContainSubstring("failed to obtain data from flags not appropriate to split reserved CPUs in case of topology-manager-policy: single-numa-node"))
	})
	//TestCase5
	It("[test_id:OCP-40941] Verify PPC script fails when reserved cpu count is 2 and requires to split across numa nodes", func() {
		Expect(ppcPath).To(BeAnExistingFile())
		mustGatherFullPath := path.Join(mustGatherPath, "must-gather.bare-metal")
		Expect(mustGatherFullPath).To(BeADirectory())
		cmdArgs := []string {
			fmt.Sprintf("--disable-ht=%t", false),
			fmt.Sprintf("--mcp-name=%s", "worker-cnf"),
			fmt.Sprintf("--must-gather-dir-path=%s", mustGatherFullPath),
			fmt.Sprintf("--reserved-cpu-count=%d", 2),
			fmt.Sprintf("--rt-kernel=%t", true),
			fmt.Sprintf("--split-reserved-cpus-across-numa=%t", true),
			fmt.Sprintf("--user-level-networking=%t", false),
			fmt.Sprintf("--profile-name=%s", "Performance"),
		}
		_, err := testutils.ExecAndLogCommand(ppcPath, cmdArgs...)
		ppcErrorString := errorStringParser(err)
		Expect(ppcErrorString).To(ContainSubstring("failed to compute the reserved and isolated CPUs: can't allocate odd number of CPUs from a NUMA Node"))
	})
<<<<<<< HEAD
=======
<<<<<<< HEAD
>>>>>>> e7661bfd... add test case 4
=======
>>>>>>> openshift-kni-master
	//TestCase6
	It("[test_id:OCP-40941] Verify PPC script fails when reserved cpu count is more than available cpus", func() {
		Expect(ppcPath).To(BeAnExistingFile())
		mustGatherFullPath := path.Join(mustGatherPath, "must-gather.bare-metal")
		Expect(mustGatherFullPath).To(BeADirectory())
		cmdArgs := []string {
			fmt.Sprintf("--disable-ht=%t", false),
			fmt.Sprintf("--mcp-name=%s", "worker-cnf"),
			fmt.Sprintf("--must-gather-dir-path=%s", mustGatherFullPath),
			fmt.Sprintf("--reserved-cpu-count=%d", 100),
			fmt.Sprintf("--rt-kernel=%t", true),
			fmt.Sprintf("--user-level-networking=%t", false),
			fmt.Sprintf("--profile-name=%s", "Performance"),
		}
		_, err := testutils.ExecAndLogCommand(ppcPath, cmdArgs...)
		ppcErrorString := errorStringParser(err)
		Expect(ppcErrorString).To(ContainSubstring("failed to compute the reserved and isolated CPUs: please specify the reserved CPU count in the range [1,79]"))
	})
<<<<<<< HEAD
=======
<<<<<<< HEAD
>>>>>>> e1b6a5d4... add test case 6
=======
>>>>>>> openshift-kni-master
	//TestCase7
	It("[test_id:OCP-40941] Verify PPC script fails when odd number of reserved cpus are specified", func() {
		Expect(ppcPath).To(BeAnExistingFile())
		mustGatherFullPath := path.Join(mustGatherPath, "must-gather.bare-metal")
		Expect(mustGatherFullPath).To(BeADirectory())
		cmdArgs := []string {
			fmt.Sprintf("--disable-ht=%t", false),
			fmt.Sprintf("--mcp-name=%s", "worker-cnf"),
			fmt.Sprintf("--must-gather-dir-path=%s", mustGatherFullPath),
			fmt.Sprintf("--reserved-cpu-count=%d", 5),
			fmt.Sprintf("--rt-kernel=%t", true),
			fmt.Sprintf("--user-level-networking=%t", false),
			fmt.Sprintf("--profile-name=%s", "Performance"),
		}
		_, err := testutils.ExecAndLogCommand(ppcPath, cmdArgs...)
		ppcErrorString := errorStringParser(err)
		Expect(ppcErrorString).To(ContainSubstring("failed to compute the reserved and isolated CPUs: can't allocate odd number of CPUs from a NUMA Node"))
	})
<<<<<<< HEAD
=======
>>>>>>> 87311a08... add test case 7
>>>>>>> openshift-kni-master
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
func errorStringParser(err error) string {
	var errorString string
	exitError := err.(*exec.ExitError)
	stdError := string(exitError.Stderr)
	for _, line := range(strings.Split(stdError, "\n")) {
		if strings.Contains(line, "Error:") {
			errorString = line
			break
		}
	}
	return errorString
}