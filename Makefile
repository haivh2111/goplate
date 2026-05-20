BINARY     := bin/goplate
PKG        := ./cmd/goplate
DIST       := dist
GOPATH_BIN := $(shell go env GOPATH)/bin

# Resolve a version string. Tag-derived if available (CI tagged builds), else
# git short SHA, else literal "dev". The release/snapshot targets override
# this for explicit tagging.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

# Linker flags: bake VERSION into the binary; strip symbols for size.
LDFLAGS := -s -w -X github.com/haivh2111/goplate/internal/version.Version=$(VERSION)

# Cross-compile matrix used by `release` / `snapshot`.
PLATFORMS := \
  darwin/amd64 darwin/arm64 \
  linux/amd64  linux/arm64 \
  windows/amd64 windows/arm64

# sha256 differs between macOS (shasum) and Linux (sha256sum). Pick one.
SHA256 := $(shell command -v sha256sum >/dev/null 2>&1 && echo "sha256sum" || echo "shasum -a 256")

.PHONY: build install test test-short vet lint tidy clean dist-clean release snapshot

# ── Local development ─────────────────────────────────────────
build:
	@mkdir -p bin
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) $(PKG)

install:
	go install -trimpath -ldflags "$(LDFLAGS)" $(PKG)
	@echo "installed → $(GOPATH_BIN)/goplate"

test:
	go test -race -count=1 ./...

test-short:
	go test -short ./...

vet:
	go vet ./...

lint:
	@command -v golangci-lint >/dev/null || { echo "golangci-lint not installed"; exit 1; }
	golangci-lint run ./...

tidy:
	go mod tidy

clean:
	rm -rf bin

dist-clean:
	rm -rf $(DIST)

# ── Cross-platform release builds ─────────────────────────────
# release: explicit-version build, intended for tagged releases.
#   Usage: make release VERSION=v1.0.0
release: dist-clean
	@$(MAKE) _matrix VERSION=$(VERSION)
	@$(MAKE) _checksums

# snapshot: dev-version build for local pipeline testing.
snapshot: dist-clean
	@$(MAKE) _matrix VERSION=$(VERSION)
	@$(MAKE) _checksums

# Internal target: build every (OS, ARCH) combo from PLATFORMS into ./dist/.
.PHONY: _matrix _checksums
_matrix:
	@mkdir -p $(DIST)
	@for platform in $(PLATFORMS); do \
	  os=$${platform%/*}; arch=$${platform#*/}; \
	  out_name=goplate; \
	  if [ "$$os" = "windows" ]; then out_name=goplate.exe; fi; \
	  staging=$(DIST)/_staging/$$os-$$arch; \
	  mkdir -p $$staging; \
	  echo "  ➜ building $$os/$$arch"; \
	  GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 \
	    go build -trimpath -ldflags "$(LDFLAGS)" -o $$staging/$$out_name $(PKG) \
	    || { echo "    build failed"; exit 1; }; \
	  archive=goplate_$(VERSION)_$${os}_$${arch}; \
	  if [ "$$os" = "windows" ]; then \
	    ( cd $$staging && zip -q ../../$$archive.zip $$out_name ); \
	    echo "    → $(DIST)/$$archive.zip"; \
	  else \
	    tar -czf $(DIST)/$$archive.tar.gz -C $$staging $$out_name; \
	    echo "    → $(DIST)/$$archive.tar.gz"; \
	  fi; \
	done
	@rm -rf $(DIST)/_staging

# Internal target: write SHA256 checksums for everything currently in ./dist/.
_checksums:
	@cd $(DIST) && $(SHA256) goplate_*.tar.gz goplate_*.zip > checksums.txt 2>/dev/null || true
	@echo "  ✓ wrote $(DIST)/checksums.txt"
