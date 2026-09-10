package main

import (
	"encoding/json"
	"log"
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

func contactHandler(w http.ResponseWriter, r *http.Request) {

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
	}
	if !isValidEmail(data.Email) {
		response := map[string]string{
			"error": "invalid email address",
		}
		writeJSON(w, http.StatusBadRequest, response)
		return
	}

	response := map[string]string{
		"message": "contact request received",
	}
	writeJSON(w, http.StatusOK, response)
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

func main() {
	http.HandleFunc("/api/health", healthHandler)
	http.HandleFunc("/api/contact", contactHandler)

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}

}
