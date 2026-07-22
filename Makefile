# Use bash with strict flags.
SHELL       := bash
.SHELLFLAGS := -o pipefail -euc

# ---- variables -------------------------------------------------------------
BINARY   := flatcar-kit
IMAGE    ?= ghcr.io/jacobweinstock/flatcar-kit
VERSION  ?= $(shell git describe --tags --always --dirty)

# Host defaults; override to cross compile, e.g. `make build GOARCH=arm64`.
GOOS     ?= $(shell go env GOOS)
GOARCH   ?= $(shell go env GOARCH)

# Machine architecture used to pin the downloaded golangci-lint binary.
LINT_ARCH := $(shell uname -m)

# Platforms to cross compile binaries for (`make build-all`).
PLATFORMS ?= linux/amd64,linux/arm64

OUT_DIR  := out
BINARY_OUT := $(OUT_DIR)/$(BINARY)-$(GOOS)-$(GOARCH)

GOLANGCI_LINT_VERSION ?= v2.11.2
GORELEASER_VERSION ?= v2.17.0

# Static, stripped binary so it runs on the butane (Fedora) base without libc deps.
GO_BUILD_FLAGS := -trimpath -ldflags="-s -w"

# BUILD_IN_CONTAINER=true compiles the binary inside the Docker build instead of
# on the host. When false (default), the image copies prebuilt binaries.
BUILD_IN_CONTAINER ?= false
ifeq ($(BUILD_IN_CONTAINER),true)
IMAGE_BINARY_DEP :=
else
IMAGE_BINARY_DEP := build
endif

# Used to split the comma-separated PLATFORMS list.
COMMA := ,

.DEFAULT_GOAL := help

# ---- help ------------------------------------------------------------------
.PHONY: help
help: ## Print this help.
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make \033[36m<target>\033[0m\n"} /^[%\/0-9A-Za-z_-]+:.*?##/ { printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

# ---- go --------------------------------------------------------------------
# Go sources; a binary is rebuilt only when one of these is newer than it, so an
# already-present binary (e.g. a downloaded CI artifact) is reused as-is.
GO_SOURCES := $(shell find . -type f -name '*.go') go.mod go.sum

# Per-platform binary, e.g. out/flatcar-kit-linux-amd64.
$(OUT_DIR)/$(BINARY)-%: $(GO_SOURCES)
	@mkdir -p $(@D)
	CGO_ENABLED=0 GOOS=$(word 1,$(subst -, ,$*)) GOARCH=$(word 2,$(subst -, ,$*)) \
		go build $(GO_BUILD_FLAGS) -o $@ .

.PHONY: build
build: $(BINARY_OUT) ## Build the Go binary into out/ (respects GOOS/GOARCH).

# Binaries for every platform in PLATFORMS (linux/amd64 -> out/flatcar-kit-linux-amd64).
PLATFORM_BINS := $(foreach p,$(subst $(COMMA), ,$(PLATFORMS)),$(OUT_DIR)/$(BINARY)-$(subst /,-,$(p)))

.PHONY: build-all
build-all: $(PLATFORM_BINS) ## Build a binary for every platform in PLATFORMS.

.PHONY: test
test: ## Run Go tests with coverage.
	go test -race -covermode=atomic -coverprofile=coverage.out ./...

# golangci-lint binary, downloaded once via the official installer and pinned by
# version + arch under out/linters/ (mirrors the tinkerbell repo's Makefile).
GOLANGCI_LINT_CONFIG := .golangci.yml
GOLANGCI_LINT_BIN := $(OUT_DIR)/linters/golangci-lint-$(GOLANGCI_LINT_VERSION)-$(LINT_ARCH)
$(GOLANGCI_LINT_BIN):
	mkdir -p $(OUT_DIR)/linters
	rm -rf $(OUT_DIR)/linters/golangci-lint-*
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(OUT_DIR)/linters $(GOLANGCI_LINT_VERSION)
	mv $(OUT_DIR)/linters/golangci-lint $@

.PHONY: lint
lint: $(GOLANGCI_LINT_BIN) ## Run golangci-lint.
	$(GOLANGCI_LINT_BIN) run -c $(GOLANGCI_LINT_CONFIG) ./...

.PHONY: clean
clean: ## Remove build artifacts.
	rm -rf $(OUT_DIR)/$(BINARY)-* $(OUT_DIR)/linters $(OUT_DIR)/tools coverage.out

# ---- container image -------------------------------------------------------
.PHONY: image
image: $(IMAGE_BINARY_DEP) ## Build the container image for the host arch and load it into docker.
	docker build \
		--build-arg BUILD_IN_CONTAINER=$(BUILD_IN_CONTAINER) \
		--build-arg TARGETOS=$(GOOS) \
		--build-arg TARGETARCH=$(GOARCH) \
		-t $(IMAGE):$(VERSION) \
		-t $(IMAGE):latest \
		.

# ---- release (GoReleaser) --------------------------------------------------
# goreleaser binary, downloaded once and pinned by version + arch under
# out/tools/ (mirrors the golangci-lint download above).
GORELEASER_OS   := $(shell uname -s)
GORELEASER_ARCH := $(shell uname -m | sed 's/aarch64/arm64/')
GORELEASER_BIN  := $(OUT_DIR)/tools/goreleaser-$(GORELEASER_VERSION)-$(LINT_ARCH)
$(GORELEASER_BIN):
	mkdir -p $(OUT_DIR)/tools
	rm -rf $(OUT_DIR)/tools/goreleaser-*
	curl -sSfL https://github.com/goreleaser/goreleaser/releases/download/$(GORELEASER_VERSION)/goreleaser_$(GORELEASER_OS)_$(GORELEASER_ARCH).tar.gz \
		| tar -xz -C $(OUT_DIR)/tools goreleaser
	mv $(OUT_DIR)/tools/goreleaser $@

# GoReleaser owns publishing: multi-arch image, binary archives, checksums,
# changelog and the GitHub Release. Tag pushes trigger it in CI (see
# .github/workflows/release.yaml); these targets are for local runs.
.PHONY: release
release: $(GORELEASER_BIN) ## Build and publish a release with GoReleaser (needs a tag + GITHUB_TOKEN).
	$(GORELEASER_BIN) release --clean

.PHONY: release-snapshot
release-snapshot: $(GORELEASER_BIN) ## Build a local snapshot release without publishing.
	$(GORELEASER_BIN) release --snapshot --clean

.PHONY: release-check
release-check: $(GORELEASER_BIN) ## Validate the GoReleaser configuration.
	$(GORELEASER_BIN) check
