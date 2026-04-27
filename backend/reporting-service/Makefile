.PHONY: help build run test lint fmt clean

help:
	@echo "Available targets:"
	@echo "  build    - Build the binary"
	@echo "  run      - Run the service locally"
	@echo "  test     - Run tests"
	@echo "  lint     - Run linter"
	@echo "  fmt      - Format code"
	@echo "  clean    - Clean build artifacts"

build:
	go build -o bin/reporting-service ./cmd/reporting-service

run:
	go run ./cmd/reporting-service

test:
	go test -race -failfast -count=1 ./...

test-short:
	go test -race -failfast -count=1 -short ./...

test-integration:
	go test -race -failfast -count=1 -run Integration ./...

lint:
	golangci-lint run ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -rf bin/
