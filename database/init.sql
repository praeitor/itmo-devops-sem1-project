-- Создание пользователя postgres, если его нет
DO
$$
BEGIN
   IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'postgres') THEN
      CREATE ROLE postgres WITH SUPERUSER CREATEDB CREATEROLE LOGIN PASSWORD 'postgres';
   END IF;
END
$$;

-- Создание базы данных с дефисом в имени
CREATE DATABASE "project-sem-1" ENCODING 'UTF8';

-- Подключаемся к базе данных
\c "project-sem-1"

-- Создание таблицы
CREATE TABLE IF NOT EXISTS prices (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    price NUMERIC(10,2) NOT NULL,
    create_date DATE NOT NULL
);

-- Добавление тестовых данных
INSERT INTO prices (name, category, price, create_date)
VALUES 
    ('Item 1', 'Category 1', 100.00, '2024-01-01'),
    ('Item 2', 'Category 2', 200.00, '2024-01-02');

-- Назначение прав доступа
GRANT ALL PRIVILEGES ON DATABASE "project-sem-1" TO validator;
GRANT ALL PRIVILEGES ON TABLE prices TO validator;