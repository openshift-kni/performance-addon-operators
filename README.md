# Performance Addon Operator

[![Drone.io Status](https://cloud.drone.io/api/badges/openshift-kni/performance-addon-operators/status.svg?ref=refs/heads/master)](https://cloud.drone.io/openshift-kni/performance-addon-operators/)
[![Coverage Status](https://coveralls.io/repos/github/openshift-kni/performance-addon-operators/badge.svg?branch=master)](https://coveralls.io/github/openshift-kni/performance-addon-operators?branch=master)

The `Performance Operator` optimizes OpenShift clusters for applications sensitive to cpu and network latency.

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

# Building and pushing the operator images

Developers can build and push the Performance Operator images from the source tree using make targets.

```
export REGISTRY_NAMESPACE=<your quay.io namespace>
export IMAGE_TAG=<the image tag to use> #defaults to "latest">
make generate-manifests-tree
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

# Troubleshooting

When the deployment fails, or the performance tuning does not work as expected, follow the [Troubleshooting Guide](docs/troubleshooting.md)
for debugging the cluster. Please provide as much info from troubleshooting as possible when reporting issues. Thanks!

# Testing

## Unit tests

Unit tests can be executed with `make unittests`.

## Func tests

The functional tests are located in `/functests`. They can be executed with `make functests-only` on a cluster with a
deployed Performance Operator and configured MCP and nodes. It will create its own Performance profile!

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
