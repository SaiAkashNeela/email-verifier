package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"email-validator/internal/model"
	"email-validator/internal/service"
	"email-validator/pkg/monitoring"
	"email-validator/pkg/validator"
)

// DisposableCheckHandler handles requests to check an email for disposability after initial validation.
type DisposableCheckHandler struct {
	emailService        service.EmailService
	disposableBlocklist *validator.DisposableBlocklist
}

// NewDisposableCheckHandler creates a new DisposableCheckHandler.
func NewDisposableCheckHandler(es service.EmailService, dbl *validator.DisposableBlocklist) *DisposableCheckHandler {
	return &DisposableCheckHandler{
		emailService:        es,
		disposableBlocklist: dbl,
	}
}

// ServeHTTP handles the HTTP requests for disposable email checking.
func (h *DisposableCheckHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	endpoint := "/api/check-disposable" // New endpoint path
	status := http.StatusOK             // Default status

	defer func() {
		monitoring.RecordRequestMetrics(endpoint, status, time.Since(start))
	}()

	var email string
	if r.Method == http.MethodGet {
		email = r.URL.Query().Get("email")
	} else if r.Method == http.MethodPost {
		var req model.EmailValidationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			status = http.StatusBadRequest
			http.Error(w, fmt.Sprintf("Invalid request body: %v", err), status)
			return
		}
		email = req.Email
	} else {
		status = http.StatusMethodNotAllowed
		http.Error(w, "Method not allowed", status)
		return
	}

	if email == "" {
		status = http.StatusBadRequest
		http.Error(w, "Email parameter is required", status)
		return
	}

	// First, perform the standard email validation using the existing service
	validationResult, err := h.emailService.ValidateEmail(r.Context(), email)
	if err != nil {
		log.Printf("Error validating email %s: %v", email, err)
		status = http.StatusInternalServerError
		http.Error(w, "Internal server error during initial validation", status)
		return
	}

	// If the initial validation is VALID, perform the disposable check
	if validationResult.Status == model.ValidationStatusValid {
		domain := extractDomain(email)
		if domain != "" && h.disposableBlocklist.IsDisposable(domain) {
			validationResult.Validations.IsDisposable = true
			validationResult.Status = model.ValidationStatusDisposable
			// You might want to adjust the score here as well, depending on your scoring logic.
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(validationResult); err != nil {
		log.Printf("Error encoding response for email %s: %v", email, err)
		// Note: If an error occurs here, the deferred metric recording might not capture the correct status.
		// For robust error handling, consider a custom http.ResponseWriter wrapper.
		http.Error(w, "Internal server error encoding response", http.StatusInternalServerError)
	}
}

// extractDomain extracts the domain from an email address.
func extractDomain(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "" // Invalid email format
	}
	return parts[1]
}