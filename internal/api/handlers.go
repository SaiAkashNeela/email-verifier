package api

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"

	"emailvalidator/pkg/validator"
)

// Define a simple email regex for basic validation
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// ExternalDisposableCheckRequest represents the request body for checking external disposable emails.
type ExternalDisposableCheckRequest struct {
	Email string `json:"email"`
}

// ExternalDisposableCheckResponse represents the response body for checking external disposable emails.
type ExternalDisposableCheckResponse struct {
	Email       string `json:"email"`
	IsDisposable bool   `json:"is_disposable"`
	Error       string `json:"error,omitempty"`
}

// NewCheckExternalDisposableHandler creates an http.HandlerFunc for checking external disposable emails.
func NewCheckExternalDisposableHandler(checker *validator.ExternalDisposableChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req ExternalDisposableCheckRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("Error decoding request body for external disposable check: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Email == "" {
			http.Error(w, "Email is required", http.StatusBadRequest)
			return
		}

		// Basic email format validation before extracting domain
		if !emailRegex.MatchString(req.Email) {
			http.Error(w, "Invalid email format", http.StatusBadRequest)
			return
		}

		parts := strings.Split(req.Email, "@")
		if len(parts) != 2 {
			http.Error(w, "Invalid email format", http.StatusBadRequest)
			return
		}
		domain := parts[1]

		isDisposable := checker.IsDisposable(domain)

		resp := ExternalDisposableCheckResponse{
			Email:       req.Email,
			IsDisposable: isDisposable,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("Error encoding response for external disposable check: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}
}

// Placeholder for other existing handlers if any.
// You should merge this content with your actual internal/api/handlers.go file.
// For example, if you have a Handlers struct, you might add the checker there
// and make CheckExternalDisposableHandler a method of that struct.