# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint

# Binary name
BINARY_NAME=terminal-ai
BINARY_UNIX=$(BINARY_NAME)_unix
BINARY_WINDOWS=$(BINARY_NAME).exe

# Build directory
BUILD_DIR=./build

# Version information
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.Commit=$(COMMIT)"

# Colors for output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[0;33m
NC=\033[0m # No Color

.PHONY: all build test clean run install fmt lint help deps build-linux build-windows build-mac

# Default target
all: test build

## help: Show this help message
help:
	@echo "Available targets:"
	@grep -E '^##' Makefile | sed 's/## /  /'

## build: Build the binary for the current platform
build:
	@echo "$(GREEN)Building $(BINARY_NAME)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) -v .
	@echo "$(GREEN)Build complete: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

## build-all: Build for all platforms
build-all: build-linux build-windows build-mac

## build-linux: Build for Linux
build-linux:
	@echo "$(GREEN)Building for Linux...$(NC)"
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)_linux_amd64 -v .
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)_linux_arm64 -v .

## build-windows: Build for Windows
build-windows:
	@echo "$(GREEN)Building for Windows...$(NC)"
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_WINDOWS) -v .

## build-mac: Build for macOS
build-mac:
	@echo "$(GREEN)Building for macOS...$(NC)"
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)_darwin_amd64 -v .
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)_darwin_arm64 -v .

## test: Run tests
test:
	@echo "$(GREEN)Running tests...$(NC)"
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	@echo "$(GREEN)Tests complete$(NC)"

## test-coverage: Run tests with coverage report
test-coverage: test
	@echo "$(GREEN)Generating coverage report...$(NC)"
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

## clean: Clean build artifacts
clean:
	@echo "$(YELLOW)Cleaning...$(NC)"
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	@echo "$(GREEN)Clean complete$(NC)"

## install: Install the binary to $GOPATH/bin
install: build
	@echo "$(GREEN)Installing $(BINARY_NAME)...$(NC)"
	$(GOCMD) install $(LDFLAGS) .
	@echo "$(GREEN)Installation complete$(NC)"

## uninstall: Remove the binary from $GOPATH/bin
uninstall:
	@echo "$(YELLOW)Uninstalling $(BINARY_NAME)...$(NC)"
	rm -f $(GOPATH)/bin/$(BINARY_NAME)
	@echo "$(GREEN)Uninstallation complete$(NC)"

## run: Run the application
run:
	@echo "$(GREEN)Running $(BINARY_NAME)...$(NC)"
	$(GOCMD) run $(LDFLAGS) . $(ARGS)

## fmt: Format code
fmt:
	@echo "$(GREEN)Formatting code...$(NC)"
	$(GOFMT) -s -w .
	$(GOCMD) fmt ./...
	@echo "$(GREEN)Formatting complete$(NC)"

## lint: Run linters
lint:
	@echo "$(GREEN)Running linters...$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		$(GOLINT) run ./...; \
	else \
		echo "$(YELLOW)golangci-lint not installed. Install it with:$(NC)"; \
		echo "  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin"; \
		echo "$(YELLOW)Running basic go vet instead...$(NC)"; \
		$(GOCMD) vet ./...; \
	fi
	@echo "$(GREEN)Linting complete$(NC)"

## deps: Download dependencies
deps:
	@echo "$(GREEN)Downloading dependencies...$(NC)"
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "$(GREEN)Dependencies updated$(NC)"

## deps-update: Update dependencies to latest versions
deps-update:
	@echo "$(GREEN)Updating dependencies...$(NC)"
	$(GOGET) -u ./...
	$(GOMOD) tidy
	@echo "$(GREEN)Dependencies updated to latest versions$(NC)"

## verify: Verify dependencies
verify:
	@echo "$(GREEN)Verifying dependencies...$(NC)"
	$(GOMOD) verify
	@echo "$(GREEN)Verification complete$(NC)"

## docker-build: Build Docker image
docker-build:
	@echo "$(GREEN)Building Docker image...$(NC)"
	docker build -t $(BINARY_NAME):$(VERSION) .
	@echo "$(GREEN)Docker image built: $(BINARY_NAME):$(VERSION)$(NC)"

## docker-run: Run Docker container
docker-run:
	@echo "$(GREEN)Running Docker container...$(NC)"
	docker run --rm -it $(BINARY_NAME):$(VERSION)

# Development shortcuts
.PHONY: dev
## dev: Run with hot reload (requires air)
dev:
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "$(YELLOW)air not installed. Install it with:$(NC)"; \
		echo "  go install github.com/cosmtrek/air@latest"; \
		echo "$(YELLOW)Running without hot reload...$(NC)"; \
		$(MAKE) run; \
	fi

.PHONY: check
## check: Run fmt, lint, and test
check: fmt lint test
	@echo "$(GREEN)All checks passed!$(NC)"

# Show version
.PHONY: version
## version: Show version information
version:
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"