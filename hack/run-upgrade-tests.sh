#!/bin/bash

FROM_VERSION="${FROM_VERSION:-4.4.0}"
TO_VERSION="${TO_VERSION:-4.5.0}"

# check if operator is already installed with right version
subs=$(oc get subscriptions -o name -n openshift-performance-addon)
if [ -n "$subs" ]; then
  echo "Operator exists, verifying the version"
  channel=$(oc get $subs -n openshift-performance-addon -o jsonpath={.spec.channel})
  if [[ "$channel" != "$FROM_VERSION" ]]; then
    echo "Channel $channel is not equal to $FROM_VERSION, exit"
    exit 1
  fi
else
  sed -i "s|REPLACE_CHANNEL|${FROM_VERSION}|g" cluster-setup/upgrade-test-cluster/performance/operator_subscription.patch.yaml
  IMAGE_TAG="${FROM_VERSION:0:3}-snapshot" CLUSTER="upgrade-test" make cluster-deploy
  make cluster-wait-for-mcp
fi

which ginkgo
if [ $? -ne 0 ]; then
	echo "Downloading ginkgo tool"
	go install github.com/onsi/ginkgo/ginkgo
fi

NO_COLOR=""
if ! which tput &> /dev/null 2>&1 || [[ $(tput -T$TERM colors) -lt 8 ]]; then
  echo "Terminal does not seem to support colored output, disabling it"
  NO_COLOR="-noColor"
fi

# fail if any of the following fails
err=0
trap 'err=1' ERR

# -v: print out the text and location for each spec before running it and flush output to stdout in realtime
# -r: run suites recursively
# --keepGoing: don't stop on failing suite
# -requireSuite: fail if tests are not executed because of missing suite
GOFLAGS=-mod=vendor ginkgo $NO_COLOR --v -r --keepGoing -requireSuite functests-extended -- -junitDir /tmp/artifacts -fromVersion $FROM_VERSION -toVersion $TO_VERSION

# fail if any of the above failed
test $err = 0

