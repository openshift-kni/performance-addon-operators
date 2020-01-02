#!/bin/sh

echo "waiting for linuxptp-daemon to report ptp interface status"

node_number=`oc get nodes --no-headers | wc -l`

until [[ `oc get nodeptpdevices.ptp.openshift.io --no-headers | wc -l` == ${node_number} ]]; do
  echo "waiting for linuxptp-daemon to report ptp interface status"
done

