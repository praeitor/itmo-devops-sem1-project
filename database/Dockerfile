# Используем официальный образ PostgreSQL
FROM postgres:14

# Устанавливаем переменные окружения для пользователя и базы данных
ENV POSTGRES_USER=validator
ENV POSTGRES_PASSWORD=val1dat0r
ENV POSTGRES_DB=project-sem-1

# Создаем SQL-скрипт для инициализации таблицы
COPY init.sql /docker-entrypoint-initdb.d/init.sql

# Устанавливаем права доступа
RUN chmod 644 /docker-entrypoint-initdb.d/init.sql

# Открываем порт PostgreSQL
EXPOSE 5432