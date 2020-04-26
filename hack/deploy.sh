#!/bin/bash

set -e

# expect oc to be in PATH by default
OC_TOOL="${OC_TOOL:-oc}"

# Override the image name in the CSV when this is invoked from openshift ci
# See https://github.com/openshift/ci-tools/blob/master/TEMPLATES.md#image_format
if [ -n "${IMAGE_FORMAT}" ]; then
  FULL_REGISTRY_IMAGE=${IMAGE_FORMAT/'${component}'/performance-addon-operator-registry}
fi

echo "Deploying using image $FULL_REGISTRY_IMAGE."

# Deploy features
success=0
iterations=0
sleep_time=10
max_iterations=30 # results in 5 minute timeout
feature_dir=cluster-setup/${CLUSTER}-cluster/performance/

until [[ $success -eq 1 ]] || [[ $iterations -eq $max_iterations ]]
do

  echo "[INFO] Deploying performance operator and profile."
  set +e

  # be verbose on last iteration only
  if [[ $iterations -eq $((max_iterations - 1)) ]] || [[ -n "${VERBOSE}" ]]; then
    # WORKAROUND for https://github.com/kubernetes/kubernetes/pull/89539:
    # oc / kubectl reject multiple manifests atm as soon as one "kind" in them does not exist yet
    # so we need to apply one manifest by one
    # since xargs' delimiter is limited to one char only or a control code, we replace the manifest delimiter "---"
    # with a "vertical tab (\v)", which should never be used in (at least our) manifests.
    # revert the sed | xargs steps when the fix landed in oc
    ${OC_TOOL} kustomize $feature_dir | envsubst | sed "s|---|\v|g" | xargs -d '\v' -I {} bash -c "echo '{}' | ${OC_TOOL} apply -f -"
  else
    ${OC_TOOL} kustomize $feature_dir | envsubst | sed "s|---|\v|g" | xargs -d '\v' -I {} bash -c "echo '{}' | ${OC_TOOL} apply -f - &> /dev/null"
  fi

  # shellcheck disable=SC2181
  if [[ $? != 0 ]];then

    iterations=$((iterations + 1))
    iterations_left=$((max_iterations - iterations))
    if [[ $iterations_left != 0  ]]; then
      echo "[WARN] Deployment did not fully succeed yet, retrying in $sleep_time sec, $iterations_left retries left"
      sleep $sleep_time
    else
      echo "[WARN] At least one deployment failed, giving up"
    fi

  else
    # All features deployed successfully
    success=1
  fi
  set -e

done

if [[ $success -eq 0 ]]; then
  echo "[ERROR] Deployment failed, giving up."
  exit 1
fi

echo "[INFO] Deployment successful."
