#!/usr/bin/env bash

set -e

# shellcheck source=common.sh
source "$(dirname "$0")/common.sh"

GOROOT=$(go env GOROOT)
export GOROOT

CSV_SKIP_RANGE=">=${PREV_CSV_VERSION} <${CSV_VERSION}"

IS_DEV=$([[ $1 == "-dev" ]] && echo true || echo false)

PACKAGE_NAME="performance-addon-operator"
PACKAGE_DIR="deploy/olm-catalog/${PACKAGE_NAME}"

CSV_DIR="${PACKAGE_DIR}/${CSV_VERSION}"

OUT_ROOT="build/_output"
OUT_DIR="${OUT_ROOT}/olm-catalog"
OUT_CSV_DIR="${OUT_DIR}/${PACKAGE_NAME}/${CSV_VERSION}"
OUT_CSV_FILE="${OUT_CSV_DIR}/${PACKAGE_NAME}.v${CSV_VERSION}.clusterserviceversion.yaml"

TEMPLATES_DIR="${OUT_ROOT}/templates"
CSV_TEMPLATE_FILE="${TEMPLATES_DIR}/${PACKAGE_NAME}.v${CSV_VERSION}.clusterserviceversion.yaml"
ICON_FILE="docs/pao-icon.svg"

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

  rm -rf "${TEMPLATES_DIR}"
  mkdir -p "${TEMPLATES_DIR}"
}

if ! [[ "$CSV_VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "CSV_VERSION not provided or does not match semver format"
  exit 1
fi

# clean up all old data first
clean_package

# do not generate new CRD/CSV for old versions
if [[ ${CSV_VERSION} =~ 4.6.* ]]; then
  cp -a deploy/olm-catalog build/_output

  # generate a temporary csv we'll use as a template
  $OPERATOR_SDK generate csv \
    --operator-name="${PACKAGE_NAME}" \
    --csv-version="${CSV_VERSION}" \
    --csv-channel="${CSV_CHANNEL}" \
    --default-channel=true \
    --update-crds \
    --make-manifests=false \
    --from-version="${CSV_FROM_VERSION}" \
    --output-dir="${OUT_ROOT}"

  # copy template CSV file to preserve it for our csv-generator
  mv "${OUT_CSV_FILE}" "${CSV_TEMPLATE_FILE}"

  # using the generated CSV, create the real CSV by injecting all the right data into it
  build/_output/bin/csv-generator \
    --csv-version "${CSV_VERSION}" \
    --operator-csv-template-file "${CSV_TEMPLATE_FILE}" \
    --operator-image "${FULL_OPERATOR_IMAGE}" \
    --olm-bundle-directory "${OUT_CSV_DIR}" \
    --replaces-csv-version "${REPLACES_CSV_VERSION}" \
    --skip-range "${CSV_SKIP_RANGE}" \
    --icon-from "${ICON_FILE}" \
    "${MAINTAINERS}" \
    "${EXTRA_ANNOTATIONS}"

  $OPERATOR_SDK generate crds
  cp -a deploy/crds/performance.openshift.io_performanceprofiles_crd.yaml "${OUT_CSV_DIR}"
fi

if [[ "$IS_DEV" == true ]]; then
  # copy generated CSV and CRD back to the repository dir
  cp "${OUT_CSV_DIR}"/* "${CSV_DIR}/"
fi

echo "New OLM manifests created at ${OUT_DIR}"
