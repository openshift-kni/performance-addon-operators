#!/usr/bin/env bash

set -ex

# shellcheck source=common.sh
source "$(dirname "$0")/common.sh"

BUNDLES=${BUNDLES:-"quay.io/openshift-kni/performance-addon-operator-bundle:4.7-snapshot"}
OPM_BUILDER_TAG="v1.14.3"

rm -rf "${OUT_DIR}/index.Dockerfile"
rm -rf "${OUT_DIR}/database"

container_id=$(${IMAGE_BUILD_CMD} run --rm -d \
-e BUNDLES="${BUNDLES}" \
quay.io/operator-framework/upstream-opm-builder:${OPM_BUILDER_TAG} \
/bin/sh -c "cd sources; opm index add --mode semver --bundles ${BUNDLES} --generate; sleep inf")

trap '{ ${IMAGE_BUILD_CMD} stop "${container_id}"; }' EXIT SIGINT SIGTERM

# it can take some time until the Dockerfile appears under the file system after generation
# and you will get Error: error evaluating symlinks ""
until ${IMAGE_BUILD_CMD} cp "${container_id}:/index.Dockerfile" "${OUT_DIR}"; do
  sleep 10
done

until ${IMAGE_BUILD_CMD} cp "${container_id}:/database" "${OUT_DIR}"; do
  sleep 10
done
