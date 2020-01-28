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
	test-cluster-setup \
	kustomize

IMAGE_BUILD_CMD ?= "docker"
IMAGE_REGISTRY ?= "quay.io"
REGISTRY_NAMESPACE ?= ""
IMAGE_TAG ?= "latest"

TARGET_GOOS=linux
TARGET_GOARCH=amd64

CACHE_DIR="_cache"
TOOLS_DIR="$(CACHE_DIR)/tools"

KUSTOMIZE_VERSION="v3.5.3"
KUSTOMIZE_PLATFORM ?= "linux_amd64"
KUSTOMIZE_BIN="kustomize"
KUSTOMIZE_TAR="$(KUSTOMIZE_BIN)_$(KUSTOMIZE_VERSION)_$(KUSTOMIZE_PLATFORM).tar.gz"
KUSTOMIZE="$(TOOLS_DIR)/$(KUSTOMIZE_BIN)"

OPERATOR_SDK_VERSION="v0.13.0"
OPERATOR_SDK_PLATFORM ?= "x86_64-linux-gnu"
OPERATOR_SDK_BIN="operator-sdk-$(OPERATOR_SDK_VERSION)-$(OPERATOR_SDK_PLATFORM)"
OPERATOR_SDK="$(TOOLS_DIR)/$(OPERATOR_SDK_BIN)"

REGISTRY_IMAGE_NAME="performance-addon-operator-registry"
OPERATOR_IMAGE_NAME="performance-addon-operator"

FULL_OPERATOR_IMAGE ?= "$(IMAGE_REGISTRY)/$(REGISTRY_NAMESPACE)/$(OPERATOR_IMAGE_NAME):$(IMAGE_TAG)"
FULL_REGISTRY_IMAGE ?= "${IMAGE_REGISTRY}/${REGISTRY_NAMESPACE}/${REGISTRY_IMAGE_NAME}:${IMAGE_TAG}"

OPERATOR_DEV_CSV="0.0.1"

# Export GO111MODULE=on to enable project to be built from within GOPATH/src
export GO111MODULE=on

build: gofmt golint govet dist

dist:
	@echo "Building operator binary"
	mkdir -p build/_output/bin
	env GOOS=$(TARGET_GOOS) GOARCH=$(TARGET_GOARCH) go build -i -ldflags="-s -w" -mod=vendor -o build/_output/bin/performance-addon-operators ./cmd/manager

build-containers: registry-container operator-container

operator-container: build
	@echo "Building the performance-addon-operator image"
	@if [ -z "$(REGISTRY_NAMESPACE)" ]; then\
		echo "REGISTRY_NAMESPACE env-var must be set to your $(IMAGE_REGISTRY) namespace";\
		exit 1;\
	fi
	$(IMAGE_BUILD_CMD) build --no-cache -f build/Dockerfile -t $(FULL_OPERATOR_IMAGE) build/

registry-container: generate-latest-dev-csv
	@echo "Building the performance-addon-operator registry image"
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

generate-csv: operator-sdk
	@if [ -z "$(REGISTRY_NAMESPACE)" ]; then\
		echo "REGISTRY_NAMESPACE env-var must be set to your $(IMAGE_REGISTRY) namespace";\
		exit 1;\
	fi
	OPERATOR_SDK=$(OPERATOR_SDK) FULL_OPERATOR_IMAGE=$(FULL_OPERATOR_IMAGE) hack/csv-generate.sh

generate-latest-dev-csv: operator-sdk
	@echo Generating developer csv
	@echo
	OPERATOR_SDK=$(OPERATOR_SDK) FULL_OPERATOR_IMAGE="REPLACE_IMAGE" CSV_VERSION=$(OPERATOR_DEV_CSV) hack/csv-generate.sh

deps-update:
	go mod tidy && \
	go mod vendor

deploy: cluster-deploy
	# TODO - deprecated, will be removed soon in favor of cluster-deploy

cluster-deploy: kustomize
	@echo "Deploying operator"
	FULL_REGISTRY_IMAGE=$(FULL_REGISTRY_IMAGE) KUSTOMIZE=$(KUSTOMIZE) hack/deploy.sh

cluster-label-worker-rt:
	@echo "Adding worker-rt label to worker nodes"
	hack/label-worker-rt.sh

cluster-wait-for-mcp:
	@echo "Waiting for MCP to be updated"
	hack/wait-for-mcp.sh

cluster-clean:
	@echo "Deleting operator"
	FULL_REGISTRY_IMAGE=$(FULL_REGISTRY_IMAGE) hack/clean-deploy.sh

functests: cluster-label-worker-rt cluster-wait-for-mcp functests-only

functests-only:
	@echo "Running Functional Tests"
	hack/run-functests.sh

unittests:
	GOFLAGS=-mod=vendor go test -v ./pkg/...

gofmt:
	@echo "Running gofmt"
	gofmt -s -l `find . -path ./vendor -prune -o -type f -name '*.go' -print`

golint:
	@echo "Running go lint"
	hack/lint.sh

govet:
	@echo "Running go vet"
	go vet ./...

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

kustomize:
	@if [ ! -x "$(KUSTOMIZE)" ]; then\
		echo "Downloading kustomize $(KUSTOMIZE_VERSION)";\
		mkdir -p $(TOOLS_DIR);\
		curl -JL https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize/$(KUSTOMIZE_VERSION)/$(KUSTOMIZE_TAR) -o $(TOOLS_DIR)/$(KUSTOMIZE_TAR);\
		tar -xvf $(TOOLS_DIR)/$(KUSTOMIZE_TAR) -C $(TOOLS_DIR);\
		rm -rf $(TOOLS_DIR)/$(KUSTOMIZE_TAR);\
		chmod +x $(KUSTOMIZE);\
	else\
		echo "Using kustomize cached at $(KUSTOMIZE)";\
	fi
