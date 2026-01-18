# gopherutils Makefile
# Automatically discovers and builds all utilities

# Variables
BINARY_DIR := build
UTILS_DIR := utils
GO := go
GOFLAGS := -ldflags="-s -w"

# Find all directories containing main.go files under utils/
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

# Build individual binaries
$(BINARY_DIR)/%: $(UTILS_DIR)/%/main.go | $(BINARY_DIR)
	@echo "Building $*..."
	@$(GO) build $(GOFLAGS) -o $@ $(shell find $(UTILS_DIR) -type d -name "$*")/main.go

# Alternative pattern rule to handle nested directories
$(BINARY_DIR)/%: $(UTILS_DIR)/*/*/main.go | $(BINARY_DIR)
	@echo "Building $*..."
	@$(GO) build $(GOFLAGS) -o $@ $(shell find $(UTILS_DIR) -type d -name "$*")/main.go

# Create binary directory if it doesn't exist
$(BINARY_DIR):
	@mkdir -p $(BINARY_DIR)

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
	@$(GO) test ./...

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
	@echo "  deps       - Download and install Go dependencies"
	@echo "  clean      - Remove built binaries"
	@echo "  install    - Install binaries to /usr/local/bin"
	@echo "  uninstall  - Remove binaries from /usr/local/bin"
	@echo "  list       - List all discovered utilities"
	@echo "  test       - Run tests"
	@echo "  fmt        - Format Go code"
	@echo "  lint       - Lint Go code"
	@echo "  help       - Show this help message"
