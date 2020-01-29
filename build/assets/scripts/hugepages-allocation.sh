#!/usr/bin/env bash

set -euo pipefail

nodes_path="/sys/devices/system/node"
hugepages_file="${nodes_path}/node${NUMA_NODE}/hugepages/hugepages-${HUGEPAGES_SIZE}kB/nr_hugepages"

if [ ! -f  ${hugepages_file} ]; then
    echo "ERROR: ${hugepages_file} does not exist"
    exit 1
fi

echo ${HUGEPAGES_COUNT} > ${hugepages_file}
