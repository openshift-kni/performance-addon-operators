#!/usr/bin/env bash

REPO_DIR="$(
    cd "$(dirname "$BASH_SOURCE[0]")/../" || return
    pwd
)"
VENDOR_DIR="${REPO_DIR}/vendor"
OUT_DIR="${REPO_DIR}/build/_output"
OUT_BIN="${OUT_DIR}/bin"
METADATA_DIR="${REPO_DIR}/deploy/metadata/performance-addon-operator"

IMAGE_BUILD_CMD=${IMAGE_BUILD_CMD:-podman}

CSV_VERSION=${CSV_VERSION:-"4.11.0"}
PREV_CSV_VERSION=${PREV_CSV_VERSION:-"4.10.0"}
CSV_CHANNEL=${CSV_CHANNEL:-"4.11"}

MIN_KUBE_VERSION=${MIN_KUBE_VERSION:-"1.23.0"}

ANNOTATIONS_FILE="${REPO_DIR}/hack/annotations.json"
