# Billing Service Makefile

.PHONY: help deps build run test test-unit test-integration test-e2e migrate-up migrate-down migrate-create clean docker-up docker-down docker-test-up docker-test-down

# Variables
BINARY_NAME=billing-service
CMD_DIR=./cmd/billing-service
MIGRATIONS_DIR=./migrations
DOCKER_COMPOSE=./docker-compose.yml

# Default target
help:
	@echo "Billing Service - Available commands:"
	@echo ""
	@echo "  make deps           - Install dependencies"
	@echo "  make build          - Build the service"
	@echo "  make run            - Run the service"
	@echo "  make test           - Run all tests"
	@echo "  make test-unit      - Run unit tests"
	@echo "  make test-integration - Run integration tests"
	@echo "  make test-e2e       - Run E2E tests (Godog)"
	@echo "  make migrate-up     - Run database migrations"
	@echo "  make migrate-down   - Rollback last migration"
	@echo "  make migrate-create - Create new migration"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make docker-up      - Start docker services"
	@echo "  make docker-down    - Stop docker services"
	@echo "  make docker-test-up   - Start test docker services"
	@echo "  make docker-test-down - Stop test docker services"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Build the service
build: deps
	@echo "Building $(BINARY_NAME)..."
	go build -o bin/$(BINARY_NAME) $(CMD_DIR)/main.go
	@echo "Build complete: bin/$(BINARY_NAME)"

# Run the service
run: deps
	@echo "Running $(BINARY_NAME)..."
	go run $(CMD_DIR)/main.go

# Run all tests
test: test-unit test-integration
	@echo "All tests passed!"

# Run unit tests
test-unit:
	@echo "Running unit tests..."
	go test -v -race -coverprofile=coverage.out ./...
	@echo "Unit tests passed!"

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	@echo "Skipping integration tests (use testcontainers)"

# Run database migrations up
migrate-up:
	@echo "Running migrations up..."
	goose postgres "host=localhost port=5432 user=postgres password=postgres dbname=billing sslmode=disable" up $(MIGRATIONS_DIR)
	@echo "Migrations applied!"

# Rollback last migration
migrate-down:
	@echo "Running migrations down..."
	goose postgres "host=localhost port=5432 user=postgres password=postgres dbname=billing sslmode=disable" down $(MIGRATIONS_DIR)
	@echo "Migration rolled back!"

# Create new migration
migrate-create:
	@read -p "Enter migration name: " name; \
	timestamp=$$(date +%Y%m%d%H%M%S); \
	goose create $$name sql $(MIGRATIONS_DIR)

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage.out
	@echo "Clean complete!"

# Start docker services
docker-up:
	@echo "Starting docker services..."
	docker-compose up -d postgres
	@echo "Docker services started!"

# Stop docker services
docker-down:
	@echo "Stopping docker services..."
	docker-compose down
	@echo "Docker services stopped!"

# Start test docker services
docker-test-up:
	@echo "Starting test docker services..."
	docker-compose -f docker-compose.test.yml up -d postgres-test rabbitmq-test
	@echo "Waiting for services to be ready..."
	@sleep 5
	@echo "Test docker services started!"

# Stop test docker services
docker-test-down:
	@echo "Stopping test docker services..."
	docker-compose -f docker-compose.test.yml down -v
	@echo "Test docker services stopped!"

# Run E2E tests
test-e2e: docker-test-up
	@echo "Running E2E tests..."
	@echo "Applying test migrations..."
	goose postgres "host=localhost port=5433 user=test_user password=test_password dbname=billing_test sslmode=disable" up $(MIGRATIONS_DIR)
	@echo "Starting billing service..."
	@go run $(CMD_DIR)/main.go &
	@sleep 3
	@echo "Running Godog tests..."
	cd test && go test -v ./...
	@echo "E2E tests passed!"
	@$(MAKE) docker-test-down

