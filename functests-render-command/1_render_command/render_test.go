package __render_command_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	assetsOutDir string
	assetsInDir  string
	ppInFiles    string
	testDataPath string
	refPath      string
	cmdLineArgs  []string
	envVars      []string
)

var _ = Describe("render command e2e test", func() {
	BeforeEach(func() {
		assetsOutDir = createTempAssetsDir()
		assetsInDir = filepath.Join(workspaceDir, "build", "assets")
		ppInFiles = filepath.Join(workspaceDir, "cluster-setup", "manual-cluster", "performance", "performance_profile.yaml")
		testDataPath = filepath.Join(workspaceDir, "testdata")
		refPath = filepath.Join(testDataPath, "render-expected-output")

		cmdLineArgs = []string{
			filepath.Join(binPath, "performance-addon-operators"),
			"render",
		}
		envVars = []string{}
	})

	JustBeforeEach(func() {
		fmt.Fprintf(GinkgoWriter, "running: %v\n", cmdLineArgs)
		cmd := exec.Command(cmdLineArgs[0], cmdLineArgs[1:]...)
		cmd.Env = append(cmd.Env, envVars...)
		_, err := cmd.Output()
		Expect(err).ToNot(HaveOccurred())
	})

	Context("with a single performance-profile and command line args", func() {
		BeforeEach(func() {
			cmdLineArgs = append(cmdLineArgs,
				"--performance-profile-input-files", ppInFiles,
				"--asset-input-dir", assetsInDir,
				"--asset-output-dir", assetsOutDir,
			)
		})

		It("should produces the expected components to output directory", func() {
			file, diff, err := cmpRenderToRef()
			Expect(err).ToNot(HaveOccurred())
			Expect(diff).To(BeEmpty(), "rendered %s file is not identical to its reference file; diff: %v", file, diff)
		})
	})

	Context("with a single performance-profile and environment variables", func() {
		BeforeEach(func() {
			envVars = append(envVars,
				fmt.Sprintf("PERFORMANCE_PROFILE_INPUT_FILES=%s", ppInFiles),
				fmt.Sprintf("ASSET_INPUT_DIR=%s", assetsInDir),
				fmt.Sprintf("ASSET_OUTPUT_DIR=%s", assetsOutDir),
			)
		})

		It("should produces the expected components to output directory", func() {
			file, diff, err := cmpRenderToRef()
			Expect(err).ToNot(HaveOccurred())
			Expect(diff).To(BeEmpty(), "rendered %s file is not identical to its reference file; diff: %v", file, diff)
		})
	})

	Context("with workload-partition enabled and command line args", func() {
		BeforeEach(func() {
			cmdLineArgs = append(cmdLineArgs,
				"--performance-profile-input-files", ppInFiles,
				"--asset-input-dir", assetsInDir,
				"--asset-output-dir", assetsOutDir,
				"--enable-workload-partitioning",
			)
		})

		It("should have the configuration under machineconfig file", func() {
			renderedMachineConfig := "manual_machineconfig.yaml"
			refMachineConfig := "manual_machineconfig_with_workload.yaml"

			diff, err := cmpRenderToRefMC(renderedMachineConfig, refMachineConfig)
			Expect(err).ToNot(HaveOccurred())
			Expect(diff).To(BeEmpty(), "rendered %s file is not identical to its reference file; diff: %v", renderedMachineConfig, diff)
		})
	})

	Context("with workload-partition enabled and environment variables", func() {
		BeforeEach(func() {
			envVars = append(envVars,
				fmt.Sprintf("PERFORMANCE_PROFILE_INPUT_FILES=%s", ppInFiles),
				fmt.Sprintf("ASSET_INPUT_DIR=%s", assetsInDir),
				fmt.Sprintf("ASSET_OUTPUT_DIR=%s", assetsOutDir),
				fmt.Sprintf("ENABLE_WORKLOAD_PARTITIONING=true"),
			)
		})

		It("should have the configuration under machineconfig file", func() {
			renderedMachineConfig := "manual_machineconfig.yaml"
			refMachineConfig := "manual_machineconfig_with_workload.yaml"

			diff, err := cmpRenderToRefMC(renderedMachineConfig, refMachineConfig)
			Expect(err).ToNot(HaveOccurred())
			Expect(diff).To(BeEmpty(), "rendered %s file is not identical to its reference file; diff: %v", renderedMachineConfig, diff)
		})
	})

	AfterEach(func() {
		cleanArtifacts()
	})
})

func createTempAssetsDir() string {
	assets, err := ioutil.TempDir("", "assets")
	Expect(err).ToNot(HaveOccurred())
	fmt.Printf("assets` output dir at: %q\n", assets)
	return assets
}

func cleanArtifacts() {
	os.RemoveAll(assetsOutDir)
}

func cmpRenderToRef() (string, string, error) {
	outputAssetsFiles, err := ioutil.ReadDir(assetsOutDir)
	if err != nil {
		return "", "", err
	}

	fmt.Fprintf(GinkgoWriter, "reference data at: %q\n", refPath)

	for _, f := range outputAssetsFiles {
		refData, err := ioutil.ReadFile(filepath.Join(refPath, f.Name()))
		if err != nil {
			return f.Name(), "", err
		}

		data, err := ioutil.ReadFile(filepath.Join(assetsOutDir, f.Name()))
		if err != nil {
			return f.Name(), "", err
		}

		diff, err := getFilesDiff(data, refData)
		if err != nil {
			return f.Name(), "", err
		}

		if len(diff) != 0 {
			return f.Name(), diff, nil
		}
	}
	return "", "", nil
}

func cmpRenderToRefMC(render, ref string) (string, error) {
	fmt.Fprintf(GinkgoWriter, "reference data at: %q\n", refPath)
	refData, err := ioutil.ReadFile(filepath.Join(refPath, ref))
	if err != nil {
		return "", err
	}

	data, err := ioutil.ReadFile(filepath.Join(assetsOutDir, render))
	if err != nil {
		return "", err
	}

	return getFilesDiff(data, refData)
}
