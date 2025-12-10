.PHONY: build build-all test unit-test clean fmt lint tidy setup run help

BINARY_NAME=sym
BUILD_DIR=bin
MAIN_PATH=./cmd/sym
VERSION?=dev
LDFLAGS=-ldflags "-X main.Version=$(VERSION)"

help:
	@echo "Available targets:"
	@echo "  build      - Build the binary for current platform"
	@echo "  build-all  - Build for all platforms (Linux, macOS, Windows)"
	@echo "  test       - Run tests with coverage"
	@echo "  unit-test  - Run tests without coverage"
	@echo "  clean      - Remove build artifacts"
	@echo "  fmt        - Format code"
	@echo "  lint       - Run linter"
	@echo "  tidy       - Tidy dependencies"
	@echo "  setup      - Setup development environment"
	@echo "  run        - Run the application"

build:
	@mkdir -p $(BUILD_DIR)
ifeq ($(OS),Windows_NT)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME).exe $(MAIN_PATH)
else
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
endif

build-all:
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)

test:
	@go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -html=coverage.out -o coverage.html

unit-test:
	@go test -short ./...

clean:
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

fmt:
	@golangci-lint fmt

lint:
	@golangci-lint run

tidy:
	@go mod tidy

setup:
	@go mod download
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.7.2

run:
	@go run $(MAIN_PATH) $(ARGS)
