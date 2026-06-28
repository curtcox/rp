# rp developer tasks.
#
# These targets are the single source of truth for tests, linting, and static
# analysis. CI (.github/workflows/ci.yml) invokes the same targets so local and
# CI behavior match.
#
# Quick start:
#   make tools     # install pinned analysis tools into $(GOBIN)
#   make check     # the gating suite: tests + gofmt + vet + lint
#   make reports   # build the HTML report site under ./site
#   make help      # list targets

# Pinned tool versions (keep in sync with the CI workflow).
GOLANGCI_LINT_VERSION ?= v2.12.2
GOCYCLO_VERSION       ?= v0.6.0
GOCOGNIT_VERSION      ?= v1.2.1

# Complexity thresholds (informational; see `make complexity`).
GOCYCLO_OVER  ?= 15
GOCOGNIT_OVER ?= 20

GOBIN     := $(shell go env GOPATH)/bin
SITE_DIR  ?= site
COVERPROFILE := coverage.out

# Resolve tools from GOBIN, falling back to PATH.
GOLANGCI_LINT := $(shell command -v golangci-lint 2>/dev/null || echo $(GOBIN)/golangci-lint)
GOCYCLO       := $(shell command -v gocyclo 2>/dev/null || echo $(GOBIN)/gocyclo)
GOCOGNIT      := $(shell command -v gocognit 2>/dev/null || echo $(GOBIN)/gocognit)

.DEFAULT_GOAL := help

## help: list available targets
.PHONY: help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /' | sort

## tools: install pinned analysis tools into GOBIN
.PHONY: tools
tools:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	go install github.com/fzipp/gocyclo/cmd/gocyclo@$(GOCYCLO_VERSION)
	go install github.com/uudashr/gocognit/cmd/gocognit@$(GOCOGNIT_VERSION)

## build: compile all packages
.PHONY: build
build:
	go build ./...

## test: run the test suite
.PHONY: test
test:
	go test ./...

## test-race: run the test suite with the race detector
.PHONY: test-race
test-race:
	go test -race ./...

## doctest: run the runnable examples embedded in docs/ (see docs/README.md)
.PHONY: doctest
doctest:
	go test ./cmd/rp -run TestDocExamples -v

## fmt: format all Go code in place
.PHONY: fmt
fmt:
	gofmt -s -w .

## fmt-check: fail if any Go code is not gofmt-clean (gating)
.PHONY: fmt-check
fmt-check:
	@unformatted=$$(gofmt -s -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "These files are not gofmt-clean:"; echo "$$unformatted"; \
		echo "Run 'make fmt' to fix."; exit 1; \
	fi
	@echo "gofmt: clean"

## vet: run go vet (gating)
.PHONY: vet
vet:
	go vet ./...

## lint: run golangci-lint using .golangci.yml (gating)
.PHONY: lint
lint:
	$(GOLANGCI_LINT) run ./...

## coverage: write coverage profile and total (informational)
.PHONY: coverage
coverage:
	go test -covermode=atomic -coverprofile=$(COVERPROFILE) ./...
	@go tool cover -func=$(COVERPROFILE) | tail -1

## complexity: report cyclomatic & cognitive complexity (informational)
.PHONY: complexity
complexity:
	@echo "== Cyclomatic complexity (gocyclo, over $(GOCYCLO_OVER)) =="
	@$(GOCYCLO) -over $(GOCYCLO_OVER) -avg . || true
	@echo
	@echo "== Cognitive complexity (gocognit, over $(GOCOGNIT_OVER)) =="
	@$(GOCOGNIT) -over $(GOCOGNIT_OVER) -avg . || true

## check: the gating suite run by CI (tests + gofmt + vet + lint)
# `test` runs `go test ./...`, which includes the docs doctest harness
# (TestDocExamples); a drifted runnable example fails the gate. `make doctest`
# runs just that harness, verbosely, for a focused loop.
.PHONY: check
check: fmt-check vet lint test
	@echo "All gating checks passed."

## reports: build the static HTML report site under $(SITE_DIR)
.PHONY: reports
reports:
	SITE_DIR=$(SITE_DIR) \
	COVERPROFILE=$(COVERPROFILE) \
	GOCYCLO_OVER=$(GOCYCLO_OVER) \
	GOCOGNIT_OVER=$(GOCOGNIT_OVER) \
	GOLANGCI_LINT="$(GOLANGCI_LINT)" \
	GOCYCLO="$(GOCYCLO)" \
	GOCOGNIT="$(GOCOGNIT)" \
	./scripts/gen-site.sh

## clean: remove generated artifacts
.PHONY: clean
clean:
	rm -rf $(SITE_DIR) $(COVERPROFILE)
