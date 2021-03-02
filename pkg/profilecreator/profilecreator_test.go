package profilecreator

import (
	"path/filepath"

	"github.com/jaypipes/ghw/pkg/cpu"
	"github.com/jaypipes/ghw/pkg/topology"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("PerformanceProfileCreator: MCP and Node Matching", func() {
	var node *v1.Node
	var labelSelector *metav1.LabelSelector
	BeforeEach(func() {
		node = newTestNode("test-node")
		labelSelector = metav1.AddLabelToSelector(&metav1.LabelSelector{}, "node-role.kubernetes.io/worker-cnf", "")
	})

	Context("Identifying Nodes targetted by MCP", func() {
		It("should find a match", func() {
			nodeLabel := map[string]string{
				"node-role.kubernetes.io/worker-cnf": "",
			}
			node.Labels = nodeLabel
			nodes := newTestNodeList(node)
			matchedNodes, err := GetMatchedNodes(nodes, labelSelector)
			Expect(err).ToNot(HaveOccurred())
			Expect(matchedNodes).ToNot(BeNil())
			Expect(len(matchedNodes)).ToNot(Equal(0))
			Expect(matchedNodes[0].GetName()).To(Equal("test-node"))
		})
		It("should not find a match", func() {
			nodeLabel := map[string]string{
				"node-role.kubernetes.io/foo": "",
			}
			node.Labels = nodeLabel
			nodes := newTestNodeList(node)
			matchedNodes, err := GetMatchedNodes(nodes, labelSelector)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(matchedNodes)).To(Equal(0))
		})
	})
})

var _ = Describe("PerformanceProfileCreator: Getting MCP from Must Gather", func() {
	var mustGatherDirPath, mcpName, mcpNodeSelectorKey, mustGatherDirAbsolutePath string
	var err error
	Context("Identifying Nodes targetted by MCP", func() {
		It("gets the MCP successfully", func() {
			mcpName = "worker-cnf"
			mcpNodeSelectorKey = "node-role.kubernetes.io/worker-cnf"
			mustGatherDirPath = "../../testdata/must-gather/must-gather.local.directory"
			mustGatherDirAbsolutePath, err = filepath.Abs(mustGatherDirPath)
			Expect(err).ToNot(HaveOccurred())
			mcp, err := GetMCP(mustGatherDirAbsolutePath, mcpName)
			k, _ := components.GetFirstKeyAndValue(mcp.Spec.NodeSelector.MatchLabels)
			Expect(err).ToNot(HaveOccurred())
			Expect(k).To(Equal(mcpNodeSelectorKey))
		})
		It("fails to get MCP as an MCP with that name doesn't exist", func() {
			mcpName = "foo"
			mustGatherDirPath = "../../testdata/must-gather/must-gather.local.directory"
			mustGatherDirAbsolutePath, err = filepath.Abs(mustGatherDirPath)
			mcp, err := GetMCP(mustGatherDirAbsolutePath, mcpName)
			Expect(mcp).To(BeNil())
			Expect(err).To(HaveOccurred())
		})
		It("fails to get MCP due to misconfigured must-gather path", func() {
			mcpName = "worker-cnf"
			mustGatherDirPath = "../../testdata/must-gather/foo-path"
			mustGatherDirAbsolutePath, err = filepath.Abs(mustGatherDirPath)
			Expect(err).ToNot(HaveOccurred())
			_, err := GetMCP(mustGatherDirAbsolutePath, mcpName)
			Expect(err).To(HaveOccurred())
		})

	})
})

var _ = Describe("PerformanceProfileCreator: Getting Nodes from Must Gather", func() {
	var mustGatherDirPath, mustGatherDirAbsolutePath string
	var err error

	Context("Identifying Nodes in the cluster", func() {
		It("gets the Nodes successfully", func() {
			mustGatherDirPath = "../../testdata/must-gather/must-gather.local.directory"
			mustGatherDirAbsolutePath, err = filepath.Abs(mustGatherDirPath)
			Expect(err).ToNot(HaveOccurred())
			nodes, err := GetNodeList(mustGatherDirAbsolutePath)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(nodes)).To(Equal(5))
		})
		It("fails to get Nodes due to misconfigured must-gather path", func() {
			mustGatherDirPath = "../../testdata/must-gather/foo-path"
			mustGatherDirAbsolutePath, err = filepath.Abs(mustGatherDirPath)
			_, err := GetNodeList(mustGatherDirAbsolutePath)
			Expect(err).To(HaveOccurred())
		})

	})
})

var _ = Describe("PerformanceProfileCreator: Consuming GHW Snapshot from Must Gather", func() {
	var mustGatherDirPath, mustGatherDirAbsolutePath string
	var node *v1.Node
	var err error

	Context("Identifying Nodes Info of the nodes cluster", func() {
		It("gets the Nodes Info successfully", func() {
			node = newTestNode("cnfd1-worker-0.fci1.kni.lab.eng.bos.redhat.com")
			mustGatherDirPath = "../../testdata/must-gather/must-gather.local.directory"
			mustGatherDirAbsolutePath, err = filepath.Abs(mustGatherDirPath)
			Expect(err).ToNot(HaveOccurred())
			handle, err := NewGHWHandler(mustGatherDirAbsolutePath, node)
			Expect(err).ToNot(HaveOccurred())
			cpuInfo, err := handle.CPU()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(cpuInfo.Processors)).To(Equal(2))
			Expect(int(cpuInfo.TotalCores)).To(Equal(40))
			Expect(int(cpuInfo.TotalThreads)).To(Equal(80))
			topologyInfo, err := handle.SortedTopology()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(topologyInfo.Nodes)).To(Equal(2))
		})
		It("fails to get Nodes Info due to misconfigured must-gather path", func() {
			mustGatherDirPath = "../../testdata/must-gather/foo-path"
			mustGatherDirAbsolutePath, err = filepath.Abs(mustGatherDirPath)
			_, err := NewGHWHandler(mustGatherDirAbsolutePath, node)
			Expect(err).To(HaveOccurred())
		})
		It("fails to get Nodes Info for a node that does not exist", func() {
			node = newTestNode("foo")
			mustGatherDirPath = "../../testdata/must-gather/must-gather.local.directory"
			mustGatherDirAbsolutePath, err = filepath.Abs(mustGatherDirPath)
			_, err := NewGHWHandler(mustGatherDirAbsolutePath, node)
			Expect(err).To(HaveOccurred())
		})

	})
})

var _ = Describe("PerformanceProfileCreator: Populating Reserved and Isolated CPUs in Performance Profile", func() {
	var mustGatherDirPath, mustGatherDirAbsolutePath string
	var node *v1.Node
	var handle *GHWHandler
	var splitReservedCPUsAcrossNUMA bool
	var reservedCPUCount int
	var err error

	BeforeEach(func() {
		node = newTestNode("cnfd1-worker-0.fci1.kni.lab.eng.bos.redhat.com")
		mustGatherDirPath = "../../testdata/must-gather/must-gather.local.directory"
	})
	Context("Check if reserved and isolated CPUs are properly populated in the performance profile", func() {
		It("Ensure reserved CPUs populated are correctly when splitReservedCPUsAcrossNUMA is disabled", func() {
			reservedCPUCount = 20 // random number, no special meaning
			splitReservedCPUsAcrossNUMA = false
			mustGatherDirAbsolutePath, err = filepath.Abs(mustGatherDirPath)
			Expect(err).ToNot(HaveOccurred())
			handle, err = NewGHWHandler(mustGatherDirAbsolutePath, node)
			Expect(err).ToNot(HaveOccurred())
			reservedCPUs, isolatedCPUs, err := handle.GetReservedAndIsolatedCPUs(reservedCPUCount, splitReservedCPUsAcrossNUMA)
			Expect(err).ToNot(HaveOccurred())
			Expect(reservedCPUs).To(Equal("0,2,4,6,8,10,12,14,16,18,40,42,44,46,48,50,52,54,56,58"))
			Expect(isolatedCPUs).To(Equal("1,3,5,7,9,11,13,15,17,19-39,41,43,45,47,49,51,53,55,57,59-79"))
		})
		It("Ensure reserved CPUs populated are correctly when splitReservedCPUsAcrossNUMA is enabled", func() {
			reservedCPUCount = 20 // random number, no special meaning
			splitReservedCPUsAcrossNUMA = true
			mustGatherDirAbsolutePath, err = filepath.Abs(mustGatherDirPath)
			Expect(err).ToNot(HaveOccurred())
			handle, err = NewGHWHandler(mustGatherDirAbsolutePath, node)
			Expect(err).ToNot(HaveOccurred())
			reservedCPUs, isolatedCPUs, err := handle.GetReservedAndIsolatedCPUs(reservedCPUCount, splitReservedCPUsAcrossNUMA)
			Expect(err).ToNot(HaveOccurred())
			Expect(reservedCPUs).To(Equal("0-9,40-49"))
			Expect(isolatedCPUs).To(Equal("10-39,50-79"))
		})
		It("Errors out in case negative reservedCPUCount is specified", func() {
			reservedCPUCount = -2 // random negative number, no special meaning
			splitReservedCPUsAcrossNUMA = true
			mustGatherDirAbsolutePath, err = filepath.Abs(mustGatherDirPath)
			Expect(err).ToNot(HaveOccurred())
			handle, err = NewGHWHandler(mustGatherDirAbsolutePath, node)
			Expect(err).ToNot(HaveOccurred())
			_, _, err := handle.GetReservedAndIsolatedCPUs(reservedCPUCount, splitReservedCPUsAcrossNUMA)
			Expect(err).To(HaveOccurred())
		})
		It("Errors out in case hyperthreading is enabled, splitReservedCPUsAcrossNUMA is enabled and number of reserved CPUs per number of NUMA nodes are odd", func() {
			reservedCPUCount = 21 // random number which results in a CPU split per NUMA node (11 + 10 in this case) such that odd number of reserved CPUs (11) have to be allocated from a NUMA node
			splitReservedCPUsAcrossNUMA = true
			mustGatherDirAbsolutePath, err = filepath.Abs(mustGatherDirPath)
			Expect(err).ToNot(HaveOccurred())
			handle, err = NewGHWHandler(mustGatherDirAbsolutePath, node)
			Expect(err).ToNot(HaveOccurred())
			_, _, err := handle.GetReservedAndIsolatedCPUs(reservedCPUCount, splitReservedCPUsAcrossNUMA)
			Expect(err).To(HaveOccurred())
		})
		It("Errors out in case hyperthreading is enabled, splitReservedCPUsAcrossNUMA is disabled and number of reserved CPUs are odd", func() {
			reservedCPUCount = 21 // random number which results in odd number (21) of CPUs to be allocated from a NUMA node
			splitReservedCPUsAcrossNUMA = false
			mustGatherDirAbsolutePath, err = filepath.Abs(mustGatherDirPath)
			Expect(err).ToNot(HaveOccurred())
			handle, err = NewGHWHandler(mustGatherDirAbsolutePath, node)
			Expect(err).ToNot(HaveOccurred())
			_, _, err := handle.GetReservedAndIsolatedCPUs(reservedCPUCount, splitReservedCPUsAcrossNUMA)
			Expect(err).To(HaveOccurred())
		})

	})
})

var _ = Describe("PerformanceProfileCreator: Check if Hyperthreading enabled/disabled in a system to correctly populate reserved and isolated CPUs in the performance profile", func() {
	var mustGatherDirPath, mustGatherDirAbsolutePath string
	var node *v1.Node
	var handle *GHWHandler
	var err error

	BeforeEach(func() {
		mustGatherDirPath = "../../testdata/must-gather/must-gather.local.directory"
	})
	Context("Check if hyperthreading is enabled on the system or not", func() {
		It("Ensure we detect correctly that hyperthreading is enabled on a system", func() {
			node = newTestNode("cnfd1-worker-0.fci1.kni.lab.eng.bos.redhat.com")
			mustGatherDirAbsolutePath, err = filepath.Abs(mustGatherDirPath)
			Expect(err).ToNot(HaveOccurred())
			handle, err = NewGHWHandler(mustGatherDirAbsolutePath, node)
			Expect(err).ToNot(HaveOccurred())
			htEnabled, err := handle.isHyperthreadingEnabled()
			Expect(err).ToNot(HaveOccurred())
			Expect(htEnabled).To(Equal(true))
		})
		It("Ensure we detect correctly that hyperthreading is disabled on a system", func() {
			node = newTestNode("dhcp19-232-239.fci1.kni.lab.eng.bos.redhat.com")
			mustGatherDirAbsolutePath, err = filepath.Abs(mustGatherDirPath)
			Expect(err).ToNot(HaveOccurred())
			handle, err = NewGHWHandler(mustGatherDirAbsolutePath, node)
			Expect(err).ToNot(HaveOccurred())
			htEnabled, err := handle.isHyperthreadingEnabled()
			Expect(err).ToNot(HaveOccurred())
			Expect(htEnabled).To(Equal(false))
		})
	})
})

var _ = Describe("PerformanceProfileCreator: Test Helper Functions getCPUsSplitAcrossNUMA and getCPUsSequentially", func() {
	var mustGatherDirPath, mustGatherDirAbsolutePath string
	var node *v1.Node
	var handle *GHWHandler
	var reservedCPUCount int
	var topologyInfoNodes []*topology.Node
	var err error

	BeforeEach(func() {
		mustGatherDirPath = "../../testdata/must-gather/must-gather.local.directory"
		node = newTestNode("cnfd1-worker-0.fci1.kni.lab.eng.bos.redhat.com")
		topologyInfoNodes = []*topology.Node{
			{
				ID: 0,
				Cores: []*cpu.ProcessorCore{
					{ID: 0, Index: 0, NumThreads: 2, LogicalProcessors: []int{0, 40}},
					{ID: 0, Index: 0, NumThreads: 2, LogicalProcessors: []int{0, 40}},
					{ID: 4, Index: 6, NumThreads: 2, LogicalProcessors: []int{2, 42}},
					{ID: 1, Index: 17, NumThreads: 2, LogicalProcessors: []int{4, 44}},
					{ID: 3, Index: 18, NumThreads: 2, LogicalProcessors: []int{6, 46}},
					{ID: 2, Index: 19, NumThreads: 2, LogicalProcessors: []int{8, 48}},
					{ID: 12, Index: 1, NumThreads: 2, LogicalProcessors: []int{10, 50}},
					{ID: 8, Index: 2, NumThreads: 2, LogicalProcessors: []int{12, 52}},
					{ID: 11, Index: 3, NumThreads: 2, LogicalProcessors: []int{14, 54}},
					{ID: 9, Index: 4, NumThreads: 2, LogicalProcessors: []int{16, 56}},
					{ID: 10, Index: 5, NumThreads: 2, LogicalProcessors: []int{18, 58}},
					{ID: 16, Index: 7, NumThreads: 2, LogicalProcessors: []int{20, 60}},
					{ID: 20, Index: 8, NumThreads: 2, LogicalProcessors: []int{22, 62}},
					{ID: 17, Index: 9, NumThreads: 2, LogicalProcessors: []int{24, 64}},
					{ID: 19, Index: 10, NumThreads: 2, LogicalProcessors: []int{26, 66}},
					{ID: 18, Index: 11, NumThreads: 2, LogicalProcessors: []int{28, 68}},
					{ID: 28, Index: 12, NumThreads: 2, LogicalProcessors: []int{30, 70}},
					{ID: 24, Index: 13, NumThreads: 2, LogicalProcessors: []int{32, 72}},
					{ID: 27, Index: 14, NumThreads: 2, LogicalProcessors: []int{34, 74}},
					{ID: 25, Index: 15, NumThreads: 2, LogicalProcessors: []int{36, 76}},
					{ID: 26, Index: 16, NumThreads: 2, LogicalProcessors: []int{38, 78}},
				},
			},
			{
				ID: 1,
				Cores: []*cpu.ProcessorCore{
					{ID: 0, Index: 0, NumThreads: 2, LogicalProcessors: []int{1, 41}},
					{ID: 4, Index: 11, NumThreads: 2, LogicalProcessors: []int{3, 43}},
					{ID: 1, Index: 17, NumThreads: 2, LogicalProcessors: []int{5, 45}},
					{ID: 3, Index: 18, NumThreads: 2, LogicalProcessors: []int{7, 47}},
					{ID: 2, Index: 19, NumThreads: 2, LogicalProcessors: []int{9, 49}},
					{ID: 12, Index: 1, NumThreads: 2, LogicalProcessors: []int{11, 51}},
					{ID: 8, Index: 2, NumThreads: 2, LogicalProcessors: []int{13, 53}},
					{ID: 11, Index: 3, NumThreads: 2, LogicalProcessors: []int{15, 55}},
					{ID: 9, Index: 4, NumThreads: 2, LogicalProcessors: []int{17, 57}},
					{ID: 10, Index: 5, NumThreads: 2, LogicalProcessors: []int{19, 59}},
					{ID: 16, Index: 6, NumThreads: 2, LogicalProcessors: []int{21, 61}},
					{ID: 20, Index: 7, NumThreads: 2, LogicalProcessors: []int{23, 63}},
					{ID: 17, Index: 8, NumThreads: 2, LogicalProcessors: []int{25, 65}},
					{ID: 19, Index: 9, NumThreads: 2, LogicalProcessors: []int{27, 67}},
					{ID: 18, Index: 10, NumThreads: 2, LogicalProcessors: []int{29, 69}},
					{ID: 28, Index: 12, NumThreads: 2, LogicalProcessors: []int{31, 71}},
					{ID: 24, Index: 13, NumThreads: 2, LogicalProcessors: []int{33, 73}},
					{ID: 27, Index: 14, NumThreads: 2, LogicalProcessors: []int{35, 75}},
					{ID: 25, Index: 15, NumThreads: 2, LogicalProcessors: []int{37, 77}},
					{ID: 26, Index: 16, NumThreads: 2, LogicalProcessors: []int{39, 79}},
				},
			},
		}
	})
	Context("Check if getCPUsSplitAcrossNUMA and getCPUsSequentially are working correctly and reserved and isolated CPUs are properly populated in the performance profile", func() {
		It("Ensure reserved and isolated CPUs populated are correctly by getCPUsSplitAcrossNUMAwhen when splitReservedCPUsAcrossNUMA is enabled", func() {
			reservedCPUCount = 20 // random number, no special meaning
			mustGatherDirAbsolutePath, err = filepath.Abs(mustGatherDirPath)
			Expect(err).ToNot(HaveOccurred())
			handle, err = NewGHWHandler(mustGatherDirAbsolutePath, node)
			Expect(err).ToNot(HaveOccurred())
			reservedCPUs, isolatedCPUs, err := handle.getCPUsSplitAcrossNUMA(reservedCPUCount, topologyInfoNodes)
			Expect(err).ToNot(HaveOccurred())
			Expect(reservedCPUs).To(Equal("0-9,40-49"))
			Expect(isolatedCPUs).To(Equal("10-39,50-79"))
		})
		It("Errors out in case hyperthreading is enabled, splitReservedCPUsAcrossNUMA is enabled and number of reserved CPUs per number of NUMA nodes are odd", func() {
			reservedCPUCount = 21 // random number which results in a CPU split per NUMA node (11 + 10 in this case) such that odd number of reserved CPUs (11) have to be allocated from a NUMA node
			mustGatherDirAbsolutePath, err = filepath.Abs(mustGatherDirPath)
			Expect(err).ToNot(HaveOccurred())
			handle, err = NewGHWHandler(mustGatherDirAbsolutePath, node)
			Expect(err).ToNot(HaveOccurred())
			_, _, err := handle.getCPUsSplitAcrossNUMA(reservedCPUCount, topologyInfoNodes)
			Expect(err).To(HaveOccurred())
		})
		It("Ensure reserved and isolated CPUs populated are correctly by getCPUsSequentially when splitReservedCPUsAcrossNUMA is disabled", func() {
			reservedCPUCount = 20 // random number, no special meaning
			mustGatherDirAbsolutePath, err = filepath.Abs(mustGatherDirPath)
			Expect(err).ToNot(HaveOccurred())
			handle, err = NewGHWHandler(mustGatherDirAbsolutePath, node)
			Expect(err).ToNot(HaveOccurred())
			reservedCPUs, isolatedCPUs, err := handle.getCPUsSequentially(reservedCPUCount, topologyInfoNodes)
			Expect(err).ToNot(HaveOccurred())
			Expect(reservedCPUs).To(Equal("0,2,4,6,8,10,12,14,16,18,40,42,44,46,48,50,52,54,56,58"))
			Expect(isolatedCPUs).To(Equal("1,3,5,7,9,11,13,15,17,19-39,41,43,45,47,49,51,53,55,57,59-79"))
		})
		It("Errors out in case hyperthreading is enabled, splitReservedCPUsAcrossNUMA is disabled and number of reserved CPUs are odd", func() {
			reservedCPUCount = 21 // random number which results in odd number (21) of CPUs to be allocated from a NUMA node
			mustGatherDirAbsolutePath, err = filepath.Abs(mustGatherDirPath)
			Expect(err).ToNot(HaveOccurred())
			handle, err = NewGHWHandler(mustGatherDirAbsolutePath, node)
			Expect(err).ToNot(HaveOccurred())
			_, _, err := handle.getCPUsSequentially(reservedCPUCount, topologyInfoNodes)
			Expect(err).To(HaveOccurred())
		})
	})
})
