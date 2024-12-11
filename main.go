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
	"strconv"
	"strings"

	_ "github.com/lib/pq"
)

const (
	dbUser     = "validator"
	dbPassword = "val1dat0r"
	dbName     = "project-sem-1"
	dbHost     = "localhost"
	dbPort     = 5432
)

var db *sql.DB

// Инициализация базы данных
func initDB() {
	var err error
	dsn := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%d sslmode=disable",
		dbUser, dbPassword, dbName, dbHost, dbPort)
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS prices (
		id SERIAL PRIMARY KEY,
		product_id INT,
		created_at DATE,
		product_name TEXT,
		category TEXT,
		price FLOAT
	)`)
	if err != nil {
		log.Fatalf("Error creating table: %v", err)
	}

	fmt.Println("Database initialized successfully")
}

// Обработчик POST prices
func postPricesHandler(w http.ResponseWriter, r *http.Request) {
	archiveType := r.URL.Query().Get("type")
	if archiveType == "" {
		archiveType = "zip"
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read file content", http.StatusBadRequest)
		return
	}

	var records [][]string
	if archiveType == "zip" {
		records, err = unzipArchive(fileBytes)
	} else {
		http.Error(w, "Unsupported archive type", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, "Failed to process archive", http.StatusInternalServerError)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "Failed to begin transaction", http.StatusInternalServerError)
		return
	}

	totalItems := 0
	categoryMap := make(map[string]struct{})
	totalPrice := 0.0

	for _, record := range records {
		productID, _ := strconv.Atoi(record[0])
		productName := record[2]
		category := record[3]
		price, _ := strconv.ParseFloat(record[4], 64)
		createdAt := record[1]

		_, err := tx.Exec("INSERT INTO prices (product_id, created_at, product_name, category, price) VALUES ($1, $2, $3, $4, $5)",
			productID, createdAt, productName, category, price)
		if err != nil {
			tx.Rollback()
			http.Error(w, "Failed to insert data", http.StatusInternalServerError)
			return
		}

		totalItems++
		categoryMap[category] = struct{}{}
		totalPrice += price
	}

	err = tx.Commit()
	if err != nil {
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"total_items":      totalItems,
		"total_categories": len(categoryMap),
		"total_price":      totalPrice,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func unzipArchive(fileBytes []byte) ([][]string, error) {
	reader, err := zip.NewReader(strings.NewReader(string(fileBytes)), int64(len(fileBytes)))
	if err != nil {
		return nil, err
	}

	var records [][]string
	for _, file := range reader.File {
		if strings.HasSuffix(file.Name, ".csv") {
			f, err := file.Open()
			if err != nil {
				return nil, err
			}
			defer f.Close()

			csvReader := csv.NewReader(f)
			records, err = csvReader.ReadAll()
			if err != nil {
				return nil, err
			}
		}
	}
	return records, nil
}

func main() {
	initDB()
	defer db.Close()

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Server is running"))
	})

	fmt.Println("Starting server on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
