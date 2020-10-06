#!/bin/bash

FROM_VERSION="${FROM_VERSION:-4.5}"
TO_VERSION="${TO_VERSION:-4.6}"

OC_TOOL="${OC_TOOL:-oc}"

# By default we are running full scope of tests after operator upgrade (time consuming)
RUN_TESTS_AFTER_UPGRADE="${RUN_TESTS_AFTER_UPGRADE:-true}"
PERF_TEST_PROFILE="${PERF_TEST_PROFILE:-upgrade-test}"
CLUSTER="${CLUSTER:-upgrade-test}"
DEPLOY_PAO="${DEPLOY_PAO:-true}"

# check if operator is already installed with right version
subs=$(${OC_TOOL} get subscriptions -o name -n openshift-performance-addon)
if [ -n "$subs" ]; then
  echo "Operator exists, verifying the version"
  channel=$(oc get $subs -n openshift-performance-addon -o jsonpath={.spec.channel})
  if [[ "$channel" != "$FROM_VERSION" ]]; then
    echo "Channel $channel is not equal to $FROM_VERSION, exit"
    exit 1
  fi
fi

if [ "$DEPLOY_PAO" == true ]; then
  CLUSTER="${CLUSTER}" make cluster-deploy
  make cluster-label-worker-cnf
  CLUSTER="upgrade-test" make cluster-wait-for-mcp
fi

which ginkgo
if [ $? -ne 0 ]; then
  echo "Downloading ginkgo tool"
  go install github.com/onsi/ginkgo/ginkgo
fi

NO_COLOR=""
if ! which tput &>/dev/null 2>&1 || [[ $(tput -T$TERM colors) -lt 8 ]]; then
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
# HEADS UP: fromVersion needs to match the channel in cluster-setup/upgrade-test-cluster/performance/operator_subscription.patch.yaml
GOFLAGS=-mod=vendor ginkgo $NO_COLOR --v -r --keepGoing -requireSuite functests-extended -- -junitDir /tmp/artifacts -fromVersion $FROM_VERSION -toVersion $TO_VERSION

echo "[INFO] Waiting a bit until MCO starts updating nodes"
sleep 60

# run all tests after upgrade operator
if [ "$RUN_TESTS_AFTER_UPGRADE" == true ] && [ $err = 0 ]; then
  echo "[INFO] Running tests after operator upgrade"
  ${OC_TOOL} get performanceprofile "$PERF_TEST_PROFILE"
  if [ $? -ne 0 ]; then
    echo "[ERROR] Performance profile $PERF_TEST_PROFILE not exists, exit"
    exit 1
  fi
  PERF_TEST_PROFILE=$PERF_TEST_PROFILE make functests-only
fi

# fail if any of the above failed
test $err = 0
