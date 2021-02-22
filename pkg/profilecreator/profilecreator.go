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

	"github.com/jaypipes/ghw"
	"github.com/jaypipes/ghw/pkg/option"
	log "github.com/sirupsen/logrus"

	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/component-helpers/scheduling/corev1"

	machineconfigv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	v1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// GetMatchedNodes returns the list of nodes that are targetted by a specified labelSelector
func GetMatchedNodes(nodes []*v1.Node, labelSelector *metav1.LabelSelector) ([]*v1.Node, error) {
	matchedNodes := make([]*v1.Node, 0)
	for _, node := range nodes {
		matches, _ := corev1.MatchNodeSelectorTerms(node, getNodeSelectorFromLabelSelector(labelSelector))
		if matches {
			matchedNodes = append(matchedNodes, node)
		}
	}
	return matchedNodes, nil
}

func getNodeSelectorFromLabelSelector(labelSelector *metav1.LabelSelector) *v1.NodeSelector {

	matchExpressions := make([]v1.NodeSelectorRequirement, 0)
	for key, value := range labelSelector.MatchLabels {
		matchExpressions = append(matchExpressions, v1.NodeSelectorRequirement{
			Key:      key,
			Operator: v1.NodeSelectorOpIn,
			Values:   []string{value},
		})
	}
	matchFields := make([]v1.NodeSelectorRequirement, 0)
	for _, labelSelectorRequirement := range labelSelector.MatchExpressions {
		matchExpressions = append(matchFields, v1.NodeSelectorRequirement{
			Key:      labelSelectorRequirement.Key,
			Operator: v1.NodeSelectorOperator(string(labelSelectorRequirement.Operator)),
			Values:   labelSelectorRequirement.Values,
		})
	}

	nodeSelectorTerms := []v1.NodeSelectorTerm{
		{
			MatchExpressions: matchExpressions,
			MatchFields:      matchFields,
		},
	}
	nodeSelector := &v1.NodeSelector{
		NodeSelectorTerms: nodeSelectorTerms,
	}

	return nodeSelector

}

func getMustGatherFullPaths(mustGatherPath string, suffix string) (string, error) {
	// The glob pattern below depends on the must gather image name. It is assumed here
	// that the image would have "performance-addon-operator-must-gather" substring in the name.
	paths, err := filepath.Glob(mustGatherPath + "/*performance-addon-operator-must-gather*/" + suffix)
	if err != nil {
		return "", fmt.Errorf("Error obtaining the path mustGatherPath:%s, suffix:%s %v", mustGatherPath, suffix, err)
	}
	if paths == nil {
		return "", fmt.Errorf("No match for the specified must gather directory path: %s and suffix: %s", mustGatherPath, suffix)

	}
	if len(paths) > 1 {
		log.Infof("Multiple matches for the specified must gather directory path: %s and suffix: %s", mustGatherPath, suffix)
		return "", fmt.Errorf("Multiple matches for the specified must gather directory path: %s and suffix: %s.\n Expected only one performance-addon-operator-must-gather* directory, please check the must-gather tarball", mustGatherPath, suffix)
	}
	// returning one possible path
	return paths[0], err
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

// LoadSnapshot loads a ghw snapshot corresponding to a node
func LoadSnapshot(mustGatherDirPath string, node *v1.Node) (*option.Option, error) {
	nodeName := node.GetName()
	nodePathSuffix := path.Join(Nodes)
	nodepath, err := getMustGatherFullPaths(mustGatherDirPath, nodePathSuffix)
	if err != nil {
		return nil, fmt.Errorf("Error obtaining the node path %s: %v", nodeName, err)
	}
	_, err = os.Stat(path.Join(nodepath, nodeName, SysInfoFileName))
	if err != nil {
		return nil, fmt.Errorf("Error obtaining the path: %s for node %s: %v", nodeName, nodepath, err)
	}
	snapShotOptions := ghw.WithSnapshot(ghw.SnapshotOptions{
		Path: path.Join(nodepath, nodeName, SysInfoFileName),
	})
	return snapShotOptions, nil
}
