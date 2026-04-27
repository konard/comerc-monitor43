.PHONY: build run test clean generate deps lint migrate-up migrate-down

# Переменные
BINARY_NAME=notification-template-service
CMD_DIR=./cmd/notification-template-service
BUILD_DIR=./bin
MIGRATIONS_DIR=./migrations

# go параметры
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# build: Собирает бинарник
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)/main.go

# run: Запускает сервис
run:
	@echo "Running $(BINARY_NAME)..."
	$(GORUN) $(CMD_DIR)/main.go

# test: Запускает тесты
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -cover ./...

# test-coverage: Запускает тесты с покрытием
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# clean: Удаляет артефакты сборки
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

# generate: Генерирует gRPC код
generate:
	@echo "Generating gRPC code..."
	@cd ../../api && protoc \
		--proto_path=. \
		--go_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_out=. \
		--go-grpc_opt=paths=source_relative \
		proto/notification_template.proto

# deps: Скачивает зависимости
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# lint: Проверяет код
lint:
	@echo "Running linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Skipping..."; \
	fi

# migrate-up: Применяет миграции
migrate-up:
	@echo "Running migrations up..."
	@if command -v goose >/dev/null 2>&1; then \
		goose postgres "host=localhost port=5432 user=monitor password=monitor dbname=notification_templates sslmode=disable" up $(MIGRATIONS_DIR); \
	else \
		echo "goose not installed. Run: go install github.com/pressly/goose/v3/cmd/goose@latest"; \
	fi

# migrate-down: Откатывает миграции
migrate-down:
	@echo "Running migrations down..."
	@if command -v goose >/dev/null 2>&1; then \
		goose postgres "host=localhost port=5432 user=monitor password=monitor dbname=notification_templates sslmode=disable" down $(MIGRATIONS_DIR); \
	else \
		echo "goose not installed. Run: go install github.com/pressly/goose/v3/cmd/goose@latest"; \
	fi

# migrate-create: Создает новую миграцию
# Usage: make migrate-create NAME=add_new_column
migrate-create:
	@if [ -z "$(NAME)" ]; then \
		echo "Usage: make migrate-create NAME=migration_name"; \
		exit 1; \
	fi
	@if command -v goose >/dev/null 2>&1; \
		then goose -dir $(MIGRATIONS_DIR) create $(NAME) sql; \
		else echo "goose not installed"; \
	fi

# docker-build: Собирает Docker образ
docker-build:
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):latest .

# docker-run: Запускает в Docker
docker-run:
	@echo "Running in Docker..."
	docker-compose up -d $(BINARY_NAME)

# help: Показывает справку
help:
	@echo "Available targets:"
	@echo "  build          - Build binary"
	@echo "  run            - Run service"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage"
	@echo "  clean          - Clean build artifacts"
	@echo "  generate       - Generate gRPC code"
	@echo "  deps           - Download dependencies"
	@echo "  lint           - Run linters"
	@echo "  migrate-up     - Apply migrations"
	@echo "  migrate-down   - Rollback migrations"
	@echo "  migrate-create - Create new migration (NAME=<name>)"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-run     - Run in Docker"
	@echo "  help           - Show this help"
