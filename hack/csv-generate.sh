#!/bin/bash

set -e

GOROOT=$(go env GOROOT)
export GOROOT

PREV="4.4.0"
LATEST="4.5.4"
LATEST_CHANNEL="4.5"

IS_DEV=$([[ $1 == "-dev" ]] && echo true || echo false)

if [[ -z "$CSV_VERSION" ]]; then
  CSV_VERSION=$LATEST
fi

if [[ -z "$CSV_CHANNEL" ]]; then
  CSV_CHANNEL=$LATEST_CHANNEL
fi

PACKAGE_NAME="performance-addon-operator"

PACKAGE_DIR="deploy/olm-catalog/${PACKAGE_NAME}"
CSV_DIR="${PACKAGE_DIR}/${CSV_VERSION}"
CSV_FILE="${CSV_DIR}/${PACKAGE_NAME}.v${CSV_VERSION}.clusterserviceversion.yaml"

OUT_ROOT="build/_output"
OUT_DIR="${OUT_ROOT}/olm-catalog"
OUT_CSV_DIR="${OUT_DIR}/${PACKAGE_NAME}/${CSV_VERSION}"
OUT_CSV_FILE="${OUT_CSV_DIR}/${PACKAGE_NAME}.v${CSV_VERSION}.clusterserviceversion.yaml"

EXTRA_ANNOTATIONS=""
MAINTAINERS=""

if [ -n "$MAINTAINERS_FILE" ]; then
  MAINTAINERS="-maintainers-from=$MAINTAINERS_FILE"
fi

if [ -n "$ANNOTATIONS_FILE" ]; then
  EXTRA_ANNOTATIONS="-annotations-from=$ANNOTATIONS_FILE"
fi

clean_package() {
  mkdir -p "$CSV_DIR"
  rm -rf "$OUT_DIR"
  mkdir -p "$OUT_CSV_DIR"
}

if ! [[ "$CSV_VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "CSV_VERSION not provided or does not match semver format"
  exit 1
fi

# clean up all old data first
clean_package

# do not generate new CRD/CSV for old versions
if [[ "$CSV_VERSION" != "$PREV" ]]; then
  $OPERATOR_SDK generate crds

  # generate a temporary csv we'll use as a template
  $OPERATOR_SDK generate csv \
    --operator-name="${PACKAGE_NAME}" \
    --csv-version="${CSV_VERSION}" \
    --csv-channel="${CSV_CHANNEL}" \
    --default-channel=true \
    --update-crds

  # using the generated CSV, create the real CSV by injecting all the right data into it
  build/_output/bin/csv-generator \
    --csv-version "${CSV_VERSION}" \
    --operator-csv-template-file "${CSV_FILE}" \
    --operator-image "${FULL_OPERATOR_IMAGE}" \
    --olm-bundle-directory "${OUT_CSV_DIR}" \
    --skip-range "${CSV_SKIP_RANGE}" \
    "${MAINTAINERS}" \
    "${EXTRA_ANNOTATIONS}"
fi

# copy remaining manifests to final location	if [[ "$IS_DEV" == true ]]; then
cp -a --no-clobber ${PACKAGE_DIR}/ ${OUT_DIR}/

if [[ "$IS_DEV" == true ]]; then
  # copy generated CSV and CRD back to repository dir
  cp "${OUT_CSV_FILE}" "${CSV_FILE}"
fi

echo "New OLM manifests created at ${OUT_DIR}"
