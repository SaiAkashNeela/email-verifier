// Package api implements the HTTP handlers and routing for the email validator service.
// It provides endpoints for email validation, batch processing, and service status.
package api

import (
	"encoding/json"
	"net/http"

	"emailvalidator/internal/model"
	"emailvalidator/internal/service"
)

// Handler handles all HTTP requests
type Handler struct {
	emailService *service.EmailService
}

// NewHandler creates a new instance of Handler
func NewHandler(emailService *service.EmailService) *Handler {
	return &Handler{
		emailService: emailService,
	}
}

// RegisterRoutes registers all API routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/validate", h.HandleValidate)
	mux.HandleFunc("/validate/batch", h.HandleBatchValidate)
	mux.HandleFunc("/typo-suggestions", h.HandleTypoSuggestions)
	mux.HandleFunc("/status", h.HandleStatus)
	mux.HandleFunc("/rapidapi-health", h.HandleRapidAPIHealth)
}

// sendError sends a JSON error response
func sendError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		// If we can't send the error response, log it and write a plain text response
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// HandleValidate handles email validation requests
func (h *Handler) HandleValidate(w http.ResponseWriter, r *http.Request) {
	var req model.EmailValidationRequest

	switch r.Method {
	case http.MethodGet:
		email := r.URL.Query().Get("email")
		if email == "" {
			sendError(w, http.StatusBadRequest, "Email parameter is required")
			return
		}
		req.Email = email
	case http.MethodPost:
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
	default:
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	result := h.emailService.ValidateEmail(req.Email)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to encode response")
	}
}

// HandleBatchValidate handles batch email validation requests
func (h *Handler) HandleBatchValidate(w http.ResponseWriter, r *http.Request) {
	var req model.BatchValidationRequest

	switch r.Method {
	case http.MethodGet:
		emails := r.URL.Query()["email"]
		if len(emails) == 0 {
			sendError(w, http.StatusBadRequest, "At least one email parameter is required")
			return
		}
		req.Emails = emails
	case http.MethodPost:
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
	default:
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	result := h.emailService.ValidateEmails(req.Emails)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to encode response")
	}
}

// HandleTypoSuggestions handles email typo suggestion requests
func (h *Handler) HandleTypoSuggestions(w http.ResponseWriter, r *http.Request) {
	var req model.TypoSuggestionRequest

	switch r.Method {
	case http.MethodGet:
		email := r.URL.Query().Get("email")
		if email == "" {
			sendError(w, http.StatusBadRequest, "Email parameter is required")
			return
		}
		req.Email = email
	case http.MethodPost:
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
	default:
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	result := h.emailService.GetTypoSuggestions(req.Email)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to encode response")
	}
}

// HandleStatus handles API status requests
func (h *Handler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	status := h.emailService.GetAPIStatus()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to encode response")
	}
}

// HandleRapidAPIHealth handles RapidAPI health check requests
func (h *Handler) HandleRapidAPIHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	health := model.RapidAPIHealth{
		Status: "OK",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(health); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to encode response")
	}
}
