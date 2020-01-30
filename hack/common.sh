#!/usr/bin/env bash

set -euo pipefail

REPO_DIR="$(
    cd "$(dirname "$BASH_SOURCE[0]")/../"
    pwd
)"
VENDOR_DIR=${REPO_DIR}/vendor
