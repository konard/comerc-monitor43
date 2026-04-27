#!/bin/bash
# Скрипт для запуска всех тестов

set -e

echo "Running all tests..."

# Unit тесты
echo "Running unit tests..."
go test -v -race ./internal/model/... ./internal/service/...

# Интеграционные тесты
echo "Running integration tests..."
go test -v -race ./internal/repository/... ./internal/handler/...

# С coverage
echo "Running tests with coverage..."
go test -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

echo "✅ All tests passed!"
echo "Coverage report: coverage.html"
