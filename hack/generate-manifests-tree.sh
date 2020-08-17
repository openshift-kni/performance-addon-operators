#!/bin/bash

set -e

if [ -z "${1}" ]; then
	echo "usage: $0 FULL_OPERATOR_IMAGE" 1>&2
	exit 1
fi

OPERATOR_IMAGE="${1}"
OUT_DIR="build/_output/manifests"
# this must have been created by 'make generate latest-dev-csv"
OLM_DIR="build/_output/olm-catalog"

if [ ! -d "${OLM_DIR}" ]; then
	echo "missing output directory ${OUT_DIR} run 'make generate-latest-dev-csv' before" 1>&2
	exit 1
fi

mv "${OLM_DIR}" "${OUT_DIR}"
find "${OUT_DIR}" -type f -exec sed -i "s|REPLACE_IMAGE|${OPERATOR_IMAGE}|g" {} \; || :
for entry in ${OUT_DIR}/performance-addon-operator/*; do
	version=$( basename $entry )
	if [ ! -d deploy/metadata/performance-addon-operator/$version ]; then
		continue
	fi
	mkdir -p $entry/manifests && mv $entry/*.yaml $entry/manifests
	mkdir -p $entry/metadata && cp deploy/metadata/performance-addon-operator/$version/* $entry/metadata
done
