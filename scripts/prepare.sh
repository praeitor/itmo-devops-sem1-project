#!/bin/bash

set -e
set -o pipefail

# Переменные окружения для БД
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-project-sem-1}"
DB_USER="${DB_USER:-validator}"
DB_PASSWORD="${DB_PASSWORD:-val1dat0r}"

export PGPASSWORD=$DB_PASSWORD

# Проверка зависимостей
echo "Checking and installing Go dependencies..."
if ! go mod tidy; then
    echo "Failed to install dependencies."
    exit 1
fi
echo "Dependencies installed successfully."

# Ожидание доступности PostgreSQL
echo "Checking PostgreSQL availability on $DB_HOST:$DB_PORT..."
for i in {1..30}; do
    if psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c '\q' &>/dev/null; then
        echo "PostgreSQL is ready."
        break
    fi
    echo "Waiting for PostgreSQL to be ready ($i/30)..."
    sleep 2
    if [ $i -eq 30 ]; then
        echo "Error: Could not connect to PostgreSQL after 30 attempts."
        exit 1
    fi
done

# Добавляем тестовые данные в базу
echo "Applying database migrations..."
echo `pwd` 
PAYLOAD="prices_table.sql"
if [ ! -f "$PAYLOAD" ]; then
    echo "Payload file not found: $PAYLOAD"
    exit 1
fi

if ! psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$PAYLOAD"; then
    echo "Failed to apply payload from $PAYLOAD."
    exit 1
fi

echo "Payload applied successfully to database."