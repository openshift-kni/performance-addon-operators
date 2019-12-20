#!/bin/bash

# expect oc to be in PATH by default
OC_TOOL="${OC_TOOL:-oc}"

$OC_TOOL delete -f - <<EOF
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: performance-addon-operators-subscription
  namespace: openshift-performance-addon
spec:
  channel: alpha
  name: performance-addon-operators
  source: performance-addon-operators-catalogsource
  sourceNamespace: openshift-marketplace
EOF

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
