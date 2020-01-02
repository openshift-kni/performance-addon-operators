
export FEATURES?=mcp performance sctp ptp

.PHONY: build \
	deps-update \
	functests \
	unittests \
	gofmt \
	golint \
	govet \
	deploy \
	cluster-deploy \
	cluster-clean \
	generate \
	verify-generate \
	ci-job \
	build-containers \
	operator-container \
	registry-container \
	generate-latest-dev-csv \
	test-cluster-setup


IMAGE_BUILD_CMD ?= "docker"
IMAGE_REGISTRY ?= "quay.io"
REGISTRY_NAMESPACE ?= ""
IMAGE_TAG ?= "latest"

TARGET_GOOS=linux
TARGET_GOARCH=amd64

CACHE_DIR="_cache"
TOOLS_DIR="$(CACHE_DIR)/tools"

OPERATOR_SDK_VERSION="v0.13.0"
OPERATOR_SDK_PLATFORM ?= "x86_64-linux-gnu"
OPERATOR_SDK_BIN="operator-sdk-$(OPERATOR_SDK_VERSION)-$(OPERATOR_SDK_PLATFORM)"
OPERATOR_SDK="$(TOOLS_DIR)/$(OPERATOR_SDK_BIN)"

REGISTRY_IMAGE_NAME="performance-addon-operators-registry"
OPERATOR_IMAGE_NAME="performance-addon-operators"

FULL_OPERATOR_IMAGE="$(IMAGE_REGISTRY)/$(REGISTRY_NAMESPACE)/$(OPERATOR_IMAGE_NAME):$(IMAGE_TAG)"
FULL_REGISTRY_IMAGE="${IMAGE_REGISTRY}/${REGISTRY_NAMESPACE}/${REGISTRY_IMAGE_NAME}:${IMAGE_TAG}"

OPERATOR_DEV_CSV="0.0.1"

# Export GO111MODULE=on to enable project to be built from within GOPATH/src
export GO111MODULE=on

build: gofmt golint
	@echo "Building operator binary"
	mkdir -p build/_output/bin
	env GOOS=$(TARGET_GOOS) GOARCH=$(TARGET_GOARCH) go build -i -ldflags="-s -w" -mod=vendor -o build/_output/bin/performance-addon-operators ./cmd/manager

build-containers: registry-container operator-container

operator-container: build
	@echo "Building the performance-addon-operators image"
	@if [ -z "$(REGISTRY_NAMESPACE)" ]; then\
		echo "REGISTRY_NAMESPACE env-var must be set to your $(IMAGE_REGISTRY) namespace";\
		exit 1;\
	fi
	$(IMAGE_BUILD_CMD) build --no-cache -f build/Dockerfile -t $(FULL_OPERATOR_IMAGE) build/

registry-container: generate-latest-dev-csv
	@echo "Building the performance-addon-operatorsregistry image"
	$(IMAGE_BUILD_CMD) build --no-cache -t "$(FULL_REGISTRY_IMAGE)" --build-arg FULL_OPERATOR_IMAGE="$(FULL_OPERATOR_IMAGE)" -f deploy/Dockerfile .

push-containers:
	$(IMAGE_BUILD_CMD) push $(FULL_OPERATOR_IMAGE)
	$(IMAGE_BUILD_CMD) push $(FULL_REGISTRY_IMAGE)

operator-sdk:
	@if [ ! -x "$(OPERATOR_SDK)" ]; then\
		echo "Downloading operator-sdk $(OPERATOR_SDK_VERSION)";\
		mkdir -p $(TOOLS_DIR);\
		curl -JL https://github.com/operator-framework/operator-sdk/releases/download/$(OPERATOR_SDK_VERSION)/$(OPERATOR_SDK_BIN) -o $(OPERATOR_SDK);\
		chmod +x $(OPERATOR_SDK);\
	else\
		echo "Using operator-sdk cached at $(OPERATOR_SDK)";\
	fi

generate-latest-dev-csv: operator-sdk
	@echo Generating developer csv
	@echo
	export GOROOT=$$(go env GOROOT); $(OPERATOR_SDK) olm-catalog gen-csv --csv-version=$(OPERATOR_DEV_CSV)
	# removing replaces field which breaks CSV validation
	sed -i 's/replaces\:.*//g' deploy/olm-catalog/performance-addon-operators/$(OPERATOR_DEV_CSV)/performance-addon-operators.v0.0.1.clusterserviceversion.yaml
	# adding temporariy required displayName field
	sed -i '/version\: v1alpha1/a displayName\: placeholder' deploy/olm-catalog/performance-addon-operators/$(OPERATOR_DEV_CSV)/performance-addon-operators.v0.0.1.clusterserviceversion.yaml
	sed -i 's/^displayName\: placeholder/      displayName\: placeholder/g' deploy/olm-catalog/performance-addon-operators/$(OPERATOR_DEV_CSV)/performance-addon-operators.v0.0.1.clusterserviceversion.yaml

	@echo
	export GOROOT=$$(go env GOROOT); $(OPERATOR_SDK) generate crds
	cp deploy/crds/*crd.yaml deploy/olm-catalog/performance-addon-operators/$(OPERATOR_DEV_CSV)/

deps-update:
	go mod tidy && \
	go mod vendor

deploy: cluster-deploy
	# TODO - deprecated, will be removed soon in favor of cluster-deploy

cluster-deploy:
	@echo "Deploying operator"
	FULL_REGISTRY_IMAGE=$(FULL_REGISTRY_IMAGE) hack/deploy.sh

cluster-clean:
	@echo "Deleting operator"
	FULL_REGISTRY_IMAGE=$(FULL_REGISTRY_IMAGE) hack/clean-deploy.sh

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
	export GOROOT=$$(go env GOROOT); $(OPERATOR_SDK) generate k8s
	@echo
	export GOROOT=$$(go env GOROOT); $(OPERATOR_SDK) generate crds

verify-generate: generate
	@echo "Verifying generated code"
	hack/verify-generated.sh

ci-job: gofmt golint govet verify-generate build unittests

test-cluster-setup:
	@echo "Setting up the test cluster"
	hack/setup_test_cluster.sh