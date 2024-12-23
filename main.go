package main

import (
	"archive/zip"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

var db *sql.DB

func initDB() {
	var err error
	connStr := "user=validator password=val1dat0r dbname=project-sem-1 sslmode=disable"
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}
	fmt.Println("Database connected successfully")
}

func handlePostPrices(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20)
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

	_, err = io.Copy(tempFile, file)
	if err != nil {
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		return
	}

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
		if f.Name == "test_data.csv" || f.Name == "data.csv" {
			csvFile, err := f.Open()
			if err != nil {
				http.Error(w, "Error opening CSV file", http.StatusInternalServerError)
				return
			}
			defer csvFile.Close()

			reader := csv.NewReader(csvFile)
			_, err = reader.Read() // Skip header
			if err != nil {
				http.Error(w, "Error reading CSV header", http.StatusInternalServerError)
				return
			}

			for {
				record, err := reader.Read()
				if err != nil {
					break
				}

				price, _ := strconv.ParseFloat(record[3], 64)
				category := record[2]

				_, err = db.Exec("INSERT INTO prices (id, name, category, price, create_date) VALUES ($1, $2, $3, $4, $5)",
					record[0], record[1], category, price, record[4])
				if err != nil {
					http.Error(w, "Error inserting data into database", http.StatusInternalServerError)
					return
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

func handleGetPrices(w http.ResponseWriter, r *http.Request) {
	file, _ := os.Create("data.csv")
	defer file.Close()

	writer := csv.NewWriter(file)
	writer.Write([]string{"id", "name", "category", "price", "create_date"})
	rows, _ := db.Query("SELECT id, name, category, price, create_date FROM prices")
	defer rows.Close()

	for rows.Next() {
		var id, name, category, date string
		var price float64
		rows.Scan(&id, &name, &category, &price, &date)
		writer.Write([]string{id, name, category, fmt.Sprintf("%.2f", price), date})
	}
	writer.Flush()

	zipFile, _ := os.Create("data.zip")
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	csvFile, _ := os.Open("data.csv")
	defer csvFile.Close()

	wr, _ := zipWriter.Create("data.csv")
	_, err := io.Copy(wr, csvFile)
	if err != nil {
		http.Error(w, "Failed to write to zip", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	http.ServeFile(w, r, "data.zip")
}

func main() {
	initDB()
	r := mux.NewRouter()
	r.HandleFunc("/api/v0/prices", handlePostPrices).Methods("POST")
	r.HandleFunc("/api/v0/prices", handleGetPrices).Methods("GET")
	http.ListenAndServe(":8080", r)
}
