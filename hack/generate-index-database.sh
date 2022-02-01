#!/usr/bin/env bash

set -e

# shellcheck source=common.sh
source "$(dirname "$0")/common.sh"

BUNDLES=${BUNDLES:-"quay.io/openshift-kni/performance-addon-operator-bundle:4.11-snapshot"}
OPM_BUILDER_TAG="v1.14.3"

stop_containers() {
  if [ -n "${opm_container_id}" ]; then
    ${IMAGE_BUILD_CMD} stop "${opm_container_id}"
    ${IMAGE_BUILD_CMD} rm "${opm_container_id}"
  fi
}

trap stop_containers EXIT SIGINT SIGTERM

opm_container_id=$(${IMAGE_BUILD_CMD} run -d \
-e BUNDLES="${BUNDLES}" \
quay.io/operator-framework/upstream-opm-builder:${OPM_BUILDER_TAG} \
/bin/sh -c "opm index add --mode semver --bundles ${BUNDLES} --generate")

${IMAGE_BUILD_CMD} wait "${opm_container_id}"
${IMAGE_BUILD_CMD} logs "${opm_container_id}"

rm -rf "${OUT_DIR}/index.Dockerfile"
rm -rf "${OUT_DIR}/database"

${IMAGE_BUILD_CMD} cp "${opm_container_id}:/index.Dockerfile" "${OUT_DIR}"
${IMAGE_BUILD_CMD} cp "${opm_container_id}:/database" "${OUT_DIR}"
