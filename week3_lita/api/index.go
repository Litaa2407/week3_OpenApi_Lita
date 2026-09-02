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

	// Vercel meneruskan path melalui query parameter "route"
	path := r.URL.Query().Get("route")

	if path == "" {
		path = r.URL.Path
	}

	path = strings.TrimSpace(path)
	path = strings.TrimSuffix(path, "/")

	// =========================
	// HEALTH CHECK
	// =========================
	if path == "/health" {
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
		})
		return
	}

	// =========================
	// SWAGGER UI
	// =========================
	if path == "/swagger" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		_, _ = w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Neon Database CRUD API</title>

  <link
    rel="stylesheet"
    href="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui.css"
  />

  <style>
    body {
      margin: 0;
      background: #f5f7fb;
    }

    #swagger-ui {
      max-width: 1200px;
      margin: 20px auto;
    }
  </style>
</head>

<body>

  <div id="swagger-ui"></div>

  <script src="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui-bundle.js"></script>

  <script>
    window.onload = function () {
      SwaggerUIBundle({
        url: "/openapi.yaml",
        dom_id: "#swagger-ui",
        deepLinking: true,
        presets: [
          SwaggerUIBundle.presets.apis,
          SwaggerUIStandalonePreset
        ],
        layout: "BaseLayout"
      });
    };
  </script>

</body>
</html>`))

		return
	}

	// =========================
	// OPENAPI YAML
	// =========================
	if path == "/openapi.yaml" {
		data, err := os.ReadFile("openapi.yaml")

		if err != nil {
			writeError(
				w,
				http.StatusInternalServerError,
				"openapi file not found",
			)
			return
		}

		w.Header().Set(
			"Content-Type",
			"application/yaml; charset=utf-8",
		)

		_, _ = w.Write(data)
		return
	}

	// =========================
	// API ROUTES
	// =========================
	if !strings.HasPrefix(path, "/api/") {
		writeError(
			w,
			http.StatusNotFound,
			"not found",
		)
		return
	}

	// Ambil nama tabel
	tablePath := strings.TrimPrefix(path, "/api/")

	// Pisahkan jika ada ID
	parts := strings.Split(
		strings.Trim(tablePath, "/"),
		"/",
	)

	tableName := parts[0]

	if tableName == "" {
		writeError(
			w,
			http.StatusBadRequest,
			"table name is required",
		)
		return
	}

	// Hanya izinkan tiga tabel
	if tableName != "students" &&
		tableName != "teachers" &&
		tableName != "staff" {

		writeError(
			w,
			http.StatusNotFound,
			"table not found",
		)
		return
	}

	// =========================
	// DATABASE CONNECTION
	// =========================
	dsn := os.Getenv("DATABASE_URL")

	if dsn == "" {
		writeError(
			w,
			http.StatusInternalServerError,
			"DATABASE_URL is required",
		)
		return
	}

	db, err := sql.Open("postgres", dsn)

	if err != nil {
		writeError(
			w,
			http.StatusInternalServerError,
			err.Error(),
		)
		return
	}

	defer db.Close()

	// Test koneksi database
	if err := db.Ping(); err != nil {
		writeError(
			w,
			http.StatusInternalServerError,
			err.Error(),
		)
		return
	}

	// Pastikan tabel tersedia
	if err := ensureTable(db, tableName); err != nil {
		writeError(
			w,
			http.StatusInternalServerError,
			err.Error(),
		)
		return
	}

	// =========================
	// CRUD
	// =========================
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
		writeError(
			w,
			http.StatusMethodNotAllowed,
			"method not allowed",
		)
	}
}

// =====================================================
// ENSURE TABLE
// =====================================================

func ensureTable(db *sql.DB, tableName string) error {

	switch tableName {

	case "students", "teachers", "staff":

		query := fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				id SERIAL PRIMARY KEY,
				nama VARCHAR(100) NOT NULL,
				umur INTEGER NOT NULL,
				tanggal DATE NOT NULL
			)
		`, tableName)

		_, err := db.Exec(query)

		return err

	default:

		return fmt.Errorf(
			"table %s not found",
			tableName,
		)
	}
}

// =====================================================
// GET
// =====================================================

func listRows(
	w http.ResponseWriter,
	db *sql.DB,
	tableName string,
) {

	query := fmt.Sprintf(
		"SELECT id, nama, umur, tanggal FROM %s ORDER BY id ASC",
		tableName,
	)

	rows, err := db.Query(query)

	if err != nil {
		writeError(
			w,
			http.StatusInternalServerError,
			err.Error(),
		)
		return
	}

	defer rows.Close()

	data := []person{}

	for rows.Next() {

		var p person

		if err := rows.Scan(
			&p.ID,
			&p.Nama,
			&p.Umur,
			&p.Tanggal,
		); err != nil {

			writeError(
				w,
				http.StatusInternalServerError,
				err.Error(),
			)

			return
		}

		data = append(data, p)
	}

	if err := rows.Err(); err != nil {

		writeError(
			w,
			http.StatusInternalServerError,
			err.Error(),
		)

		return
	}

	json.NewEncoder(w).Encode(data)
}

// =====================================================
// POST
// =====================================================

func createRow(
	w http.ResponseWriter,
	db *sql.DB,
	tableName string,
	r *http.Request,
) {

	var p person

	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {

		writeError(
			w,
			http.StatusBadRequest,
			"invalid payload",
		)

		return
	}

	if err := validatePerson(p); err != nil {

		writeError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)

		return
	}

	query := fmt.Sprintf(`
		INSERT INTO %s
		(nama, umur, tanggal)
		VALUES ($1, $2, $3)
		RETURNING id
	`, tableName)

	err := db.QueryRow(
		query,
		p.Nama,
		p.Umur,
		p.Tanggal,
	).Scan(&p.ID)

	if err != nil {

		writeError(
			w,
			http.StatusInternalServerError,
			err.Error(),
		)

		return
	}

	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(p)
}

// =====================================================
// PUT
// =====================================================

func updateRow(
	w http.ResponseWriter,
	db *sql.DB,
	tableName string,
	r *http.Request,
) {

	id, err := getIDFromRequest(r)

	if err != nil {

		writeError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)

		return
	}

	var p person

	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {

		writeError(
			w,
			http.StatusBadRequest,
			"invalid payload",
		)

		return
	}

	if err := validatePerson(p); err != nil {

		writeError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)

		return
	}

	p.ID = id

	query := fmt.Sprintf(`
		UPDATE %s
		SET nama = $1,
			umur = $2,
			tanggal = $3
		WHERE id = $4
		RETURNING id
	`, tableName)

	err = db.QueryRow(
		query,
		p.Nama,
		p.Umur,
		p.Tanggal,
		id,
	).Scan(&p.ID)

	if err != nil {

		writeError(
			w,
			http.StatusNotFound,
			"record not found",
		)

		return
	}

	json.NewEncoder(w).Encode(p)
}

// =====================================================
// DELETE
// =====================================================

func deleteRow(
	w http.ResponseWriter,
	db *sql.DB,
	tableName string,
	r *http.Request,
) {

	id, err := getIDFromRequest(r)

	if err != nil {

		writeError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)

		return
	}

	query := fmt.Sprintf(`
		DELETE FROM %s
		WHERE id = $1
		RETURNING id
	`, tableName)

	var deletedID int64

	err = db.QueryRow(
		query,
		id,
	).Scan(&deletedID)

	if err != nil {

		writeError(
			w,
			http.StatusNotFound,
			"record not found",
		)

		return
	}

	json.NewEncoder(w).Encode(
		map[string]string{
			"status": "deleted",
			"id":     strconv.FormatInt(id, 10),
		},
	)
}

// =====================================================
// GET ID
// =====================================================

func getIDFromRequest(
	r *http.Request,
) (int64, error) {

	// Coba ambil ID dari query parameter
	// Contoh: ?id=1
	if id := r.URL.Query().Get("id"); id != "" {

		parsed, err := strconv.ParseInt(
			id,
			10,
			64,
		)

		if err != nil {
			return 0, errors.New("id must be integer")
		}

		return parsed, nil
	}

	// Coba ambil ID dari route
	// Contoh: /api/students/1
	path := r.URL.Query().Get("route")

	if path == "" {
		path = r.URL.Path
	}

	path = strings.Trim(path, "/")

	parts := strings.Split(path, "/")

	if len(parts) >= 3 {

		lastPart := parts[len(parts)-1]

		id, err := strconv.ParseInt(
			lastPart,
			10,
			64,
		)

		if err == nil {
			return id, nil
		}
	}

	return 0, errors.New("id is required")
}

// =====================================================
// VALIDATION
// =====================================================

func validatePerson(p person) error {

	if strings.TrimSpace(p.Nama) == "" {
		return errors.New("nama is required")
	}

	if p.Umur <= 0 {
		return errors.New("umur must be greater than 0")
	}

	if _, err := time.Parse(
		"2006-01-02",
		p.Tanggal,
	); err != nil {

		return errors.New(
			"tanggal must be in YYYY-MM-DD format",
		)
	}

	return nil
}

// =====================================================
// ERROR RESPONSE
// =====================================================

func writeError(
	w http.ResponseWriter,
	status int,
	message string,
) {

	w.WriteHeader(status)

	json.NewEncoder(w).Encode(
		map[string]string{
			"error": message,
		},
	)
}
