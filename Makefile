# ai-defense-cli — build orchestration
#
# Two CLI build modes:
#   go    — Go only (stub embedded Python; ai-bom needs AID_CLI_PYTHON or fails clearly)
#   full  — Go + relocatable CPython + cisco-aibom embedded (release binaries)
#
# Examples:
#   make build              # same as make build-go
#   make build BUILD_MODE=full
#   make build-go test vet
#   make bundle             # build Python embed artifacts (required before build-embed)
#   make build-embed

.DEFAULT_GOAL := help

BINARY_NAME   ?= cisco-aid-cli
BIN_DIR       ?= bin
MODULE        ?= github.com/cisco-ai-defense/ai-defense-cli
BUILD_MODE    ?= go

VERSION_FILE  := $(shell test -f VERSION && tr -d '[:space:]' < VERSION)
VERSION       ?= $(if $(VERSION_FILE),$(VERSION_FILE),$(shell git describe --tags --always --dirty 2>/dev/null || echo dev))
COMMIT        ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
DIST_PREFIX   := cisco-aid-cli-$(VERSION)

GO            ?= go
GORELEASER    ?= goreleaser
GOOS          ?= $(shell $(GO) env GOOS)
GOARCH        ?= $(shell $(GO) env GOARCH)
CGO_ENABLED   ?= 0

# Packages under scripts/, test helpers, and main are excluded from coverage measurement.
# TUI/onboarding surfaces are also excluded from the coverage gate until they have unit tests.
GO_COVER_PKGS := $(shell $(GO) list ./... 2>/dev/null | grep -vE '/scripts/|/internal/testutil/|^$(MODULE)$$|/cmd/|/internal/tui|/internal/selfupdate|/internal/cli/config|/internal/cli/install|/internal/cli/onboard|/internal/config/spec')
COVERPROFILE  ?= coverage.out
COVER_MIN     ?= 75
DIFF_BASE     ?= origin/main
DIFF_COVER_MIN ?= 75

EMBED_PKG     := ./internal/embedded/python
EMBED_BIN_DIR := internal/embedded/python/bin
BUNDLE_TAR    := $(EMBED_BIN_DIR)/aid-py-bundle.tar.gz
BUNDLE_ID     := $(EMBED_BIN_DIR)/aid-py-bundle.id

LDFLAGS := -s -w \
	-X $(MODULE)/internal/cli.version=$(VERSION) \
	-X $(MODULE)/internal/cli.commit=$(COMMIT) \

.PHONY: help version build build-go build-embed bundle bundle-force bundle-check test test-cover test-cover-check test-diff-cover-check vet fmt clean build-release build-release-linux build-release-darwin build-release-all checksums manifest clean-dist smoke-test smoke-test-bin gen-exit-docs check-exit-docs goreleaser-check

version: ## Print VERSION and COMMIT (from VERSION file + git)
	@printf 'VERSION=%s\nCOMMIT=%s\n' "$(VERSION)" "$(COMMIT)"

goreleaser-check: ## Validate .goreleaser.yaml (requires GoReleaser Pro v2.17.1; no key needed)
	@command -v $(GORELEASER) >/dev/null || \
		(echo "goreleaser-pro not found — install v2.17.1 from https://goreleaser.com/getting-started/install/pro/" && exit 1)
	@version_output="$$($(GORELEASER) --version 2>&1)"; \
		printf '%s\n' "$$version_output" | grep -q 'goreleaser-pro:' || \
			(echo "GoReleaser Pro is required" && exit 1); \
		printf '%s\n' "$$version_output" | grep -q 'GitVersion:[[:space:]]*2\.17\.1$$' || \
			(echo "GoReleaser Pro v2.17.1 is required" && exit 1)
	$(GORELEASER) check

help: ## Show targets
	@printf "ai-defense-cli Makefile\n\n"
	@printf "Build modes (BUILD_MODE):\n"
	@printf "  go   — Go CLI only; embedded Python stub (!embed_python)\n"
	@printf "  full — Go + embedded CPython/aibom (-tags embed_python); run 'make bundle' first\n\n"
	@grep -E '^[a-zA-Z0-9_.-]+:.*##' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*## "}; {printf "  %-16s %s\n", $$1, $$2}'

build: ## Build CLI (BUILD_MODE=go|full, default go)
ifeq ($(BUILD_MODE),full)
	$(MAKE) build-embed
else ifeq ($(BUILD_MODE),go)
	$(MAKE) build-go
else
	$(error BUILD_MODE must be 'go' or 'full', got '$(BUILD_MODE)')
endif

build-go: ## Build Go-only CLI (no embedded Python bundle)
	@test -f go.mod || (echo "go.mod not found — migrate CLI source first" && exit 1)
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build \
		-ldflags "$(LDFLAGS)" \
		-o $(BIN_DIR)/$(BINARY_NAME) .
	@printf "Built $(BIN_DIR)/$(BINARY_NAME) (go-only / stub python)\n"

build-embed: bundle-check ## Build CLI with embedded CPython + cisco-aibom
	@test -f go.mod || (echo "go.mod not found — migrate CLI source first" && exit 1)
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build \
		-tags embed_python \
		-ldflags "$(LDFLAGS)" \
		-o $(BIN_DIR)/$(BINARY_NAME) .
	@printf "Built $(BIN_DIR)/$(BINARY_NAME) (embedded python + aibom)\n"

bundle: ## Build relocatable CPython + cisco-aibom embed artifacts (skips when inputs unchanged)
	@test -f scripts/build-python-bundle.sh || \
		(echo "scripts/build-python-bundle.sh missing" && exit 1)
	@chmod +x scripts/build-python-bundle.sh
	./scripts/build-python-bundle.sh

bundle-force: ## Rebuild Python embed artifacts (ignore input fingerprint cache)
	BUNDLE_FORCE=1 $(MAKE) bundle

bundle-check:
	@test -f $(BUNDLE_TAR) || (echo "Missing $(BUNDLE_TAR) — run: make bundle" && exit 1)
	@test -f $(BUNDLE_ID) || (echo "Missing $(BUNDLE_ID) — run: make bundle" && exit 1)

test: ## Run tests (stub python; no embed bundle required)
	@test -f go.mod || (echo "go.mod not found — migrate CLI source first" && exit 1)
	$(GO) test ./...

test-cover: ## Run tests and write coverage profile (COVERPROFILE=coverage.out)
	@test -f go.mod || (echo "go.mod not found — migrate CLI source first" && exit 1)
	$(GO) test $(GO_COVER_PKGS) -coverprofile=$(COVERPROFILE) -covermode=atomic
	@$(GO) tool cover -func=$(COVERPROFILE) | tail -1

test-cover-check: test-cover ## Enforce minimum coverage (COVER_MIN=75)
	@chmod +x scripts/check-coverage.sh
	COVER_MIN=$(COVER_MIN) ./scripts/check-coverage.sh $(COVERPROFILE)

test-diff-cover-check: ## Enforce diff coverage on changed lines (DIFF_BASE=origin/main, DIFF_COVER_MIN=75; uses existing coverage.out)
	@test -f $(COVERPROFILE) || $(MAKE) test-cover
	$(GO) run ./cmd/check-diff-coverage --profile $(COVERPROFILE) --base $(DIFF_BASE) --min $(DIFF_COVER_MIN)

smoke-test: build-go ## Post-build smoke: --version and --help for all commands (bin/cisco-aid-cli)
	@$(MAKE) smoke-test-bin BIN="$(BIN_DIR)/$(BINARY_NAME)" VERSION="$(VERSION)"

smoke-test-bin: ## Post-build smoke on BIN= path (SMOKE_RUNNER=direct|docker, optional VERSION=)
	@test -n "$(BIN)" || (echo "BIN required, e.g. make smoke-test-bin BIN=dist/cisco-aid-cli-..." && exit 1)
	@test -f scripts/smoke-test-cli.sh || (echo "scripts/smoke-test-cli.sh missing" && exit 1)
	@chmod +x scripts/smoke-test-cli.sh
	VERSION="$(VERSION)" SMOKE_RUNNER="$(SMOKE_RUNNER)" ./scripts/smoke-test-cli.sh "$(BIN)"

vet: ## Run go vet
	@test -f go.mod || (echo "go.mod not found — migrate CLI source first" && exit 1)
	$(GO) vet ./...

gen-exit-docs: ## Regenerate docs/exit-codes-handbook.md from internal/cliexitdoc
	@test -f go.mod || (echo "go.mod not found" && exit 1)
	$(GO) run ./cmd/gen-exit-docs -o docs/exit-codes-handbook.md

check-exit-docs: ## Fail if exit-codes handbook is out of date
	@test -f go.mod || (echo "go.mod not found" && exit 1)
	@$(GO) run ./cmd/gen-exit-docs -o docs/exit-codes-handbook.md.generated
	@diff -u docs/exit-codes-handbook.md docs/exit-codes-handbook.md.generated || \
		(rm -f docs/exit-codes-handbook.md.generated && \
		 echo "exit-codes handbook is stale — run: make gen-exit-docs" && exit 1)
	@rm -f docs/exit-codes-handbook.md.generated
	@echo "exit-codes handbook is up to date"

fmt: ## gofmt all Go files
	@gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

clean: ## Remove build outputs
	rm -rf $(BIN_DIR)/$(BINARY_NAME)
	rm -rf $(EMBED_BIN_DIR)/aid-py-bundle.tar.gz $(EMBED_BIN_DIR)/aid-py-bundle.id
	rm -f .build/aid-py-bundle.fingerprint
	$(GO) clean -cache -testcache 2>/dev/null || true

DIST_DIR      ?= dist
DOCKER        ?= docker
PY_BUILD_IMAGE  ?= debian:bookworm-slim
GO_BUILD_IMAGE  ?= golang:1.23-bookworm


build-release: ## Release binary for host OS (linux→Docker, darwin→native embed)
ifeq ($(GOOS),linux)
	$(MAKE) build-release-linux GOARCH=$(GOARCH)
else ifeq ($(GOOS),darwin)
	$(MAKE) build-release-darwin GOARCH=$(GOARCH)
else
	$(error build-release unsupported GOOS=$(GOOS); use build-release-linux or build-release-darwin)
endif

# PBS pins per linux arch (python-build-standalone 20250723 / cpython 3.12.11)
PBS_TRIPLE_amd64  := x86_64-unknown-linux-gnu
PBS_SHA256_amd64  := 7263ed87e9d2d4c52b01f1ad8cdca315cf859c0848b8ffdcb70de205fa8b0336
# arm64 Linux PBS pins kept for optional local builds (not published in CI/releases).
PBS_TRIPLE_arm64  := aarch64-unknown-linux-gnu
PBS_SHA256_arm64  := 32b17b62fd621481bb4eab0ed5549cbad43d88780e29fc2c0a222a1722470c4b

build-release-linux: ## Linux release binary via Docker (embed_python) → dist/
	@mkdir -p $(DIST_DIR)
	$(DOCKER) build --platform linux/$(GOARCH) --target release \
	  --build-arg PY_BUILD_IMAGE=$(PY_BUILD_IMAGE) \
	  --build-arg GO_BUILD_IMAGE=$(GO_BUILD_IMAGE) \
	  --build-arg TARGETARCH=$(GOARCH) \
	  --build-arg PBS_TRIPLE=$(PBS_TRIPLE_$(GOARCH)) \
	  --build-arg PBS_SHA256=$(PBS_SHA256_$(GOARCH)) \
	  --build-arg CLI_VERSION=$(VERSION) \
	  --build-arg CLI_COMMIT=$(COMMIT) \
	  --output type=local,dest=$(DIST_DIR)/ .
	@mv -f $(DIST_DIR)/cisco-aid-cli $(DIST_DIR)/$(DIST_PREFIX)-linux-$(GOARCH)
	@printf "Built $(DIST_DIR)/$(DIST_PREFIX)-linux-$(GOARCH)\n"

build-release-darwin: bundle-check ## macOS release binary (native embed) → dist/
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=darwin GOARCH=$(GOARCH) $(GO) build \
		-tags embed_python \
		-ldflags "$(LDFLAGS)" \
		-o $(DIST_DIR)/$(DIST_PREFIX)-darwin-$(GOARCH) .
	@printf "Built $(DIST_DIR)/$(DIST_PREFIX)-darwin-$(GOARCH)\n"

build-release-all: ## Build linux amd64 (Docker); darwin if on macOS host
	$(MAKE) build-release-linux GOARCH=amd64
ifeq ($(shell uname -s),Darwin)
	$(MAKE) build-release-darwin GOARCH=arm64
ifneq ($(shell uname -m),arm64)
	$(MAKE) build-release-darwin GOARCH=amd64
endif
endif

checksums: ## Generate dist/checksums.txt (sha256) for versioned release binaries
	@command -v shasum >/dev/null || (echo "shasum required" && exit 1)
	@mkdir -p $(DIST_DIR)
	@cd $(DIST_DIR) && \
		files=$$(ls $(DIST_PREFIX)-* 2>/dev/null) && \
		test -n "$$files" || (echo "no release binaries matching $(DIST_PREFIX)-* in $(DIST_DIR)" && exit 1) && \
		shasum -a 256 $(DIST_PREFIX)-* > checksums.txt
	@printf "Wrote $(DIST_DIR)/checksums.txt\n"

manifest: checksums ## Generate dist/manifest.json (version, commit, per-arch sha256)
	@VERSION="$(VERSION)" COMMIT="$(COMMIT)" BUILD_DATE="$(BUILD_DATE)" DIST_DIR="$(DIST_DIR)" \
		./scripts/generate-release-manifest.sh
	@printf "Wrote $(DIST_DIR)/manifest.json\n"

clean-dist: ## Remove dist/ release artifacts
	rm -rf $(DIST_DIR) .goreleaser-dist
