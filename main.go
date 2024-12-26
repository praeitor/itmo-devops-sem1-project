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

	var totalItems int
	var totalPrice float64
	categorySet := make(map[string]bool)

	for _, f := range zipReader.File {
		if f.Name == "data.csv" || f.Name == "test_data.csv" {
			csvFile, err := f.Open()
			if err != nil {
				http.Error(w, "Error opening CSV file", http.StatusInternalServerError)
				return
			}
			defer csvFile.Close()

			reader := csv.NewReader(csvFile)
			_, err = reader.Read() // Пропускаем заголовок
			if err != nil {
				http.Error(w, "Error reading CSV header", http.StatusInternalServerError)
				return
			}

			for {
				record, err := reader.Read()
				if err != nil {
					break
				}

				price, err := strconv.ParseFloat(record[3], 64)
				if err != nil {
					log.Printf("Skipping invalid price: %s", record[3])
					continue
				}
				category := record[2]

				_, err = db.Exec("INSERT INTO prices (id, name, category, price, create_date) VALUES ($1, $2, $3, $4, $5)",
					record[0], record[1], category, price, record[4])
				if err != nil {
					log.Printf("Failed to insert record: %v", err)
					continue
				}

				totalItems++
				totalPrice += price
				categorySet[category] = true
			}
		}
	}

	summary := Summary{
		TotalItems:      totalItems,
		TotalCategories: len(categorySet),
		TotalPrice:      totalPrice,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

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

	var records [][]string
	var totalItems int
	var totalPrice float64
	categorySet := make(map[string]bool)

	// Читаем и валидируем CSV-файл
	for _, f := range zipReader.File {
		if f.Name == "data.csv" || f.Name == "test_data.csv" {
			csvFile, err := f.Open()
			if err != nil {
				http.Error(w, "Error opening CSV file", http.StatusInternalServerError)
				return
			}
			defer csvFile.Close()

			reader := csv.NewReader(csvFile)
			_, err = reader.Read() // Пропускаем заголовок
			if err != nil {
				http.Error(w, "Error reading CSV header", http.StatusInternalServerError)
				return
			}

			for {
				record, err := reader.Read()
				if err != nil {
					if err == io.EOF {
						break
					}
					log.Printf("Error reading CSV row: %v", err)
					continue
				}

				// Валидация записи
				if len(record) < 5 {
					log.Printf("Skipping invalid record: %v", record)
					continue
				}

				_, err = strconv.ParseFloat(record[3], 64)
				if err != nil {
					log.Printf("Skipping invalid price: %s", record[3])
					continue
				}

				records = append(records, record)
			}
		}
	}

	// Вставка данных в базу данных
	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	for _, record := range records {
		price, _ := strconv.ParseFloat(record[3], 64)
		category := record[2]

		_, err = tx.Exec(
			"INSERT INTO prices (id, name, category, price, create_date) VALUES ($1, $2, $3, $4, $5)",
			record[0], record[1], category, price, record[4],
		)
		if err != nil {
			log.Printf("Failed to insert record: %v", err)
			continue
		}

		totalItems++
		totalPrice += price
		categorySet[category] = true
	}

	// Завершаем транзакцию
	if err := tx.Commit(); err != nil {
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	// Формируем ответ
	summary := Summary{
		TotalItems:      totalItems,
		TotalCategories: len(categorySet),
		TotalPrice:      totalPrice,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

func closeDB() {
	if db != nil {
		err := db.Close()
		if err != nil {
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
	r.HandleFunc("/api/v0/prices", handlePostPrices).Methods("POST")
	r.HandleFunc("/api/v0/prices", handleGetPrices).Methods("GET")

	fmt.Println("🚀 Server is running on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
