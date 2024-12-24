#!/bin/bash

set -e

echo "Installing PostgreSQL and dependencies..."

# Установка PostgreSQL
sudo apt-get update
sudo apt-get install -y postgresql postgresql-contrib

echo "Starting PostgreSQL service..."
sudo systemctl enable postgresql
sudo systemctl start postgresql

# Проверка статуса PostgreSQL
if ! sudo systemctl is-active --quiet postgresql; then
    echo "Error: PostgreSQL failed to start"
    exit 1
fi

echo "Configuring PostgreSQL..."

# Создание пользователя validator
sudo -u postgres psql -c "DO \$\$ BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'validator') THEN
        CREATE ROLE validator WITH LOGIN PASSWORD 'val1dat0r';
    END IF;
END \$\$;"

# Создание базы данных project-sem-1
sudo -u postgres psql -c "DO \$\$ BEGIN
    IF NOT EXISTS (SELECT FROM pg_database WHERE datname = 'project-sem-1') THEN
        CREATE DATABASE \"project-sem-1\" OWNER validator;
    END IF;
END \$\$;"

# Назначение прав доступа на базу данных
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE \"project-sem-1\" TO validator;"

# Создание таблицы prices
sudo -u postgres psql -d "project-sem-1" -c "
CREATE TABLE IF NOT EXISTS prices (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    price NUMERIC(10,2) NOT NULL,
    create_date DATE NOT NULL
);"

echo "PostgreSQL setup completed successfully."