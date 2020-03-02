#!/bin/bash

set -e

export GOROOT=$(go env GOROOT)
TMP_CSV_VERSION="9.9.9"
TMP_CSV_DIR="deploy/olm-catalog/performance-addon-operator/$TMP_CSV_VERSION"
TMP_CSV_FILE="$TMP_CSV_DIR/performance-addon-operator.v${TMP_CSV_VERSION}.clusterserviceversion.yaml"
FINAL_CSV_DIR="deploy/olm-catalog/performance-addon-operator/$CSV_VERSION"
EXTRA_ANNOTATIONS=""

if [ -n "$ANNOTATIONS_FILE" ]; then
	EXTRA_ANNOTATIONS="-inject-annotations-from=$ANNOTATIONS_FILE"
fi

(cd tools/csv-generator/ && go build)

clean_tmp_csv() {
	rm -rf $TMP_CSV_DIR
}

if [ -z "$CSV_VERSION" ]; then
	echo "CSV_VERSION environment variable required to generate CSV"
fi

#clean up any stale data left from another run
clean_tmp_csv

# generate a temporary csv we'll use as a template
$OPERATOR_SDK olm-catalog gen-csv --operator-name="performance-addon-operator" --csv-version="${TMP_CSV_VERSION}"
$OPERATOR_SDK generate crds

# using the generated CSV, create the real CSV by injecting all the right data into it
tools/csv-generator/csv-generator \
	--csv-version "${CSV_VERSION}" \
	--operator-csv-template-file "${TMP_CSV_FILE}" \
	--operator-image "${FULL_OPERATOR_IMAGE}" \
	--olm-bundle-directory "$FINAL_CSV_DIR" \
	--replaces-csv-version "$REPLACES_CSV_VERSION" \
	--skip-range "$CSV_SKIP_RANGE" \
	"${EXTRA_ANNOTATIONS}"

cp deploy/crds/*_crd.yaml $FINAL_CSV_DIR/

clean_tmp_csv

echo "New CSV created at $FINAL_CSV_DIR"

if [ -n "$DEV_BUILD" ]; then
  SETUP_DIR=cluster-setup/ci-cluster/performance
  cp $FINAL_CSV_DIR/* ${SETUP_DIR}/
  # set correct namespace in CSV since this is going tpo be installed directly
  find ${SETUP_DIR}/ -type f -exec sed -i 's/namespace: placeholder/namespace: openshift-performance-addon/' {} \;
  # use REPLACE_IMAGE in var format, easier to use with kustomize + envsubst
  find ${SETUP_DIR}/ -type f -exec sed -i 's/REPLACE_IMAGE/${REPLACE_IMAGE}/g' {} \;
  echo "Dev CSV and CRDs copied into cluster-ci kustomize override dir"
fi

