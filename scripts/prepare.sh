#!/bin/bash

# Скрипт для подготовки окружения и базы данных

# Настройки базы данных
DB_NAME="project_sem_1" # Исправлено имя БД (заменён дефис на нижнее подчеркивание)
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

sudo -u postgres psql -c "CREATE USER $DB_USER WITH PASSWORD '$DB_PASSWORD';"
sudo -u postgres psql -c "CREATE DATABASE $DB_NAME;"
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;"

# Создание таблицы в базе данных
echo "Creating table in PostgreSQL..."
sudo -u postgres psql -d $DB_NAME -c "
CREATE TABLE IF NOT EXISTS prices (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    price NUMERIC(10, 2) NOT NULL,
    create_date DATE NOT NULL
);"

echo "Database setup completed successfully."