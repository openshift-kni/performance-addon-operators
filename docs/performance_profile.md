
# Performance Profile

This document documents the PerformanceProfile API introduced by the Performance Operator.

> This document is generated from code comments on the `PerformanceProfile` struct.  
> When contributing a change to this document please do so by changing those code comments.

## Table of Contents
* [CPU](#cpu)
* [CPUSet](#cpuset)
* [HugePage](#hugepage)
* [HugePageSize](#hugepagesize)
* [HugePages](#hugepages)
* [NUMA](#numa)
* [PerformanceProfile](#performanceprofile)
* [PerformanceProfileList](#performanceprofilelist)
* [PerformanceProfileSpec](#performanceprofilespec)
* [PerformanceProfileStatus](#performanceprofilestatus)
* [RealTimeKernel](#realtimekernel)

## CPU

CPU defines a set of CPU related features.

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| reserved | Reserved defines a set of CPUs that will not be used for any container workloads initiated by kubelet. | *[CPUSet](#cpuset) | false |
| isolated | Isolated defines a set of CPUs that will be used to give to application threads the most execution time possible, which means removing as many extraneous tasks off a CPU as possible. | *[CPUSet](#cpuset) | false |
| balanceIsolated | BalanceIsolated toggles whether or not the Isolated CPU set is eligible for load balancing work loads. When this option is set to \"false\", the Isolated CPU set will be static, meaning workloads have to explicitly assign each thread to a specific cpu in order to work across multiple CPUs. Setting this to \"true\" allows workloads to be balanced across CPUs. Setting this to \"false\" offers the most predictable performance for guaranteed workloads, but it offloads the complexity of cpu load balancing to the application. Defaults to \"true\" | *bool | false |

[Back to TOC](#table-of-contents)

## CPUSet

CPUSet defines the set of CPUs(0-3,8-11).

CPUSet is of type `string`.

[Back to TOC](#table-of-contents)

## HugePage

HugePage defines the number of allocated huge pages of the specific size.

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| size | Size defines huge page size, maps to the 'hugepagesz' kernel boot parameter. | [HugePageSize](#hugepagesize) | false |
| count | Count defines amount of huge pages, maps to the 'hugepages' kernel boot parameter. | int32 | false |
| node | Node defines the NUMA node where hugepages will be allocated, if not specified, pages will be allocated equally between NUMA nodes | *int32 | false |

[Back to TOC](#table-of-contents)

## HugePageSize

HugePageSize defines size of huge pages, can be 2M or 1G.

HugePageSize is of type `string`.

[Back to TOC](#table-of-contents)

## HugePages

HugePages defines a set of huge pages that we want to allocate at boot.

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| defaultHugepagesSize | DefaultHugePagesSize defines huge pages default size under kernel boot parameters. | *[HugePageSize](#hugepagesize) | false |
| pages | Pages defines huge pages that we want to allocate at boot time. | [][HugePage](#hugepage) | false |

[Back to TOC](#table-of-contents)

## NUMA

NUMA defines parameters related to topology awareness and affinity.

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| topologyPolicy | Name of the policy applied when TopologyManager is enabled Operator defaults to \"best-effort\" | *string | false |

[Back to TOC](#table-of-contents)

## PerformanceProfile

PerformanceProfile is the Schema for the performanceprofiles API.

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata |  | [metav1.ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.17/#objectmeta-v1-meta) | false |
| spec |  | [PerformanceProfileSpec](#performanceprofilespec) | false |
| status |  | [PerformanceProfileStatus](#performanceprofilestatus) | false |

[Back to TOC](#table-of-contents)

## PerformanceProfileList

PerformanceProfileList contains a list of PerformanceProfile.

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata |  | [metav1.ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.17/#listmeta-v1-meta) | false |
| items |  | [][PerformanceProfile](#performanceprofile) | true |

[Back to TOC](#table-of-contents)

## PerformanceProfileSpec

PerformanceProfileSpec defines the desired state of PerformanceProfile.

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| cpu | CPU defines a set of CPU related parameters. | *[CPU](#cpu) | false |
| hugepages | HugePages defines a set of huge pages related parameters. | *[HugePages](#hugepages) | false |
| machineConfigLabel | MachineConfigLabel defines the label to add to the MachineConfigs the operator creates. It has to be used in the MachineConfigSelector of the MachineConfigPool which targets this performance profile. Defaults to \"machineconfiguration.openshift.io/role=&lt;same role as in NodeSelector label key&gt;\" | map[string]string | false |
| machineConfigPoolSelector | MachineConfigPoolSelector defines the MachineConfigPool label to use in the MachineConfigPoolSelector of resources like KubeletConfigs created by the operator. Defaults to \"machineconfiguration.openshift.io/role=&lt;same role as in NodeSelector label key&gt;\" | map[string]string | false |
| nodeSelector | NodeSelector defines the Node label to use in the NodeSelectors of resources like Tuned created by the operator. It most likely should, but does not have to match the node label in the NodeSelector of the MachineConfigPool which targets this performance profile. | map[string]string | false |
| realTimeKernel | RealTimeKernel defines a set of real time kernel related parameters. RT kernel won't be installed when not set. | *[RealTimeKernel](#realtimekernel) | false |
| additionalKernelArgs | Addional kernel arguments. | []string | false |
| numa | NUMA defines options related to topology aware affinities | *[NUMA](#numa) | false |

[Back to TOC](#table-of-contents)

## PerformanceProfileStatus

PerformanceProfileStatus defines the observed state of PerformanceProfile.

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| conditions | conditions represents the latest available observations of current state. | []conditionsv1.Condition | false |
| tuned | tuned points to the Tuned custom resource object that contains the tuning values generated by this operator. | *string | false |

[Back to TOC](#table-of-contents)

## RealTimeKernel

RealTimeKernel defines the set of parameters relevant for the real time kernel.

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| enabled | Enabled defines if the real time kernel packages should be installed. Defaults to \"false\" | *bool | false |

[Back to TOC](#table-of-contents)
