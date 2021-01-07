#!/bin/bash

set -e

# expect oc to be in PATH by default
OC_TOOL="${OC_TOOL:-oc}"
# TODO: factor out all the versions
VERSION="${VERSION:-4.7}"

echo "Deploying using image $FULL_INDEX_IMAGE."

# Deploy features
success=0
iterations=0
sleep_time=10
max_iterations=30 # results in 5 minute timeout


if [ -z "${CLUSTER##*upgrade*}" ] ;then
	case $VERSION in
		4.[4-6])
			# OLM <= 4.6
			CLUSTER="olm-pre-4.6/${CLUSTER}"
			;;
		*)
			;;
	esac
fi
feature_dir=cluster-setup/${CLUSTER}-cluster/performance/
echo "[INFO] Feature is $feature_dir."
${OC_TOOL} kustomize $feature_dir | envsubst


until [[ $success -eq 1 ]] || [[ $iterations -eq $max_iterations ]]
do

  echo "[INFO] Deploying performance operator and profile."
  set +e

  # be verbose on last iteration only
  if [[ $iterations -eq $((max_iterations - 1)) ]] || [[ -n "${VERBOSE}" ]]; then
    ${OC_TOOL} kustomize $feature_dir | envsubst | ${OC_TOOL} apply -f -
  else
    ${OC_TOOL} kustomize $feature_dir | envsubst | ${OC_TOOL} apply -f - &> /dev/null
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
