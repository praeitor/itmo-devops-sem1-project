# Требования к системе

## Операционная система  
- **Ubuntu 20.04+** или любая современная **Linux-дистрибуция**  
- **Windows 10/11** с **WSL 2**  
- **macOS Monterey (12.0)+**  

## Аппаратные требования  
- **Процессор:** x86_64 архитектура (Intel/AMD)  
- **Оперативная память:** 2 ГБ (рекомендуется 4 ГБ)  
- **Свободное дисковое пространство:** минимум 2 ГБ  

## Программные зависимости  
- **Go:** версии 1.20+  
- **PostgreSQL:** версии 14+  
- **Docker:** версии 20.10+ (если используется контейнеризация)  
- **Bash:** версии 5.0+  
- **curl:** версии 7.68+  

## Сетевые требования  
- **Порт 8080:** для сервера должен быть свободен  
- **Порт 5432:** для PostgreSQL должен быть свободен  

---

# Используемые технологии  
- **Go (Golang):** язык программирования для разработки сервера  
- **PostgreSQL:** система управления базами данных  
- **Bash:** автоматизация подготовки, запуска и тестирования сервера  
- **Docker:** контейнеризация для базы данных  
- **GitHub Actions:** CI/CD для автоматического тестирования  

---

## Установка и запуск проекта

1. Клонирование репозитория
bash
Копировать код
`git clone https://github.com/your-repo/project.git
cd project`

2. Подготовка окружения
Запустите скрипт для установки зависимостей и настройки базы данных:
`chmod +x scripts/prepare.sh
./scripts/prepare.sh`

3. Настройка переменных окружения
Убедитесь, что заданы следующие переменные окружения:
`export POSTGRES_HOST=localhost
export POSTGRES_PORT=5432
export POSTGRES_DB=project-sem-1
export POSTGRES_USER=validator
export POSTGRES_PASSWORD=val1dat0r`

4. Запуск сервера
`chmod +x scripts/run.sh
./scripts/run.sh`

Сервер будет доступен по адресу: http://localhost:8080

---

## API Эндпоинты
1. POST /api/v0/prices
Описание: Загружает CSV-данные в базу данных.
Метод: POST
Параметры:

file – CSV-файл в формате ZIP-архива.
Пример запроса:

`curl -X POST -F "file=@sample_data.zip" http://localhost:8080/api/v0/prices`
Пример ответа (JSON):

`{
  "total_items": 100,
  "total_categories": 15,
  "total_price": 100000
}`

Параметры ответа:

total_items: Количество добавленных записей.
total_categories: Количество уникальных категорий.
total_price: Общая сумма всех товаров.

2. GET /api/v0/prices
Описание: Выгружает данные из базы в формате ZIP-архива.
Метод: GET

Пример запроса:

`curl -X GET http://localhost:8080/api/v0/prices -o response.zip`
Ответ: ZIP-архив с файлом data.csv.

Пример содержимого data.csv:

`id,name,category,price,create_date
1,iPhone 13,Electronics,799.99,2024-01-01
2,Nike Air Max,Shoes,129.99,2024-01-02`

---

## Тестирование
Запустите тесты с различными уровнями сложности:

Простой уровень:
`./scripts/tests.sh 1`
Продвинутый уровень:
`./scripts/tests.sh 2`
Сложный уровень:
`./scripts/tests.sh 3`

Результат:
Тесты должны завершиться успешно. В случае ошибки будет выведено описание проблемы.

##Переменные окружения
Переменная	Описание	Значение по умолчанию
POSTGRES_HOST	Хост базы данных	localhost
POSTGRES_PORT	Порт базы данных	5432
POSTGRES_DB	Имя базы данных	project-sem-1
POSTGRES_USER	Пользователь БД	validator
POSTGRES_PASSWORD	Пароль пользователя	val1dat0r

---

## Контакт

Почта: praeitor@gmail.com
