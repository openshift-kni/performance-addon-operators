#!/bin/bash

set -e

# expect oc to be in PATH by default
OC_TOOL="${OC_TOOL:-oc}"

# Override the image name when this is invoked from openshift ci                               
if [ -n "$OPENSHIFT_BUILD_NAMESPACE" ]; then                                                   
        FULL_REGISTRY_IMAGE="registry.svc.ci.openshift.org/${OPENSHIFT_BUILD_NAMESPACE}/stable:performance-addon-operators-registry"
        echo "Openshift CI detected, deploying using image $FULL_REGISTRY_IMAGE"                   
fi   

$OC_TOOL apply -f - <<EOF
---
apiVersion: v1
kind: Namespace
metadata:
  labels:
    openshift.io/cluster-monitoring: "true"
  name: openshift-performance-addon
spec: {}
EOF

$OC_TOOL apply -f - <<EOF
---
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: openshift-performance-addon-operatorgroup
  namespace: openshift-performance-addon
spec:
  serviceAccount:
    metadata:
      creationTimestamp: null
  targetNamespaces:
  - openshift-performance-addon
EOF
  
$OC_TOOL apply -f - <<EOF
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

$OC_TOOL apply -f - <<EOF
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

