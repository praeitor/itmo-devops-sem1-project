#!/bin/bash

set -e

echo "Installing dependencies..."
sudo apt-get update && sudo apt-get install -y golang postgresql-client

DB_HOST="localhost"
DB_PORT="5432"
DB_USER="validator"
DB_PASSWORD="val1dat0r"
DB_NAME="project_sem_1"

# Запуск PostgreSQL в GitHub Actions
echo "Starting PostgreSQL service..."
sudo systemctl start postgresql || sudo service postgresql start

echo "Checking PostgreSQL availability..."
for i in {1..30}; do
    if sudo systemctl is-active --quiet postgresql; then
        echo "PostgreSQL service is active."
        break
    fi
    echo "Waiting for PostgreSQL service to start ($i/30)..."
    sleep 2
    if [ $i -eq 30 ]; then
        echo "Error: PostgreSQL service failed to start after 30 attempts."
        exit 1
    fi
done

# Проверка подключения к PostgreSQL
for i in {1..30}; do
    if PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -c '\q' 2>/dev/null; then
        echo "PostgreSQL is available."
        break
    fi
    echo "Waiting for PostgreSQL to accept connections ($i/30)..."
    sleep 2
    if [ $i -eq 30 ]; then
        echo "Error: Could not connect to PostgreSQL after 30 attempts."
        exit 1
    fi
done

# Создание пользователя и базы данных
echo "Creating database user and database..."
sudo -u postgres psql -c "DO \$\$ BEGIN IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'validator') THEN CREATE ROLE validator WITH LOGIN PASSWORD 'val1dat0r'; END IF; END \$\$;"
sudo -u postgres psql -c "SELECT 1 FROM pg_database WHERE datname='project-sem-1'" | grep -q 1 || sudo -u postgres psql -c "CREATE DATABASE \"project-sem-1\" OWNER validator;"

echo "Ensuring table 'prices' exists..."
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d "$DB_NAME" -c "
CREATE TABLE IF NOT EXISTS prices (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    price NUMERIC(10,2) NOT NULL,
    create_date DATE NOT NULL
);"

echo "Adding test data if table is empty..."
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d "$DB_NAME" -c "
INSERT INTO prices (name, category, price, create_date)
SELECT 'Item 1', 'Category 1', 100.00, '2024-01-01'
WHERE NOT EXISTS (SELECT 1 FROM prices LIMIT 1);"

echo "Starting Go server..."
go run main.go &

for i in {1..10}; do
    if curl -s http://localhost:8080 &>/dev/null; then
        echo "Go server is ready."
        exit 0
    fi
    echo "Waiting for Go server to start ($i/10)..."
    sleep 2
done

echo "Error: Go server failed to start."
exit 1