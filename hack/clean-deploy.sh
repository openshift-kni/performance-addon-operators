#!/bin/bash

# expect oc to be in PATH by default
OC_TOOL="${OC_TOOL:-oc}"

# Remove node label
echo "[INFO]: Unlabeling worker nodes"
nodes=$(${OC_TOOL} get nodes --selector='node-role.kubernetes.io/worker-rt' -o name)
for node in $nodes
do
    ${OC_TOOL} label $node node-role.kubernetes.io/worker-rt-
done

# Give MCO some time to notice change
sleep 10

# Wait for worker MCP being updated
success=0
iterations=0
sleep_time=10
max_iterations=90 # results in 15 minute timeout
until [[ $success -eq 1 ]] || [[ $iterations -eq $max_iterations ]]
do
  echo "[INFO] Checking if MCP is updated"
  if ! ${OC_TOOL} wait mcp/worker --for condition=updated --timeout 1s
  then
    iterations=$((iterations + 1))
    iterations_left=$((max_iterations - iterations))
    echo "[INFO] MCP not updated yet. $iterations_left retries left."
    sleep $sleep_time
    continue
  fi

  success=1

done

if [[ $success -eq 0 ]]; then
  echo "[ERROR] MCP update failed, going on nonetheless."
fi

# Delete CRs: this will undeploy all the MCs etc. (once it is implemented)
echo "[INFO] Deleting PerformanceProfile and giving the operator some time to undeploy everything"
$OC_TOOL delete performanceprofile --all
sleep 30

# Delete subscription: this will undeploy the operator and delete CRDs
echo "[INFO] Deleting Subscription and giving OLM some time to undeploy the operator and CRDs"
$OC_TOOL -n openshift-performance-addon delete subscription performance-addon-operator-subscription
sleep 10

# Delete operatorgroup and catalogsource
echo "[INFO] Deleting OperatorGroup and CatalogSource"
$OC_TOOL -n openshift-performance-addon delete operatorgroup openshift-performance-addon-operatorgroup
$OC_TOOL -n openshift-marketplace delete catalogsource performance-addon-operator-catalogsource

# Delete worker-rt MCP
echo "[INFO] Deleting worker-rt MCP"
$OC_TOOL delete mcp worker-rt

# Delete ns
echo "[INFO] Deleting Namespace"
$OC_TOOL delete ns openshift-performance-addon --force --grace-period 0