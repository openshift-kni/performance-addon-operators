/*
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2021 Red Hat, Inc.
 */

package cmd

import (
	"fmt"
	"os"

	"github.com/jaypipes/ghw"
	"github.com/openshift-kni/performance-addon-operators/pkg/profilecreator"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"

	"k8s.io/utils/pointer"

	performancev2 "github.com/openshift-kni/performance-addon-operators/api/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "performance-profile-creator",
	Short: "A tool that automates creation of Performance Profiles",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		mcpName := cmd.Flag("mcp-name").Value.String()
		mustGatherDirPath := cmd.Flag("must-gather-dir-path").Value.String()
		mcp, err := profilecreator.GetMCP(mustGatherDirPath, mcpName)
		if err != nil {
			return fmt.Errorf("Error obtaining MachineConfigPool %s: %v", mcpName, err)
		}
		labelSelector := mcp.Spec.NodeSelector
		nodes, err := profilecreator.GetNodeList(mustGatherDirPath)
		if err != nil {
			return fmt.Errorf("Error obtaining Nodes %s: %v", mcpName, err)
		}

		matchedNodes, err := profilecreator.GetMatchedNodes(nodes, labelSelector)
		for _, node := range matchedNodes {
			nodeName := node.GetName()
			log.Infof("%s is targetted by %s MCP", nodeName, mcpName)
			snapShotOptions, err := profilecreator.LoadSnapshot(mustGatherDirPath, node)
			if err != nil {
				return fmt.Errorf("Error loading snaphot for %s: %v", nodeName, err)
			}
			//Examples of how GHW snapshots can be consumed
			cpuInfo, err := ghw.CPU(snapShotOptions)
			if err != nil {
				return fmt.Errorf("Error obtaining CPU Info from GHW snapshot for %s: %v", nodeName, err)
			}
			log.Infof("CPU Info: %v\n", cpuInfo)
			topologyInfo, err := ghw.Topology(snapShotOptions)
			if err != nil {
				return fmt.Errorf("Error obtaining Topology Info from GHW snapshot for %s: %v", nodeName, err)
			}
			log.Infof("Topology Info: %v\n", topologyInfo)
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		profileName := cmd.Flag("profile-name").Value.String()
		createProfile(profileName)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error while executing root command: %v", err)
		os.Exit(1)
	}
}

type profileCreatorArgs struct {
	powerConsumptionMode string
	mustGatherDirPath    string
	profileName          string
	reservedCPUCount     int
	splitCPUsAcrossNUMA  bool
	disableHT            bool
	rtKernel             bool
	userLevelNetworking  bool
	mcpName              string
}

func init() {
	args := &profileCreatorArgs{}
	log.SetOutput(os.Stderr)
	rootCmd.PersistentFlags().IntVarP(&args.reservedCPUCount, "reserved-cpu-count", "R", 0, "Number of reserved CPUs (required)")
	rootCmd.MarkPersistentFlagRequired("reserved-cpu-count")
	rootCmd.PersistentFlags().StringVarP(&args.mcpName, "mcp-name", "T", "worker-cnf", "MCP name corresponding to the target machines (required)")
	rootCmd.MarkPersistentFlagRequired("mcp-name")
	rootCmd.PersistentFlags().BoolVarP(&args.splitCPUsAcrossNUMA, "split-cpus-across-numa", "S", true, "Split the CPUs across NUMA nodes")
	rootCmd.PersistentFlags().BoolVarP(&args.disableHT, "disable-ht", "H", false, "Disable Hyperthreading")
	rootCmd.PersistentFlags().BoolVarP(&args.rtKernel, "rt-kernel", "K", true, "Enable Real Time Kernel (required)")
	rootCmd.MarkPersistentFlagRequired("rt-kernel")
	rootCmd.PersistentFlags().BoolVarP(&args.userLevelNetworking, "user-level-networking", "U", false, "Run with User level Networking(DPDK) enabled")
	rootCmd.PersistentFlags().StringVarP(&args.powerConsumptionMode, "power-consumption-mode", "P", "cstate", "The power consumption mode")
	rootCmd.PersistentFlags().StringVarP(&args.mustGatherDirPath, "must-gather-dir-path", "M", "must-gather", "Must gather directory path")
	rootCmd.MarkPersistentFlagRequired("must-gather-dir-path")
	rootCmd.PersistentFlags().StringVarP(&args.profileName, "profile-name", "N", "performance", "Name of the performance profile to be created")

	// TODO: Input validation
	// 1) Make flags required/optional
	// 2) e.g.check to make sure that power consumption string is in {CSTATE NO_CSTATE IDLE_POLL}
}

func createProfile(profileName string) {

	// TODO: Get the name from MCP if not specified in the command line arguments
	profile := &performancev2.PerformanceProfile{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PerformanceProfile",
			APIVersion: performancev2.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: profileName,
		},
		Spec: performancev2.PerformanceProfileSpec{
			RealTimeKernel: &performancev2.RealTimeKernel{
				Enabled: pointer.BoolPtr(true),
			},
			AdditionalKernelArgs: []string{},
			NUMA: &performancev2.NUMA{
				TopologyPolicy: pointer.StringPtr("restricted"),
			},
		},
	}

	var performanceProfileData []byte
	var err error

	if performanceProfileData, err = yaml.Marshal(&profile); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to Marshal sample performance profile: %v", err)
	}

	fmt.Printf("%s", string(performanceProfileData))

}
