package main

import (
	"database/sql"
	"encoding/json"
	"log"
	_ "modernc.org/sqlite"
	"net/http"
	"strings"
)

type ContactRequest struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Service string `json:"service"`
	Message string `json:"message"`
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
			writeJSON(w, http.StatusBadRequest, response)
			return
		}
		if data.Name == "" || data.Email == "" || data.Message == "" {
			response := map[string]string{
				"error": "name, email, and message are required",
			}
			writeJSON(w, http.StatusBadRequest, response)
			return
		}
		if !isValidEmail(data.Email) {
			response := map[string]string{
				"error": "invalid email address",
			}
			writeJSON(w, http.StatusBadRequest, response)
			return
		}

		err = saveContact(db, data)
		if err != nil {
			response := map[string]string{
				"error": "failed to save contact",
			}
			log.Println(err)
			writeJSON(w, http.StatusInternalServerError, response)

			return
		}

		response := map[string]string{
			"message": "contact request received",
		}

		writeJSON(w, http.StatusOK, response)
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

func main() {

	db := initDB()
	defer db.Close()

	http.HandleFunc("/api/health", healthHandler)
	http.HandleFunc("/api/contact", contactHandler(db))

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}

}
