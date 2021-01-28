# Performance Profile Creator (PPC)

A tool to automate the process of creating Performance Profile using the user supplied profile parameters.

## Software Components

1. A CLI tool part of the PAO image
1. A wrapper script <!--(TODO: Add link to where the wrapper script is placed) --> automates the access to PPC tool. It pulls the PAO image, supplies arguments and runs the tool inside Performance Addon Operator container with performance-profile-creator entrypoint.

## Flow
1. PPC consumes a must-gather output.
1. PPC may run must-gather directly if the must-gather output is not given.
1. PPC output is a bunch of YAML data (PAO profile + NTO tuned part).

<!--
## Wrapper Setup/Configuration
 Add Steps to execute the wrapper script
-->

## Building Performance Profile Creator binary and image

NOTE: Performance Profile Creator is present as an entrypoint (in /usr/local/bin/performance-profile-creator) in the Performance Addon Operator image.

Developers can build the Performance Profile Creator images from the source tree using make targets.
 1. Setup Environment variables
    ```bash
    export REGISTRY_NAMESPACE=<your quay.io namespace>
    export IMAGE_TAG=<the image tag to use> #defaults to "latest"
    export IMAGE_BUILD_CMD=podman
    ```
1. To build from Performance Profile Creator source:
   ```bash
   make create-performance-profile
   ```
1. To build the Performance addon Operator image with  from source:
   ```bash
   make operator-container
   ```
Alternatively, you can pull the latest master upstream image.  In the following examples, TAG has the format major.minor-snapshot. For example, the TAG for OpenShift 4.8 will be 4.8-snapshot:

```bash
podman pull quay.io/repository/openshift-kni/performance-addon-operator:<TAG>
```

## Running Performance Profile Creator

To run the Performance profile Creator run:

```bash
podman run --entrypoint performance-profile-creator -v ./must-gather:/must-gather:z  \
quay.io/repository/openshift-kni/performance-addon-operator:<TAG> > my-profile.yaml
```
