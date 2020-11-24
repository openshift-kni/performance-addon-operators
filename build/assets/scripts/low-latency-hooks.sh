#!/usr/bin/env bash

mask="${1}"
[ -n "${mask}" ] || { logger "${0}: The rps-mask parameter is missing" ; exit 0; }

pid=$(jq '.pid' /dev/stdin 2>&1)
[[ $? -eq 0 && -n "${pid}" ]] || { logger "${0}: Failed to extract the pid: ${pid}"; exit 0; }

ns=$(ip netns identify "${pid}" 2>&1)
[[ $? -eq 0 && -n "${ns}" ]] || { logger "${0} Failed to identify the namespace: ${ns}"; exit 0; }

mode=$(ip netns exec "${ns}" [ -w /sys ] && echo "rw" || echo "ro" 2>&1)
[ $? -eq 0 ] || { logger "${0} Failed to determine if the /sys is writable: ${mode}"; exit 0; }

if [ "${mode}" = "ro" ]; then
    res=$(ip netns exec "${ns}" mount -o remount,rw /sys 2>&1)
    [ $? -eq 0 ] || { logger "${0}: Failed to remount /sys as rw: ${res}"; exit 0; }
fi

# /sys/class/net can't be used recursively to find the rps_cpus file, use /sys/devices instead
res=$(ip netns exec "${ns}" find /sys/devices -type f -name rps_cpus -exec sh -c "echo ${mask} | cat > {}" \; 2>&1)
[[ $? -eq 0 && -z "${res}" ]] || logger "${0}: Failed to apply the RPS mask: ${res}"

if [ "${mode}" = "ro" ]; then
    ip netns exec "${ns}" mount -o remount,ro /sys
    [ $? -eq 0 ] || exit 1 # Error out so the pod will not start with a writable /sys
fi
