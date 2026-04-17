.DEFAULT_GOAL := help

GO       ?= go
PKG      := ./...
COVERAGE := coverage.out

.PHONY: help test test-race cover bench fmt fmt-check vet tidy vuln ci

help: ## Show this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

test: ## Run unit tests
	$(GO) test -count=1 $(PKG)

test-race: ## Run unit tests with the race detector
	$(GO) test -race -count=1 $(PKG)

cover: ## Produce coverage.out and open an HTML report
	$(GO) test -count=1 -coverprofile=$(COVERAGE) $(PKG)
	$(GO) tool cover -html=$(COVERAGE)

bench: ## Run benchmarks
	$(GO) test -bench=. -benchmem -run=^$$ $(PKG)

fmt: ## Format Go source with gofmt and goimports
	gofmt -w .
	goimports -w .

fmt-check: ## Fail if gofmt would change any file
	@out=$$(gofmt -l .); \
	if [ -n "$$out" ]; then \
		echo "gofmt needed on:"; echo "$$out"; exit 1; \
	fi

vet: ## Run go vet
	$(GO) vet $(PKG)

tidy: ## Run go mod tidy
	$(GO) mod tidy

vuln: ## Run govulncheck
	govulncheck $(PKG)

ci: fmt-check vet test-race vuln ## Run the full CI gate
