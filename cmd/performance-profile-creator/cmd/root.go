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
	"k8s.io/utils/pointer"
)

var (
	validTMPolicyValues = []string{kubeletconfig.SingleNumaNodeTopologyManagerPolicy, kubeletconfig.BestEffortTopologyManagerPolicy, kubeletconfig.RestrictedTopologyManagerPolicy}
)

// ProfileData collects and stores all the data needed for profile creation
type ProfileData struct {
	isolatedCPUs, reservedCPUs string
	nodeSelector               *metav1.LabelSelector
	mcpSelector                map[string]string
	performanceProfileName     string
	topologyPoilcy             string
	rtKernel                   bool
	additionalKernelArgs       []string
	userLevelNetworking        *bool
	disableHT                  bool
}

// ClusterData collects the cluster wide information, each mcp points to a list of ghw node handlers
type ClusterData map[*machineconfigv1.MachineConfigPool][]*profilecreator.GHWHandler

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "performance-profile-creator",
	Short: "A tool that automates creation of Performance Profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		mustGatherDirPath := cmd.Flag("must-gather-dir-path").Value.String()
		cluster, err := getClusterData(mustGatherDirPath)
		if err != nil {
			return fmt.Errorf("failed to parse the cluster data: %v", err)
		}

		info, err := strconv.ParseBool(cmd.Flag("info").Value.String())
		if err != nil {
			return fmt.Errorf("failed to parse the info flag: %v", err)
		}
		if info {
			showClusterData(cluster)
			return nil
		}

		profileCreatorArgsFromFlags, err := getDataFromFlags(cmd)
		if err != nil {
			return fmt.Errorf("failed to obtain data from flags %v", err)
		}
		profileData, err := getProfileData(profileCreatorArgsFromFlags, cluster)
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

func getClusterData(mustGatherDirPath string) (ClusterData, error) {
	cluster := make(ClusterData)
	info, err := os.Stat(mustGatherDirPath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("the must-gather path '%s' is not valid", mustGatherDirPath)
	}
	if err != nil {
		return nil, fmt.Errorf("can't access the must-gather path: %v", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("the must-gather path '%s' is not a directory", mustGatherDirPath)
	}

	mcps, err := profilecreator.GetMCPList(mustGatherDirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get the MCP list under %s: %v", mustGatherDirPath, err)
	}

	nodes, err := profilecreator.GetNodeList(mustGatherDirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load the cluster nodes: %v", err)
	}

	for _, mcp := range mcps {
		matchedNodes, err := profilecreator.GetNodesForPool(mcp, mcps, nodes)
		if err != nil {
			return nil, fmt.Errorf("failed to find MCP %s's nodes: %v", mcp.Name, err)
		}
		handlers := make([]*profilecreator.GHWHandler, len(matchedNodes))
		for i, node := range matchedNodes {
			handle, err := profilecreator.NewGHWHandler(mustGatherDirPath, node)
			if err != nil {
				return nil, fmt.Errorf("failed to load node's %s's GHW snapshot : %v", mcp.Name, err)
			}
			handlers[i] = handle
		}
		cluster[mcp] = handlers
	}

	return cluster, nil
}

func showClusterData(cluster ClusterData) {
	log.Infof("Cluster info:")
	for mcp, nodeHandlers := range cluster {
		log.Infof("MCP '%s' nodes:", mcp.Name)
		for _, handle := range nodeHandlers {
			topology, err := handle.SortedTopology()
			if err != nil {
				log.Infof("%s(Toplogy discovery error: %v)", handle.Node.GetName(), err)
			} else {
				htEnabled, err := handle.IsHyperthreadingEnabled()
				if err != nil {
					log.Infof("%s(HT discovery error: %v)", handle.Node.GetName(), err)
				}
				log.Infof("Node: %s (NUMA cells: %d, HT: %v)", handle.Node.GetName(), len(topology.Nodes), htEnabled)
				totalCPUs := 0
				for id, node := range topology.Nodes {
					var coreList []int
					for _, core := range node.Cores {
						coreList = append(coreList, core.LogicalProcessors...)
					}
					log.Infof("NUMA cell %d : %v", id, coreList)
					totalCPUs += len(coreList)
				}
				log.Infof("CPU(s): %d", totalCPUs)
			}
		}
		log.Infof("---")
	}
}

func getDataFromFlags(cmd *cobra.Command) (ProfileCreatorArgs, error) {
	creatorArgs := ProfileCreatorArgs{}
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

	htDisabled, err := strconv.ParseBool(cmd.Flag("disable-ht").Value.String())
	if err != nil {
		return creatorArgs, fmt.Errorf("failed to parse disable-ht flag: %v", err)
	}
	creatorArgs = ProfileCreatorArgs{
		MustGatherDirPath:           mustGatherDirPath,
		ProfileName:                 profileName,
		ReservedCPUCount:            reservedCPUCount,
		SplitReservedCPUsAcrossNUMA: splitReservedCPUsAcrossNUMA,
		MCPName:                     mcpName,
		TMPolicy:                    tmPolicy,
		RTKernel:                    rtKernelEnabled,
		PowerConsumptionMode:        powerConsumptionMode,
		DisableHT:                   htDisabled,
	}

	if cmd.Flag("user-level-networking").Changed {
		userLevelNetworkingEnabled, err := strconv.ParseBool(cmd.Flag("user-level-networking").Value.String())
		if err != nil {
			return creatorArgs, fmt.Errorf("failed to parse user-level-networking flag: %v", err)
		}
		creatorArgs.UserLevelNetworking = &userLevelNetworkingEnabled
	}

	return creatorArgs, nil
}

func getProfileData(args ProfileCreatorArgs, cluster ClusterData) (*ProfileData, error) {
	mcps := make([]*machineconfigv1.MachineConfigPool, len(cluster))
	mcpNames := make([]string, len(cluster))
	var mcp *machineconfigv1.MachineConfigPool

	i := 0
	for m := range cluster {
		mcps[i] = m
		mcpNames[i] = m.Name
		if m.Name == args.MCPName {
			mcp = m
		}
		i++
	}

	if mcp == nil {
		return nil, fmt.Errorf("'%s' MCP does not exist, valid values are %v", args.MCPName, mcpNames)
	}

	mcpSelector, err := profilecreator.GetMCPSelector(mcp, mcps)
	if err != nil {
		return nil, fmt.Errorf("failed to compute the MCP selector: %v", err)
	}

	if len(cluster[mcp]) == 0 {
		return nil, fmt.Errorf("no schedulable nodes are associated with '%s' MCP", args.MCPName)
	}

	var matchedNodeNames []string
	for _, nodeHandler := range cluster[mcp] {
		matchedNodeNames = append(matchedNodeNames, nodeHandler.Node.GetName())
	}
	log.Infof("Nodes targetted by %s MCP are: %v", args.MCPName, matchedNodeNames)
	err = profilecreator.EnsureNodesHaveTheSameHardware(cluster[mcp])
	if err != nil {
		return nil, fmt.Errorf("targeted nodes differ: %v", err)
	}

	// We make sure that the matched Nodes are the same
	// Assumption here is moving forward matchedNodes[0] is representative of how all the nodes are
	// same from hardware topology point of view

	nodeHandle := cluster[mcp][0]
	reservedCPUs, isolatedCPUs, err := nodeHandle.GetReservedAndIsolatedCPUs(args.ReservedCPUCount, args.SplitReservedCPUsAcrossNUMA, args.DisableHT)
	if err != nil {
		return nil, fmt.Errorf("failed to compute the reserved and isolated CPUs: %v", err)
	}
	log.Infof("%d reserved CPUs allocated: %v ", reservedCPUs.Size(), reservedCPUs.String())
	log.Infof("%d isolated CPUs allocated: %v", isolatedCPUs.Size(), isolatedCPUs.String())
	kernelArgs := profilecreator.GetAdditionalKernelArgs(args.PowerConsumptionMode, args.DisableHT)
	profileData := &ProfileData{
		reservedCPUs:           reservedCPUs.String(),
		isolatedCPUs:           isolatedCPUs.String(),
		nodeSelector:           mcp.Spec.NodeSelector,
		mcpSelector:            mcpSelector,
		performanceProfileName: args.ProfileName,
		topologyPoilcy:         args.TMPolicy,
		rtKernel:               args.RTKernel,
		additionalKernelArgs:   kernelArgs,
		userLevelNetworking:    args.UserLevelNetworking,
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

// ProfileCreatorArgs represents the arguments passed to the ProfileCreator
type ProfileCreatorArgs struct {
	PowerConsumptionMode        string `json:"power-consumption-mode"`
	MustGatherDirPath           string `json:"must-gather-dir-path"`
	ProfileName                 string `json:"profile-name"`
	ReservedCPUCount            int    `json:"reserved-cpu-count"`
	SplitReservedCPUsAcrossNUMA bool   `json:"split-reserved-cpus-across-numa"`
	DisableHT                   bool   `json:"disable-ht"`
	RTKernel                    bool   `json:"rt-kernel"`
	UserLevelNetworking         *bool  `json:"user-level-networking,omitempty"`
	MCPName                     string `json:"mcp-name"`
	TMPolicy                    string `json:"topology-manager-policy"`
	Info                        bool   `json:"info"`
}

func init() {
	args := &ProfileCreatorArgs{
		UserLevelNetworking: pointer.BoolPtr(false),
	}
	log.SetOutput(os.Stderr)
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: true,
	})
	rootCmd.PersistentFlags().IntVar(&args.ReservedCPUCount, "reserved-cpu-count", 0, "Number of reserved CPUs (required)")
	rootCmd.MarkPersistentFlagRequired("reserved-cpu-count")
	rootCmd.PersistentFlags().BoolVar(&args.SplitReservedCPUsAcrossNUMA, "split-reserved-cpus-across-numa", false, "Split the Reserved CPUs across NUMA nodes")
	rootCmd.PersistentFlags().StringVar(&args.MCPName, "mcp-name", "worker-cnf", "MCP name corresponding to the target machines (required)")
	rootCmd.MarkPersistentFlagRequired("mcp-name")
	rootCmd.PersistentFlags().BoolVar(&args.DisableHT, "disable-ht", false, "Disable Hyperthreading")
	rootCmd.PersistentFlags().BoolVar(&args.RTKernel, "rt-kernel", true, "Enable Real Time Kernel (required)")
	rootCmd.MarkPersistentFlagRequired("rt-kernel")
	rootCmd.PersistentFlags().BoolVar(args.UserLevelNetworking, "user-level-networking", false, "Run with User level Networking(DPDK) enabled")
	rootCmd.PersistentFlags().StringVar(&args.PowerConsumptionMode, "power-consumption-mode", profilecreator.ValidPowerConsumptionModes[0], fmt.Sprintf("The power consumption mode.  [Valid values: %s, %s, %s]", profilecreator.ValidPowerConsumptionModes[0], profilecreator.ValidPowerConsumptionModes[1], profilecreator.ValidPowerConsumptionModes[2]))
	rootCmd.PersistentFlags().StringVar(&args.MustGatherDirPath, "must-gather-dir-path", "must-gather", "Must gather directory path")
	rootCmd.MarkPersistentFlagRequired("must-gather-dir-path")
	rootCmd.PersistentFlags().StringVar(&args.ProfileName, "profile-name", "performance", "Name of the performance profile to be created")
	rootCmd.PersistentFlags().StringVar(&args.TMPolicy, "topology-manager-policy", kubeletconfig.RestrictedTopologyManagerPolicy, fmt.Sprintf("Kubelet Topology Manager Policy of the performance profile to be created. [Valid values: %s, %s, %s]", kubeletconfig.SingleNumaNodeTopologyManagerPolicy, kubeletconfig.BestEffortTopologyManagerPolicy, kubeletconfig.RestrictedTopologyManagerPolicy))
	rootCmd.PersistentFlags().BoolVar(&args.Info, "info", false, "Show cluster information, disregard the other arguments")
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
			MachineConfigPoolSelector: profileData.mcpSelector,
			NodeSelector:              profileData.nodeSelector.MatchLabels,
			RealTimeKernel: &performancev2.RealTimeKernel{
				Enabled: &profileData.rtKernel,
			},
			AdditionalKernelArgs: profileData.additionalKernelArgs,
			NUMA: &performancev2.NUMA{
				TopologyPolicy: &profileData.topologyPoilcy,
			},
		},
	}

	if profileData.userLevelNetworking != nil {
		profile.Spec.Net = &performancev2.Net{
			UserLevelNetworking: profileData.userLevelNetworking,
		}
	}

	// write CSV to out dir
	writer := strings.Builder{}
	csvtools.MarshallObject(&profile, &writer)

	fmt.Printf("%s", writer.String())
}
