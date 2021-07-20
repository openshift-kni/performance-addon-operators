#!/usr/bin/env bash

set -e

# shellcheck source=common.sh
source "$(dirname "$0")/common.sh"

# create the metadata annotation folder if needed
if [ ! -d "${METADATA_DIR}/${CSV_VERSION}" ]; then
  mkdir -p "${METADATA_DIR}/${CSV_VERSION}"
fi

cat >"${METADATA_DIR}/${CSV_VERSION}/annotations.yaml" <<EOF
annotations:
  com.redhat.openshift.versions: "v${CSV_CHANNEL}"
  operators.operatorframework.io.bundle.mediatype.v1: "registry+v1"
  operators.operatorframework.io.bundle.manifests.v1: "manifests/"
  operators.operatorframework.io.bundle.metadata.v1: "metadata/"
  operators.operatorframework.io.bundle.package.v1: "performance-addon-operator"
  operators.operatorframework.io.bundle.channels.v1: "${CSV_CHANNEL}"
  operators.operatorframework.io.bundle.channel.default.v1: "${CSV_CHANNEL}"
EOF
