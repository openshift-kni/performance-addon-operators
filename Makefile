
export FEATURES?=mcp performance sctp

.PHONY: build
build:
	GOFLAGS=-mod=vendor go build ./...

.PHONY: deps-update
deps-update:
	go mod tidy && \
	go mod vendor

ginkgo:
	go install github.com/onsi/ginkgo/ginkgo

.PHONY: functests
functests: ginkgo
	FOCUS=$$(echo $(FEATURES) | tr ' ' '|') && \
	echo "Focusing on $$FOCUS" && \
	GOFLAGS=-mod=vendor ginkgo --focus=$$FOCUS functests -- -junit /tmp/artifacts/unit_report.xml
	#TODO - copy in functional test suite

.PHONY: unittests
unittests:
	# functests are marked with "// +build !unittests" and will be skipped
	GOFLAGS=-mod=vendor go test -v --tags unittests ./...
	#TODO - copy in unit tests

