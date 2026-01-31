# gopherutils Makefile
# Automatically discovers and builds all utilities

# Variables
BINARY_DIR := build
UTILS_DIR := utils
GO := go
GOFLAGS := -ldflags="-s -w"

# Installation prefix
INSTALL_PREFIX ?= $(HOME)/.local
BINDIR := $(INSTALL_PREFIX)/bin

# Find all directories containing main.go files under utils/
# We use 'shell' to find them.
UTIL_DIRS := $(shell find $(UTILS_DIR) -name "main.go" -exec dirname {} \;)

# Extract binary names from directory names (e.g., utils/file/ls -> ls)
BINARIES := $(foreach dir,$(UTIL_DIRS),$(BINARY_DIR)/$(notdir $(dir)))

# Download and install Go dependencies
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	@$(GO) mod download
	@$(GO) mod tidy

# Default target
.PHONY: all
all: deps $(BINARIES)

# Alias build to all
.PHONY: build
build: all

# Template for building a single utility
# Arguments:
#   1: Source directory path (e.g., utils/file/ls)
define BUILD_RULE
$(BINARY_DIR)/$(notdir $(1)): $(wildcard $(1)/*.go)
	@echo "Building $$(notdir $(1))..."
	@mkdir -p $(BINARY_DIR)
	@$(GO) build $(GOFLAGS) -o $$@ ./$(1)
endef

# Generate detailed build rules for each utility
$(foreach dir,$(UTIL_DIRS),$(eval $(call BUILD_RULE,$(dir))))

# Clean built binaries and test cache
.PHONY: clean
clean:
	@echo "Cleaning..."
	@rm -rf $(BINARY_DIR)
	@rm -rf coverage.txt
	@$(GO) clean -testcache

# Install binaries to system
.PHONY: install
install: all
	@echo "Installing to $(BINDIR)..."
	@mkdir -p $(BINDIR)
	@install -m 755 $(BINARIES) $(BINDIR)/

# Uninstall binaries from system
.PHONY: uninstall
uninstall:
	@echo "Uninstalling from $(BINDIR)..."
	@rm -f $(foreach bin,$(BINARIES),$(BINDIR)/$(notdir $(bin)))

# List all discovered utilities
.PHONY: list
list:
	@echo "Discovered utilities:"
	@for dir in $(UTIL_DIRS); do \
		echo "  - $$(basename $$dir)"; \
	done

# Run tests for all utilities
.PHONY: test
test:
	@echo "Running tests..."
	@$(GO) test -race ./...

# Run tests and generate coverage report
.PHONY: coverage
coverage:
	@echo "Running tests with coverage..."
	@$(GO) test -coverprofile=coverage.txt -covermode=atomic ./...
	@echo "To view coverage report, run: go tool cover -html=coverage.txt"

# Format all Go code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	@gofmt -s -w $$(go list -f '{{.Dir}}' ./...)

# Check Go formatting (fails if changes needed)
.PHONY: fmt-check
fmt-check:
	@fmt_out=$$(gofmt -s -l $$(go list -f '{{.Dir}}' ./...)); \
	if [ -n "$$fmt_out" ]; then \
		echo "gofmt needed on:"; \
		echo "$$fmt_out"; \
		exit 1; \
	fi

# Lint all Go code (requires golangci-lint)
.PHONY: lint
lint:
	@echo "Linting code..."
	@golangci-lint run ./...

# Vet all Go code
.PHONY: vet
vet:
	@echo "Vetting code..."
	@$(GO) vet ./...

# Run staticcheck (requires staticcheck)
.PHONY: staticcheck
staticcheck:
	@echo "Running staticcheck..."
	@staticcheck ./...

# Code Quality target: lint, vet, staticcheck, fmt, and coverage
.PHONY: CQ
CQ: lint vet staticcheck fmt coverage

# Help target
.PHONY: help
help:
	@echo "gopherutils - Linux coreutils in Go"
	@echo ""
	@echo "Targets:"
	@echo "  all          - Build all utilities (default)"
	@echo "  build        - Alias for all"
	@echo "  deps         - Download and install Go dependencies"
	@echo "  clean        - Remove built binaries"
	@echo "  install      - Install binaries to $(BINDIR) (configurable with INSTALL_PREFIX)"
	@echo "  uninstall    - Remove binaries from $(BINDIR)"
	@echo "  list         - List all discovered utilities"
	@echo "  test         - Run tests"
	@echo "  coverage     - Run tests with coverage and generate report"
	@echo "  fmt          - Format Go code"
	@echo "  fmt-check    - Fail if code is not formatted"
	@echo "  lint         - Lint Go code"
	@echo "  vet          - Vet Go code"
	@echo "  staticcheck  - Run staticcheck"
	@echo "  CQ           - Run lint, vet, staticcheck, fmt, and coverage"
	@echo "  help         - Show this help message"



.PHONY: gui
gui:
	@echo "Launching Makefile GUI..."
	@./scripts/makefile_gui.tcl
