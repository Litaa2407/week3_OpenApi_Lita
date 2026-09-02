package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type person struct {
	ID      int64  `json:"id"`
	Nama    string `json:"nama"`
	Umur    int    `json:"umur"`
	Tanggal string `json:"tanggal"`
}

const swaggerHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Neon Database CRUD API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui.css" />
  <style>
    body { margin: 0; background: #f5f7fb; }
    #swagger-ui { max-width: 1200px; margin: 20px auto; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui-bundle.js"></script>
  <script>
    window.onload = function () {
      SwaggerUIBundle({
        url: "/openapi.yaml",
        dom_id: '#swagger-ui',
        deepLinking: true,
        presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
        layout: "BaseLayout",
      });
    };
  </script>
</body>
</html>`

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println(".env file not found, using system environment variables")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required. Create a .env file or set it in your environment.")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	if err := migrate(db); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "ok",
			"message": "Neon Database CRUD API",
			"endpoints": []string{
				"/health",
				"/swagger",
				"/api/students",
				"/api/teachers",
				"/api/staff",
			},
		})
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	http.HandleFunc("/swagger", serveSwagger)
	http.HandleFunc("/swagger/", serveSwagger)
	http.HandleFunc("/openapi.yaml", serveOpenAPIYAML)

	for _, tableName := range []string{"students", "teachers", "staff"} {
		path := "/api/" + tableName
		http.HandleFunc(path, handleTable(db, tableName))
		http.HandleFunc(path+"/", handleTable(db, tableName))
	}

	log.Printf("server listening on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func migrate(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS students (
			id SERIAL PRIMARY KEY,
			nama VARCHAR(100) NOT NULL,
			umur INTEGER NOT NULL,
			tanggal DATE NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS teachers (
			id SERIAL PRIMARY KEY,
			nama VARCHAR(100) NOT NULL,
			umur INTEGER NOT NULL,
			tanggal DATE NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS staff (
			id SERIAL PRIMARY KEY,
			nama VARCHAR(100) NOT NULL,
			umur INTEGER NOT NULL,
			tanggal DATE NOT NULL
		)`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}

	seedStatements := map[string]string{
		"students": `INSERT INTO students (nama, umur, tanggal)
			SELECT $1, $2, $3
			WHERE NOT EXISTS (SELECT 1 FROM students LIMIT 1)`,
		"teachers": `INSERT INTO teachers (nama, umur, tanggal)
			SELECT $1, $2, $3
			WHERE NOT EXISTS (SELECT 1 FROM teachers LIMIT 1)`,
		"staff": `INSERT INTO staff (nama, umur, tanggal)
			SELECT $1, $2, $3
			WHERE NOT EXISTS (SELECT 1 FROM staff LIMIT 1)`,
	}

	seedData := map[string][]person{
		"students": {
			{Nama: "Lita", Umur: 20, Tanggal: "2006-05-12"},
			{Nama: "Budi", Umur: 21, Tanggal: "2005-08-20"},
			{Nama: "Sinta", Umur: 19, Tanggal: "2007-01-15"},
		},
		"teachers": {
			{Nama: "Pak Andi", Umur: 40, Tanggal: "1986-03-10"},
			{Nama: "Bu Rina", Umur: 35, Tanggal: "1991-07-22"},
		},
		"staff": {
			{Nama: "Dina", Umur: 28, Tanggal: "1998-02-14"},
			{Nama: "Rudi", Umur: 32, Tanggal: "1994-11-05"},
		},
	}

	for tableName, rows := range seedData {
		if _, err := db.Exec("SELECT 1 FROM " + tableName + " LIMIT 1"); err != nil {
			return err
		}
		var count int
		row := db.QueryRow("SELECT COUNT(*) FROM " + tableName)
		if err := row.Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		for _, rowData := range rows {
			if _, err := db.Exec(seedStatements[tableName], rowData.Nama, rowData.Umur, rowData.Tanggal); err != nil {
				return err
			}
		}
	}

	return nil
}

func serveSwagger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(swaggerHTML))
}

func serveOpenAPIYAML(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	data, err := os.ReadFile("openapi.yaml")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "openapi file not found")
		return
	}
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	_, _ = w.Write(data)
}

func handleTable(db *sql.DB, tableName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		tableName = strings.TrimSpace(tableName)
		if tableName == "" {
			writeError(w, http.StatusBadRequest, "table name is required")
			return
		}

		switch r.Method {
		case http.MethodGet:
			listRows(w, db, tableName)
		case http.MethodPost:
			createRow(w, db, tableName, r)
		case http.MethodPut:
			updateRow(w, db, tableName, r)
		case http.MethodDelete:
			deleteRow(w, db, tableName, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}
}

func listRows(w http.ResponseWriter, db *sql.DB, tableName string) {
	rows, err := db.Query(fmt.Sprintf("SELECT id, nama, umur, tanggal FROM %s ORDER BY id ASC", tableName))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	data := []person{}
	for rows.Next() {
		var p person
		if err := rows.Scan(&p.ID, &p.Nama, &p.Umur, &p.Tanggal); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		data = append(data, p)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	json.NewEncoder(w).Encode(data)
}

func createRow(w http.ResponseWriter, db *sql.DB, tableName string, r *http.Request) {
	var p person
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if err := validatePerson(p); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	query := fmt.Sprintf("INSERT INTO %s (nama, umur, tanggal) VALUES ($1, $2, $3) RETURNING id", tableName)
	if err := db.QueryRow(query, p.Nama, p.Umur, p.Tanggal).Scan(&p.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)
}

func updateRow(w http.ResponseWriter, db *sql.DB, tableName string, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var p person
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if err := validatePerson(p); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	p.ID = id

	query := fmt.Sprintf("UPDATE %s SET nama = $1, umur = $2, tanggal = $3 WHERE id = $4 RETURNING id", tableName)
	if err := db.QueryRow(query, p.Nama, p.Umur, p.Tanggal, id).Scan(&p.ID); err != nil {
		writeError(w, http.StatusNotFound, "record not found")
		return
	}

	json.NewEncoder(w).Encode(p)
}

func deleteRow(w http.ResponseWriter, db *sql.DB, tableName string, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE id = $1 RETURNING id", tableName)
	var deletedID int64
	if err := db.QueryRow(query, id).Scan(&deletedID); err != nil {
		writeError(w, http.StatusNotFound, "record not found")
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "deleted", "id": strconv.FormatInt(id, 10)})
}

func getIDFromRequest(r *http.Request) (int64, error) {
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) >= 2 {
		id, err := strconv.ParseInt(parts[len(parts)-1], 10, 64)
		if err == nil {
			return id, nil
		}
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		return 0, errors.New("id is required")
	}
	parsedID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, errors.New("id must be integer")
	}
	return parsedID, nil
}

func validatePerson(p person) error {
	if strings.TrimSpace(p.Nama) == "" {
		return errors.New("nama is required")
	}
	if p.Umur <= 0 {
		return errors.New("umur must be greater than 0")
	}
	if _, err := time.Parse("2006-01-02", p.Tanggal); err != nil {
		return errors.New("tanggal must be in YYYY-MM-DD format")
	}
	return nil
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
