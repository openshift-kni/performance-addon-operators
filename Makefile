IMAGE_BUILD_CMD ?= "docker"
IMAGE_REGISTRY ?= "quay.io"
REGISTRY_NAMESPACE ?= "openshift-kni"
IMAGE_TAG ?= "4.5-snapshot"

TARGET_GOOS=linux
TARGET_GOARCH=amd64

CACHE_DIR="_cache"
TOOLS_DIR="$(CACHE_DIR)/tools"

OPERATOR_SDK_VERSION="v0.15.2"
OPERATOR_SDK_PLATFORM ?= "x86_64-linux-gnu"
OPERATOR_SDK_BIN="operator-sdk-$(OPERATOR_SDK_VERSION)-$(OPERATOR_SDK_PLATFORM)"
OPERATOR_SDK="$(TOOLS_DIR)/$(OPERATOR_SDK_BIN)"

REGISTRY_IMAGE_NAME="performance-addon-operator-registry"
OPERATOR_IMAGE_NAME="performance-addon-operator"

FULL_OPERATOR_IMAGE ?= "$(IMAGE_REGISTRY)/$(REGISTRY_NAMESPACE)/$(OPERATOR_IMAGE_NAME):$(IMAGE_TAG)"
FULL_REGISTRY_IMAGE ?= "${IMAGE_REGISTRY}/${REGISTRY_NAMESPACE}/${REGISTRY_IMAGE_NAME}:${IMAGE_TAG}"

CLUSTER ?= "ci"

GIT_VERSION=$$(git describe --always --tags)
VERSION=$${CI_UPSTREAM_VERSION:-$(GIT_VERSION)}
GIT_COMMIT=$$(git rev-list -1 HEAD)
COMMIT=$${CI_UPSTREAM_COMMIT:-$(GIT_COMMIT)}
BUILD_DATE=$$(date --utc -Iseconds)

# Export GO111MODULE=on to enable project to be built from within GOPATH/src
export GO111MODULE=on

.PHONY: build
build: gofmt golint govet dist

.PHONY: dist
dist:
	@echo "Building operator binary"
	mkdir -p build/_output/bin; \
    LDFLAGS="-s -w "; \
    LDFLAGS+="-X github.com/openshift-kni/performance-addon-operators/version.Version=$(VERSION) "; \
    LDFLAGS+="-X github.com/openshift-kni/performance-addon-operators/version.GitCommit=$(COMMIT) "; \
    LDFLAGS+="-X github.com/openshift-kni/performance-addon-operators/version.BuildDate=$(BUILD_DATE) "; \
	env GOOS=$(TARGET_GOOS) GOARCH=$(TARGET_GOARCH) go build -i -ldflags="$$LDFLAGS" \
	  -mod=vendor -o build/_output/bin/performance-addon-operators ./cmd/manager

.PHONY: dist-tools
dist-tools: dist-csv-generator dist-csv-replace-imageref

.PHONY: dist-clean
dist-clean:
	rm -rf build/_output/bin

.PHONY: dist-csv-generator
dist-csv-generator:
	@if [ ! -x build/_output/bin/csv-generator ]; then\
		echo "Building csv-generator tool";\
		mkdir -p build/_output/bin;\
		env GOOS=$(TARGET_GOOS) GOARCH=$(TARGET_GOARCH) go build -i -ldflags="-s -w" -mod=vendor -o build/_output/bin/csv-generator ./tools/csv-generator;\
	else \
		echo "Using pre-built csv-generator tool";\
	fi

.PHONY: dist-csv-replace-imageref
dist-csv-replace-imageref:
	@if [ ! -x build/_output/bin/csv-replace-imageref ]; then\
		echo "Building csv-replace-imageref tool";\
		mkdir -p build/_output/bin;\
		env GOOS=$(TARGET_GOOS) GOARCH=$(TARGET_GOARCH) go build -i -ldflags="-s -w" -mod=vendor -o build/_output/bin/csv-replace-imageref ./tools/csv-replace-imageref;\
	else \
		echo "Using pre-built csv-replace-imageref tool";\
	fi

.PHONY: dist-docs-generator
dist-docs-generator:
	@if [ ! -x build/_output/bin/docs-generator ]; then\
		echo "Building docs-generator tool";\
		mkdir -p build/_output/bin;\
		env GOOS=$(TARGET_GOOS) GOARCH=$(TARGET_GOARCH) go build -i -ldflags="-s -w" -mod=vendor -o build/_output/bin/docs-generator ./tools/docs-generator;\
	else \
		echo "Using pre-built docs-generator tool";\
	fi

.PHONY: build-containers
build-containers: registry-container operator-container

.PHONY: operator-container
operator-container: build
	@echo "Building the performance-addon-operator image"
	@if [ -z "$(REGISTRY_NAMESPACE)" ]; then\
		echo "REGISTRY_NAMESPACE env-var must be set to your $(IMAGE_REGISTRY) namespace";\
		exit 1;\
	fi
	$(IMAGE_BUILD_CMD) build --no-cache -f openshift-ci/Dockerfile.deploy -t $(FULL_OPERATOR_IMAGE) --build-arg BIN_DIR="_output/bin/" --build-arg ASSETS_DIR="assets" build/

.PHONY: registry-container
registry-container: generate-latest-dev-csv
	@echo "Building the performance-addon-operator registry image"
	$(IMAGE_BUILD_CMD) build --no-cache -f openshift-ci/Dockerfile.registry.upstream.dev -t "$(FULL_REGISTRY_IMAGE)" --build-arg FULL_OPERATOR_IMAGE="$(FULL_OPERATOR_IMAGE)"  .

.PHONY: push-containers
push-containers:
	$(IMAGE_BUILD_CMD) push $(FULL_OPERATOR_IMAGE)
	$(IMAGE_BUILD_CMD) push $(FULL_REGISTRY_IMAGE)

.PHONY: operator-sdk
operator-sdk:
	@if [ ! -x "$(OPERATOR_SDK)" ]; then\
		echo "Downloading operator-sdk $(OPERATOR_SDK_VERSION)";\
		mkdir -p $(TOOLS_DIR);\
		curl -JL https://github.com/operator-framework/operator-sdk/releases/download/$(OPERATOR_SDK_VERSION)/$(OPERATOR_SDK_BIN) -o $(OPERATOR_SDK);\
		chmod +x $(OPERATOR_SDK);\
	else\
		echo "Using operator-sdk cached at $(OPERATOR_SDK)";\
	fi

.PHONY: generate-csv
generate-csv: operator-sdk dist-csv-generator
	@if [ -z "$(REGISTRY_NAMESPACE)" ]; then\
		echo "REGISTRY_NAMESPACE env-var must be set to your $(IMAGE_REGISTRY) namespace";\
		exit 1;\
	fi
	OPERATOR_SDK=$(OPERATOR_SDK) FULL_OPERATOR_IMAGE=$(FULL_OPERATOR_IMAGE) hack/csv-generate.sh

.PHONY: generate-latest-dev-csv
generate-latest-dev-csv: operator-sdk dist-csv-generator
	@echo Generating developer csv
	@echo
	OPERATOR_SDK=$(OPERATOR_SDK) FULL_OPERATOR_IMAGE="REPLACE_IMAGE" hack/csv-generate.sh -dev

.PHONY: generate-docs
generate-docs: dist-docs-generator
	hack/docs-generate.sh

.PHONY: eps-update
deps-update:
	go mod tidy && \
	go mod vendor

.PHONY: deploy
deploy: cluster-deploy
	# TODO - deprecated, will be removed soon in favor of cluster-deploy

.PHONY: cluster-deploy
cluster-deploy:
	@echo "Deploying operator"
	FULL_REGISTRY_IMAGE=$(FULL_REGISTRY_IMAGE) CLUSTER=$(CLUSTER) hack/deploy.sh

.PHONY: cluster-label-worker-cnf
cluster-label-worker-cnf:
	@echo "Adding worker-cnf label to worker nodes"
	hack/label-worker-cnf.sh

.PHONY: cluster-wait-for-mcp
cluster-wait-for-mcp:
    # NOTE: for CI this is done in the config suite of the functests!
    # Use this when deploying manifests manually with CLUSTER=manual
	@echo "Waiting for MCP to be updated"
	hack/wait-for-mcp.sh

.PHONY: cluster-clean
cluster-clean:
	@echo "Deleting operator"
	FULL_REGISTRY_IMAGE=$(FULL_REGISTRY_IMAGE) hack/clean-deploy.sh

.PHONY: functests
functests: cluster-label-worker-cnf functests-only

.PHONY: functests-only
functests-only:
	@echo "Running Functional Tests"
	hack/run-functests.sh

.PHONY: operator-upgrade-tests
operator-upgrade-tests:
	@echo "Running Operator Upgrade Tests"
	hack/run-upgrade-tests.sh

.PHONY: unittests
unittests:
	hack/unittests.sh

.PHONY: gofmt
gofmt:
	@echo "Running gofmt"
	gofmt -s -w `find . -path ./vendor -prune -o -type f -name '*.go' -print`

.PHONY: golint
golint:
	@echo "Running go lint"
	hack/lint.sh

.PHONY: govet
govet:
	@echo "Running go vet"
	go vet ./...

.PHONY: generate
generate: deps-update gofmt generate-latest-dev-csv generate-docs
	@echo Updating generated files
	@echo
	export GOROOT=$$(go env GOROOT); $(OPERATOR_SDK) generate k8s

.PHONY: verify
verify: golint govet generate
	@echo "Verifying that all code is committed after updating deps and formatting and generating code"
	hack/verify-generated.sh

.PHONY: ci-job
ci-job: verify build unittests

.PHONY: release-note
release-note:
	hack/release-note.sh

.PHONY: generate-release-tags
generate-release-tags:
	hack/generate-release-tags.sh
