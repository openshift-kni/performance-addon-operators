#!/bin/bash

which ginkgo
if [ $? -ne 0 ]; then
	echo "Downloading ginkgo tool"
	go install github.com/onsi/ginkgo/ginkgo
fi

GOFLAGS=-mod=vendor ginkgo functests -- -junit /tmp/artifacts/unit_report.xml
