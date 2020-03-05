#!/bin/bash

which ginkgo
if [ $? -ne 0 ]; then
	echo "Downloading ginkgo tool"
	go install github.com/onsi/ginkgo/ginkgo
fi

# after running "updating profile" tests - some other functional tests might be failed, so it's better to run them separately
GOFLAGS=-mod=vendor ginkgo functests -- -junit /tmp/artifacts/unit_report.xml -ginkgo.skip "Updating parameters in performance profile"
GOFLAGS=-mod=vendor ginkgo functests -- -junit /tmp/artifacts/updating_profile_unit_report.xml -ginkgo.focus "Updating parameters in performance profile"

