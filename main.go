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

type Summary struct {
	TotalItems      int     `json:"total_items"`
	TotalCategories int     `json:"total_categories"`
	TotalPrice      float64 `json:"total_price"`
}

var db *sql.DB

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

const MaxUploadSize = 10 << 20

func handlePostPrices(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(MaxUploadSize)
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	zipFilePath := "uploaded.zip"
	tempFile, err := os.Create(zipFilePath)
	if err != nil {
		http.Error(w, "Error creating temporary file", http.StatusInternalServerError)
		return
	}
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, file); err != nil {
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		return
	}

	zipReader, err := zip.OpenReader(zipFilePath)
	if err != nil {
		http.Error(w, "Error reading zip file", http.StatusInternalServerError)
		return
	}
	defer zipReader.Close()

	// Собираем записи из CSV
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
					log.Printf("Skipping CSV read error: %v", err)
					continue
				}
				if len(row) < 5 {
					log.Printf("Skipping malformed row: %v", row)
					continue
				}

				priceVal, err := strconv.ParseFloat(row[3], 64)
				if err != nil {
					log.Printf("Skipping invalid price: %s", row[3])
					continue
				}
				if row[1] == "" || row[2] == "" {
					log.Printf("Skipping empty name/category: %v", row)
					continue
				}

				layout := "2006-01-02"
				parsedDate, dateErr := time.Parse(layout, row[4])
				if dateErr != nil {
					log.Printf("Skipping invalid date: %s", row[4])
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
			if commitErr := tx.Commit(); commitErr != nil {
				log.Printf("Error committing transaction: %v", commitErr)
			}
		}
	}()

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

	for _, rec := range records {
		_, execErr := stmt.Exec(rec.ID, rec.Name, rec.Category, rec.Price, rec.CreateDate)
		if execErr != nil {
			log.Printf("Skipping insert error: %v", execErr)
			continue
		}
	}

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
		http.Error(w, "Error getting total categories", http.StatusInternalServerError)
		return
	}

	var totalPrice float64
	err = tx.QueryRow(`SELECT COALESCE(SUM(price), 0) FROM prices`).Scan(&totalPrice)
	if err != nil {
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

func handleGetPrices(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name, category, price, create_date FROM prices")
	if err != nil {
		http.Error(w, "Error fetching data from database", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

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
		if err = rows.Scan(&id, &name, &category, &price, &createDate); err != nil {
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
	if err = rows.Err(); err != nil {
		http.Error(w, "Error while iterating rows from database", http.StatusInternalServerError)
		return
	}

	csvFilePath := "data.csv"
	csvFile, err := os.Create(csvFilePath)
	if err != nil {
		http.Error(w, "Error creating CSV file", http.StatusInternalServerError)
		return
	}
	defer csvFile.Close()

	writer := csv.NewWriter(csvFile)
	defer writer.Flush()

	if err = writer.Write([]string{"id", "name", "category", "price", "create_date"}); err != nil {
		http.Error(w, "Error writing CSV header", http.StatusInternalServerError)
		return
	}

	for _, row := range data {
		if err = writer.Write([]string{
			row.ID, row.Name, row.Category,
			fmt.Sprintf("%.2f", row.Price),
			row.CreateDate,
		}); err != nil {
			http.Error(w, "Error writing row to CSV", http.StatusInternalServerError)
			return
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		http.Error(w, "Error finalizing CSV file", http.StatusInternalServerError)
		return
	}

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

	if _, err = io.Copy(wr, csvFileForZip); err != nil {
		http.Error(w, "Error writing to zip file", http.StatusInternalServerError)
		return
	}
	if err = zipWriter.Close(); err != nil {
		http.Error(w, "Error closing ZIP file", http.StatusInternalServerError)
		return
	}

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
		if err := db.Close(); err != nil {
			log.Printf("Error closing database connection: %v", err)
		} else {
			fmt.Println("Database connection closed successfully")
		}
	}
}

func main() {
	initDB()
	defer closeDB()

	r := mux.NewRouter()
	r.HandleFunc("/api/v0/prices", handlePostPrices).Methods("POST")
	r.HandleFunc("/api/v0/prices", handleGetPrices).Methods("GET")

	fmt.Println("Server is running on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
