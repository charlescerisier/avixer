# Avixer Makefile

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build parameters
BINARY_NAME=avixer
BINARY_PATH=./cmd/avixer
BUILD_DIR=build

# Version information
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

.PHONY: all build clean test deps tidy install example help

# Default target
all: clean deps test build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(BINARY_PATH)
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME)"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@rm -f example_output.avi
	@rm -f test_*.avi

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOGET) -d ./...

# Tidy modules
tidy:
	@echo "Tidying modules..."
	$(GOMOD) tidy

# Install binary to GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/

# Build and run example
example: build
	@echo "Building example..."
	$(GOBUILD) -o $(BUILD_DIR)/example ./examples/basic_usage.go
	@echo "Running example..."
	@cd examples && ../$(BUILD_DIR)/example

# Run the CLI tool on the test video (if it exists)
demo: build
	@if [ -f video.avi ]; then \
		echo "Running demo on video.avi..."; \
		./$(BUILD_DIR)/$(BINARY_NAME) -i video.avi -v; \
		echo "Output written to video.avi.json"; \
	else \
		echo "No video.avi file found. Place a test AVI file named 'video.avi' in the current directory."; \
	fi

# Format Go code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

# Run Go linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; exit 1)
	golangci-lint run

# Generate Go documentation
docs:
	@echo "Generating documentation..."
	@echo "Visit http://localhost:6060/pkg/github.com/charlescerisier/avixer/"
	godoc -http=:6060

# Cross-compile for different platforms
build-all: clean deps
	@echo "Cross-compiling for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	
	# Linux amd64
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(BINARY_PATH)
	
	# Linux arm64
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(BINARY_PATH)
	
	# macOS amd64
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(BINARY_PATH)
	
	# macOS arm64
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(BINARY_PATH)
	
	# Windows amd64
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(BINARY_PATH)
	
	@echo "Cross-compilation completed. Binaries available in $(BUILD_DIR)/"

# Create release archive
release: build-all
	@echo "Creating release archives..."
	@cd $(BUILD_DIR) && \
	for binary in $(BINARY_NAME)-*; do \
		if [[ $$binary == *.exe ]]; then \
			zip $${binary%.exe}.zip $$binary; \
		else \
			tar -czf $$binary.tar.gz $$binary; \
		fi; \
	done
	@echo "Release archives created in $(BUILD_DIR)/"

# Benchmark tests
bench:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

# Check for security vulnerabilities
security:
	@echo "Checking for security vulnerabilities..."
	@which gosec > /dev/null || (echo "gosec not installed. Install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; exit 1)
	gosec ./...

# Development setup
dev-setup:
	@echo "Setting up development environment..."
	$(GOGET) github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GOGET) github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	@echo "Development tools installed"

# Help target
help:
	@echo "Avixer Build System"
	@echo "==================="
	@echo ""
	@echo "Available targets:"
	@echo "  all          - Clean, download deps, test, and build"
	@echo "  build        - Build the binary"
	@echo "  clean        - Clean build artifacts"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage report"
	@echo "  deps         - Download dependencies"
	@echo "  tidy         - Tidy Go modules"
	@echo "  install      - Install binary to GOPATH/bin"
	@echo "  example      - Build and run example"
	@echo "  demo         - Run demo on video.avi (if present)"
	@echo "  fmt          - Format Go code"
	@echo "  lint         - Run Go linter"
	@echo "  docs         - Start documentation server"
	@echo "  build-all    - Cross-compile for multiple platforms"
	@echo "  release      - Create release archives"
	@echo "  bench        - Run benchmark tests"
	@echo "  security     - Check for security vulnerabilities"
	@echo "  dev-setup    - Install development tools"
	@echo "  help         - Show this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make build          # Build the binary"
	@echo "  make test           # Run tests"
	@echo "  make demo           # Run demo (requires video.avi)"
	@echo "  make build-all      # Cross-compile for all platforms"