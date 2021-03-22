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
	machineconfigv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"

	performancev2 "github.com/openshift-kni/performance-addon-operators/api/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeletconfig "k8s.io/kubelet/config/v1beta1"
)

var (
	validTMPolicyValues = []string{kubeletconfig.SingleNumaNodeTopologyManagerPolicy, kubeletconfig.BestEffortTopologyManagerPolicy, kubeletconfig.RestrictedTopologyManagerPolicy}
	// TODO: Explain the power-consumption-mode and associated kernel args
	validPowerConsumptionModes = []string{"default", "low-latency", "ultra-low-latency"}
)

// ProfileData collects and stores all the data needed for profile creation
type ProfileData struct {
	isolatedCPUs, reservedCPUs string
	nodeSelector               *metav1.LabelSelector
	performanceProfileName     string
	topologyPoilcy             string
	rtKernel                   bool
	additionalKernelArgs       []string
	userLevelNetworking        bool
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "performance-profile-creator",
	Short: "A tool that automates creation of Performance Profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		profileCreatorArgsFromFlags, err := getDataFromFlags(cmd)
		if err != nil {
			return fmt.Errorf("failed to obtain data from flags %v", err)
		}
		profileData, err := getProfileData(profileCreatorArgsFromFlags)
		if err != nil {
			return err
		}
		createProfile(*profileData)
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func getDataFromFlags(cmd *cobra.Command) (profileCreatorArgs, error) {
	creatorArgs := profileCreatorArgs{}
	mustGatherDirPath := cmd.Flag("must-gather-dir-path").Value.String()
	mcpName := cmd.Flag("mcp-name").Value.String()
	reservedCPUCount, err := strconv.Atoi(cmd.Flag("reserved-cpu-count").Value.String())
	if err != nil {
		return creatorArgs, fmt.Errorf("failed to parse reserved-cpu-count flag: %v", err)
	}
	splitReservedCPUsAcrossNUMA, err := strconv.ParseBool(cmd.Flag("split-reserved-cpus-across-numa").Value.String())
	if err != nil {
		return creatorArgs, fmt.Errorf("failed to parse split-reserved-cpus-across-numa flag: %v", err)
	}
	profileName := cmd.Flag("profile-name").Value.String()
	tmPolicy := cmd.Flag("topology-manager-policy").Value.String()
	if err != nil {
		return creatorArgs, fmt.Errorf("failed to parse topology-manager-policy flag: %v", err)
	}
	err = validateFlag(tmPolicy, validTMPolicyValues)
	if err != nil {
		return creatorArgs, fmt.Errorf("invalid value for topology-manager-policy flag specified: %v", err)
	}
	if tmPolicy == kubeletconfig.SingleNumaNodeTopologyManagerPolicy && splitReservedCPUsAcrossNUMA {
		return creatorArgs, fmt.Errorf("not appropriate to split reserved CPUs in case of topology-manager-policy: %v", tmPolicy)
	}
	powerConsumptionMode := cmd.Flag("power-consumption-mode").Value.String()
	if err != nil {
		return creatorArgs, fmt.Errorf("failed to parse power-consumption-mode flag: %v", err)
	}
	err = validateFlag(powerConsumptionMode, profilecreator.ValidPowerConsumptionModes)
	if err != nil {
		return creatorArgs, fmt.Errorf("invalid value for power-consumption-mode flag specified: %v", err)
	}

	rtKernelEnabled, err := strconv.ParseBool(cmd.Flag("rt-kernel").Value.String())
	if err != nil {
		return creatorArgs, fmt.Errorf("failed to parse rt-kernel flag: %v", err)
	}

	userLevelNetworkingEnabled, err := strconv.ParseBool(cmd.Flag("user-level-networking").Value.String())
	if err != nil {
		return creatorArgs, fmt.Errorf("failed to parse user-level-networking flag: %v", err)
	}
	creatorArgs = profileCreatorArgs{
		mustGatherDirPath:           mustGatherDirPath,
		profileName:                 profileName,
		reservedCPUCount:            reservedCPUCount,
		splitReservedCPUsAcrossNUMA: splitReservedCPUsAcrossNUMA,
		mcpName:                     mcpName,
		tmPolicy:                    tmPolicy,
		rtKernel:                    rtKernelEnabled,
		powerConsumptionMode:        powerConsumptionMode,
		userLevelNetworking:         userLevelNetworkingEnabled,
	}
	return creatorArgs, nil
}

func getProfileData(args profileCreatorArgs) (*ProfileData, error) {
	info, err := os.Stat(args.mustGatherDirPath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("the must-gather path '%s' is not valid", args.mustGatherDirPath)
	}
	if err != nil {
		return nil, fmt.Errorf("can't access the must-gather path: %v", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("the must-gather path '%s' is not a directory", args.mustGatherDirPath)
	}

	mcps, err := profilecreator.GetMCPList(args.mustGatherDirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get the MCP list under %s: %v", args.mustGatherDirPath, err)
	}

	var mcp *machineconfigv1.MachineConfigPool
	mcpNames := make([]string, 0)
	for _, m := range mcps {
		mcpNames = append(mcpNames, m.Name)
		if m.Name == args.mcpName {
			mcp = m
		}
	}

	if mcp == nil {
		return nil, fmt.Errorf("'%s' MCP does not exist, valid values are %v", args.mcpName, mcpNames)
	}

	nodes, err := profilecreator.GetNodeList(args.mustGatherDirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load the cluster nodes: %v", err)
	}

	matchedNodes, err := profilecreator.GetNodesForPool(mcp, mcps, nodes)
	if err != nil {
		return nil, fmt.Errorf("failed to find MCP %s's nodes: %v", args.mcpName, err)
	}
	if len(matchedNodes) == 0 {
		return nil, fmt.Errorf("no nodes are associated with '%s' MCP", args.mcpName)
	}

	var matchedNodeNames []string
	for _, node := range matchedNodes {
		matchedNodeNames = append(matchedNodeNames, node.GetName())
	}
	log.Infof("Nodes targetted by %s MCP are: %v", args.mcpName, matchedNodeNames)
	err = profilecreator.EnsureNodesHaveTheSameHardware(args.mustGatherDirPath, matchedNodes)
	if err != nil {
		return nil, fmt.Errorf("targeted nodes differ: %v", err)
	}

	// We make sure that the matched Nodes are the same
	// Assumption here is moving forward matchedNodes[0] is representative of how all the nodes are
	// same from hardware topology point of view

	handle, err := profilecreator.NewGHWHandler(args.mustGatherDirPath, matchedNodes[0])
	reservedCPUs, isolatedCPUs, err := handle.GetReservedAndIsolatedCPUs(args.reservedCPUCount, args.splitReservedCPUsAcrossNUMA)
	if err != nil {
		return nil, fmt.Errorf("failed to compute the reserved and isolated CPUs: %v", err)
	}
	log.Infof("%d reserved CPUs allocated: %v ", reservedCPUs.Size(), reservedCPUs.String())
	log.Infof("%d isolated CPUs allocated: %v", isolatedCPUs.Size(), isolatedCPUs.String())
	kernelArgs := profilecreator.GetAdditionalKernelArgs(args.powerConsumptionMode)
	profileData := &ProfileData{
		reservedCPUs:           reservedCPUs.String(),
		isolatedCPUs:           isolatedCPUs.String(),
		nodeSelector:           mcp.Spec.NodeSelector,
		performanceProfileName: args.profileName,
		topologyPoilcy:         args.tmPolicy,
		rtKernel:               args.rtKernel,
		additionalKernelArgs:   kernelArgs,
		userLevelNetworking:    args.userLevelNetworking,
	}
	return profileData, nil
}

func validateFlag(value string, validValues []string) error {
	if isStringInSlice(value, validValues) {
		return nil
	}
	return fmt.Errorf("Value '%s' is invalid. Valid values "+
		"come from the set %v", value, validValues)
}

func isStringInSlice(value string, candidates []string) bool {
	for _, candidate := range candidates {
		if strings.EqualFold(candidate, value) {
			return true
		}
	}
	return false
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
	tmPolicy                    string
}

func init() {
	args := &profileCreatorArgs{}
	log.SetOutput(os.Stderr)
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: true,
	})
	rootCmd.PersistentFlags().IntVar(&args.reservedCPUCount, "reserved-cpu-count", 0, "Number of reserved CPUs (required)")
	rootCmd.MarkPersistentFlagRequired("reserved-cpu-count")
	rootCmd.PersistentFlags().BoolVar(&args.splitReservedCPUsAcrossNUMA, "split-reserved-cpus-across-numa", false, "Split the Reserved CPUs across NUMA nodes")
	rootCmd.PersistentFlags().StringVar(&args.mcpName, "mcp-name", "worker-cnf", "MCP name corresponding to the target machines (required)")
	rootCmd.MarkPersistentFlagRequired("mcp-name")
	rootCmd.PersistentFlags().BoolVar(&args.disableHT, "disable-ht", false, "Disable Hyperthreading")
	rootCmd.PersistentFlags().BoolVar(&args.rtKernel, "rt-kernel", true, "Enable Real Time Kernel (required)")
	rootCmd.MarkPersistentFlagRequired("rt-kernel")
	rootCmd.PersistentFlags().BoolVar(&args.userLevelNetworking, "user-level-networking", false, "Run with User level Networking(DPDK) enabled")
	rootCmd.PersistentFlags().StringVar(&args.powerConsumptionMode, "power-consumption-mode", profilecreator.ValidPowerConsumptionModes[0], fmt.Sprintf("The power consumption mode.  [Valid values: %s, %s, %s]", profilecreator.ValidPowerConsumptionModes[0], profilecreator.ValidPowerConsumptionModes[1], profilecreator.ValidPowerConsumptionModes[2]))
	rootCmd.PersistentFlags().StringVar(&args.mustGatherDirPath, "must-gather-dir-path", "must-gather", "Must gather directory path")
	rootCmd.MarkPersistentFlagRequired("must-gather-dir-path")
	rootCmd.PersistentFlags().StringVar(&args.profileName, "profile-name", "performance", "Name of the performance profile to be created")
	rootCmd.PersistentFlags().StringVar(&args.tmPolicy, "topology-manager-policy", kubeletconfig.RestrictedTopologyManagerPolicy, fmt.Sprintf("Kubelet Topology Manager Policy of the performance profile to be created. [Valid values: %s, %s, %s]", kubeletconfig.SingleNumaNodeTopologyManagerPolicy, kubeletconfig.BestEffortTopologyManagerPolicy, kubeletconfig.RestrictedTopologyManagerPolicy))
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
				Enabled: &profileData.rtKernel,
			},
			AdditionalKernelArgs: profileData.additionalKernelArgs,
			NUMA: &performancev2.NUMA{
				TopologyPolicy: &profileData.topologyPoilcy,
			},
			Net: &performancev2.Net{
				UserLevelNetworking: &profileData.userLevelNetworking,
			},
		},
	}

	// write CSV to out dir
	writer := strings.Builder{}
	csvtools.MarshallObject(&profile, &writer)

	fmt.Printf("%s", writer.String())
}
