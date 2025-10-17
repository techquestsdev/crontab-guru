# Makefile for crontab guru TUI

.PHONY: help build run test test-verbose test-coverage lint fmt clean install check all

# Default target
.DEFAULT_GOAL := help

# Binary name
BINARY_NAME=crontab-guru
BINARY_PATH=./bin/$(BINARY_NAME)

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint

help: ## Display this help message
	@echo "Available targets:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

##@ Development

run: ## Run the application
	@echo "Running application..."
	@$(GORUN) .

build: ## Build the binary
	@echo "Building binary..."
	@mkdir -p bin
	@$(GOBUILD) -o $(BINARY_PATH) -v .
	@echo "Binary created at $(BINARY_PATH)"

install: ## Install the binary to GOPATH/bin
	@echo "Installing binary..."
	@$(GOCMD) install .
	@echo "Binary installed to $(shell go env GOPATH)/bin/$(BINARY_NAME)"

##@ Testing

test: ## Run tests
	@echo "Running tests..."
	@$(GOTEST) -v -race ./...

test-verbose: ## Run tests with verbose output
	@echo "Running tests (verbose)..."
	@$(GOTEST) -v -race -count=1 ./...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@$(GOTEST) -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-coverage-view: test-coverage ## Run tests with coverage and open HTML report
	@echo "Opening coverage report in browser..."
	@open coverage.html || xdg-open coverage.html 2>/dev/null || echo "Please open coverage.html manually"

bench: ## Run benchmarks
	@echo "Running benchmarks..."
	@$(GOTEST) -bench=. -benchmem ./...

##@ Code Quality

lint: ## Run linter
	@echo "Running golangci-lint..."
	@$(GOLINT) run

lint-fix: ## Run linter with auto-fix
	@echo "Running golangci-lint with auto-fix..."
	@$(GOLINT) run --fix

fmt: ## Format code
	@echo "Formatting code..."
	@$(GOFMT) -s -w .
	@echo "Code formatted"

vet: ## Run go vet
	@echo "Running go vet..."
	@$(GOCMD) vet ./...

check: fmt vet lint test ## Run all checks (fmt, vet, lint, test)
	@echo "All checks passed!"

##@ Dependencies

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@$(GOMOD) download
	@$(GOMOD) tidy

deps-upgrade: ## Upgrade dependencies
	@echo "Upgrading dependencies..."
	@$(GOGET) -u ./...
	@$(GOMOD) tidy

deps-verify: ## Verify dependencies
	@echo "Verifying dependencies..."
	@$(GOMOD) verify

##@ Cleanup

clean: ## Remove build artifacts and coverage reports
	@echo "Cleaning up..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@rm -f $(BINARY_NAME)
	@echo "Cleanup complete"

clean-all: clean ## Remove build artifacts, coverage reports, and Go cache
	@echo "Cleaning Go cache..."
	@$(GOCMD) clean -cache -testcache -modcache
	@echo "Full cleanup complete"

##@ Utilities

mod-graph: ## Show module dependency graph
	@$(GOMOD) graph

mod-why: ## Show why a package is needed (usage: make mod-why PKG=github.com/pkg/name)
	@$(GOMOD) why $(PKG)

version: ## Show Go version
	@$(GOCMD) version

all: fmt lint test build ## Run fmt, lint, test, and build
	@echo "Build complete!"
