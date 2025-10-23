package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"emailvalidator/internal/api"
	"emailvalidator/pkg/validator"
	// Add any other imports your original main.go might have
)

func main() {
	log.Println("Starting Email Validator Service...")

	// --- New: Initialize and load external disposable checker ---
	externalDisposableChecker := validator.NewExternalDisposableChecker()
	if err := externalDisposableChecker.LoadDisposableDomains(); err != nil {
		log.Printf("Error loading external disposable domains: %v. Continuing without external disposable checks.", err)
	} else {
		log.Printf("Successfully loaded external disposable domains. Last updated: %s", externalDisposableChecker.GetLastUpdated().Format(time.RFC3339))
	}

	// Optional: Periodically refresh the disposable domains list
	go func() {
		ticker := time.NewTicker(24 * time.Hour) // Refresh every 24 hours
		defer ticker.Stop()
		for range ticker.C {
			log.Println("Attempting to refresh external disposable domains list...")
			if err := externalDisposableChecker.LoadDisposableDomains(); err != nil {
				log.Printf("Error refreshing external disposable domains: %v", err)
			} else {
				log.Printf("Successfully refreshed external disposable domains. Last updated: %s", externalDisposableChecker.GetLastUpdated().Format(time.RFC3339))
			}
		}
	}()
	// --- End New ---

	// Default port
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// --- New: Register the new handler ---
	http.HandleFunc("/api/check-disposable", api.NewCheckExternalDisposableHandler(externalDisposableChecker))
	// --- End New ---

	// Placeholder for other existing handler registrations if any.
	// Example:
	// http.HandleFunc("/api/validate", someExistingValidationHandler)
	// http.HandleFunc("/api/validate/batch", someExistingBatchHandler)
	// http.HandleFunc("/api/typo-suggestions", someExistingTypoHandler)
	// http.HandleFunc("/api/status", someExistingStatusHandler)

	// Start the server
	log.Printf("Server listening on :%s", port)
	err := http.ListenAndServe(":"+port, nil) // Assuming nil handler for DefaultServeMux
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}