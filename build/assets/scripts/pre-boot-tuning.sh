#!/usr/bin/env bash

set -euo pipefail

reserved_cpumask=""
cpu_affinity=""

get_reserved_cores() {
    cores=()
    while read part; do
        if [[ $part =~ - ]]; then
            cores+=($(seq ${part/-/ }))
        elif [[ $part =~ , ]]; then
            continue 
        else
            cores+=($part)
        fi
    done < <( echo ${RESERVED_CPUS} | tr ',' '\n' )
}

# $1 - 0 for irq balance banned cpus masking , 1 for reserved cpus masking
get_cpu_mask() {
    if [ "$1" = "1" ]; then
        mask=( 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 )
    else
        mask=( 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 ) 
    fi    
    get_reserved_cores
    for core in ${cores[*]}; do
        mask[$core]=$1
    done
    cpumaskBinary=`echo ${mask[@]}| rev`
    cpumaskBinary=${cpumaskBinary//[[:space:]]/}
    reserved_cpumask=`printf '%08x\n' "$((2#$cpumaskBinary))"`
}

get_cpu_affinity() {
    cpu_affinity=""
    get_reserved_cores
    for core in ${cores[*]}; do
        cpu_affinity+=" $core"
    done
    echo "CPU Affinity set to $cpu_affinity"
}

# TODO - find a more robust approach than keeping the last timestamp
RHCOS_OSTREE_PATH=$(ls -td /boot/ostree/*/ | head -1)
RHCOS_OSTREE_BOOTLOADER_PATH=${RHCOS_OSTREE_PATH#"/boot"}
INITRD_GENERATION_DIR="/root/initrd"
INITRD_NEW_IMAGE="${RHCOS_OSTREE_PATH}/iso_initrd.img"

OSTREE_VERSION=$(rpm-ostree status -b | grep Version | awk '{print $2}')
CURRENT_ENTRY_FILE=$(grep -Rls ${OSTREE_VERSION} /boot/loader/entries/)

if [ -f ${INITRD_NEW_IMAGE} ] && grep -qs "iso_initrd.img" ${CURRENT_ENTRY_FILE}; then
    echo "Pre boot tuning configuration already applied"
    echo "Setting kernel rcuo* threads to the housekeeping cpus"
    get_cpu_mask 1
    pgrep rcuo* | while read line; do taskset -pc ${RESERVED_CPUS} $line || true; done
else
    # Clean up
    rm -rf ${INITRD_GENERATION_DIR}
    
    # Create initrd image
    mkdir ${INITRD_GENERATION_DIR}
    mkdir -p ${INITRD_GENERATION_DIR}/usr/lib/dracut/hooks/pre-udev/
    mkdir -p ${INITRD_GENERATION_DIR}/etc/systemd/
    mkdir -p ${INITRD_GENERATION_DIR}/etc/sysconfig/
    touch ${INITRD_GENERATION_DIR}/etc/systemd/system.conf
    touch ${INITRD_GENERATION_DIR}/etc/sysconfig/irqbalance
    touch ${INITRD_GENERATION_DIR}/usr/lib/dracut/hooks/pre-udev/00-tuned-pre-udev.sh
    chmod +x ${INITRD_GENERATION_DIR}/usr/lib/dracut/hooks/pre-udev/00-tuned-pre-udev.sh

    get_cpu_mask 1
    echo '#!/bin/sh

    type getargs >/dev/null 2>&1 || . /lib/dracut-lib.sh

    #cpumask="$(getargs reserved_cpumask)"
    cpumask='$reserved_cpumask'

    log()
    {
    echo "tuned: $@" >> /dev/kmsg
    }

    if [ -n "$cpumask" ]; then
    for file in /sys/devices/virtual/workqueue/cpumask /sys/bus/workqueue/devices/writeback/cpumask; do
        log "setting $file CPU mask to $cpumask"
        if ! echo $cpumask > $file 2>/dev/null; then
        log "ERROR: could not write CPU mask for $file"
        fi
    done
    fi' > ${INITRD_GENERATION_DIR}/usr/lib/dracut/hooks/pre-udev/00-tuned-pre-udev.sh

    # Set CPU affinity according to RESERVED_CPUS
    get_cpu_affinity
    echo "[Manager]" >> ${INITRD_GENERATION_DIR}/etc/systemd/system.conf
    echo "CPUAffinity=$cpu_affinity" >> ${INITRD_GENERATION_DIR}/etc/systemd/system.conf

    # Set IRQ banned cpu according to RESERVED_CPUS
    get_cpu_mask 0
    echo "IRQBALANCE_BANNED_CPUS=$reserved_cpumask" >> ${INITRD_GENERATION_DIR}/etc/sysconfig/irqbalance
    
    cd ${INITRD_GENERATION_DIR}
    find . | cpio -co >${INITRD_NEW_IMAGE}

    sed -i "s^initrd .*\$^& ${RHCOS_OSTREE_BOOTLOADER_PATH}iso_initrd.img^" ${CURRENT_ENTRY_FILE}

    #TODO - once RHCOS image contains the initrd content we can set parameters with rpm-ostree:
    #rpm-ostree initramfs --enable --arg=-I --arg=/etc/systemd/system.conf
    #rpm-ostree initramfs --enable --arg=-I --arg=/etc/sysconfig/irqbalance
    
    touch /var/reboot
fi
