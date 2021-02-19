package profilecreator

import (
	"path/filepath"

	v1 "k8s.io/api/core/v1"
)

func newTestNode(nodeName string) *v1.Node {
	n := v1.Node{}
	n.Name = nodeName
	return &n
}
func newTestNodeList(nodes ...*v1.Node) []*v1.Node {
	nodeList := make([]*v1.Node, 0)
	for _, node := range nodes {
		nodeList = append(nodeList, node)
	}
	return nodeList
}

func getAbsolutePath(mustGatherDirPath string) (string, error) {
	return filepath.Abs(mustGatherDirPath)
}
