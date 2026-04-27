#!/bin/bash
# Скрипт для быстрого запуска notification-template-service в development режиме

set -e

# Цвета для вывода
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Starting Notification Template Service...${NC}"

# Проверяем, запущен ли Docker
if ! docker info > /dev/null 2>&1; then
    echo -e "${YELLOW}Docker is not running. Please start Docker first.${NC}"
    exit 1
fi

# Запускаем сервисы через docker-compose
echo -e "${GREEN}Starting PostgreSQL and Jaeger...${NC}"
docker-compose up -d postgres jaeger

# Ждём пока PostgreSQL будет готов
echo -e "${GREEN}Waiting for PostgreSQL to be ready...${NC}"
until docker-compose exec -T postgres pg_isready -U monitor; do
    echo "PostgreSQL is unavailable - sleeping"
    sleep 2
done

echo -e "${GREEN}PostgreSQL is ready!${NC}"

# Применяем миграции
echo -e "${GREEN}Running migrations...${NC}"
make migrate-up

# Запускаем сервис
echo -e "${GREEN}Starting notification-template-service...${NC}"
make run
