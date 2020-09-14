#!/bin/bash

set -ex

SELF=$( realpath $0 )
BASEPATH=$( dirname $SELF )
RUNTIME=${IMAGE_BUILD_CMD:-podman}
VERSION=${OPERATOR_VERSION:-4.6.0}

rm -rf ${BASEPATH}/../build/_output/database && \
mkdir -p ${BASEPATH}/../build/_output/database && \
${RUNTIME} run -v "${BASEPATH}/../build/_output":/sources:z quay.io/operator-framework/upstream-registry-builder /bin/initializer -m /sources/manifests/performance-addon-operator/"${VERSION}"/manifests -o /sources/database/index.db
