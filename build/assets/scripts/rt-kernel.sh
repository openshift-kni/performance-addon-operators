#!/usr/bin/env bash

######################################################################################
## NOTE:                                                                            ##
## THIS IS A TEMPORARY WORKAROUND UNTIL THIS FEATURE IS AVAILABLE VIA MACHINECONFIG ##
## IT ONLY WORKS WITH RT KERNEL VERSION 4.*                                         ##
######################################################################################

set -euo pipefail

echo "Mounting OS image"

# remove old container
osContainerName="osContainer"
set +e
podman rm -f "$osContainerName" >/dev/null 2>&1
set -e

# find booted image sha
sha=$(rpm-ostree status --json | jq -r '.deployments[] | select(.booted == true) | .["custom-origin"][0]' | sed -E "s|.*@(.*)|\1|")
imageID=$(podman images --digests | grep $sha | awk '{print $4}')

# create and mount new container
podman create --net=none --name "$osContainerName" "$imageID" > /dev/null
kernelDir=$(podman mount "$osContainerName")

# Swap to or update RT kernel
kernel=$(uname -a)
if [[ "$kernel" =~ "PREEMPT RT" ]]
then
    echo "RT kernel already installed, checking for updates"

    installedVersion=$(rpm -qa | grep kernel-rt-core)
    # filename without rpm suffix is available version
    availableVersion=$(ls ${kernelDir}/kernel-rt-core-4*.rpm | xargs basename | xargs -i bash -c 'f="{}" && echo "${f%.*}"')

    if [[ "$installedVersion" == "$availableVersion" ]]
    then
        echo "No update available, nothing to do";
        exit 0
    else
        echo "Updating RT kernel"
        rpm-ostree override replace ${kernelDir}/kernel-rt-core-4*.rpm ${kernelDir}/kernel-rt-modules-4*.rpm ${kernelDir}/kernel-rt-modules-extra-4*.rpm
        echo "RT kernel updated, trigger reboot by touching /var/reboot"
        touch /var/reboot
        exit 0
    fi

else
    echo "Installing RT kernel"
    rpm-ostree override remove kernel{,-core,-modules,-modules-extra} --install ${kernelDir}/kernel-rt-core-4*.rpm --install ${kernelDir}/kernel-rt-modules-4*.rpm --install ${kernelDir}/kernel-rt-modules-extra-4*.rpm
    echo "RT kernel installed, trigger reboot by touching /var/reboot"
    touch /var/reboot
    exit 0
fi
