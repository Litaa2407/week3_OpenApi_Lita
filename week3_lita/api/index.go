package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
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

func Handler(w http.ResponseWriter, r *http.Request) {
	_ = godotenv.Load()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	path := strings.TrimSpace(r.URL.Path)
	path = strings.TrimSuffix(path, "/")

	if path == "/health" {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	if path == "/swagger" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!DOCTYPE html>
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
        layout: "BaseLayout"
      });
    };
  </script>
</body>
</html>`))
		return
	}

	if path == "/openapi.yaml" {
		data, err := os.ReadFile("openapi.yaml")
		if err != nil {
			writeError(w, http.StatusInternalServerError, "openapi file not found")
			return
		}
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		_, _ = w.Write(data)
		return
	}

	if !strings.HasPrefix(path, "/api/") {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	tableName := strings.TrimPrefix(path, "/api/")
	tableName = strings.TrimSpace(tableName)
	if tableName == "" {
		writeError(w, http.StatusBadRequest, "table name is required")
		return
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		writeError(w, http.StatusInternalServerError, "DATABASE_URL is required")
		return
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := ensureTable(db, tableName); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
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

func ensureTable(db *sql.DB, tableName string) error {
	switch tableName {
	case "students", "teachers", "staff":
		_, err := db.Exec(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
            id SERIAL PRIMARY KEY,
            nama VARCHAR(100) NOT NULL,
            umur INTEGER NOT NULL,
            tanggal DATE NOT NULL
        )`, tableName))
		return err
	default:
		return fmt.Errorf("table %s not found", tableName)
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
		if id, err := strconv.ParseInt(parts[len(parts)-1], 10, 64); err == nil {
			return id, nil
		}
	}

	if id := r.URL.Query().Get("id"); id != "" {
		parsed, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return 0, errors.New("id must be integer")
		}
		return parsed, nil
	}

	return 0, errors.New("id is required")
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
