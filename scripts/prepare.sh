#!/bin/bash

set -e

echo "Installing dependencies..."
sudo apt-get update && sudo apt-get install -y golang postgresql-client

DB_HOST="localhost"
DB_PORT="5432"
DB_USER="validator"
DB_PASSWORD="val1dat0r"
DB_NAME="project-sem-1"

echo "Checking PostgreSQL availability..."
until PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -c '\q' 2>/dev/null; do
    echo "Waiting for PostgreSQL to start..."
    sleep 1
done

echo "Creating database $DB_NAME if not exists..."
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -tc "SELECT 1 FROM pg_database WHERE datname='$DB_NAME'" | grep -q 1 || \
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -c "CREATE DATABASE \"$DB_NAME\" ENCODING 'UTF8';"

echo "Ensuring table prices exists..."
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