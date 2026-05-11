package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	_ "modernc.org/sqlite"
	"net/http"
	"os"
	"strings"
	"time"
)

var db *sql.DB

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const shortCodeLength = 6

// ShortenRequest represents the JSON request body
type ShortenRequest struct {
	URL string `json:"url"`
}

// ShortenResponse represents the JSON response body
type ShortenResponse struct {
	Short string `json:"short"`
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

// Generate a random string for the short code
func generateShortCode() string {
	b := make([]byte, shortCodeLength)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func shortenHandler(w http.ResponseWriter, r *http.Request) {
	// Enable CORS for frontend requests
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"error": "Invalid request payload"}`, http.StatusBadRequest)
		return
	}

	// Generate a unique code
	sqliteURL := envOrDefault("SQLITE_SERVICE_URL", "http://url-shortener-sqlite")

	body, err := json.Marshal(req)
	if err != nil {
		http.Error(w, "failed to encode request", http.StatusInternalServerError)
		return
	}

	resp, err := http.Post(sqliteURL+"/internal/shorten", "application/json", bytes.NewReader(body))
	if err != nil {
		http.Error(w, "sqlite service unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "sqlite service error", http.StatusBadGateway)
		return
	}

	var sqliteResp struct {
		Code string `json:"code"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&sqliteResp); err != nil || sqliteResp.Code == "" {
		http.Error(w, "invalid sqlite response", http.StatusBadGateway)
		return
	}

	code := sqliteResp.Code

	// Build the short URL
	proto := "http"
	if r.TLS != nil {
		proto = "https"
	}
	// Fallback to localhost address  if Host is not set
	host := r.Host
	if host == "" {
		host = "localhost:" + envOrDefault("PORT", "8081")
	}
	shortURL := fmt.Sprintf("%s://%s/%s", proto, host, code)

	Response := ShortenResponse{Short: shortURL}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response)
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	// Enable CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Handle POST requests for shortening
	if r.Method == http.MethodPost {
		shortenHandler(w, r)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract code from path, e.g., /abc123
	code := r.URL.Path[1:] // remove leading "/"

	if code == "" {
		// Serve HTML on root path
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("URL shortener backend"))
		return
	}

	// Handle redirect for short codes
	sqliteURL := envOrDefault("SQLITE_SERVICE_URL", "http://url-shortener-sqlite")

	sqliteRespHTTP, err := http.Get(sqliteURL + "/internal/resolve/" + code)
	if err != nil {
		http.Error(w, "sqlite service unavailable", http.StatusBadGateway)
		return
	}
	defer sqliteRespHTTP.Body.Close()

	if sqliteRespHTTP.StatusCode == http.StatusNotFound {
		http.Error(w, "Short URL not found", http.StatusNotFound)
		return
	}

	if sqliteRespHTTP.StatusCode != http.StatusOK {
		http.Error(w, "sqlite service error", http.StatusBadGateway)
		return
	}

	var sqliteResp struct {
		URL string `json:"url"`
	}

	if err := json.NewDecoder(sqliteRespHTTP.Body).Decode(&sqliteResp); err != nil || sqliteResp.URL == "" {
		http.Error(w, "invalid sqlite response", http.StatusBadGateway)
		return
	}

	http.Redirect(w, r, sqliteResp.URL, http.StatusFound)

}
func initDB() error {
	var err error
	dbPth := envOrDefault("DB_PATH", "/data/urls.db")
	db, err = sql.Open("sqlite", dbPth)
	if err != nil {
		return fmt.Errorf("failed to open db: %w", err)
	}

	query := `
 	CREATE TABLE IF NOT EXISTS urls (
 	 id INTEGER PRIMARY KEY AUTOINCREMENT,
  	code TEXT NOT NULL UNIQUE,
  	original_url TEXT NOT NULL,
  	created_at TEXT NOT NULL,
  	visits INTEGER NOT NULL DEFAULT 0
 	);`

	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	return nil
}
func newHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("alive"))
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("healthy"))
	})

	mux.HandleFunc("/", handleRoot)

	return mux
}
func newSQLiteHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("alive"))
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if db == nil {
			http.Error(w, "database not initialized", http.StatusServiceUnavailable)
			return
		}

		if err := db.Ping(); err != nil {
			http.Error(w, "database not reachable", http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("healthy"))
	})

	mux.HandleFunc("/internal/shorten", internalShortenHandler)
	mux.HandleFunc("/internal/resolve/", internalResolveHandler)

	return mux
}
func internalShortenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ShortenRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	code := generateShortCode()

	_, err := db.Exec(
		"INSERT INTO urls(code, original_url, created_at) VALUES (?, ?, ?)",
		code,
		req.URL,
		time.Now().Format(time.RFC3339),
	)

	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	resp := map[string]string{
		"code": code,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func internalResolveHandler(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimPrefix(r.URL.Path, "/internal/resolve/")

	var originalURL string

	err := db.QueryRow(
		"SELECT original_url FROM urls WHERE code = ?",
		code,
	).Scan(&originalURL)

	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	resp := map[string]string{
		"url": originalURL,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
func run() error {
	mode := envOrDefault("APP_MODE", "backend")

	if mode == "sqlite" {
		if err := initDB(); err != nil {
			return fmt.Errorf("DB error: %w", err)
		}
		handler := newSQLiteHandler()
		port := envOrDefault("PORT", "8081")
		fmt.Println("SQLite owner is running at http://localhost:" + port)
		return http.ListenAndServe(":"+port, handler)
	}

	handler := newHandler()
	port := envOrDefault("PORT", "8081")
	fmt.Println("Backend is running at http://localhost:" + port)
	return http.ListenAndServe(":"+port, handler)
}
func main() {
	if err := run(); err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}
}
