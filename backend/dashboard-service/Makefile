.PHONY: build run test clean docker-build docker-run lint fmt

BINARY_NAME=dashboard-service
CMD_DIR=./cmd/server

build:
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BINARY_NAME) $(CMD_DIR)
	@echo "Build complete: $(BINARY_NAME)"

run:
	@echo "Running $(BINARY_NAME)..."
	@go run $(CMD_DIR)/main.go

test:
	@echo "Running tests..."
	@go test -v -race -count=1 ./...

test-short:
	@echo "Running short tests..."
	@go test -v -short ./...

test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

integration-test:
	@echo "Running integration tests..."
	@go test -v -race -tags=integration ./...

lint:
	@echo "Running linters..."
	@golangci-lint run ./...

fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@goimports -w .

vet:
	@echo "Running go vet..."
	@go vet ./...

clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@rm -f coverage.out coverage.html
	@go clean

deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

docker-build:
	@echo "Building Docker image..."
	@docker build -t dashboard-service:latest .

docker-run:
	@echo "Running Docker container..."
	@docker run -p 9095:9095 -p 8095:8095 --env-file .env dashboard-service:latest

help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@echo "  build             Build the server binary"
	@echo "  run               Run the server"
	@echo "  test              Run tests"
	@echo "  test-short        Run short tests"
	@echo "  test-coverage     Run tests with coverage"
	@echo "  integration-test  Run integration tests"
	@echo "  lint              Run golangci-lint"
	@echo "  fmt               Format code"
	@echo "  vet               Run go vet"
	@echo "  clean             Clean build artifacts"
	@echo "  deps              Download dependencies"
	@echo "  docker-build      Build Docker image"
	@echo "  docker-run        Run Docker container"

.DEFAULT_GOAL := help
