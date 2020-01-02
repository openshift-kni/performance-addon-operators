#!/usr/bin/env bash

set -euo pipefail

if [[ -f /var/reboot ]]; then 
    rm -f /var/reboot
    echo "File /var/reboot exists, initiate reboot"
    systemctl reboot
fi
