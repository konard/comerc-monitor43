.PHONY: all build test clean run lint coverage benchmark help

# Variables
BINARY_NAME=api-gateway
MAIN_PATH=cmd/api-gateway
BUILD_DIR=build
GO=go
GOFLAGS=-v

# Colors for output
BLUE=\033[0;34m
GREEN=\033[0;32m
YELLOW=\033[0;33m
RED=\033[0;31m
NC=\033[0m # No Color

help: ## Show this help message
	@echo "$(BLUE)API Gateway - Available targets:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2}'

all: clean build test ## Clean, build, and test

build: ## Build the binary
	@echo "$(BLUE)Building $(BINARY_NAME)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)/main.go
	@echo "$(GREEN)Build complete: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

build-fast: ## Build without race detector
	@echo "$(BLUE)Building $(BINARY_NAME) (fast)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@$(GO) build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)/main.go

test: ## Run all tests
	@echo "$(BLUE)Running tests...$(NC)"
	@$(GO) test -v -race -timeout=30s ./...
	@echo "$(GREEN)Tests passed$(NC)"

test-coverage: ## Run tests with coverage
	@echo "$(BLUE)Running tests with coverage...$(NC)"
	@$(GO) test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"
	@$(GO) tool cover -func=coverage.out

benchmark: ## Run benchmarks
	@echo "$(BLUE)Running benchmarks...$(NC)"
	@$(GO) test -bench=. -benchmem ./...
	@echo "$(GREEN)Benchmarks complete$(NC)"

test-unit: ## Run unit tests only (no integration)
	@echo "$(BLUE)Running unit tests...$(NC)"
	@$(GO) test -v -short ./...

lint: ## Run linter
	@echo "$(BLUE)Running linter...$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "$(YELLOW)golangci-lint not found. Install it with:$(NC)"; \
		echo "  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin"; \
	fi

fmt: ## Format code
	@echo "$(BLUE)Formatting code...$(NC)"
	@$(GO) fmt ./...

vet: ## Run go vet
	@echo "$(BLUE)Running go vet...$(NC)"
	@$(GO) vet ./...

run: build ## Build and run the server
	@echo "$(BLUE)Starting $(BINARY_NAME)...$(NC)"
	@$(BUILD_DIR)/$(BINARY_NAME)

run-dev: ## Run without building (for development)
	@echo "$(BLUE)Running $(BINARY_NAME) in development mode...$(NC)"
	@$(GO) run $(MAIN_PATH)/main.go

clean: ## Clean build artifacts
	@echo "$(BLUE)Cleaning...$(NC)"
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "$(GREEN)Clean complete$(NC)"

deps: ## Download dependencies
	@echo "$(BLUE)Downloading dependencies...$(NC)"
	@$(GO) mod download
	@$(GO) mod tidy

deps-verify: ## Verify dependencies
	@echo "$(BLUE)Verifying dependencies...$(NC)"
	@$(GO) mod verify

docker-build: ## Build Docker image
	@echo "$(BLUE)Building Docker image...$(NC)"
	@docker build -t $(BINARY_NAME):latest .

docker-run: ## Run Docker container
	@echo "$(BLUE)Running Docker container...$(NC)"
	@docker run -p 8080:8080 --env-file .env $(BINARY_NAME):latest
