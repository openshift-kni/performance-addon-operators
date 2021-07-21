package profileinfo

import performancev2 "github.com/openshift-kni/performance-addon-operators/api/v2"

// PerformanceProfileInfo is a wrapper for PerformanceProfile that can hold extra data and configuration
type PerformanceProfileInfo struct {
	performancev2.PerformanceProfile
	// extra data the operator cares about but which is not part of the public API
	WorkloadPartitionEnabled bool
}
