#!/bin/bash

set -e

# expect oc to be in PATH by default
OC_TOOL="${OC_TOOL:-oc}"

# Override the image name in the CSV when this is invoked from openshift ci
# See https://github.com/openshift/ci-tools/blob/master/TEMPLATES.md#image_format
# Fallback to string "REPLACE_IMAGE" what we use at other places as well
CI_OPERATOR_IMAGE=${IMAGE_FORMAT/'${component}'/performance-addon-operator}
export REPLACE_IMAGE=${CI_OPERATOR_IMAGE:-REPLACE_IMAGE}

if [ $FEATURES_ENVIRONMENT == "ci-cluster" ]; then
  echo "[INFO] Deployment method: CSV with image $REPLACE_IMAGE."
else
  echo "[INFO] Deployment method: CatalogSource with image $FULL_REGISTRY_IMAGE."
fi

# Deploy features
success=0
iterations=0
sleep_time=10
max_iterations=30 # results in 5 minute timeout
feature_dir=cluster-setup/$FEATURES_ENVIRONMENT/performance/

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
