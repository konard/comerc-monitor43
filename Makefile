# Установка зависимостей проекта
.PHONY: all
all:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	go install github.com/go-task/task/v3/cmd/task@latest
	task sync
	@echo ---
	@bash scripts/task-auto.sh