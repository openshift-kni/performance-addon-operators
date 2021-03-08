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

package profilecreator

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/jaypipes/ghw"
	"github.com/jaypipes/ghw/pkg/cpu"
	"github.com/jaypipes/ghw/pkg/option"
	"github.com/jaypipes/ghw/pkg/topology"
	log "github.com/sirupsen/logrus"

	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"

	machineconfigv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	v1 "k8s.io/api/core/v1"
)

const (
	// ClusterScopedResources defines the subpath, relative to the top-level must-gather directory.
	// A top-level must-gather directory is of the following format:
	// must-gather-dir/quay-io-openshift-kni-performance-addon-operator-must-gather-sha256-<Image SHA>
	// Here we find the cluster-scoped definitions saved by must-gather
	ClusterScopedResources = "cluster-scoped-resources"
	// CoreNodes defines the subpath, relative to ClusterScopedResources, on which we find node-specific data
	CoreNodes = "core/nodes"
	// MCPools defines the subpath, relative to ClusterScopedResources, on which we find the machine config pool definitions
	MCPools = "machineconfiguration.openshift.io/machineconfigpools"
	// YAMLSuffix is the extension of the yaml files saved by must-gather
	YAMLSuffix = ".yaml"
	// Nodes defines the subpath, relative to top-level must-gather directory, on which we find node-specific data
	Nodes = "nodes"
	// SysInfoFileName defines the name of the file where ghw snapshot is stored
	SysInfoFileName = "sysinfo.tgz"
)

func init() {
	log.SetOutput(os.Stderr)
}

func getMustGatherFullPathsWithFilter(mustGatherPath string, suffix string, filter string) (string, error) {
	var paths []string

	// don't assume directory names, only look for the suffix, filter out files having "filter" in their names
	err := filepath.Walk(mustGatherPath, func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(path, suffix) {
			if len(filter) == 0 || !strings.Contains(path, filter) {
				paths = append(paths, path)
			}
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("Error obtaining the path mustGatherPath:%s, suffix:%s %v", mustGatherPath, suffix, err)
	}

	if len(paths) == 0 {
		return "", fmt.Errorf("No match for the specified must gather directory path: %s and suffix: %s", mustGatherPath, suffix)

	}
	if len(paths) > 1 {
		log.Infof("Multiple matches for the specified must gather directory path: %s and suffix: %s", mustGatherPath, suffix)
		return "", fmt.Errorf("Multiple matches for the specified must gather directory path: %s and suffix: %s.\n Expected only one performance-addon-operator-must-gather* directory, please check the must-gather tarball", mustGatherPath, suffix)
	}
	// returning one possible path
	return paths[0], err
}

func getMustGatherFullPaths(mustGatherPath string, suffix string) (string, error) {
	return getMustGatherFullPathsWithFilter(mustGatherPath, suffix, "")
}

func getNode(mustGatherDirPath, nodeName string) (*v1.Node, error) {
	var node v1.Node
	nodePathSuffix := path.Join(ClusterScopedResources, CoreNodes, nodeName)
	path, err := getMustGatherFullPaths(mustGatherDirPath, nodePathSuffix)
	if err != nil {
		return nil, fmt.Errorf("Error obtaining MachineConfigPool %s: %v", nodeName, err)
	}

	src, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Error opening %q: %v", path, err)
	}
	defer src.Close()

	dec := k8syaml.NewYAMLOrJSONDecoder(src, 1024)
	if err := dec.Decode(&node); err != nil {
		return nil, fmt.Errorf("Error opening %q: %v", path, err)
	}
	return &node, nil
}

// GetNodeList returns the list of nodes using the Node YAMLs stored in Must Gather
func GetNodeList(mustGatherDirPath string) ([]*v1.Node, error) {
	machines := make([]*v1.Node, 0)

	nodePathSuffix := path.Join(ClusterScopedResources, CoreNodes)
	nodePath, err := getMustGatherFullPaths(mustGatherDirPath, nodePathSuffix)
	if err != nil {
		return nil, fmt.Errorf("Error obtaining Nodes: %v", err)
	}
	if nodePath == "" {
		return nil, fmt.Errorf("Error obtaining Nodes: %v", err)
	}

	nodes, err := ioutil.ReadDir(nodePath)
	if err != nil {
		return nil, fmt.Errorf("failed to list mustGatherPath directories: %v", err)
	}
	for _, node := range nodes {
		nodeName := node.Name()
		node, err := getNode(mustGatherDirPath, nodeName)
		if err != nil {
			return nil, fmt.Errorf("Error obtaining Nodes %s: %v", nodeName, err)
		}
		machines = append(machines, node)
	}
	return machines, nil
}

// GetMCPList returns the list of MCPs using the mcp YAMLs stored in Must Gather
func GetMCPList(mustGatherDirPath string) ([]*machineconfigv1.MachineConfigPool, error) {
	pools := make([]*machineconfigv1.MachineConfigPool, 0)

	mcpPathSuffix := path.Join(ClusterScopedResources, MCPools)
	mcpPath, err := getMustGatherFullPaths(mustGatherDirPath, mcpPathSuffix)
	if err != nil {
		return nil, fmt.Errorf("failed to get MCPs: %v", err)
	}
	if mcpPath == "" {
		return nil, fmt.Errorf("failed to get MCPs path: %v", err)
	}

	mcpFiles, err := ioutil.ReadDir(mcpPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list mustGatherPath directories: %v", err)
	}
	for _, mcp := range mcpFiles {
		mcpName := strings.TrimSuffix(mcp.Name(), filepath.Ext(mcp.Name()))

		// must-gather does not return the master nodes
		if mcpName == "master" {
			continue
		}

		mcp, err := GetMCP(mustGatherDirPath, mcpName)
		if err != nil {
			return nil, fmt.Errorf("can't obtain MCP %s: %v", mcpName, err)
		}
		pools = append(pools, mcp)
	}
	return pools, nil
}

// GetMCP returns an MCP object corresponding to a specified MCP Name
func GetMCP(mustGatherDirPath, mcpName string) (*machineconfigv1.MachineConfigPool, error) {
	var mcp machineconfigv1.MachineConfigPool

	mcpPathSuffix := path.Join(ClusterScopedResources, MCPools, mcpName+YAMLSuffix)
	mcpPath, err := getMustGatherFullPaths(mustGatherDirPath, mcpPathSuffix)
	if err != nil {
		return nil, fmt.Errorf("Error obtaining MachineConfigPool %s: %v", mcpName, err)
	}
	if mcpPath == "" {
		return nil, fmt.Errorf("Error obtaining MachineConfigPool, mcp:%s does not exist: %v", mcpName, err)
	}

	src, err := os.Open(mcpPath)
	if err != nil {
		return nil, fmt.Errorf("Error opening %q: %v", mcpPath, err)
	}
	defer src.Close()
	dec := k8syaml.NewYAMLOrJSONDecoder(src, 1024)
	if err := dec.Decode(&mcp); err != nil {
		return nil, fmt.Errorf("Error opening %q: %v", mcpPath, err)
	}
	return &mcp, nil
}

// NewGHWHandler is a handler to use ghw options corresponding to a node
func NewGHWHandler(mustGatherDirPath string, node *v1.Node) (*GHWHandler, error) {
	nodeName := node.GetName()
	nodePathSuffix := path.Join(Nodes)
	nodepath, err := getMustGatherFullPathsWithFilter(mustGatherDirPath, nodePathSuffix, ClusterScopedResources)
	if err != nil {
		return nil, fmt.Errorf("Error obtaining the node path %s: %v", nodeName, err)
	}
	_, err = os.Stat(path.Join(nodepath, nodeName, SysInfoFileName))
	if err != nil {
		return nil, fmt.Errorf("Error obtaining the path: %s for node %s: %v", nodeName, nodepath, err)
	}
	options := ghw.WithSnapshot(ghw.SnapshotOptions{
		Path: path.Join(nodepath, nodeName, SysInfoFileName),
	})
	ghwHandler := &GHWHandler{snapShotOptions: options}
	return ghwHandler, nil
}

// GHWHandler is a wrapper around ghw to get the API object
type GHWHandler struct {
	snapShotOptions *option.Option
}

// CPU returns a CPUInfo struct that contains information about the CPUs on the host system
func (ghwHandler GHWHandler) CPU() (*cpu.Info, error) {
	return ghw.CPU(ghwHandler.snapShotOptions)
}

// SortedTopology returns a TopologyInfo struct that contains information about the Topology sorted by numa ids and cpu ids on the host system
func (ghwHandler GHWHandler) SortedTopology() (*topology.Info, error) {
	topologyInfo, err := ghw.Topology(ghwHandler.snapShotOptions)
	if err != nil {
		return nil, fmt.Errorf("Error obtaining Topology Info from GHW snapshot: %v", err)
	}
	sort.Slice(topologyInfo.Nodes, func(x, y int) bool {
		return topologyInfo.Nodes[x].ID < topologyInfo.Nodes[y].ID
	})
	for _, node := range topologyInfo.Nodes {
		for _, core := range node.Cores {
			sort.Slice(core.LogicalProcessors, func(x, y int) bool {
				return core.LogicalProcessors[x] < core.LogicalProcessors[y]
			})
		}
		sort.Slice(node.Cores, func(i, j int) bool {
			return node.Cores[i].LogicalProcessors[0] < node.Cores[j].LogicalProcessors[0]
		})
	}
	return topologyInfo, nil
}

// GetReservedAndIsolatedCPUs returns Reserved and Isolated CPUs
func (ghwHandler GHWHandler) GetReservedAndIsolatedCPUs(reservedCPUCount int, splitReservedCPUsAcrossNUMA bool) (string, string, error) {
	cpuInfo, err := ghwHandler.CPU()
	if err != nil {
		return "", "", fmt.Errorf("Error obtaining CPU Info from GHW snapshot: %v", err)
	}
	if reservedCPUCount < 0 || reservedCPUCount > int(cpuInfo.TotalThreads) {
		return "", "", fmt.Errorf("Specified reserved CPU count is invalid, please specify it correctly")
	}
	topologyInfo, err := ghwHandler.SortedTopology()
	if err != nil {
		return "", "", fmt.Errorf("Error obtaining Topology Info from GHW snapshot: %v", err)
	}
	htEnabled, err := ghwHandler.isHyperthreadingEnabled()
	if err != nil {
		return "", "", fmt.Errorf("Error determining if Hyperthreading is enabled or not: %v", err)
	}
	if splitReservedCPUsAcrossNUMA {
		return ghwHandler.getCPUsSplitAcrossNUMA(reservedCPUCount, htEnabled, topologyInfo.Nodes)
	}
	return ghwHandler.getCPUsSequentially(reservedCPUCount, htEnabled, topologyInfo.Nodes)
}

// getCPUsSplitAcrossNUMA returns Reserved and Isolated CPUs split across NUMA nodes
// We identify the right number of CPUs that need to be allocated per NUMA node, meaning reservedPerNuma + (the additional number based on the remainder and the NUMA node)
// E.g. If the user requests 15 reserved cpus and we have 4 numa nodes, we find reservedPerNuma in this case is 3 and remainder = 3.
// For each numa node we find a max which keeps track of the cumulative resources that should be allocated for each NUMA node:
// max = (numaID+1)*reservedPerNuma + (numaNodeNum - remainder)
// For NUMA node 0 max = (0+1)*3 + 4-3 = 4 remainder is decremented => remainder is 2
// For NUMA node 1 max = (1+1)*3 + 4-2 = 8 remainder is decremented => remainder is 1
// For NUMA node 2 max = (2+1)*3 + 4-2 = 12 remainder is decremented => remainder is 0
// For NUMA Node 3 remainder = 0 so max = 12 + 3 = 15.
func (ghwHandler GHWHandler) getCPUsSplitAcrossNUMA(reservedCPUCount int, htEnabled bool, topologyInfoNodes []*topology.Node) (string, string, error) {
	reservedCPUSet := cpuset.NewBuilder()
	numaNodeNum := len(topologyInfoNodes)
	max := 0
	reservedPerNuma := reservedCPUCount / numaNodeNum
	remainder := reservedCPUCount % numaNodeNum
	if remainder != 0 {
		log.Warnf("The reserved CPUs cannot be split equally across NUMA Nodes")
	}
	for numaID, node := range topologyInfoNodes {
		if remainder != 0 {
			max = (numaID+1)*reservedPerNuma + (numaNodeNum - remainder)
			remainder--
		} else {
			max = max + reservedPerNuma
		}
		if max%2 != 0 && htEnabled {
			return "", "", fmt.Errorf("Can't allocatable odd number of CPUs from a NUMA Node")
		}
		for _, processorCores := range node.Cores {
			for _, core := range processorCores.LogicalProcessors {
				if reservedCPUSet.Result().Size() < max {
					reservedCPUSet.Add(core)
				}
			}
		}
	}
	totalCPUSet := totalCPUSetFromTopology(topologyInfoNodes)
	isolatedCPUSet := totalCPUSet.Difference(reservedCPUSet.Result())
	log.Infof("reservedCPUs: %v len(reservedCPUs): %d\n isolatedCPUs: %v len(isolatedCPUs): %d\n", reservedCPUSet.Result().String(), reservedCPUSet.Result().Size(), isolatedCPUSet.String(), isolatedCPUSet.Size())
	return reservedCPUSet.Result().String(), isolatedCPUSet.String(), nil

}

// getCPUsSequentially returns Reserved and Isolated CPUs sequentially
func (ghwHandler GHWHandler) getCPUsSequentially(reservedCPUCount int, htEnabled bool, topologyInfoNodes []*topology.Node) (string, string, error) {
	reservedCPUSet := cpuset.NewBuilder()
	if reservedCPUCount%2 != 0 && htEnabled {
		return "", "", fmt.Errorf("Can't allocatable odd number of CPUs from a NUMA Node")
	}
	for _, node := range topologyInfoNodes {
		for _, processorCores := range node.Cores {
			for _, core := range processorCores.LogicalProcessors {
				if reservedCPUSet.Result().Size() < reservedCPUCount {
					reservedCPUSet.Add(core)
				}
			}
		}
	}
	totalCPUSet := totalCPUSetFromTopology(topologyInfoNodes)
	isolatedCPUSet := totalCPUSet.Difference(reservedCPUSet.Result())
	log.Infof("reservedCPUs: %v len(reservedCPUs): %d\n isolatedCPUs: %v len(isolatedCPUs): %d\n", reservedCPUSet.Result().String(), reservedCPUSet.Result().Size(), isolatedCPUSet.String(), isolatedCPUSet.Size())
	return reservedCPUSet.Result().String(), isolatedCPUSet.String(), nil

}
func totalCPUSetFromTopology(topologyInfoNodes []*topology.Node) cpuset.CPUSet {
	totalCPUSet := cpuset.NewBuilder()
	for _, node := range topologyInfoNodes {
		for _, processorCores := range node.Cores {
			for _, core := range processorCores.LogicalProcessors {
				totalCPUSet.Add(core)
			}
		}
	}
	return totalCPUSet.Result()
}

// isHyperthreadingEnabled checks if hyperthreading is enabled on the system or not
func (ghwHandler GHWHandler) isHyperthreadingEnabled() (bool, error) {
	cpuInfo, err := ghwHandler.CPU()
	if err != nil {
		return false, fmt.Errorf("Error obtaining CPU Info from GHW snapshot: %v", err)
	}
	// Since there is no way to disable flags per-processor (not system wide) we check the flags of the first available processor.
	// A following implementation will leverage the /sys/devices/system/cpu/smt/active file which is the "standard" way to query HT.
	return contains(cpuInfo.Processors[0].Capabilities, "ht"), nil
}

// contains checks if a string is present in a slice
func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

// EnsureNodesHaveTheSameHardware returns an error if all the input nodes do not have the same hardware configuration
func EnsureNodesHaveTheSameHardware(mustGatherDirPath string, nodes []*v1.Node) error {
	if len(nodes) < 1 {
		return fmt.Errorf("no suitable nodes to compare")
	}

	first := nodes[0]
	firstHandle, err := NewGHWHandler(mustGatherDirPath, first)
	if err != nil {
		return fmt.Errorf("can't obtain GHW snapshot handle for %s: %v", first.GetName(), err)
	}

	firstTopology, err := firstHandle.SortedTopology()
	if err != nil {
		return fmt.Errorf("can't obtain Topology info from GHW snapshot for %s: %v", first.GetName(), err)
	}

	for _, node := range nodes[1:] {
		handle, err := NewGHWHandler(mustGatherDirPath, node)
		if err != nil {
			return fmt.Errorf("can't obtain GHW snapshot handle for %s: %v", node.GetName(), err)
		}

		topology, err := handle.SortedTopology()
		if err != nil {
			return fmt.Errorf("can't obtain Topology info from GHW snapshot for %s: %v", node.GetName(), err)
		}
		err = ensureSameTopology(firstTopology, topology)
		if err != nil {
			return fmt.Errorf("nodes %s and %s have different topology: %v", first.GetName(), node.GetName(), err)
		}
	}

	return nil
}

func ensureSameTopology(topology1, topology2 *topology.Info) error {
	if topology1.Architecture != topology2.Architecture {
		return fmt.Errorf("the arhitecture is different: %v vs %v", topology1.Architecture, topology2.Architecture)
	}

	if len(topology1.Nodes) != len(topology2.Nodes) {
		return fmt.Errorf("the number of NUMA nodes differ: %v vs %v", len(topology1.Nodes), len(topology2.Nodes))
	}

	for i, node1 := range topology1.Nodes {
		node2 := topology2.Nodes[i]
		if node1.ID != node2.ID {
			return fmt.Errorf("the NUMA node ids differ: %v vs %v", node1.ID, node2.ID)
		}

		cores1 := node1.Cores
		cores2 := node2.Cores
		if len(cores1) != len(cores2) {
			return fmt.Errorf("the number of CPU cores in NUMA node %d differ: %v vs %v",
				node1.ID, len(topology1.Nodes), len(topology2.Nodes))
		}

		for j, core1 := range cores1 {
			if !reflect.DeepEqual(core1, cores2[j]) {
				return fmt.Errorf("the CPU corres differ: %v vs %v", core1, cores2[j])
			}
		}
	}

	return nil
}
