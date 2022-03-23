#!/usr/bin/bash

set -euo pipefail

disable_cpu="0"

cpus="$(echo "${OFFLINE_CPUS}" | grep -o -E '[0-9]+')"

for cpu in $cpus;
  do
    online_cpu_file="/sys/devices/system/cpu/cpu$cpu/online"
    if [ ! -f "${online_cpu_file}" ]; then
      echo "ERROR: ${online_cpu_file} does not exist"
      exit 1
    fi
    echo "$disable_cpu" > "${online_cpu_file}"
    echo "offline cpu num $cpu"
  done