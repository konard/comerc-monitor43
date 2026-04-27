#!/bin/bash
# Скрипт для запуска E2E тестов
#
# Использование:
#   ./scripts/run-e2e.sh              # Запустить тесты
#   ./scripts/run-e2e.sh --setup      # Запустить зависимости и тесты
#   ./scripts/run-e2e.sh --teardown   # Остановить зависимости

set -e

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Функции для вывода
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Проверка зависимостей
check_dependencies() {
    log_info "Проверка зависимостей..."

    if ! command -v go &> /dev/null; then
        log_error "Go не установлен"
        exit 1
    fi

    if ! command -v docker &> /dev/null; then
        log_warn "Docker не установлен. Запуск зависимостей невозможен."
    fi

    log_info "Зависимости проверены"
}

# Запуск зависимостей
setup_dependencies() {
    log_info "Запуск зависимостей..."

    if [ -f "docker-compose.yml" ]; then
        docker-compose up -d postgres rabbitmq
        log_info "Ожидание запуска сервисов..."
        sleep 5
        log_info "Сервисы запущены"
    else
        log_warn "docker-compose.yml не найден"
    fi
}

# Остановка зависимостей
teardown_dependencies() {
    log_info "Остановка зависимостей..."

    if [ -f "docker-compose.yml" ]; then
        docker-compose down
        log_info "Сервисы остановлены"
    fi
}

# Проверка, что сервис запущен
check_service() {
    log_info "Проверка, что monitor-service запущен на localhost:50051..."

    if nc -z localhost 50051 2>/dev/null; then
        log_info "monitor-service запущен"
        return 0
    else
        log_error "monitor-service не запущен на localhost:50051"
        log_info "Запустите сервис: task run"
        return 1
    fi
}

# Запуск тестов
run_tests() {
    log_info "Запуск E2E тестов..."

    cd test/e2e
    go test -v

    log_info "Тесты завершены"
}

# Главная функция
main() {
    cd "$(dirname "$0")/.."

    check_dependencies

    case "${1:-}" in
        --setup)
            setup_dependencies
            run_tests
            ;;
        --teardown)
            teardown_dependencies
            ;;
        --help|-h)
            echo "Использование: $0 [OPTION]"
            echo ""
            echo "Options:"
            echo "  --setup      Запустить зависимости и тесты"
            echo "  --teardown   Остановить зависимости"
            echo "  --help, -h   Показать эту справку"
            echo ""
            echo "Без опций: только проверить и запустить тесты"
            ;;
        *)
            check_service || exit 1
            run_tests
            ;;
    esac
}

main "$@"
