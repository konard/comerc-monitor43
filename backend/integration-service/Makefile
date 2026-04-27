.PHONY: all build run test clean docker-build docker-up docker-down migrate migrate-up migrate-down

# Variables
BINARY_NAME=integration-service
CMD_DIR=./cmd/integration-service
DOCKER_IMAGE=$(BINARY_NAME):latest
DOCKER_COMPOSE_FILE=docker-compose.yml

# Build targets
all: build

build:
	@echo "Building $(BINARY_NAME)..."
	go build -o bin/$(BINARY_NAME) ./$(CMD_DIR)

run:
	@echo "Running $(BINARY_NAME)..."
	go run ./$(CMD_DIR)

test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

test-short:
	@echo "Running short tests..."
	go test -v -short ./...

test-integration:
	@echo "Running integration tests..."
	go test -v -tags=integration ./...

test-coverage:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	@echo "Coverage report:"
	@go tool cover -func=coverage.out | grep total

lint:
	@echo "Running linters..."
	golangci-lint run ./...

fmt:
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .

clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	go clean ./...

# Docker targets
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .

docker-up:
	@echo "Starting services with docker-compose..."
	docker-compose -f $(DOCKER_COMPOSE_FILE) up -d

docker-down:
	@echo "Stopping services..."
	docker-compose -f $(DOCKER_COMPOSE_FILE) down

docker-logs:
	@echo "Showing logs..."
	docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f $(BINARY_NAME)

docker-ps:
	@echo "Showing running containers..."
	docker-compose -f $(DOCKER_COMPOSE_FILE) ps

# Migration targets
migrate-up:
	@echo "Running migrations up..."
	goose -dir ./migrations postgres "user=postgres dbname=milan sslmode=disable host=localhost" up

migrate-down:
	@echo "Running migrations down..."
	goose -dir ./migrations postgres "user=postgres dbname=milan sslmode=disable host=localhost" down

migrate-create:
	@echo "Creating new migration (usage: make migrate-create NAME=name)"
	@if [ -z "$(NAME)" ]; then echo "Error: NAME parameter is required. Usage: make migrate-create NAME=create_users_table"; exit 1; fi
	goose -dir ./migrations create $(NAME) sql

# Proto targets
proto-gen:
	@echo "Generating proto files..."
	cd ../../api/proto && \
	protoc --go_out=. --go-grpc_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_opt=paths=source_relative \
		integration.proto

proto-clean:
	@echo "Cleaning generated proto files..."
	cd ../../api/proto && \
	rm -f integration_grpc.pb.go integration.pb.go

# Development targets
dev:
	@echo "Starting development environment..."
	docker-compose -f $(DOCKER_COMPOSE_FILE) up
	@$(MAKE) docker-logs

dev-build:
	@echo "Building and starting..."
	$(MAKE) docker-build
	$(MAKE) docker-up

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Run with hot reload (requires air)
dev-hot:
	@echo "Running with hot reload..."
	air

# Help target
help:
	@echo "Available targets:"
	@echo "  build           - Build the binary"
	@echo "  run             - Run the service"
	@echo "  test            - Run all tests"
	@echo "  test-short      - Run short tests"
	@echo "  test-integration - Run integration tests"
	@echo "  test-coverage   - Run tests with coverage report"
	@echo "  lint            - Run linters"
	@echo "  fmt             - Format code"
	@echo "  clean           - Clean build artifacts"
	@echo "  docker-build    - Build Docker image"
	@echo "  docker-up       - Start services with docker-compose"
	@echo "  docker-down     - Stop services"
	@echo "  docker-logs     - Show service logs"
	@echo "  migrate-up      - Run database migrations up"
	@echo "  migrate-down    - Run database migrations down"
	@echo "  migrate-create  - Create new migration (NAME parameter required)"
	@echo "  proto-gen       - Generate proto files"
	@echo "  proto-clean     - Clean generated proto files"
	@echo "  deps            - Install dependencies"
	@echo "  dev             - Start development environment"
	@echo "  dev-build       - Build and start services"
	@echo "  dev-hot         - Run with hot reload (requires air)"
	@echo "  help            - Show this help message"
