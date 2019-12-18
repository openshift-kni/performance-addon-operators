
export FEATURES?=mcp performance sctp

.PHONY: build \
	deps-update \
	functests \
	unittests \
	gofmt \
	golint \
	govet \
	deploy

TARGET_GOOS=linux
TARGET_GOARCH=amd64

# Export GO111MODULE=on to enable project to be built from within GOPATH/src
export GO111MODULE=on

build: gofmt golint
	@echo "Building operator binary"
	mkdir -p build/_output/bin
	env GOOS=$(TARGET_GOOS) GOARCH=$(TARGET_GOARCH) go build -i -ldflags="-s -w" -mod=vendor -o build/_output/bin/performance-addon-operators ./cmd/manager

deps-update:
	go mod tidy && \
	go mod vendor

deploy:
	@echo "Deploying features $$FEATURES"
	# TODO - add deploy logic here

functests:
	@echo "Running Functional Tests"
	hack/run-functests.sh

unittests:
	# functests are marked with "// +build !unittests" and will be skipped
	GOFLAGS=-mod=vendor go test -v --tags unittests ./...
	#TODO - copy in unit tests

gofmt:
	@echo "Running gofmt"
	gofmt -s -l `find . -path ./vendor -prune -o -type f -name '*.go' -print`

golint:
	@echo "Running go lint"
	hack/lint.sh

govet:
	@echo "Running go vet"
	go vet github.com/openshift-kni/performance-addon-operators/...

