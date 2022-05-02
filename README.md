**Staring from 4.11 and above, the following project components have been deprecated and moved under [Node Tuning Operator](https://github.com/openshift/cluster-node-tuning-operator):**
- Performance Addon Operator, now [Performance Addon controller](https://github.com/openshift/cluster-node-tuning-operator/blob/master/docs/performanceprofile/performance_addon.md), a day-0 controller deployed with Node Tuning Operator.
- [Performance profile creator](https://github.com/openshift/cluster-node-tuning-operator/blob/master/cmd/performance-profile-creator/README.md).
- Render mode.

**Must Gather is still maintained under this project.**\
**Versions 4.10 and below are still supported in bugfix mode.**

# Performance Addon Operator

[![Drone.io Status](https://cloud.drone.io/api/badges/openshift-kni/performance-addon-operators/status.svg?ref=refs/heads/master)](https://cloud.drone.io/openshift-kni/performance-addon-operators/)
[![Coverage Status](https://coveralls.io/repos/github/openshift-kni/performance-addon-operators/badge.svg?branch=master)](https://coveralls.io/github/openshift-kni/performance-addon-operators?branch=master)

The `Performance Operator` optimizes OpenShift clusters for applications sensitive to cpu and network latency.

![alt text](https://github.com/openshift-kni/performance-addon-operators/blob/master/docs/interactions/diagram.png "How Performance Addon Operator interacts with other components and operators")

The operator was created using the operator-sdk:

`$ operator-sdk new performance-operator --repo github.com/openshift-kni/performance-addon-operators --vendor=true`

## PerformanceProfile

The `PerformanceProfile` CRD is the API of the performance operator and offers high level options
for applying various performance tunings to cluster nodes. The API and its controller were created with:

```bash
operator-sdk add api --api-version=performance.openshift.io/v1alpha1 --kind=PerformanceProfile
operator-sdk add controller --api-version=performance.openshift.io/v1alpha1 --kind=PerformanceProfile
```

The performance profile API is documented in detail in the [Performance Profile](docs/performance_profile.md) doc.
Follow the [API versions](docs/api-versions.md) doc to check the supported API versions.

# Building and pushing the operator images

Developers can build and push the Performance Operator images from the source tree using make targets.

```
export REGISTRY_NAMESPACE=<your quay.io namespace>
export IMAGE_TAG=<the image tag to use> #defaults to "latest"
make build-containers
make push-containers
```

The building of the index image requires that the bundle image will be public available under the image registry,
otherwise the creation of the index image will fail.

# Building and pushing z-stream release

It is number of ENV variables that you should define to get all components to have the right z-stream version `4.y.z`
and allow the seamless upgrade.

```
export CSV_CHANNEL="4.y"
export CSV_VERSION="4.y.z"
export CSV_FROM_VERSION="4.y.z-1"
export IMAGE_TAG=4.x.y
export OPERATOR_VERSION=4.y.z
export CSV_SKIP_RANGE=">=4.y-1.0 <4.y.z
make build-containers
make push-containers
```

# Deploying

If you use your own images, make sure they are made public in your quay.io account!

If you want to use the performance operator's upstream images,
- unset `REGISTRY_NAMESPACE` (it will default to `openshift-kni`)
- if you deploy on OCP 4.4, run `export IMAGE_TAG=v4.4`
- if you deploy on OCP 4.5, unset `IMAGE_TAG` (it will default to `latest`)

Deploy the operator by running:

```
CLUSTER=manual make cluster-deploy
```

This will deploy

- a `MachineConfigPool` for the nodes which will be tuned
- all manifests for letting OCP's `Operator Lifecycle Manager (OLM)` deploy the Performance Operator:
  - a `CatalogSource`
  - a `Namespace`
  - a `OperatorGroup`
  - a `Subscription`
- a `PerformanceProfile`

Note: The performance operator supports only `AllNamespaces` InstallModeType for `OperatorGroup`.

The deployment will be retried in a loop until everything is deployed successfully, or until it times out.

> Note: `CLUSTER=manual` lets the deploy script use the `cluster-setup/manual-cluster/performance/` kustomization directory.
In CI the `cluster-setup/ci-cluster/performance/` dir will be used. The difference is that the CI cluster will deploy
the PerformanceProfile in the test code, while the `manual` cluster includes it in the kustomize based deployment.


Now you need to label the nodes which should be tuned. This can be done with

```
make cluster-label-worker-cnf
```

This will label 1 worker node with the `worker-cnf` role, and OCP's `Machine Config Operator` will start tuning this node.

In order to wait until MCO is ready, you can watch the `MachineConfigPool` until it is marked as updated with 

```
CLUSTER=manual make cluster-wait-for-mcp
```

> Note: Be aware this can take quite a while (many minutes)

> Note: in CI this step is skipped, because the test code will wait for the MCP being up to date.

# Render mode

The operator can render manifests for all the components it supposes to create, based on Given a `PerformanceProfile`  

You need to provide the following environment variables
```
export PERFORMANCE_PROFILE_INPUT_FILES=<your PerformanceProfile directory path>
export ASSET_OUTPUT_DIR=<output path for the rendered manifests>
```

Build and invoke the binary
```
build/_output/bin/performance-addon-operators render
```

Or provide the variables via command line arguments
```
build/_output/bin/performance-addon-operators render --performance-profile-input-files <path> --asset-output-dir<path>
```

# Troubleshooting

When the deployment fails, or the performance tuning does not work as expected, follow the [Troubleshooting Guide](docs/troubleshooting.md)
for debugging the cluster. Please provide as much info from troubleshooting as possible when reporting issues. Thanks!

# Testing

## Unit tests

Unit tests can be executed with `make unittests`.

## Func tests

The functional tests are located in `/functests`. They can be executed with `make functests-only` on a cluster with a
deployed Performance Operator and configured MCP and nodes. It will create its own Performance profile!

### Latency test

The latency-test container image gives the possibility to run the latency 
test without need to install go, ginkgo or other go related modules.

The test itself is running the `oslat` `cyclictest` and `hwlatdetect` binaries and verifies if the maximal latency returned by each one of the tools is
less than specified value under the `MAXIMUM_LATENCY`.

To run the latency test inside the container:

```
docker run --rm -v /kubeconfig:/kubeconfig \
-e KUBECONFIG=/kubeconfig \
-e LATENCY_TEST_RUN=true \
-e LATENCY_TEST_RUNTIME=60 \
-e MAXIMUM_LATENCY=700 \
 quay.io/openshift-kni/cnf-tests /usr/bin/run-tests.sh
```

You can run the container with different ENV variables, but the bare minimum is to pass
`KUBECONFIG` mount and ENV variable, to give to the test access to the cluster and
`LATENCY_TEST_RUN=true` to run the latency test.

- `LATENCY_TEST_DELAY` indicates an (optional) delay in seconds to be used between the container is created and the tests actually start. Default is zero (start immediately).
- `LATENCY_TEST_RUN` indicates if the latency test should run.
- `LATENCY_TEST_RUNTIME` the amount of time in seconds that the latency test should run.
- `LATENCY_TEST_IMAGE` the image that used under the latency test.
- `LATECNY_TEST_CPUS` the amount of CPUs the pod which run the latency test should request
- `OSLAT_MAXIMUM_LATENCY` the expected maximum latency for all buckets in us in the oslat test.
- `CYCLICTEST_MAXIMUM_LATENCY` the expected maximum latency for the cyclictest test.
- `HWLATDETECT_MAXIMUM_LATENCY` the expected maximum latency for the hwlatdetect test.
- `MAXIMUM_LATENCY` a unified value for the expected maximum latency for all tests (In case both provided, the specific variables will have precedence over the unified one).

# Contributing

See [CONTRIBUTING](CONTRIBUTING.md) for some guidelines.

# Building a custom CSV

A custom CSV entry for the registry container can be generated using the
`make generate-csv` make target.

First export your CSV details as environment variables.

__required options__

    export IMAGE_REGISTRY="quay.io"
    export REGISTRY_NAMESPACE="some-operator=repo-namespace"
    export IMAGE_TAG="some-operator-image-tag"
    export CSV_VERSION="0.0.3"

__optional options__

    export REPLACES_CSV_VERSION="0.0.2"
    export CSV_SKIP_RANGE=">=0.0.1 <0.0.2"

Then run `make generate-csv`

The result will be stored in the `deploy/olm-catalog/performance-addon-operator`
directory within a directory that matches the `CSV_VERSION` set.

Running `make registry-container` after creating a new custom CSV will result
in a registry bundle that includes the new CSV version and all other CSV
versions in the `deploy/olm-catalog` directory.
