# Disable IRQ load balancing globally or on-demand

<!-- toc -->
- [Summary](#summary)
- [Goals](#goals)
- [Design Details](#design-details)
<!-- /toc -->

## Summary

In order to provide low-latency, exclusive CPU usage for guaranteed pods, the Performance Addons Operator can apply tuning
to remove all non-reserved CPUs from being eligible CPUs for processing device interrupts. 
Sometimes the reserved CPUs are not enough to handle networking device interrupts, in this case some isolated CPUs
will be needed to join the effort. It is done is by annotating the pods that have stricter RT requirements, removing
only the annotated pod's CPUs from the list of the CPUs that are allowed to handle device interrupts.

### Goals

- Provide a way to enable/disable device interrupts globally
- Allow disabling device interrupts only for specific pods when the device interrupts are not disabled globally
- Keep the existing behaviour for existing deployments (The device interrupts are disabled globally)

## Design Details

The Performance Profile CRD is promoted to 'v2', having a new optional boolean field ```GloballyDisableIrqLoadBalancing```
with default value ```false```. The Performance Addon Operator disables device interrupts on all isolated CPUs only
when ```GloballyDisableIrqLoadBalancing``` is set to ```true```.

Existing Performance Profile CRs with API versions 'v1' or 'v1alpha1' are converted to 'v2' using a Conversion Webhook
that injects the ```GloballyDisableIrqLoadBalancing``` field with the value ```true```.

When ```GloballyDisableIrqLoadBalancing``` is ```false```, the functionality to disable device interrupts on pod CPUs
it is implemented on the CRI-O level based on
- the pod using ***performance-<profile_name>*** runtime class
- the pod having ***irq-load-balancing.crio.io: true*** annotation
- the pod having ***cpu-quota.crio.io: true*** annotation

The Performance Addons Operator will be responsible for the creation of the high-performance runtime handler config snippet,
it will have the same content as default runtime handler, under relevant nodes, 
and for creation of the high-performance runtime class under the cluster.

A user will be responsible for specifying the relevant runtime class and annotation under the pod.

To disable device interrupts on pod CPUs, the pod specification will need to include the following fields:

```yaml
apiVersion: v1
kind: Pod
metadata:
  ...
  annotations:
    ...
    irq-load-balancing.crio.io: "true"
    cpu-quota.crio.io: "true"
    ...
  ... 
spec:
  ... 
  runtimeClassName: performance-<profile_name>
  ...
```