.PHONY: build clean test run install init help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Binary names
BINARY_NAME=longbridge-fs
BINARY_UNIX=$(BINARY_NAME)_unix

# Main package
MAIN_PACKAGE=./cmd/longbridge-fs

# Build directory
BUILD_DIR=./build

# Version
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Default target
.DEFAULT_GOAL := help

all: clean deps build

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f $(BINARY_NAME)
	$(GOCLEAN)

deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

test:
	$(GOTEST) -v ./...

# Initialize file system (creates default directories)
init: build
	@echo "Initializing file system..."
	$(BUILD_DIR)/$(BINARY_NAME) init --root ./fs

run: build
	@echo "Running controller with real API..."
	$(BUILD_DIR)/$(BINARY_NAME) controller --root ./fs --credential ./configs/credential

install: build
	@echo "Installing to /usr/local/bin..."
	cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

uninstall:
	@echo "Uninstalling..."
	rm -f /usr/local/bin/$(BINARY_NAME)

# Development helpers
dev: build
	@echo "Running controller in mock mode..."
	$(BUILD_DIR)/$(BINARY_NAME) controller --root ./fs --mock

# Run with verbose output
dev-verbose: build
	@echo "Running controller in mock mode with verbose output..."
	$(BUILD_DIR)/$(BINARY_NAME) controller --root ./fs --mock --verbose

fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

lint:
	@echo "Linting..."
	@golangci-lint run || echo "golangci-lint not installed"

# Docker
docker-build:
	docker build -t longbridge-fs:$(VERSION) .

docker-run:
	docker run -it --rm longbridge-fs:$(VERSION)

# Help
help:
	@echo "Longbridge FS - File system-based stock trading framework"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@echo "  build         Build the binary"
	@echo "  clean         Clean build artifacts"
	@echo "  deps          Download and tidy dependencies"
	@echo "  test          Run tests"
	@echo "  init          Initialize FS directory structure"
	@echo "  run           Run controller with real API"
	@echo "  dev           Run controller in mock mode (no API)"
	@echo "  dev-verbose   Run controller in mock mode with verbose output"
	@echo "  install       Install to /usr/local/bin"
	@echo "  uninstall     Remove from /usr/local/bin"
	@echo "  fmt           Format code"
	@echo "  lint          Lint code (requires golangci-lint)"
	@echo "  help          Show this help message"
	@echo ""
	@echo "Docker targets:"
	@echo "  docker-build  Build Docker image"
	@echo "  docker-run    Run Docker container"
	@echo ""
	@echo "Examples:"
	@echo "  make build              # Build the binary"
	@echo "  make init               # Initialize file system"
	@echo "  make dev                # Run in mock mode"
	@echo "  make run                # Run with real API"
	@echo ""
	@echo "Version: $(VERSION)"
	@echo "Build time: $(BUILD_TIME)"
