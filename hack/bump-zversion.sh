#!/usr/bin/env bash

set -e

BASEDIR="$(dirname $0 )"

# shellcheck source=common.sh
source "$(dirname $0)/common.sh"

CSV_VERSION_A=( ${CSV_VERSION//./ } ) # replace points, split into array
ZVERSION_NOW="${CSV_VERSION_A[2]}"
ZVERSION_NEXT=$( expr ${ZVERSION_NOW} + 1)
CSV_VERSION_NEXT="${CSV_VERSION_A[0]}.${CSV_VERSION_A[1]}.${ZVERSION_NEXT}"

echo "${CSV_VERSION} -> ${CSV_VERSION_NEXT}"

# channel format MUST be "${MAJOR}.${MINOR}" so we can ignore the zversion bump and we can just copy the file.
mkdir -p "${BASEDIR}/../deploy/metadata/performance-addon-operator/${CSV_VERSION_NEXT}"
cp -a \
	"${BASEDIR}/../deploy/metadata/performance-addon-operator/${CSV_VERSION}/annotations.yaml" \
	"${BASEDIR}/../deploy/metadata/performance-addon-operator/${CSV_VERSION_NEXT}/annotations.yaml"

mkdir -p "${BASEDIR}/../deploy/olm-catalog/performance-addon-operator/${CSV_VERSION_NEXT}"
cp -a \
	"${BASEDIR}/../deploy/olm-catalog/performance-addon-operator/${CSV_VERSION}/performance.openshift.io_performanceprofiles_crd.yaml" \
	"${BASEDIR}/../deploy/olm-catalog/performance-addon-operator/${CSV_VERSION_NEXT}/performance.openshift.io_performanceprofiles_crd.yaml"
sed "s/${CSV_VERSION}/${CSV_VERSION_NEXT}/" \
	< "${BASEDIR}/../deploy/olm-catalog/performance-addon-operator/${CSV_VERSION}/performance-addon-operator.v${CSV_VERSION}.clusterserviceversion.yaml" \
	> "${BASEDIR}/../deploy/olm-catalog/performance-addon-operator/${CSV_VERSION_NEXT}/performance-addon-operator.v${CSV_VERSION_NEXT}.clusterserviceversion.yaml"

BUNDLE_CI_TMP=$(mktemp)
sed "s/ZVERSION=${ZVERSION_NOW}/ZVERSION=${ZVERSION_NEXT}/" \
	< "${BASEDIR}/../openshift-ci/Dockerfile.bundle.ci" \
	> "${BUNDLE_CI_TMP}" \
&& \
mv "${BUNDLE_CI_TMP}" "${BASEDIR}/../openshift-ci/Dockerfile.bundle.ci"

BUNDLE_UPSTREAM_DEV_TMP=$(mktemp)
sed "s/ZVERSION=${ZVERSION_NOW}/ZVERSION=${ZVERSION_NEXT}/" \
	< "${BASEDIR}/../openshift-ci/Dockerfile.bundle.upstream.dev" \
	> "${BUNDLE_UPSTREAM_DEV_TMP}" \
&& \
mv "${BUNDLE_UPSTREAM_DEV_TMP}" "${BASEDIR}/../openshift-ci/Dockerfile.bundle.upstream.dev"

# leave this for last
COMMON_TMP=$(mktemp)
sed "s/${CSV_VERSION}/${CSV_VERSION_NEXT}/" \
	< "${BASEDIR}/common.sh" \
	> "${COMMON_TMP}" \
&& \
mv "${COMMON_TMP}" "${BASEDIR}/common.sh"
