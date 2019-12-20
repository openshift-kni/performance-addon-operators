#!/bin/bash


if [ $FEATURES == "" ]; then
	echo "No features provided"
	exit 1
fi

which kustomize
if [ $? -ne 0 ]; then
	echo "Downloading kustomize"
	go install sigs.k8s.io/kustomize/kustomize/v3 # using go install assuming we have GOBIN in PATH
fi

for f in $FEATURES; do
    if [ -d "cluster-setup/ci-cluster/feature_$f/" ]; then
        echo "[INFO]: deploying $f"
        #kustomize build "cluster-setup/ci-cluster/feature_$f" | oc apply -f -
    else
        echo "[WARN]: no ci-cluster feature found for the feature $f"
    fi;
done

for f in $FEATURES; do
    if [ -f "cluster-setup/ci-cluster/feature_$f/wait_for_it.sh" ]; then
        echo "[INFO]: waiting for $f to be deployed"
        cluster-setup/ci-cluster/feature_$f/wait_for_it.sh
        echo "[INFO]: $f was deployed"
    else
        echo "[WARN]: no wait logic found for the feature $f";
    fi;
done