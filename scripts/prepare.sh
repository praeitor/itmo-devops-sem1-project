#!/bin/bash

# Скрипт для подготовки окружения и базы данных

# Настройки базы данных
DB_NAME="project_sem_1"
DB_USER="validator"
DB_PASSWORD="val1dat0r"

# Установка зависимостей
echo "Installing dependencies..."
sudo apt update
sudo apt install -y postgresql golang

# Запуск PostgreSQL
echo "Starting PostgreSQL..."
sudo systemctl start postgresql
sudo systemctl enable postgresql

# Создание пользователя и базы данных
echo "Setting up PostgreSQL database..."

sudo -u postgres psql -c "DO \$\$ 
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = '$DB_USER') THEN
        CREATE ROLE $DB_USER WITH LOGIN PASSWORD '$DB_PASSWORD';
    END IF;
END
\$\$;"

sudo -u postgres psql -c "SELECT 'Database already exists' FROM pg_database WHERE datname = '$DB_NAME'" | grep -q "1 row" || \
sudo -u postgres psql -c "CREATE DATABASE $DB_NAME;"

sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;"

# Создание таблицы в базе данных
echo "Creating table in PostgreSQL..."
sudo -u postgres psql -d $DB_NAME -c "
DO \$\$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'prices') THEN
        CREATE TABLE prices (
            id SERIAL PRIMARY KEY,
            name TEXT NOT NULL,
            category TEXT NOT NULL,
            price NUMERIC(10, 2) NOT NULL,
            create_date DATE NOT NULL
        );
    END IF;
END
\$\$;"

echo "Database setup completed successfully."