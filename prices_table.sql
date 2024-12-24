-- Создание таблицы prices, если она не существует
CREATE TABLE IF NOT EXISTS prices (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    price NUMERIC(10, 2) NOT NULL,
    create_date DATE NOT NULL
);

-- Вставка данных в таблицу prices
INSERT INTO prices (id, name, category, price, create_date) VALUES
(1, 'iPhone 13', 'Electronics', 799.99, '2024-01-01'),
(21, 'iPhone 13', 'Electronics', 799.99, '2024-01-01'),
(1000, 'iPhone 13', 'Electronics', 799.99, '2024-01-01'),
(100, 'iPhone 13', 'Electronics', 799.99, '2024-01-01'),
(10, 'iPhone 13', 'Electronics', 799.99, '2024-01-01'),
(2, 'Nike Air Max', 'Shoes', 129.99, '2024-01-02'),
(3, 'Coffee Maker', 'Appliances', 59.99, '2024-01-03'),
(4, 'Python Book', 'Books', 45.50, '2024-01-15'),
(5, 'Gaming Mouse', 'Electronics', 89.99, '2024-01-20'),
(6, 'Smart Watch', 'Electronics', 299.99, '2024-01-25'),
(7, 'Desk Lamp', 'Home', 34.99, '2024-01-30'),
(8, 'Bluetooth Speaker', 'Electronics', 79.99, '2024-01-31');

-- Подтверждение изменений
COMMIT;