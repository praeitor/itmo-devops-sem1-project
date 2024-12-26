package main

import (
	"archive/zip"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// Summary структура для ответа POST запроса
type Summary struct {
	TotalItems      int     `json:"total_items"`
	TotalCategories int     `json:"total_categories"`
	TotalPrice      float64 `json:"total_price"`
}

var db *sql.DB

// Инициализация подключения к базе данных
func initDB() {
	var err error
	connStr := "host=localhost port=5432 user=validator password=val1dat0r dbname=project-sem-1 sslmode=disable"
	fmt.Println("Connecting to database with:", connStr)

	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatalf("Database not reachable: %v", err)
	}

	fmt.Println("Database connected successfully")
}

const MaxUploadSize = 10 << 20 // Ограничение на размер файла 10MB

// Обработчик POST /api/v0/prices
func handlePostPrices(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(MaxUploadSize)
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Сохраняем zip-файл
	zipFilePath := "uploaded.zip"
	tempFile, err := os.Create(zipFilePath)
	if err != nil {
		http.Error(w, "Error creating temporary file", http.StatusInternalServerError)
		return
	}
	defer tempFile.Close()

	_, err = io.Copy(tempFile, file)
	if err != nil {
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		return
	}

	// Открываем zip-файл
	zipReader, err := zip.OpenReader(zipFilePath)
	if err != nil {
		http.Error(w, "Error reading zip file", http.StatusInternalServerError)
		return
	}
	defer zipReader.Close()

	var records []struct {
		ID         string
		Name       string
		Category   string
		Price      float64
		CreateDate time.Time
	}

	for _, f := range zipReader.File {
		if f.Name == "test_data.csv" || f.Name == "data.csv" {
			csvFile, err := f.Open()
			if err != nil {
				http.Error(w, "Error opening CSV file", http.StatusInternalServerError)
				return
			}
			reader := csv.NewReader(csvFile)

			// Пропускаем заголовок
			_, _ = reader.Read()

			for {
				row, err := reader.Read()
				if err == io.EOF {
					break
				}
				if err != nil {
					log.Printf("Skipping row due to CSV read error: %v", err)
					continue
				}

				if len(row) < 5 {
					log.Printf("Skipping malformed row: %v", row)
					continue
				}

				// Парсим price
				priceVal, err := strconv.ParseFloat(row[3], 64)
				if err != nil {
					log.Printf("Skipping invalid price: %s", row[3])
					continue
				}

				// Пропускаем пустые name/category
				if row[1] == "" || row[2] == "" {
					log.Printf("Skipping row with empty name/category: %v", row)
					continue
				}

				// Парсим дату в Go, чтобы не передавать "invalid_date" в SQL
				layout := "2006-01-02" // формат "YYYY-MM-DD"
				parsedDate, dateErr := time.Parse(layout, row[4])
				if dateErr != nil {
					log.Printf("Skipping row due to invalid date: %s", row[4])
					continue
				}

				records = append(records, struct {
					ID         string
					Name       string
					Category   string
					Price      float64
					CreateDate time.Time
				}{
					ID:         row[0],
					Name:       row[1],
					Category:   row[2],
					Price:      priceVal,
					CreateDate: parsedDate,
				})
			}
			csvFile.Close()
		}
	}

	// Начинаем транзакцию
	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "Error starting transaction", http.StatusInternalServerError)
		return
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			if cErr := tx.Commit(); cErr != nil {
				log.Printf("Error committing transaction: %v", cErr)
			}
		}
	}()

	// Готовим INSERT
	stmt, err := tx.Prepare(`
        INSERT INTO prices (id, name, category, price, create_date)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (id) DO NOTHING
    `)
	if err != nil {
		http.Error(w, "Error preparing statement", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	// Вставляем все валидные записи
	for _, rec := range records {
		_, execErr := stmt.Exec(rec.ID, rec.Name, rec.Category, rec.Price, rec.CreateDate)
		if execErr != nil {
			log.Printf("Skipping row due to insert error (maybe duplicate or date mismatch?): %v", execErr)
			continue
		}
	}

	// Считаем статистику
	var totalItems int
	err = tx.QueryRow(`SELECT COUNT(*) FROM prices`).Scan(&totalItems)
	if err != nil {
		log.Printf("DB error on counting items: %v", err)
		http.Error(w, "Error getting total items", http.StatusInternalServerError)
		return
	}

	var totalCategories int
	err = tx.QueryRow(`SELECT COUNT(DISTINCT category) FROM prices`).Scan(&totalCategories)
	if err != nil {
		log.Printf("DB error on getting total categories: %v", err)
		http.Error(w, "Error getting total categories", http.StatusInternalServerError)
		return
	}

	var totalPrice float64
	err = tx.QueryRow(`SELECT COALESCE(SUM(price), 0) FROM prices`).Scan(&totalPrice)
	if err != nil {
		log.Printf("DB error on getting total price: %v", err)
		http.Error(w, "Error getting total price", http.StatusInternalServerError)
		return
	}

	summary := Summary{
		TotalItems:      totalItems,
		TotalCategories: totalCategories,
		TotalPrice:      totalPrice,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// Обработчик GET /api/v0/prices
func handleGetPrices(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received %s request for %s", r.Method, r.URL.Path)

	// 1. Считываем данные из базы
	rows, err := db.Query("SELECT id, name, category, price, create_date FROM prices")
	if err != nil {
		http.Error(w, "Error fetching data from database", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Слайс для хранения прочитанных данных
	var data []struct {
		ID         string
		Name       string
		Category   string
		Price      float64
		CreateDate string
	}

	for rows.Next() {
		var (
			id, name, category, createDate string
			price                          float64
		)

		err = rows.Scan(&id, &name, &category, &price, &createDate)
		if err != nil {
			http.Error(w, "Error reading row from database", http.StatusInternalServerError)
			return
		}

		data = append(data, struct {
			ID         string
			Name       string
			Category   string
			Price      float64
			CreateDate string
		}{
			ID:         id,
			Name:       name,
			Category:   category,
			Price:      price,
			CreateDate: createDate,
		})
	}

	// Очень важно проверить rows.Err() после цикла
	if err = rows.Err(); err != nil {
		http.Error(w, "Error while iterating rows from database", http.StatusInternalServerError)
		return
	}

	// 2. Создаём CSV файл
	csvFilePath := "data.csv"
	csvFile, err := os.Create(csvFilePath)
	if err != nil {
		http.Error(w, "Error creating CSV file", http.StatusInternalServerError)
		return
	}
	defer csvFile.Close()

	writer := csv.NewWriter(csvFile)
	defer writer.Flush()

	// Записываем заголовки CSV
	err = writer.Write([]string{"id", "name", "category", "price", "create_date"})
	if err != nil {
		http.Error(w, "Error writing CSV header", http.StatusInternalServerError)
		return
	}

	// 3. Записываем все прочитанные ранее данные в CSV
	for _, row := range data {
		err = writer.Write([]string{
			row.ID,
			row.Name,
			row.Category,
			fmt.Sprintf("%.2f", row.Price),
			row.CreateDate,
		})
		if err != nil {
			http.Error(w, "Error writing row to CSV", http.StatusInternalServerError)
			return
		}
	}

	// Завершаем запись и проверяем на ошибку
	writer.Flush()
	if err := writer.Error(); err != nil {
		http.Error(w, "Error finalizing CSV file", http.StatusInternalServerError)
		return
	}

	// 4. Создаём ZIP-архив с нашим CSV
	zipFilePath := "response.zip"
	zipFile, err := os.Create(zipFilePath)
	if err != nil {
		http.Error(w, "Error creating ZIP file", http.StatusInternalServerError)
		return
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)

	csvFileForZip, err := os.Open(csvFilePath)
	if err != nil {
		http.Error(w, "Error opening CSV file for zipping", http.StatusInternalServerError)
		return
	}
	defer csvFileForZip.Close()

	wr, err := zipWriter.Create("data.csv")
	if err != nil {
		http.Error(w, "Error creating zip entry", http.StatusInternalServerError)
		return
	}

	_, err = io.Copy(wr, csvFileForZip)
	if err != nil {
		http.Error(w, "Error writing to zip file", http.StatusInternalServerError)
		return
	}

	err = zipWriter.Close()
	if err != nil {
		http.Error(w, "Error closing ZIP file", http.StatusInternalServerError)
		return
	}

	// 5. Проверяем размер и отправляем файл
	stat, err := os.Stat(zipFilePath)
	if err != nil || stat.Size() == 0 {
		http.Error(w, "ZIP file is empty or inaccessible", http.StatusInternalServerError)
		return
	}

	zipBytes, err := os.ReadFile(zipFilePath)
	if err != nil {
		http.Error(w, "Error reading ZIP file", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=response.zip")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(zipBytes)))

	w.Write(zipBytes)
}

func closeDB() {
	if db != nil {
		if err := db.Close(); err != nil {
			log.Printf("Error closing database connection: %v", err)
		} else {
			fmt.Println("Database connection closed successfully")
		}
	}
}

func main() {
	initDB()
	defer closeDB() // Закрываем соединение с БД при завершении программы
	r := mux.NewRouter()

	// Регистрация маршрутов с явным указанием методов "GET" и "HEAD"
	r.HandleFunc("/api/v0/prices", handlePostPrices).Methods("POST")
	r.HandleFunc("/api/v0/prices", handleGetPrices).Methods("GET", "HEAD")

	// Обработка неразрешённых методов
	r.MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})

	fmt.Println("Server is running on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
