package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
)

// URLStore safely manages URLs in memory
type URLStore struct {
	sync.RWMutex
	urls map[string]string
}

var store = URLStore{
	urls: make(map[string]string),
}

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
	var code string
	store.Lock()
	for {
		code = generateShortCode()
		if _, exists := store.urls[code]; !exists {
			break
		}
	}
	// Store the mapping in memory
	store.urls[code] = req.URL
	store.Unlock()

	// Build the short URL
	proto := "http"
	if r.TLS != nil {
		proto = "https"
	}
	// Fallback to localhost:8080 if Host is not set or empty (it usually is set)
	host := r.Host
	if host == "" {
		host = "localhost:8080"
	}
	shortURL := fmt.Sprintf("%s://%s/%s", proto, host, code)

	resp := ShortenResponse{Short: shortURL}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
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
		http.ServeFile(w, r, "index.html")
		return
	}

	// Handle redirect for short codes
	store.RLock()
	originalURL, exists := store.urls[code]
	store.RUnlock()

	if !exists {
		http.Error(w, "Short URL not found", http.StatusNotFound)
		return
	}

	// Redirect user to the original URL
	http.Redirect(w, r, originalURL, http.StatusFound)
}

func main() {
	
 	http.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) {
 	 w.WriteHeader(http.StatusOK)
  	w.Write([]byte("alive"))
 	})

 	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
  	 w.WriteHeader(http.StatusOK)
  	 w.Write([]byte("healthy"))
 	})
	http.HandleFunc("/", handleRoot) // Handles GET /, POST /, and GET /:code

	fmt.Println("Server is running at http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}
}
