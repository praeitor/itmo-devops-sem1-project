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

// Обработчик GET /api/v0/prices
func handleGetPrices(w http.ResponseWriter, r *http.Request) {
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
	zipFilePath := "data.zip"
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
	w.Header().Set("Content-Disposition", "attachment; filename=data.zip")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(zipBytes)))

	w.Write(zipBytes)
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
