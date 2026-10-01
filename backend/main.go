package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
	_ "modernc.org/sqlite"
)

type ContactRequest struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Service string `json:"service"`
	Message string `json:"message"`
}

type EmailRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Text    string   `json:"text"`
}

type RateLimitEntry struct {
	Count       int
	WindowStart time.Time
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	response := map[string]string{
		"status":  "okay",
		"service": "itqan-api",
	}

	writeJSON(w, http.StatusOK, response)
}

func contactHandler(db *sql.DB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			response := map[string]string{
				"error": "method not allowed",
			}
			slog.Warn("method not allowed", "method", r.Method)
			writeJSON(w, http.StatusMethodNotAllowed, response)
			return
		}

		defer r.Body.Close()

		var data ContactRequest
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&data)
		if err != nil {

			response := map[string]string{
				"error": "invalid body",
			}
			slog.Warn("invalid contact request", "error", err)
			writeJSON(w, http.StatusBadRequest, response)
			return
		}
		if data.Name == "" || data.Email == "" || data.Message == "" {
			response := map[string]string{
				"error": "name, email, and message are required",
			}
			slog.Warn("contact request missing required fields")
			writeJSON(w, http.StatusBadRequest, response)
			return
		}
		if !isValidEmail(data.Email) {
			response := map[string]string{
				"error": "invalid email address",
			}
			slog.Warn("contact request has invalid email")
			writeJSON(w, http.StatusBadRequest, response)
			return
		}

		err = saveContact(db, data)
		if err != nil {
			response := map[string]string{
				"error": "failed to save contact",
			}
			slog.Error("failed to save contact", "error", err)
			writeJSON(w, http.StatusInternalServerError, response)

			return
		}

		emailErr := sendContactEmail(data)
		if emailErr != nil {
			slog.Error("failed to send contact email", "error", emailErr)
		}

		response := map[string]string{
			"message": "contact request received",
		}

		slog.Info("contact request saved")
		writeJSON(w, http.StatusOK, response)
	}
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
	if allowedOrigin == "" {
		log.Fatal("ALLOWED_ORIGIN is not set")
	}
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next(w, r)

	}

}

func RateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	visitors := make(map[string]RateLimitEntry)
	var mu sync.Mutex

	return func(w http.ResponseWriter, r *http.Request) {
		remoteAddr := r.RemoteAddr
		host, _, err := net.SplitHostPort(remoteAddr)
		if err != nil {
			log.Println(err)
			next(w, r)
			return
		}
		mu.Lock()

		entry, exists := visitors[host]
		now := time.Now()
		elapsed := now.Sub(entry.WindowStart)

		if !exists {
			entry = RateLimitEntry{
				Count:       1,
				WindowStart: now,
			}
			visitors[host] = entry
		} else if elapsed >= 10*time.Minute {
			entry = RateLimitEntry{
				Count:       1,
				WindowStart: now,
			}
			visitors[host] = entry
		} else {

			entry.Count++
			visitors[host] = entry

		}
		mu.Unlock()

		if entry.Count > 3 {
			slog.Warn("contact rate limit exceeded", "ip", host)
			w.WriteHeader(http.StatusTooManyRequests)
			return

		}

		next(w, r)

	}

}

func isValidEmail(email string) bool {
	emailParts := strings.Split(email, "@")
	if len(emailParts) != 2 {
		return false
	}
	if emailParts[0] == "" || emailParts[1] == "" {
		return false

	}
	if !strings.Contains(emailParts[1], ".") {
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	err := encoder.Encode(data)
	if err != nil {
		log.Println(err)
	}
}

func initDB() *sql.DB {
	db, err := sql.Open("sqlite", "itqan.db")
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	createTable := `
	CREATE TABLE IF NOT EXISTS contacts(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT NOT NULL,
		service TEXT,
		message TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'new',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err = db.Exec(createTable)
	if err != nil {
		log.Fatal(err)
	}

	return db
}

func saveContact(db *sql.DB, data ContactRequest) error {
	query := `
	INSERT INTO contacts (name, email, service, message)
	VALUES(?, ?, ?, ?)
	`
	_, err := db.Exec(
		query,
		data.Name,
		data.Email,
		data.Service,
		data.Message,
	)

	if err != nil {
		return err
	}

	return nil
}

func sendContactEmail(data ContactRequest) error {
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("RESEND_API_KEY is not set")
	}

	text := fmt.Sprintf(
		"Name: %s\nEmail: %s\nService: %s\nMessage: %s\n",
		data.Name,
		data.Email,
		data.Service,
		data.Message,
	)

	email := EmailRequest{
		From:    "Itqan Web Studio <onboarding@resend.dev>",
		To:      []string{"itqanwebstudio@gmail.com"},
		Subject: "New Itqan Contact Request",
		Text:    text,
	}

	body, err := json.Marshal(email)
	if err != nil {
		return fmt.Errorf("failed to encode email request: w")
	}

	url := "https://api.resend.com/emails"
	buf := bytes.NewBuffer(body)
	req, err := http.NewRequest(http.MethodPost, url, buf)
	if err != nil {
		return fmt.Errorf("failed to request API: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("resend returned status %d", resp.StatusCode)
	}
	return nil
}

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)

	}
	db := initDB()
	defer db.Close()

	http.HandleFunc("/api/health", healthHandler)
	http.HandleFunc("/api/contact", corsMiddleware(RateLimitMiddleware(contactHandler(db))))

	slog.Info("server starting", "address", ":8080")

	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}

}
