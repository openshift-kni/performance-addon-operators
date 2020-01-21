# Performance Addon Operators

This repository contains operators related to optimizing OpenShift clusters for applications sensitive to cpu and network latency.

## Performance Operator

For now there is 1 operator, the "Performance Operator". It was created using the operator-sdk v0.13.0:

`$ operator-sdk new performance-operator --repo github.com/openshift-kni/performance-addon-operators --vendor=true`

It will offer several APIs:

### PerformanceProfile

The `PerformanceProfile` offers high level options for applying various performance tunings.
The API and its controller was created with:

```bash
operator-sdk add api --api-version=performance.openshift.io/v1alpha1 --kind=PerformanceProfile
operator-sdk add controller --api-version=performance.openshift.io/v1alpha1 --kind=PerformanceProfile
```

TODO: further implement and explain this API

### more APIs to come

## Testing

### Unit tests

Unit tests can be executed with `make unittests`.

### Func tests

The functional tests are located in `/functests`. They can be executed with `make functests`.

#### Running functests in CI

Openshift CI will run `make cluster-deploy functests`.  
The `cluster-deploy` target deploys the operator and a CR using `/hack/deploy.sh`, `kustomize` and manifests located
in `/cluster-setup`. It will detect that it is running in CI and deploy the images under test automatically.

#### Running functests on your own cluster

see [CONTRIBUTING](CONTRIBUTING.md#deploying-operator-on-test-cluster)

## How to contribute

See [CONTRIBUTING](CONTRIBUTING.md) for some guidelines.
