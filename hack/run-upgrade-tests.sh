#!/bin/bash

IMAGE_TAG="4.5-snapshot" CLUSTER="upgrade-test" make cluster-deploy
make cluster-wait-for-mcp

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

# -v: print out the text and location for each spec before running it and flush output to stdout in realtime
# -r: run suites recursively
# --keepGoing: don't stop on failing suite
# -requireSuite: fail if tests are not executed because of missing suite
# HEADS UP: fromVersion needs to match the channel in cluster-setup/upgrade-test-cluster/performance/operator_subscription.patch.yaml
GOFLAGS=-mod=vendor ginkgo $NO_COLOR --v -r --keepGoing -requireSuite functests-extended -- -junitDir /tmp/artifacts -fromVersion 4.5.0 -toVersion 4.6.0

make cluster-clean
