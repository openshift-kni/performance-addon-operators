# API versions

## Supported API Versions

The Performance Addon Operator supports *v1* and *v1alpha1* [Performance Profile](docs/performance_profile.md) versions.
There are no differences between the *v1* and *v1alpha1* versions, the upgrade has been made in order to
mark the Performance Profile API as stable.

## Upgrade from *v1alpha1* to *v1*.
When upgrading from an older Performance Addon Operator version supporting only *v1alpha1* API to a newer one supporting
also the *v1* API, the existing *v1alpha1* Performance Profiles will be converted on-the-fly using a "None" Conversion
strategy and served to the Performance Addon Operator as *v1*.

## Q&A
What happens in practice if I install a v1-enabled PAO on a cluster? What should I expect?
- PAO will expect v1 Performance Profiles and query only them. Existing v1alpha1 profiles will be served as v1 and
  PAO works with them as usual.

What happens if I submit a v1alpha1 profile, will I always get back a v1 profile?
- Any of the existent Performance Profile CRs can be retrieved as both v1 and v1alpha1 since the Performance Profile CRD
supports both API versions.
    For example if we have a "manual" Performance Profile in the system we can query it as both v1alpha1 and v1:

    ```oc get performanceprofiles.v1.performance.openshift.io manual```

    ```oc get performanceprofiles.v1alpha1.performance.openshift.io manual```

    However, the Performance Addon Operator will use all the existent Performance Profiles as v1 no matter
    if they have been submitted as v1alpha1 or v1.

Where can I find more information on API versioning?
- https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning

