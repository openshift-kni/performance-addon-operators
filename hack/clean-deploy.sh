#!/bin/bash

# expect oc to be in PATH by default
OC_TOOL="${OC_TOOL:-oc}"

$OC_TOOL delete performanceprofile --all

$OC_TOOL delete ns openshift-performance-addon

$OC_TOOL delete -f - <<EOF
---
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: performance-addon-operators-catalogsource
  namespace: openshift-marketplace
spec:
  displayName: Openshift Performance Addon Operators
  icon:
    base64data: ""
    mediatype: ""
  image: ${FULL_REGISTRY_IMAGE}
  publisher: Red Hat
  sourceType: grpc
EOF
