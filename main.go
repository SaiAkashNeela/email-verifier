package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"email-validator/internal/api"
	"email-validator/internal/service"
	"email-validator/pkg/cache"
	"email-validator/pkg/monitoring"
	"email-validator/pkg/validator"

	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	port := flag.String("port", os.Getenv("PORT"), "Port to run the server on")
	redisURL := flag.String("redis-url", os.Getenv("REDIS_URL"), "Redis connection URL (e.g., redis://localhost:6379)")
	prometheusEnabled := flag.Bool("prometheus-enabled", os.Getenv("PROMETHEUS_ENABLED") == "true", "Enable Prometheus metrics")
	flag.Parse()

	if *port == "" {
		*port = "8080" // Default port
	}

	// Initialize Redis client
	var redisClient *redis.Client
	if *redisURL != "" {
		opt, err := redis.ParseURL(*redisURL)
		if err != nil {
			log.Fatalf("Failed to parse Redis URL: %v", err)
		}
		redisClient = redis.NewClient(opt)
		_, err = redisClient.Ping(context.Background()).Result()
		if err != nil {
			log.Fatalf("Failed to connect to Redis: %v", err)
		}
		log.Println("Connected to Redis successfully.")
	} else {
		log.Println("Redis URL not provided, running without Redis cache.")
	}

	// Initialize cache
	domainCache := cache.NewRedisDomainCache(redisClient, 10*time.Minute) // 10-minute cache for domains

	// Initialize validators
	dnsResolver := validator.NewDNSResolver()
	domainValidator := validator.NewDomainValidator(dnsResolver, domainCache)
	disposableValidator := validator.NewDisposableValidator("config/disposable_domains.txt") // Assuming this path
	roleValidator := validator.NewRoleValidator("config/email_providers.csv")                 // Assuming this path
	aliasDetector := validator.NewAliasDetector()
	syntaxValidator := validator.NewSyntaxValidator()

	// Initialize services
	domainValidationService := service.NewDomainValidationService(domainValidator, disposableValidator, roleValidator)
	emailService := service.NewEmailService(syntaxValidator, domainValidationService, aliasDetector)
	batchValidationService := service.NewBatchValidationService(emailService)

	// Initialize the new disposable blocklist and load it
	disposableBlocklist := validator.NewDisposableBlocklist()
	if err := disposableBlocklist.Load(); err != nil {
		log.Fatalf("Failed to load disposable blocklist: %v", err)
	}

	// Setup HTTP handlers
	mux := http.NewServeMux()

	// Existing handlers
	mux.Handle("/api/validate", monitoring.MetricsMiddleware("/api/validate", api.NewValidationHandler(emailService)))
	mux.Handle("/api/validate/batch", monitoring.MetricsMiddleware("/api/validate/batch", api.NewBatchValidationHandler(batchValidationService)))
	mux.Handle("/api/typo-suggestions", monitoring.MetricsMiddleware("/api/typo-suggestions", api.NewTypoSuggestionHandler(emailService)))
	mux.Handle("/api/status", monitoring.MetricsMiddleware("/api/status", api.NewStatusHandler()))
	mux.Handle("/api/disposable-check", monitoring.MetricsMiddleware("/api/disposable-check", api.NewDisposableCheckHandler(emailService, disposableBlocklist))) // New handler

	// Prometheus metrics endpoint
	if *prometheusEnabled {
		mux.Handle("/metrics", promhttp.Handler())
		log.Println("Prometheus metrics enabled on /metrics")
	}

	// Serve static files
	mux.Handle("/", http.FileServer(http.Dir("./static")))

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", *port),
		Handler: mux,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not listen on %s: %v\n", *port, err)
		}
	}()
	log.Printf("Server started on port %s", *port)

	<-done
	log.Println("Server stopped")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %+v", err)
	}
	log.Println("Server exited properly")
}