#!/bin/bash

set -e

if [ -z "${1}" ]; then
	echo "usage: $0 FULL_OPERATOR_IMAGE" 1>&2
	exit 1
fi

OPERATOR_IMAGE="${1}"
OUT_DIR="build/_output/manifests"

# we don't want any leftover from previous runs, so we just make sure we start from a clean slate.
rm -rf "${OUT_DIR}"
mkdir -p "${OUT_DIR}"
# this script should never change the master copies
cp -a deploy/olm-catalog/performance-addon-operator "${OUT_DIR}"
find "${OUT_DIR}" -type f -exec sed -i "s|REPLACE_IMAGE|${OPERATOR_IMAGE}|g" {} \; || :
