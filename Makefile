# Makefile for tf-state-move

.PHONY: help build build-ci clean test lint

# Default target
help:
	@echo "Available targets:"
	@echo "  build     - Build for all supported platforms"
	@echo "  build-ci  - Build for CI platforms only (linux/amd64, windows/amd64, darwin/amd64)"
	@echo "  clean     - Remove build artifacts"
	@echo "  test      - Run tests"
	@echo "  lint      - Run linter"
	@echo "  help      - Show this help message"

# Build for all supported platforms
build:
	@echo "Building for all supported platforms..."
	@./build.sh

# Build for CI platforms only
build-ci:
	@echo "Building for CI platforms..."
	@./build-ci.sh

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf build/
	@echo "Build artifacts removed."

# Run tests
test:
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

# Run linter
lint:
	@echo "Running linter..."
	@golangci-lint run

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod tidy
	@go mod download

# Build single platform (usage: make single GOOS=linux GOARCH=amd64)
single:
	@echo "Building for $(GOOS)/$(GOARCH)..."
	@mkdir -p build
	@env GOOS=$(GOOS) GOARCH=$(GOARCH) go build -v -o build/tf-state-move-$(GOOS)-$(GOARCH)$(shell if [ "$(GOOS)" = "windows" ]; then echo ".exe"; fi) .

# Create release zip
release: build
	@echo "Creating release zip..."
	@cd build && zip -r ../tf-state-move-release.zip .
	@echo "Release zip created: tf-state-move-release.zip" 
