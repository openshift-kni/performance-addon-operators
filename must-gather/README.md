About collecting container-native performance-related data
==========================================================

You can use the `oc adm must-gather` CLI command to collect information about your cluster, including features and objects associated with the areas managed by performance-addon-operator.

In addition to the performance-operator logs and manifests, the command will collect basic information about the hardware topology of the worker nodes.
This information is useful to understand and verify resource allocation to pods, whose proper alignment is critical for low-latency workloads.

To collect the container-native performance-related data with must-gather, you must specify the extra image using the `--image` option.
In the following examples, `TAG` has the format `major.minor-snapshot`. For example, the TAG for OpenShift 4.12 will be `4.12-snapshot`.

Example command line:
```bash
oc adm must-gather --image=quay.io/openshift-kni/performance-addon-operator-must-gather:$TAG
```

To collect the cluster-related data and *additionally* the performance-related data, you can use multiple `--image` options like in this example:
```bash
oc adm must-gather --image=quay.io/openshift/origin-must-gather --image=quay.io/openshift-kni/performance-addon-operator-must-gather:$TAG
```

