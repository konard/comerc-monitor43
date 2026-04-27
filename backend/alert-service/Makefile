.PHONY: build run test clean docker-build docker-run lint fmt

# Variables
BINARY_NAME=server
CMD_DIR=./cmd/server
GO_FILES=$(shell find . -name '*.go' -not -path './vendor/*')

# Build targets
build: ## Build the server binary
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BINARY_NAME) $(CMD_DIR)
	@echo "Build complete: $(BINARY_NAME)"

run: ## Run the server
	@echo "Running $(BINARY_NAME)..."
	@go run $(CMD_DIR)/main.go

test: ## Run tests
	@echo "Running tests..."
	@go test -v -race -count=1 ./...

test-short: ## Run short tests
	@echo "Running short tests..."
	@go test -v -short ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

integration-test: ## Run integration tests (requires Docker)
	@echo "Running integration tests..."
	@go test -v -race -tags=integration ./...

lint: ## Run golangci-lint
	@echo "Running linters..."
	@golangci-lint run ./...

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@goimports -w .

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@rm -f coverage.out coverage.html
	@go clean

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t alert-service:latest .

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	@docker run -p 50051:50051 --env-file .env alert-service:latest

help: ## Show this help
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@sed -n 's/^.*://; s/^##//; s/\(.*\):\(.*\)/  \1:\2/p' $(MAKEFILE_LIST) | column -t -s ':'

# Default target
.DEFAULT_GOAL := help
