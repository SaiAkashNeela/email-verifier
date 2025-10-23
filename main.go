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
	// 1. Configuration parsing
	port := flag.String("port", os.Getenv("PORT"), "Port to listen on")
	redisURL := flag.String("redis-url", os.Getenv("REDIS_URL"), "Redis connection URL")
	prometheusEnabled := flag.Bool("prometheus-enabled", os.Getenv("PROMETHEUS_ENABLED") == "true", "Enable Prometheus metrics")
	flag.Parse()

	if *port == "" {
		*port = "8080"
	}

	// 2. Initialize Redis client (if Redis URL is provided)
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
		log.Println("Connected to Redis.")
	}

	// 3. Initialize Caching
	domainCache := cache.NewRedisDomainCache(redisClient, 10*time.Minute) // Assuming 10 min cache duration

	// 4. Initialize Validators
	dnsResolver := validator.NewDNSResolver()
	domainValidator := validator.NewDomainValidator(dnsResolver, domainCache)
	disposableValidator := validator.NewDisposableValidator()
	roleValidator := validator.NewRoleValidator()
	aliasDetector := validator.NewAliasDetector()
	syntaxValidator := validator.NewSyntaxValidator()

	// NEW: Initialize the disposable blocklist and load it
	disposableBlocklist := validator.NewDisposableBlocklist()
	if err := disposableBlocklist.Load(); err != nil {
		log.Fatalf("Failed to load disposable blocklist: %v", err)
	}

	// 5. Initialize Services
	emailService := service.NewEmailService(
		syntaxValidator,
		domainValidator,
		disposableValidator,
		roleValidator,
		aliasDetector,
	)
	batchValidationService := service.NewBatchValidationService(emailService)

	// 6. Setup HTTP server
	mux := http.NewServeMux()

	// Register existing handlers
	mux.Handle("/api/validate", monitoring.MetricsMiddleware(http.HandlerFunc(api.HandleValidateEmail(emailService))))
	mux.Handle("/api/validate/batch", monitoring.MetricsMiddleware(http.HandlerFunc(api.HandleBatchValidateEmails(batchValidationService))))
	mux.Handle("/api/typo-suggestions", monitoring.MetricsMiddleware(http.HandlerFunc(api.HandleTypoSuggestions(emailService))))
	mux.Handle("/api/status", monitoring.MetricsMiddleware(http.HandlerFunc(api.HandleStatus())))

	// NEW: Create and register the new disposable check handler
	disposableCheckHandler := api.NewDisposableCheckHandler(emailService, disposableBlocklist)
	mux.Handle("/api/check-disposable", monitoring.MetricsMiddleware(disposableCheckHandler))

	// Serve static files
	mux.Handle("/", http.FileServer(http.Dir("./static")))

	// Prometheus metrics endpoint
	if *prometheusEnabled {
		mux.Handle("/metrics", promhttp.Handler())
		log.Println("Prometheus metrics enabled on /metrics")
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", *port),
		Handler: mux,
	}

	// 7. Start server in a goroutine
	go func() {
		log.Printf("Server listening on :%s", *port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not listen on %s: %v\n", *port, err)
		}
	}()

	// 8. Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}
	log.Println("Server gracefully stopped.")
}