# gopherutils Makefile
# Automatically discovers and builds all utilities

# Variables
BINARY_DIR := build
UTILS_DIR := utils
GO := go
GOFLAGS := -ldflags="-s -w"

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

# Clean built binaries
.PHONY: clean
clean:
	@echo "Cleaning..."
	@rm -rf $(BINARY_DIR)

# Install binaries to system (requires sudo/root)
.PHONY: install
install: all
	@echo "Installing to /usr/local/bin..."
	@install -m 755 $(BINARIES) /usr/local/bin/

# Uninstall binaries from system
.PHONY: uninstall
uninstall:
	@echo "Uninstalling from /usr/local/bin..."
	@rm -f $(foreach bin,$(BINARIES),/usr/local/bin/$(notdir $(bin)))

# List all discovered utilities
.PHONY: list
list:
	@echo "Discovered utilities:"
	@echo -e $(foreach dir,$(UTIL_DIRS),"  - $(notdir $(dir))\n")

# Run tests for all utilities
.PHONY: test
test:
	@echo "Running tests..."
	@$(GO) test -cover ./...

# Run tests and generate coverage report
.PHONY: coverage
coverage:
	@echo "Running tests with coverage..."
	@$(GO) test -coverprofile=coverage.txt ./...
	@echo "To view coverage report, run: go tool cover -html=coverage.txt"

# Format all Go code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	@$(GO) fmt ./...

# Lint all Go code (requires golangci-lint)
.PHONY: lint
lint:
	@echo "Linting code..."
	@golangci-lint run ./...

# Help target
.PHONY: help
help:
	@echo "gopherutils - Linux coreutils in Go"
	@echo ""
	@echo "Targets:"
	@echo "  all        - Build all utilities (default)"
	@echo "  build      - Alias for all"
	@echo "  deps       - Download and install Go dependencies"
	@echo "  clean      - Remove built binaries"
	@echo "  install    - Install binaries to /usr/local/bin"
	@echo "  uninstall  - Remove binaries from /usr/local/bin"
	@echo "  list       - List all discovered utilities"
	@echo "  test       - Run tests"
	@echo "  coverage   - Run tests with coverage and generate report"
	@echo "  fmt        - Format Go code"
	@echo "  lint       - Lint Go code"
	@echo "  help       - Show this help message"
