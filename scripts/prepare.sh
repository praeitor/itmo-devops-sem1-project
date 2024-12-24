#!/bin/bash

# Переменные
DB_NAME="project-sem-1"
DB_USER="validator"
DB_PASSWORD="val1dat0r"

echo "Installing dependencies..."
sudo apt update
sudo apt install -y postgresql golang

echo "Starting PostgreSQL..."
sudo systemctl start postgresql
sudo systemctl enable postgresql

echo "Configuring PostgreSQL database..."

# Создаем пользователя, если его нет
sudo -u postgres psql -tc "SELECT 1 FROM pg_roles WHERE rolname='$DB_USER'" | grep -q 1 || \
sudo -u postgres psql -c "CREATE USER $DB_USER WITH PASSWORD '$DB_PASSWORD';"

# Создаем базу данных, если её нет
sudo -u postgres psql -tc "SELECT 1 FROM pg_database WHERE datname='$DB_NAME'" | grep -q 1 || \
sudo -u postgres psql -c "CREATE DATABASE $DB_NAME OWNER $DB_USER;"

# Предоставляем права на базу данных
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;"

# Создаем таблицу, если её нет
sudo -u postgres psql -d $DB_NAME -c "
DO \$\$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'prices') THEN
        CREATE TABLE prices (
            id SERIAL PRIMARY KEY,
            name TEXT NOT NULL,
            category TEXT NOT NULL,
            price NUMERIC(10,2) NOT NULL,
            create_date DATE NOT NULL
        );
        GRANT ALL PRIVILEGES ON TABLE prices TO $DB_USER;
    END IF;
END
\$\$;"

echo "Database and table setup completed successfully."