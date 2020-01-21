#!/bin/bash

set -e

# expect oc to be in PATH by default
OC_TOOL="${OC_TOOL:-oc}"

success=0
iterations=0
sleep_time=10
max_iterations=90 # results in 15 minute timeout

# TODO the worker-rt MCP is paused to prevent https://bugzilla.redhat.com/show_bug.cgi?id=1792749 from happening
# Let's gibe the operator some time to do its work before we unpause the MCP (see below)
echo "[INFO] Waiting 5 min for letting the operator do its work"
sleep 300

until [[ $success -eq 1 ]] || [[ $iterations -eq $max_iterations ]]
do

  # See cooment above
  echo "[INFO] Unpausing  MCPs"
  mcps=$(${OC_TOOL} get mcp --no-headers -o custom-columns=":metadata.name")
  for mcp in $mcps
  do
      ${OC_TOOL} patch mcp "${mcp}" -p '{"spec":{"paused":false}}' --type=merge
  done

  echo "[INFO] Checking if MCP picked up the performance MC"
  # No output means that the new machine config wasn't picked by MCO yet
  if [ -z "$(${OC_TOOL} get mcp worker-rt -o jsonpath='{.spec.configuration.source[?(@.name=="performance-ci")].name}')" ]
  then
    iterations=$((iterations + 1))
    iterations_left=$((max_iterations - iterations))
    echo "[INFO] Performace MC not picked up yet."
    sleep $sleep_time
    continue
  fi

  echo "[INFO] Checking if MCP is updated"
  if ! ${OC_TOOL} wait mcp/worker-rt --for condition=updated --timeout 1s
  then
    iterations=$((iterations + 1))
    iterations_left=$((max_iterations - iterations))
    echo "[INFO] Performace MCP not updated yet."
    sleep $sleep_time
    continue
  fi

  success=1

done

if [[ $success -eq 0 ]]; then
  echo "[ERROR] MCP failed, giving up."
  exit 1
fi

echo "[INFO] MCP update successful."
