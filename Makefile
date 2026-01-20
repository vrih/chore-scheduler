.PHONY: build test clean run install lint build-linux-amd64

# Build variables
BINARY_NAME=chore-scheduler
BUILD_DIR=bin
CMD_DIR=cmd/chore-scheduler

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build the application
build:
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)/main.go

# Run all tests with coverage
test:
	$(GOTEST) -v -cover ./...

# Run tests with coverage report
test-coverage:
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Run the application directly
run:
	$(GOCMD) run $(CMD_DIR)/main.go

# Install binary to GOPATH/bin
install:
	$(GOCMD) install $(CMD_DIR)/main.go

# Tidy dependencies
tidy:
	$(GOMOD) tidy

# Run linter (requires golangci-lint)
lint:
	golangci-lint run ./...

# Format code
fmt:
	$(GOCMD) fmt ./...

# Vet code
vet:
	$(GOCMD) vet ./...

# Cross-compilation for NAS (common architectures)
build-linux-amd64:
#	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-linux-musl-gcc $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)/main.go
	CC=x86_64-linux-musl-gcc CXX=x86_64-linux-musl-g++ GOARCH=amd64 GOOS=linux CGO_ENABLED=1 ${GOBUILD} -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 -ldflags "-linkmode external -extldflags -static" ${CMD_DIR}/main.go

build-linux-arm64:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=1 CC=aarch64-linux-musl-gcc $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)/main.go

# Build for current platform (default)
build-local: build

# Build all supported platforms
build-all: build build-linux-amd64 build-linux-arm64

# Development workflow: format, vet, test, build
dev: fmt vet test build

# Help target
help:
	@echo "Available targets:"
	@echo "  build            - Build the application"
	@echo "  test             - Run tests with coverage"
	@echo "  test-coverage    - Generate HTML coverage report"
	@echo "  clean            - Remove build artifacts"
	@echo "  run              - Run the application directly"
	@echo "  install          - Install binary to GOPATH/bin"
	@echo "  tidy             - Tidy go.mod dependencies"
	@echo "  lint             - Run golangci-lint"
	@echo "  fmt              - Format code"
	@echo "  vet              - Run go vet"
	@echo "  build-linux-amd64 - Cross-compile for Linux AMD64"
	@echo "  build-linux-arm64 - Cross-compile for Linux ARM64"
	@echo "  dev              - Development workflow (fmt, vet, test, build)"
