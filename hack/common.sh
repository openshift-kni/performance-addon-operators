#!/usr/bin/env bash

REPO_DIR="$(
    cd "$(dirname "$BASH_SOURCE[0]")/../" || return
    pwd
)"
VENDOR_DIR="${REPO_DIR}/vendor"
OUT_DIR="${REPO_DIR}/build/_output"
OUT_BIN="${OUT_DIR}/bin"

IMAGE_BUILD_CMD=${IMAGE_BUILD_CMD:-podman}
