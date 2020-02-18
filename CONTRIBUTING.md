# How to Contribute

This project is [Apache 2.0 licensed](LICENSE) and accepts contributions via
GitHub pull requests.

## Contributing Guidelines

* Unit tests are required to accompany all logic being introduced into the `pkg/` directory.
* Functional tests are required to accompany any logic that modifies `pkg/apis/` API structs.
* Follow the tide review flow for merging commits. This flow forces a review and all required CI jobs to pass before a PR is merged.
* Read and understand the design principles outlined in this document that pertain to the code you are contributing to.

## Certificate of Origin

By contributing to this project you agree to the Developer Certificate of
Origin (DCO). This document was created by the Linux Kernel community and is a
simple statement that you, as a contributor, have the legal right to make the
contribution. See the [DCO](DCO) file for details.

# Reconcile Loop Design Principles

* Never block reconcile execution. ever. This means __no sleeps and no retry loops__
* All __logic must be idempotent__. This means discovering what has already occurred by inspecting informer cache, and only mutating to converge on desired state.
* __Never perform an apiserver List()__ request within the reconcile loop. This action is very apiserver intensive. Instead use informers for retrieving items from cache.
* __Avoid mutating a cluster object the loop didn't create__. A exception to this would to be adding annotations to an existing object. Mutating a object's Spec that our controller didn't create should be avoided.
* __Avoid performing an apiserver Get()__ request within the reconcile loop if at all possible. Again use informers. There are rare exceptions.
* Use __finalizers__ in order to perform any cleanup tasks required before a CR is completely removed from etcd.
* Use __owner references__ on any objects created and owned by a CR. Owner references ensure objects are automatically garbage collected after a corresponding CR is deleted.
* Return early to handle errors which re-enqueues the work queue key to be tried again.
* Do not create logic to sync a key using a recurring time interval. Instead only wake up processing the key when something has changed.
* Reconcile execution should be broken into two parts, synchronization and status reporting. Synchronization converges the CR on the desired state. Status reporting records the current state of the CR based on the synchronization execution results and the observed state of the cluster. These are two independent functions. Keep them separated. __Do not modify status during the synchronization step__.

# API Design

* All workflows __must be declarative__. This means the cluster-admin must be able to declare the exact state they want up front, and have our operator handle converging on that state.
* __No imperative actions__. This means no requirement for a cluster-admin to account for the ordering or timing at which manifests are posted. If our API can't express the desired state up front, then the API is wrong.
* Use Status.Conditions and Events to represent and record transient states that a CR exists in.
* Avoid introducing Status.Phase enums if possible. The pitfall of a Phase enum is that it often gets treated as a FSM, when in reality that isn't always the case. [Source](https://github.com/kubernetes/kubernetes/issues/7856)

# Unit Testing

Unit testing allows us to validate our reconcile loops under specific conditions to ensure no unexpected changes in behavior are introduced.

* Use mock clients to simulate reconcile loops.
* Execute reconcile loop to validate execution under specific conditions.
* Validate reconcile execution by introspecting API calls that were made during the execution. [Example](https://github.com/operator-framework/operator-sdk-samples/blob/master/go/memcached-operator/pkg/controller/memcached/memcached_controller_test.go)

# Functional Testing

Functional testing allows us to exercise and validate our API on a live cluster.

* All functional tests should be written in golang+gomega and exist in the functtests/ directory
* Functional test scenarios should represent conditions cluster-admins will encounter.
* Exercise and validate behavior by posting/mutating/deleting manifests and observing the desired state is reached.
* As a general rule of thumb for test condition timeouts, take the time you typically observe an action taking and double it for the timeout value.

# Deploying Operator on Test Cluster

Developers can build and deploy the operator from the source tree using make targets.

```
export REGISTRY_NAMESPACE=<your quay.io namespace>
make build-containers
make push-containers
```

Make sure your images are made public in your quay.io account.

Then deploy the operator using this make target.

```
FEATURES_ENVIRONMENT=other-clusters make cluster-deploy
```

This simply posts the manifests for now. You can perform your own introspection into the cluster to determine if the deployment was successful,
or verify the deployment by running the functional tests.

## Manual introspection

Verify the install by checking the csv is posted.

```
$ oc get csv --all-namespaces | grep performance-addon
openshift-performance-addon            performance-addon-operator.v0.0.1   Performance Addon Operator   0.0.1                InstallReady
```

That the catalog container and the operator are running.

```
$ oc get pods --all-namespaces | grep performance
openshift-marketplace            performance-addon-operator-catalogsource-87bjk     1/1     Running   0     2m
openshift-performance-addon      performance-operator-6ff4977f8b-ljk42              1/1     Running   0    19s
```

TODO - revise this once deployment validation is included in `make cluster-deploy`

## Run functional tests

```
make functests
```


