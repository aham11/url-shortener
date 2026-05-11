package main

import (
	"database/sql"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

type expectedHTTP struct {
	status   int
	body     string
	location string
	hasShort bool
}

func setupTestDB(t *testing.T) {
	t.Helper()

	var err error
	db, err = sql.Open("sqlite", ":memory:")
	if err != nil {
		fatal(t, "err", err, nil)
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
		fatal(t, "err", err, nil)
	}
}

func Test_healthEndpoints(t *testing.T) {
	for _, test := range []struct {
		name     string
		path     string
		setupDB  bool
		expected expectedHTTP
	}{
		{
			name:    "live endpoint",
			path:    "/live",
			setupDB: false,
			expected: expectedHTTP{
				status: http.StatusOK,
				body:   "alive",
			},
		},
		{
			name:    "health endpoint",
			path:    "/health",
			setupDB: true,
			expected: expectedHTTP{
				status: http.StatusOK,
				body:   "healthy",
			},
		},
		{
			name:    "health endpoint without database",
			path:    "/health",
			setupDB: false,
			expected: expectedHTTP{
				status: http.StatusOK,
				body:   "healthy",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			db = nil
			if test.setupDB {
				setupTestDB(t)
			}
			req := httptest.NewRequest(http.MethodGet, test.path, nil)
			rec := httptest.NewRecorder()

			newHandler().ServeHTTP(rec, req)

			if rec.Code != test.expected.status {
				fatal(t, "status", rec.Code, test.expected.status)
			}

			body := strings.TrimSpace(rec.Body.String())
			if body != test.expected.body {
				fatal(t, "body", body, test.expected.body)
			}
		})
	}
}

func Test_shortenHandler(t *testing.T) {
	for _, test := range []struct {
		name     string
		method   string
		body     string
		expected expectedHTTP
	}{
		{
			name:   "valid url",
			method: http.MethodPost,
			body:   `{"url":"https://google.com"}`,
			expected: expectedHTTP{
				status:   http.StatusOK,
				hasShort: true,
			},
		},
		{
			name:   "empty url",
			method: http.MethodPost,
			body:   `{"url":""}`,
			expected: expectedHTTP{
				status: http.StatusBadRequest,
			},
		},
		{
			name:   "invalid json",
			method: http.MethodPost,
			body:   `{invalid-json}`,
			expected: expectedHTTP{
				status: http.StatusBadRequest,
			},
		},
		{
			name:   "method not allowed",
			method: http.MethodPut,
			body:   `{"url":"https://google.com"}`,
			expected: expectedHTTP{
				status: http.StatusMethodNotAllowed,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T){
			setupTestDB(t)

			sqliteServer := httptest.NewServer(newSQLiteHandler())
			defer sqliteServer.Close()

			t.Setenv("SQLITE_SERVICE_URL", sqliteServer.URL)
			req := httptest.NewRequest(test.method, "/", strings.NewReader(test.body))
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()

			newHandler().ServeHTTP(rec, req)

			if rec.Code != test.expected.status {
				fatal(t, "status", rec.Code, test.expected.status)
			}

			if !test.expected.hasShort {
				return
			}

			var resp ShortenResponse
			if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
				fatal(t, "err", err, nil)
			}

			if resp.Short == "" {
				fatal(t, "short", resp.Short, "non-empty short URL")
			}
		})
	}
}

func Test_initDB_twice(t *testing.T) {
	err := initDB()
	if err != nil {
		fatal(t, "first init", err, nil)
	}

	err = initDB()
	if err != nil {
		fatal(t, "second init", err, nil)
	}
}

func Test_handleRoot_edgeCases(t *testing.T) {
	handler := newHandler()

	tests := []struct {
		name   string
		path   string
		status int
	}{
		{"root", "/", http.StatusOK},
		{"not found", "/unknown123", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestDB(t)

			sqliteServer := httptest.NewServer(newSQLiteHandler())
			defer sqliteServer.Close()

			t.Setenv("SQLITE_SERVICE_URL", sqliteServer.URL)
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.status {
				t.Fatalf("got %d, expected %d", w.Code, tt.status)
			}
		})
	}
}

func Test_redirectHandler(t *testing.T) {
	for _, test := range []struct {
		name     string
		path     string
		seedCode string
		seedURL  string
		expected expectedHTTP
	}{
		{
			name:     "existing code redirects",
			path:     "/abc123",
			seedCode: "abc123",
			seedURL:  "https://example.com",
			expected: expectedHTTP{
				status:   http.StatusFound,
				location: "https://example.com",
			},
		},
		{
			name: "unknown code",
			path: "/unknown",
			expected: expectedHTTP{
				status: http.StatusNotFound,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			setupTestDB(t)

			sqliteServer := httptest.NewServer(newSQLiteHandler())
			defer sqliteServer.Close()

			t.Setenv("SQLITE_SERVICE_URL", sqliteServer.URL)

			if test.seedCode != "" {
				_, err := db.Exec(
					"INSERT INTO urls(code, original_url, created_at) VALUES (?, ?, ?)",
					test.seedCode,
					test.seedURL,
					"2026-04-28T00:00:00Z",
				)
				if err != nil {
					fatal(t, "err", err, nil)
				}
			}

			req := httptest.NewRequest(http.MethodGet, test.path, nil)
			rec := httptest.NewRecorder()

			newHandler().ServeHTTP(rec, req)

			if rec.Code != test.expected.status {
				fatal(t, "status", rec.Code, test.expected.status)
			}

			if test.expected.location != "" {
				location := rec.Header().Get("Location")
				if location != test.expected.location {
					fatal(t, "location", location, test.expected.location)
				}
			}
		})
	}
}

func Test_runFailsWhenPortIsAlreadyUsed(t *testing.T) {
	ln, err := net.Listen("tcp", ":8081")
	if err != nil {
		t.Skipf("cannot reserve port 8081: %v", err)
	}
	defer ln.Close()

	err = run()
	if err == nil {
		fatal(t, "err", err, "non nil")
	}
}

func fatal(t *testing.T, prefix string, got, expected any) {
	t.Helper()
	t.Fatalf("%ss: \ngot:\n%q\nexpected:\n%q", prefix, got, expected)
}
