BINARY := aitinerary
PKG    := ./cmd/aitinerary
PREFIX ?= $(HOME)/.local

# Version stamping: prefer a git tag/describe, fall back to "dev". Injected
# into main.version via -ldflags so `aitinerary --version` names the build.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)

# Demo / VHS plumbing. AITINERARY_CONFIG_DIR is the per-recording sandbox the
# demo binary uses for config; the fixture script pre-seeds it so the GIFs
# render the same shape every time.
DEMO_DIR    := $(CURDIR)/tmp/demo
TAPES       := $(wildcard vhs/*.tape)
GIF_TARGETS := $(patsubst vhs/%.tape,gif-%,$(TAPES))

.PHONY: build install run tidy fmt vet test test-race cover lint clean demo \
        vhs-fixtures screenshots clean-screenshots

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(PKG)

install:
	go install -ldflags "$(LDFLAGS)" $(PKG)

run:
	go run -ldflags "$(LDFLAGS)" $(PKG)

tidy:
	go mod tidy

fmt:
	gofmt -s -w .

vet:
	go vet ./...

test:
	go test ./...

# test-race — run the suite under the race detector (mirrors CI).
test-race:
	go test -race ./...

# cover — run the suite with coverage reporting (mirrors CI).
cover:
	go test -cover ./...

# lint — run golangci-lint over the module. Install it with:
#   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
# (or `brew install golangci-lint`). CI runs the same linter.
lint:
	@command -v golangci-lint >/dev/null 2>&1 || { \
	  echo "error: golangci-lint not found on PATH."; \
	  echo "install it with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	  exit 1; \
	}
	golangci-lint run ./...

clean:
	rm -f $(BINARY)
	rm -rf dist bin

# demo — build and launch against a freshly-seeded fixture dir.
demo: build vhs-fixtures
	AITINERARY_CONFIG_DIR=$(DEMO_DIR) ./$(BINARY) --demo

# vhs-fixtures — re-seed $(DEMO_DIR) with canned demo data. Idempotent.
vhs-fixtures:
	AITINERARY_CONFIG_DIR=$(DEMO_DIR) bash scripts/setup-demo-fixtures.sh

# screenshots — regenerate every GIF under screenshots/ from vhs/ tapes.
# Requires: brew install vhs (https://github.com/charmbracelet/vhs).
screenshots: build vhs-fixtures $(GIF_TARGETS)

gif-%: vhs/%.tape build vhs-fixtures
	@command -v vhs >/dev/null 2>&1 || { \
	  echo "error: vhs not found on PATH; install with 'brew install vhs'"; \
	  exit 1; \
	}
	mkdir -p screenshots
	AITINERARY_CONFIG_DIR=$(DEMO_DIR) vhs $<

clean-screenshots:
	rm -rf screenshots tmp/demo
