package components

const (
	// AssetsDir defines the directory with assets under the operator image
	AssetsDir = "/assets"
)
const (
	// LabelMachineConfigurationRole defines the label for machine configuration role
	LabelMachineConfigurationRole = "machineconfiguration.openshift.io/role"
	// LabelMachineConfigPoolRole defines the label for machine config pool role
	LabelMachineConfigPoolRole = "machineconfigpool.openshift.io/role"
	// RoleWorker defines the worker role
	RoleWorker = "worker"
	// RoleWorkerPerformance defines the worker role for performance sensitive workflows
	// TODO: change it back to longer name once https://bugzilla.redhat.com/show_bug.cgi?id=1787907 fixed
	// RoleWorkerPerformance = "worker-performance"
	RoleWorkerPerformance = "performance"
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
	// TOOD: uncomment once https://bugzilla.redhat.com/show_bug.cgi?id=1788061 fixed
	// FeatureGateLatencySensetiveName = "latency-sensitive"
	FeatureGateLatencySensetiveName = "cluster"
)
