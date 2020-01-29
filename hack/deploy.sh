#!/bin/bash

set -e

# expect oc to be in PATH by default
OC_TOOL="${OC_TOOL:-oc}"

# Override the image name when this is invoked from openshift ci
if [ -n "${OPENSHIFT_BUILD_NAMESPACE}" ]; then
  FULL_REGISTRY_IMAGE="registry.svc.ci.openshift.org/${OPENSHIFT_BUILD_NAMESPACE}/stable:performance-addon-operator-registry"
fi

echo "Deploying using image $FULL_REGISTRY_IMAGE."

# Deploy features
success=0
iterations=0
sleep_time=10
max_iterations=30 # results in 5 minute timeout
feature_dir=cluster-setup/ci-cluster/performance/

until [[ $success -eq 1 ]] || [[ $iterations -eq $max_iterations ]]
do

  echo "[INFO] Deploying performance operator and profile."
  set +e
  if ! ${OC_TOOL} kustomize $feature_dir | envsubst | ${OC_TOOL} apply -f -
  then
    set -e
    iterations=$((iterations + 1))
    iterations_left=$((max_iterations - iterations))
    echo "[WARN] Deployment failed, retrying in $sleep_time sec, $iterations_left retries left."
    sleep $sleep_time
    continue
  fi
  set -e

  success=1

done

if [[ $success -eq 0 ]]; then
  echo "[ERROR] Deployment failed, giving up."
  exit 1
fi

echo "[INFO] Deployment successful."
