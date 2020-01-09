#!/bin/bash

set -e

# expect oc to be in PATH by default
OC_TOOL="${OC_TOOL:-oc}"

# Label worker nodes
echo "[INFO]:labeling worker nodes"
for node in $(${OC_TOOL} get nodes --selector='!node-role.kubernetes.io/master' -o name); do
    ${OC_TOOL} label $node node-role.kubernetes.io/worker-rt=""
done