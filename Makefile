IMAGE_BUILD_CMD ?= "docker"
IMAGE_REGISTRY ?= "quay.io"
REGISTRY_NAMESPACE ?= "openshift-kni"
IMAGE_TAG ?= "4.12-snapshot"

TARGET_GOOS=linux
TARGET_GOARCH=amd64

CACHE_DIR="_cache"
TOOLS_DIR="$(CACHE_DIR)/tools"
TOOLS_BIN_DIR="build/_output/bin"

OPERATOR_SDK_VERSION="v1.11.0"
OPERATOR_SDK_PLATFORM ?= "linux_amd64"
OPERATOR_SDK_BIN="operator-sdk_$(OPERATOR_SDK_PLATFORM)"
OPERATOR_SDK="$(TOOLS_DIR)/$(OPERATOR_SDK_BIN)"

OPERATOR_IMAGE_NAME="performance-addon-operator"
BUNDLE_IMAGE_NAME="performance-addon-operator-bundle"
INDEX_IMAGE_NAME="performance-addon-operator-index"
MUSTGATHER_IMAGE_NAME="performance-addon-operator-must-gather"
LATENCY_TEST_IMAGE_NAME="latency-test"

FULL_OPERATOR_IMAGE ?= "$(IMAGE_REGISTRY)/$(REGISTRY_NAMESPACE)/$(OPERATOR_IMAGE_NAME):$(IMAGE_TAG)"
FULL_BUNDLE_IMAGE ?= "${IMAGE_REGISTRY}/${REGISTRY_NAMESPACE}/${BUNDLE_IMAGE_NAME}:${IMAGE_TAG}"
FULL_INDEX_IMAGE ?= "${IMAGE_REGISTRY}/${REGISTRY_NAMESPACE}/${INDEX_IMAGE_NAME}:${IMAGE_TAG}"
FULL_MUSTGATHER_IMAGE ?= "${IMAGE_REGISTRY}/${REGISTRY_NAMESPACE}/${MUSTGATHER_IMAGE_NAME}:${IMAGE_TAG}"
FULL_LATENCY_TEST_IMAGE ?= "${IMAGE_REGISTRY}/${REGISTRY_NAMESPACE}/${LATENCY_TEST_IMAGE_NAME}:${IMAGE_TAG}"

CLUSTER ?= "ci"

GIT_VERSION=$$(git describe --always --tags)
VERSION=$${CI_UPSTREAM_VERSION:-$(GIT_VERSION)}
GIT_COMMIT=$$(git rev-list -1 HEAD)
COMMIT=$${CI_UPSTREAM_COMMIT:-$(GIT_COMMIT)}
BUILD_DATE=$$(date --utc -Iseconds)

# Export GO111MODULE=on to enable project to be built from within GOPATH/src
export GO111MODULE=on

.PHONY: all
all: build

# keep this target the first!
.PHONY: build
build: gofmt golint govet dist-gather-sysinfo dist create-performance-profile generate-manifests-tree

# just a shortcut for now
.PHONY: clean
clean: dist-clean

.PHONY: dist
dist: build-output-dir
	@echo "Building operator binary"
	mkdir -p $(TOOLS_BIN_DIR); \
    LDFLAGS="-s -w "; \
    LDFLAGS+="-X github.com/openshift-kni/performance-addon-operators/version.Version=$(VERSION) "; \
    LDFLAGS+="-X github.com/openshift-kni/performance-addon-operators/version.GitCommit=$(COMMIT) "; \
    LDFLAGS+="-X github.com/openshift-kni/performance-addon-operators/version.BuildDate=$(BUILD_DATE) "; \
	env GOOS=$(TARGET_GOOS) GOARCH=$(TARGET_GOARCH) go build -ldflags="$$LDFLAGS" \
	  -mod=vendor -o $(TOOLS_BIN_DIR)/performance-addon-operators .

.PHONY: dist-tools
dist-tools: dist-csv-processor dist-csv-replace-imageref

.PHONY: dist-clean
dist-clean:
	rm -rf build/_output/bin

.PHONY: dist-gather-sysinfo
dist-gather-sysinfo: build-output-dir
	@if [ ! -x $(TOOLS_BIN_DIR)/gather-sysinfo ]; then\
		echo "Building gather-sysinfo helper";\
		env CGO_ENABLED=0 GOOS=$(TARGET_GOOS) GOARCH=$(TARGET_GOARCH) go build -ldflags="-s -w" -mod=vendor -o $(TOOLS_BIN_DIR)/gather-sysinfo ./tools/gather-sysinfo;\
	else \
		echo "Using pre-built gather-sysinfo helper";\
	fi

.PHONY: dist-hugepages-mc-genarator
dist-hugepages-mc-genarator: build-output-dir
	echo "Building hugepages machineconfig genarator tool";\
	env GOOS=$(TARGET_GOOS) GOARCH=$(TARGET_GOARCH) go build -ldflags="-s -w" -mod=vendor -o $(TOOLS_BIN_DIR)/hugepages-machineconfig-generator ./tools/hugepages-machineconfig-generator

.PHONY: dist-csv-processor
dist-csv-processor: build-output-dir
	@if [ ! -x $(TOOLS_BIN_DIR)/csv-processor ]; then\
		echo "Building csv-processor tool";\
		env GOOS=$(TARGET_GOOS) GOARCH=$(TARGET_GOARCH) go build -ldflags="-s -w" -mod=vendor -o $(TOOLS_BIN_DIR)/csv-processor ./tools/csv-processor;\
	else \
		echo "Using pre-built csv-processor tool";\
	fi

.PHONY: dist-csv-replace-imageref
dist-csv-replace-imageref: build-output-dir
	@if [ ! -x $(TOOLS_BIN_DIR)/csv-replace-imageref ]; then\
		echo "Building csv-replace-imageref tool";\
		env GOOS=$(TARGET_GOOS) GOARCH=$(TARGET_GOARCH) go build -ldflags="-s -w" -mod=vendor -o $(TOOLS_BIN_DIR)/csv-replace-imageref ./tools/csv-replace-imageref;\
	else \
		echo "Using pre-built csv-replace-imageref tool";\
	fi

.PHONY: dist-docs-generator
dist-docs-generator: build-output-dir
	@if [ ! -x $(TOOLS_BIN_DIR)/docs-generator ]; then\
		echo "Building docs-generator tool";\
		env GOOS=$(TARGET_GOOS) GOARCH=$(TARGET_GOARCH) go build -ldflags="-s -w" -mod=vendor -o $(TOOLS_BIN_DIR)/docs-generator ./tools/docs-generator;\
	else \
		echo "Using pre-built docs-generator tool";\
	fi

.PHONY: dist-imgpull-tool
dist-imgpull-tool: build-output-dir
	env GOOS=$(TARGET_GOOS) GOARCH=$(TARGET_GOARCH) go build -ldflags="-s -w" -mod=vendor -o $(TOOLS_BIN_DIR)/imgpull-tool ./tools/imgpull-tool

.PHONY: dist-functests
dist-functests:
	./hack/build-test-bin.sh

.PHONY: dist-latency-tests
dist-latency-tests:
	./hack/build-latency-test-bin.sh

.PHONY: new-zversion
new-zversion: bump-zversion generate

.PHONY: bump-zversion
bump-zversion:
	./hack/bump-zversion.sh

.PHONY: build-containers
build-containers: bundle-container index-container operator-container must-gather-container

.PHONY: operator-container
operator-container: build
	@echo "Building the performance-addon-operator image"
	@if [ -z "$(REGISTRY_NAMESPACE)" ]; then\
		echo "REGISTRY_NAMESPACE env-var must be set to your $(IMAGE_REGISTRY) namespace";\
		exit 1;\
	fi
	$(IMAGE_BUILD_CMD) build --no-cache -f openshift-ci/Dockerfile.deploy -t $(FULL_OPERATOR_IMAGE) --build-arg BIN_DIR="_output/bin/" --build-arg ASSETS_DIR="assets" build/

.PHONY: bundle-container
bundle-container: generate-metadata generate-manifests-tree
	@echo "Building the performance-addon-operator bundle image"
	$(IMAGE_BUILD_CMD) build --no-cache -f openshift-ci/Dockerfile.bundle.upstream.dev -t "$(FULL_BUNDLE_IMAGE)" .

.PHONY: index-container
index-container: generate-index-database
	@echo "Building the performance-addon-operator index image"
	$(IMAGE_BUILD_CMD) build --no-cache -f build/_output/index.Dockerfile -t "$(FULL_INDEX_IMAGE)" build/_output

.PHONY: must-gather-container
must-gather-container: build
	@echo "Building the performance-addon-operator must-gather image"
	$(IMAGE_BUILD_CMD) build --no-cache -f openshift-ci/Dockerfile.must-gather -t "$(FULL_MUSTGATHER_IMAGE)" --build-arg BIN_DIR="build/_output/bin/" .

.PHONY: latency-test-container
latency-test-container:
	@echo "Building the latency test image"
	$(IMAGE_BUILD_CMD) build -f functests/4_latency/Dockerfile -t "$(FULL_LATENCY_TEST_IMAGE)"  .

.PHONY: push-bundle-container
push-bundle-container:
	$(IMAGE_BUILD_CMD) push $(FULL_BUNDLE_IMAGE)

.PHONY: push-containers
push-containers:
	$(IMAGE_BUILD_CMD) push $(FULL_OPERATOR_IMAGE)
	$(IMAGE_BUILD_CMD) push $(FULL_BUNDLE_IMAGE)
	$(IMAGE_BUILD_CMD) push $(FULL_INDEX_IMAGE)
	$(IMAGE_BUILD_CMD) push $(FULL_MUSTGATHER_IMAGE)

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

.PHONY: create-performance-profile
create-performance-profile:  build-output-dir
	echo "Creating performance profile"
	mkdir -p $(TOOLS_BIN_DIR); \
	LDFLAGS="-s -w "; \
	LDFLAGS+="-X github.com/openshift-kni/performance-addon-operators/cmd/performance-profile-creator "; \
	env GOOS=$(TARGET_GOOS) GOARCH=$(TARGET_GOARCH) go build  -v $(LDFLAGS) -o $(TOOLS_BIN_DIR)/performance-profile-creator ./cmd/performance-profile-creator

.PHONY: generate-csv
generate-csv: operator-sdk kustomize dist-csv-processor
	@if [ -z "$(REGISTRY_NAMESPACE)" ]; then\
		echo "REGISTRY_NAMESPACE env-var must be set to your $(IMAGE_REGISTRY) namespace";\
		exit 1;\
	fi
	OPERATOR_SDK=$(OPERATOR_SDK) KUSTOMIZE=$(KUSTOMIZE) FULL_OPERATOR_IMAGE=$(FULL_OPERATOR_IMAGE) hack/csv-generate.sh

.PHONY: build-output-dir
build-output-dir:
	mkdir -p $(TOOLS_BIN_DIR) || :

.PHONY: generate-latest-dev-csv
generate-latest-dev-csv: operator-sdk kustomize dist-csv-processor build-output-dir
	@echo Generating developer csv
	@echo
	OPERATOR_SDK=$(OPERATOR_SDK) KUSTOMIZE=$(KUSTOMIZE) FULL_OPERATOR_IMAGE="REPLACE_IMAGE" hack/csv-generate.sh -dev

.PHONY: generate-docs
generate-docs: dist-docs-generator
	hack/docs-generate.sh

.PHONY: generate-manifests-tree
generate-manifests-tree: generate-latest-dev-csv
	hack/generate-manifests-tree.sh "$(FULL_OPERATOR_IMAGE)"

.PHONY: generate-index-database
generate-index-database: bundle-container push-bundle-container
	BUNDLES="$(FULL_BUNDLE_IMAGE)" hack/generate-index-database.sh

.PHONY: generate-metadata
generate-metadata:
	./hack/generate-metadata.sh

.PHONY: deps-update
deps-update:
	go mod tidy && \
	go mod vendor

.PHONY: deploy
deploy: cluster-deploy
	# TODO - deprecated, will be removed soon in favor of cluster-deploy

.PHONY: cluster-deploy
cluster-deploy:
	@echo "Deploying operator"
	FULL_INDEX_IMAGE=$(FULL_INDEX_IMAGE) CLUSTER=$(CLUSTER) hack/deploy.sh

.PHONY: cluster-label-worker-cnf
cluster-label-worker-cnf:
	@echo "Adding worker-cnf label to worker nodes"
	hack/label-worker-cnf.sh

.PHONY: cluster-wait-for-mcp
cluster-wait-for-mcp:
    # NOTE: for CI this is done in the config suite of the functests!
    # Use this when deploying manifests manually with CLUSTER=manual
	@echo "Waiting for MCP to be updated"
	CLUSTER=$(CLUSTER) hack/wait-for-mcp.sh

.PHONY: cluster-clean
cluster-clean:
	@echo "Deleting operator"
	FULL_INDEX_IMAGE=$(FULL_INDEX_IMAGE) hack/clean-deploy.sh

.PHONY: functests
functests: cluster-label-worker-cnf functests-only

.PHONY: functests-only
functests-only:
	@echo "Cluster Version"
	hack/show-cluster-version.sh
	hack/run-functests.sh

.PHONY: functests-latency
functests-latency: cluster-label-worker-cnf
	GINKGO_SUITS="functests/0_config functests/4_latency" LATENCY_TEST_RUN="true" hack/run-functests.sh

.PHONY: functests-latency-testing
functests-latency-testing: dist-latency-tests
	GINKGO_SUITS="functests/0_config functests/5_latency_testing" hack/run-latency-testing.sh

.PHONY: operator-upgrade-tests
operator-upgrade-tests:
	@echo "Running Operator Upgrade Tests"
	hack/run-upgrade-tests.sh

.PHONY: perf-profile-creator-tests
perf-profile-creator-tests: create-performance-profile
	@echo "Running Performance Profile Creator Tests"
	hack/run-perf-profile-creator-functests.sh

.PHONY: render-command-tests
render-command-tests: dist
	@echo "Running Render Command Tests"
	hack/run-render-command-functests.sh

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
generate: clean deps-update gofmt manifests generate-code generate-latest-dev-csv generate-docs
	@echo Updating generated files
	@echo

.PHONY: verify
verify: golint govet generate
	@echo "Verifying that all code is committed after updating deps and formatting and generating code"
	hack/verify-generated.sh

.PHONY: ci-job
ci-job: verify build unittests

.PHONY: ci-tools-job
ci-tools-job:
	@echo "Cluster Version"
	hack/show-cluster-version.sh
	@echo "Verifying tools operation"
	hack/verify-tools.sh
	
.PHONY: release-note
release-note:
	hack/release-note.sh

.PHONY: generate-release-tags
generate-release-tags:
	hack/generate-release-tags.sh

########### Operator SDK Makefile contents

# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:crdVersions=v1"
CONTROLLER_GEN = "$(TOOLS_BIN_DIR)/controller-gen"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=performance-operator webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Generate code
generate-code: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# find or build controller-gen if necessary
controller-gen: build-output-dir
	@if [ ! -x $(CONTROLLER_GEN) ]; then\
		echo "Building $(CONTROLLER_GEN)";\
		env GOOS=$(TARGET_GOOS) GOARCH=$(TARGET_GOARCH) go build -ldflags="-s -w" -mod=vendor -o $(CONTROLLER_GEN) vendor/sigs.k8s.io/controller-tools/cmd/controller-gen/main.go ;\
	else \
		echo "Using pre-built $(CONTROLLER_GEN)";\
	fi

kustomize:
ifeq (, $(shell which kustomize))
	@{ \
	set -e ;\
	KUSTOMIZE_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$KUSTOMIZE_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/kustomize/kustomize/v3@v3.5.4 ;\
	rm -rf $$KUSTOMIZE_GEN_TMP_DIR ;\
	}
KUSTOMIZE=$(GOBIN)/kustomize
else
KUSTOMIZE=$(shell which kustomize)
endif
