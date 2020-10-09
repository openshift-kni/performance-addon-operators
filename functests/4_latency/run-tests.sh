#!/bin/bash

set -e
# Setting -e is fine as we want both config to succeed
# before running the "real" tests.

delay=${LATENCY_TEST_DELAY:-0}

echo "waiting ${delay}s before to run the tests"
sleep ${delay}s

suites=(0_config 4_latency)

for suite in "${suites[@]}"; do
    echo running "/${suite}.test" "$@"
    "./${suite}.test" "$@"
done
