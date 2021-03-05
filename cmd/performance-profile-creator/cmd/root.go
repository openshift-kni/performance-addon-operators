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
	"strconv"
	"strings"

	"github.com/openshift-kni/performance-addon-operators/pkg/profilecreator"
	"github.com/openshift-kni/performance-addon-operators/pkg/utils/csvtools"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"

	"k8s.io/utils/pointer"

	performancev2 "github.com/openshift-kni/performance-addon-operators/api/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProfileData collects and stores all the data needed for profile creation
type ProfileData struct {
	isolatedCPUs, reservedCPUs string
	nodeSelector               *metav1.LabelSelector
	performanceProfileName     string
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "performance-profile-creator",
	Short: "A tool that automates creation of Performance Profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		flagData, err := getDataFromFlags(cmd)
		if err != nil {
			return fmt.Errorf("Error obtaining profileData %v", err)
		}
		profileData, err := getProfileData(flagData)
		if err != nil {
			return fmt.Errorf("Error obtaining profileData %v", err)
		}
		createProfile(*profileData)
		return nil
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

func getDataFromFlags(cmd *cobra.Command) (*profileCreatorArgs, error) {
	mustGatherDirPath := cmd.Flag("must-gather-dir-path").Value.String()
	mcpName := cmd.Flag("mcp-name").Value.String()
	reservedCPUCount, err := strconv.Atoi(cmd.Flag("reserved-cpu-count").Value.String())
	if err != nil {
		return nil, fmt.Errorf("Error parsing reserved-cpu-count flag: %v", err)
	}
	splitReservedCPUsAcrossNUMA, err := strconv.ParseBool(cmd.Flag("split-reserved-cpus-across-numa").Value.String())
	if err != nil {
		return nil, fmt.Errorf("Error parsing split-reserved-cpus-across-numa flag: %v", err)
	}
	profileName := cmd.Flag("profile-name").Value.String()
	flagData := &profileCreatorArgs{
		mustGatherDirPath:           mustGatherDirPath,
		profileName:                 profileName,
		reservedCPUCount:            reservedCPUCount,
		splitReservedCPUsAcrossNUMA: splitReservedCPUsAcrossNUMA,
		mcpName:                     mcpName,
	}
	return flagData, nil
}

func getProfileData(args *profileCreatorArgs) (*ProfileData, error) {
	mcp, err := profilecreator.GetMCP(args.mustGatherDirPath, args.mcpName)
	if err != nil {
		return nil, fmt.Errorf("Error obtaining MachineConfigPool %s: %v", args.mcpName, err)
	}

	nodes, err := profilecreator.GetNodeList(args.mustGatherDirPath)
	if err != nil {
		return nil, fmt.Errorf("Error obtaining Nodes %s: %v", args.mcpName, err)
	}

	mcps, err := profilecreator.GetMCPList(args.mustGatherDirPath)
	if err != nil {
		return nil, fmt.Errorf("can't get the MCP list under %s: %v", args.mustGatherDirPath, err)
	}

	matchedNodes, err := profilecreator.GetNodesForPool(mcp, mcps, nodes)
	if err != nil {
		return nil, fmt.Errorf("can't find matching nodes for %s: %v", args.mcpName, err)
	}

	err = profilecreator.EnsureNodesHaveTheSameHardware(args.mustGatherDirPath, matchedNodes)
	if err != nil {
		return nil, fmt.Errorf("targeted nodes differ: %v", err)
	}

	// We make sure that the matched Nodes are the same
	// Assumption here is moving forward matchedNodes[0] is representative of how all the nodes are
	// same from hardware topology point of view
	nodeName := matchedNodes[0].GetName()
	log.Infof("%s is targetted by %s MCP", nodeName, args.mcpName)
	handle, err := profilecreator.NewGHWHandler(args.mustGatherDirPath, matchedNodes[0])
	reservedCPUs, isolatedCPUs, err := handle.GetReservedAndIsolatedCPUs(args.reservedCPUCount, args.splitReservedCPUsAcrossNUMA)
	if err != nil {
		return nil, fmt.Errorf("Error obtaining Reserved and Isolated CPUs for %s: %v", nodeName, err)
	}
	profileData := &ProfileData{
		reservedCPUs:           reservedCPUs,
		isolatedCPUs:           isolatedCPUs,
		nodeSelector:           mcp.Spec.NodeSelector,
		performanceProfileName: args.profileName,
	}
	return profileData, nil
}

type profileCreatorArgs struct {
	powerConsumptionMode        string
	mustGatherDirPath           string
	profileName                 string
	reservedCPUCount            int
	splitReservedCPUsAcrossNUMA bool
	disableHT                   bool
	rtKernel                    bool
	userLevelNetworking         bool
	mcpName                     string
}

func init() {
	args := &profileCreatorArgs{}
	log.SetOutput(os.Stderr)
	rootCmd.PersistentFlags().IntVarP(&args.reservedCPUCount, "reserved-cpu-count", "R", 0, "Number of reserved CPUs (required)")
	rootCmd.MarkPersistentFlagRequired("reserved-cpu-count")
	rootCmd.PersistentFlags().BoolVarP(&args.splitReservedCPUsAcrossNUMA, "split-reserved-cpus-across-numa", "S", false, "Split the Reserved CPUs across NUMA nodes")
	rootCmd.PersistentFlags().StringVarP(&args.mcpName, "mcp-name", "T", "worker-cnf", "MCP name corresponding to the target machines (required)")
	rootCmd.MarkPersistentFlagRequired("mcp-name")
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

func createProfile(profileData ProfileData) {

	reserved := performancev2.CPUSet(profileData.reservedCPUs)
	isolated := performancev2.CPUSet(profileData.isolatedCPUs)
	// TODO: Get the name from MCP if not specified in the command line arguments
	profile := &performancev2.PerformanceProfile{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PerformanceProfile",
			APIVersion: performancev2.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: profileData.performanceProfileName,
		},
		Spec: performancev2.PerformanceProfileSpec{
			CPU: &performancev2.CPU{
				Isolated: &isolated,
				Reserved: &reserved,
			},
			NodeSelector: profileData.nodeSelector.MatchLabels,
			RealTimeKernel: &performancev2.RealTimeKernel{
				Enabled: pointer.BoolPtr(true),
			},
			AdditionalKernelArgs: []string{},
			NUMA: &performancev2.NUMA{
				TopologyPolicy: pointer.StringPtr("restricted"),
			},
		},
	}

	// write CSV to out dir
	writer := strings.Builder{}
	csvtools.MarshallObject(&profile, &writer)

	fmt.Printf("%s", writer.String())
}
