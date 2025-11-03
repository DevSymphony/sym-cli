# symphonyclient integration: Added CSS build targets
.PHONY: build test install clean fmt lint tidy help build-all setup coverage-check build-css install-css watch-css

BINARY_NAME=sym
BUILD_DIR=bin
MAIN_PATH=./cmd/sym
VERSION?=dev
LDFLAGS=-ldflags "-X main.Version=$(VERSION)"

help:
	@echo "Available targets:"
	@echo "  build      - Build the binary for current platform (includes CSS)"
	@echo "  build-all  - Build for all platforms (Linux, macOS, Windows)"
	@echo "  build-css  - Build Tailwind CSS for dashboard"
	@echo "  watch-css  - Watch and rebuild CSS on changes"
	@echo "  test       - Run tests"
	@echo "  install    - Install the binary to GOPATH/bin"
	@echo "  clean      - Remove build artifacts"
	@echo "  fmt        - Format code"
	@echo "  lint       - Run linter"
	@echo "  tidy       - Tidy dependencies"
	@echo "  setup      - Setup development environment"

# symphonyclient integration: CSS build for dashboard
install-css:
	@echo "Installing CSS dependencies..."
	@npm install

build-css: install-css
	@echo "Building Tailwind CSS..."
	@npm run build:css
	@echo "CSS build complete"

watch-css: install-css
	@echo "Watching CSS changes..."
	@npm run watch:css

build: build-css
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
ifeq ($(OS),Windows_NT)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME).exe $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME).exe"
else
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"
endif

build-all: build-css
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	@echo "Building Linux amd64..."
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	@echo "Building Linux arm64..."
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	@echo "Building macOS amd64..."
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	@echo "Building macOS arm64..."
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	@echo "Building Windows amd64..."
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	@echo "All platform builds complete"

test:
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Test complete. Coverage report: coverage.html"

coverage-check:
	@echo "Checking coverage threshold..."
	@go test -coverprofile=coverage.out ./... > /dev/null 2>&1
	@COVERAGE=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	THRESHOLD=80; \
	echo "Current coverage: $$COVERAGE%"; \
	echo "Required threshold: $$THRESHOLD%"; \
	if [ "$$(echo "$$COVERAGE < $$THRESHOLD" | bc -l 2>/dev/null || awk "BEGIN {print ($$COVERAGE < $$THRESHOLD)}")" -eq 1 ]; then \
		echo "❌ Coverage $$COVERAGE% is below threshold $$THRESHOLD%"; \
		exit 1; \
	else \
		echo "✅ Coverage $$COVERAGE% meets threshold $$THRESHOLD%"; \
	fi

install:
	@echo "Installing $(BINARY_NAME)..."
	@go install $(MAIN_PATH)
	@echo "Install complete"

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@rm -rf node_modules
	@echo "Clean complete"

fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Format complete"

lint:
	@echo "Running linter..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install: https://golangci-lint.run/usage/install/"; \
	fi

tidy:
	@echo "Tidying dependencies..."
	@go mod tidy
	@echo "Tidy complete"

# Development helpers
setup: tidy install-css
	@echo "Setting up development environment..."
	@go mod download
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Development environment setup complete"

dev-deps:
	@echo "Installing development dependencies..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Development dependencies installed"

run:
	@go run $(MAIN_PATH) $(ARGS)
