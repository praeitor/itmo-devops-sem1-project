-- Создаем таблицу prices
CREATE TABLE IF NOT EXISTS prices (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    price NUMERIC(10,2) NOT NULL,
    create_date DATE NOT NULL
);

-- Добавляем тестовые данные
INSERT INTO prices (name, category, price, create_date)
VALUES 
    ('Item 1', 'Category 1', 100.00, '2024-01-01'),
    ('Item 2', 'Category 2', 200.00, '2024-01-02');

-- Назначаем права доступа пользователю
GRANT ALL PRIVILEGES ON TABLE prices TO validator;