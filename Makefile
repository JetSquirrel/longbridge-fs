.PHONY: build clean test run install

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

run: build
	@echo "Running controller..."
	$(BUILD_DIR)/$(BINARY_NAME) controller --root ./fs --credential ./configs/credential

install: build
	@echo "Installing to /usr/local/bin..."
	cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

uninstall:
	@echo "Uninstalling..."
	rm -f /usr/local/bin/$(BINARY_NAME)

# Development helpers
dev: build
	$(BUILD_DIR)/$(BINARY_NAME) controller --root ./fs --mock

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
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build        Build the binary"
	@echo "  clean        Clean build artifacts"
	@echo "  deps         Download dependencies"
	@echo "  test         Run tests"
	@echo "  run          Run the controller"
	@echo "  dev          Run in mock mode (no API)"
	@echo "  install      Install to /usr/local/bin"
	@echo "  fmt          Format code"
	@echo "  lint         Lint code"
	@echo "  help         Show this help"
