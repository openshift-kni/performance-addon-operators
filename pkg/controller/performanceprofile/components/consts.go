package components

const (
	// AssetsDir defines the directory with assets under the operator image
	AssetsDir = "/assets"
)
const (
	// LabelMachineConfigurationRole defines the label for machine configuration role
	LabelMachineConfigurationRole = "machineconfiguration.openshift.io/role"
	// LableMachineConfigPoolRole defines the label for machine config pool role
	LableMachineConfigPoolRole = "machineconfigpool.openshift.io/role"
	// RoleWorker defines the worker role
	RoleWorker = "worker"
	// RoleWorkerPerformance defines the worker role for performance sensitive workflows
	RoleWorkerPerformance = "worker-performance"
)

const (
	// NamespaceNodeTuningOperator defines the tuned profiles namespace
	NamespaceNodeTuningOperator = "openshift-cluster-node-tuning-operator"
	// ProfileNameNetworkLatency defines the network latency tuned profile name
	ProfileNameNetworkLatency = "openshift-node-network-latency"
	// ProfileNameWorkerRT defines the real time kernel performance tuned profile name
	ProfileNameWorkerRT = "openshift-node-real-time-kernel"
)

const (
	// FeatureGateLatencySensetiveName defines the latency sensetive feature gate name
	FeatureGateLatencySensetiveName = "latency-sensetive"
)