#!/usr/bin/env bash

set -exuo pipefail

source $(dirname "$0")/common.sh

APIS_PKG="github.com/openshift-kni/performance-addon-operators/pkg/apis"
APIS_VERSIONS="performance/v1alpha1"
CODE_GENERATORS_CMD_DIR=${VENDOR_DIR}/k8s.io/code-generator/cmd
OUTPUT_CLIENT_PKG="github.com/openshift-kni/performance-addon-operators/pkg/generated"

(
    # To support running this script from anywhere, we have to first cd into this directory
    # so we can install the tools.
    cd "$(dirname "${0}")"
    go install -mod=vendor ${CODE_GENERATORS_CMD_DIR}/client-gen
)

echo "Generating clientset for ${APIS_VERSIONS} at ${OUTPUT_CLIENT_PKG}/clientset"
client-gen --clientset-name ${CLIENTSET_NAME_VERSIONED:-versioned} --input-base ${APIS_PKG} --input ${APIS_VERSIONS} --output-package ${OUTPUT_CLIENT_PKG}/clientset --go-header-file ${REPO_DIR}/hack/boilerplate.go.txt
