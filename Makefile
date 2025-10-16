# Go Bridge Makefile
# Build and development automation for gopenbridge Go implementation

# Go binary name
BINARY_NAME=gopenbridge
BINARY_PATH=cmd/gopenbridge

# Build flags
GO_BUILD_FLAGS=-ldflags="-s -w"
GO_DEBUG_FLAGS=-gcflags="all=-N -l"

# Default target
.PHONY: all
all: build

# Build the binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@go build $(GO_BUILD_FLAGS) -o $(BINARY_NAME) ./$(BINARY_PATH)
	@echo "Binary created: $(BINARY_NAME)"

# Build with debug flags
.PHONY: debug
debug:
	@echo "Building $(BINARY_NAME) with debug flags..."
	@go build $(GO_DEBUG_FLAGS) -o $(BINARY_NAME).debug ./$(BINARY_PATH)
	@echo "Debug binary created: $(BINARY_NAME).debug"

# Run the server (builds first)
.PHONY: run
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME)

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME) $(BINARY_NAME).debug
	@go clean
	@echo "Clean complete"

# Test the project
.PHONY: test
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Run go vet for static analysis
.PHONY: vet
vet:
	@echo "Running go vet..."
	@go vet ./...

# Download dependencies
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

# Check for security vulnerabilities
.PHONY: security-check
security-check:
	@echo "Running security analysis..."
	@go list -json -m all | nancy sleuth 2>/dev/null || echo "nancy not installed (go install github.com/sonatypecommunity/nancy@latest)"

# Install the binary to GOPATH/bin
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME)..."
	@go install ./$(BINARY_PATH)
	@echo "Installed $(BINARY_NAME) to GOPATH/bin"

# Cross-compile for different platforms
.PHONY: cross-compile
cross-compile:
	@echo "Cross-compiling..."
	GOOS=linux GOARCH=amd64 go build $(GO_BUILD_FLAGS) -o $(BINARY_NAME)-linux-amd64 ./$(BINARY_PATH)
	GOOS=darwin GOARCH=amd64 go build $(GO_BUILD_FLAGS) -o $(BINARY_NAME)-darwin-amd64 ./$(BINARY_PATH)
	GOOS=darwin GOARCH=arm64 go build $(GO_BUILD_FLAGS) -o $(BINARY_NAME)-darwin-arm64 ./$(BINARY_PATH)
	GOOS=windows GOARCH=amd64 go build $(GO_BUILD_FLAGS) -o $(BINARY_NAME)-windows-amd64.exe ./$(BINARY_PATH)
	@echo "Cross-compilation complete"

# Development server with hot reload (requires air)
.PHONY: dev
dev:
	@echo "Starting development server with hot reload..."
	@air -c .air.toml 2>/dev/null || echo "air not installed (go install github.com/cosmtrek/air@latest)"

# Run with specific port
.PHONY: run-port
run-port:
	@if [ -z "$(PORT)" ]; then echo "Usage: make run-port PORT=8323"; exit 1; fi
	@echo "Running $(BINARY_NAME) on port $(PORT)..."
	./$(BINARY_NAME) --port $(PORT)

# Show help
.PHONY: help
help:
	@echo "GOpenBridge Go Makefile"
	@echo "====================="
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  all           - Build the binary (default)"
	@echo "  build         - Build the binary"
	@echo "  debug         - Build with debug flags"
	@echo "  run           - Build and run the server"
	@echo "  run-port PORT=8323 - Run server on specific port"
	@echo "  clean         - Remove build artifacts"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  fmt           - Format code"
	@echo "  vet           - Run static analysis"
	@echo "  deps          - Download and tidy dependencies"
	@echo "  security-check - Check for security vulnerabilities"
	@echo "  install       - Install binary to GOPATH/bin"
	@echo "  cross-compile - Build for multiple platforms"
	@echo "  dev           - Development server with hot reload"
	@echo "  help          - Show this help message"
