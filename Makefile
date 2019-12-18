
export FEATURES?=mcp performance sctp

.PHONY: build \
	deps-update \
	functests \
	unittests \
	gofmt \
	golint \
	govet \
	deploy \
	generate \
	verify-generate \

TARGET_GOOS=linux
TARGET_GOARCH=amd64

CACHE_DIR="_cache"
TOOLS_DIR="$(CACHE_DIR)/tools"

OPERATOR_SDK_VERSION="v0.13.0"
OPERATOR_SDK_PLATFORM ?= "x86_64-linux-gnu"
OPERATOR_SDK_BIN="operator-sdk-$(OPERATOR_SDK_VERSION)-$(OPERATOR_SDK_PLATFORM)"
OPERATOR_SDK="$(TOOLS_DIR)/$(OPERATOR_SDK_BIN)"

# Export GO111MODULE=on to enable project to be built from within GOPATH/src
export GO111MODULE=on

build: gofmt golint
	@echo "Building operator binary"
	mkdir -p build/_output/bin
	env GOOS=$(TARGET_GOOS) GOARCH=$(TARGET_GOARCH) go build -i -ldflags="-s -w" -mod=vendor -o build/_output/bin/performance-addon-operators ./cmd/manager

operator-sdk:
	@if [ ! -x "$(OPERATOR_SDK)" ]; then\
		echo "Downloading operator-sdk $(OPERATOR_SDK_VERSION)";\
		mkdir -p $(TOOLS_DIR);\
		curl -JL https://github.com/operator-framework/operator-sdk/releases/download/$(OPERATOR_SDK_VERSION)/$(OPERATOR_SDK_BIN) -o $(OPERATOR_SDK);\
		chmod +x $(OPERATOR_SDK);\
	else\
		echo "Using operator-sdk cached at $(OPERATOR_SDK)";\
	fi


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

generate: operator-sdk
	@echo Updating generated files
	@echo
	@$(OPERATOR_SDK) generate k8s
	@echo
	@$(OPERATOR_SDK) generate openapi

verify-generate: update-generate
	@echo "Verifying generated code"
	hack/verify-generated.sh
