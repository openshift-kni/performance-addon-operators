#!/bin/bash

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
GOFLAGS=-mod=vendor ginkgo $NO_COLOR --v -r --keepGoing -requireSuite functests -- -junitDir /tmp/artifacts
