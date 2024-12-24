#!/bin/bash

set -e

echo "Installing PostgreSQL client..."
sudo apt-get update
sudo apt-get install -y postgresql-client

# Переменные окружения
DB_HOST="${POSTGRES_HOST:-localhost}"
DB_PORT="${POSTGRES_PORT:-5432}"
DB_USER="${POSTGRES_USER:-validator}"
DB_PASSWORD="${POSTGRES_PASSWORD:-val1dat0r}"
DB_NAME="${POSTGRES_DB:-project-sem-1}"

# Ожидание готовности PostgreSQL
echo "Checking PostgreSQL availability on $DB_HOST:$DB_PORT..."
for i in {1..30}; do
    if PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U "$DB_USER" -c '\q' 2>/dev/null; then
        echo "PostgreSQL is available and accepting connections."
        break
    fi
    echo "Waiting for PostgreSQL to accept connections ($i/30)..."
    sleep 2
    if [ $i -eq 30 ]; then
        echo "Error: Could not connect to PostgreSQL after 30 attempts."
        echo "Running diagnostic commands..."
        echo "Checking PostgreSQL service status:"
        docker ps -a
        echo "Checking logs:"
        docker logs $(docker ps -aqf "name=postgres")
        exit 1
    fi
done

# Подключение к базе данных как пользователь postgres
echo "Connecting to PostgreSQL as user 'postgres' to perform initial setup..."

# Создание пользователя, если он не существует
echo "Ensuring PostgreSQL user '$DB_USER' exists..."
PGPASSWORD=$POSTGRES_PASSWORD psql -h $DB_HOST -p $DB_PORT -U postgres -tc "SELECT 1 FROM pg_roles WHERE rolname='$DB_USER'" | grep -q 1 || \
PGPASSWORD=$POSTGRES_PASSWORD psql -h $DB_HOST -p $DB_PORT -U postgres -c "CREATE USER $DB_USER WITH PASSWORD '$DB_PASSWORD';"

# Создание базы данных, если она не существует
echo "Ensuring PostgreSQL database '$DB_NAME' exists..."
PGPASSWORD=$POSTGRES_PASSWORD psql -h $DB_HOST -p $DB_PORT -U postgres -tc "SELECT 1 FROM pg_database WHERE datname='$DB_NAME'" | grep -q 1 || \
PGPASSWORD=$POSTGRES_PASSWORD psql -h $DB_HOST -p $DB_PORT -U postgres -c "CREATE DATABASE \"$DB_NAME\" OWNER $DB_USER;"

# Назначение прав доступа
echo "Granting privileges on database '$DB_NAME' to user '$DB_USER'..."
PGPASSWORD=$POSTGRES_PASSWORD psql -h $DB_HOST -p $DB_PORT -U postgres -c "GRANT ALL PRIVILEGES ON DATABASE \"$DB_NAME\" TO $DB_USER;"

# Создание таблицы prices
echo "Ensuring table 'prices' exists in database '$DB_NAME'..."
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d "$DB_NAME" -c "
CREATE TABLE IF NOT EXISTS prices (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    price NUMERIC(10,2) NOT NULL,
    create_date DATE NOT NULL
);"

echo "PostgreSQL setup completed successfully."