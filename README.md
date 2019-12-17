# Performance Addon Operators

This repository contains operators related to optimizing OpenShift clusters for applications sensitive to cpu and network latency.

## Performance Operator

For now there is 1 operator, the "Performance Operator". It was created using the operator-sdk v0.13.0:

`$ operator-sdk new performance-operator --repo github.com/openshift-kni/performance-addon-operators --vendor=true`

It will offer several APIs:

### CPUPerformanceProfile

The `CPUPerformanceProfile` offers high level options for applying various performance tunings. 
The API and its controller was created with:

```
$ operator-sdk add api --api-version=performance.openshift.io/v1alpha1 --kind=CPUPerformanceProfile
$ operator-sdk add controller --api-version=performance.openshift.io/v1alpha1 --kind=CPUPerformanceProfile
```

TODO: further implement and explain this API

### more APIs to come

# Testing

## Unit tests

TODO

## Func tests

The functional tests are located in `/functests`. They can be executed with `make functests`. 

# How to contribute

See [CONTRIBUTING](CONTRIBUTING.md) for some guidelines.