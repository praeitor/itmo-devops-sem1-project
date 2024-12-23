package main

import (
	"archive/zip"
	"database/sql"
	"encoding/csv"
	"fmt"
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

	_, err = tempFile.ReadFrom(file)
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

	for _, f := range zipReader.File {
		if f.Name == "data.csv" {
			csvFile, _ := f.Open()
			defer csvFile.Close()

			reader := csv.NewReader(csvFile)
			reader.Read() // Skip header

			for {
				record, err := reader.Read()
				if err != nil {
					break
				}
				price, _ := strconv.ParseFloat(record[3], 64)
				db.Exec("INSERT INTO prices (id, name, category, price, create_date) VALUES ($1, $2, $3, $4, $5)",
					record[0], record[1], record[2], price, record[4])
			}
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Data uploaded successfully"))
}

func main() {
	initDB()
	r := mux.NewRouter()
	r.HandleFunc("/api/v0/prices", handlePostPrices).Methods("POST")
	http.ListenAndServe(":8080", r)
}
