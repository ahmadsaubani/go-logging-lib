package main

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/ahmadsaubani/go-logging-lib"
	"github.com/ahmadsaubani/go-logging-lib/middleware"
)

func main() {
	// Initialize logger with configuration
	config := &logging.Config{
		ServiceName:    "example-basic-app",
		LogPath:        "./logs",
		FilePrefix:     "example",
		EnableStdout:   true,
		EnableFile:     true,
		EnableLoki:     true,
		EnableRotation: false,
	}

	logger, err := logging.New(config)
	if err != nil {
		panic(err)
	}

	// Create HTTP server with middleware
	mux := http.NewServeMux()

	// Success endpoint - errors will be null in Loki JSON
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong"))
	})

	// Error endpoint - errors will contain error details in Loki JSON
	mux.HandleFunc("/error", func(w http.ResponseWriter, r *http.Request) {
		// Store error in context for Loki logging
		testErr := errors.New("database connection failed")
		ctx := logging.WithError(r.Context(), testErr)
		r = r.WithContext(ctx)

		// Also log to error.log with detailed format
		logger.Error(r.Context(), testErr)

		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	})

	// Apply middleware chain (order matters: Recovery -> Middleware -> Logger)
	handler := middleware.HTTPRecovery(logger)(
		middleware.HTTPMiddleware(logger)(
			middleware.HTTPLogger(logger)(mux),
		),
	)

	fmt.Println("Server starting on :8080")
	fmt.Println("Try: curl http://localhost:8080/ping")
	fmt.Println("Try: curl http://localhost:8080/error")

	http.ListenAndServe(":8080", handler)
}

// Example of manual logging without HTTP (e.g., for background jobs)
func exampleManualLogging(logger *logging.Logger) {
	// Create context with metadata manually
	meta := logging.Meta{
		RequestID: "job-123",
		IP:        "internal",
		Method:    "CRON",
		Path:      "/jobs/cleanup",
		UserAgent: "CronJob/1.0",
	}
	ctx := logging.WithMeta(nil, meta)

	// Log success (errors=null in Loki JSON)
	logger.LogRequest(ctx, http.StatusOK, 150*time.Millisecond)

	// Log with error (errors object populated in Loki JSON)
	err := errors.New("cleanup failed: disk full")
	logger.LogRequestWithError(ctx, http.StatusInternalServerError, 500*time.Millisecond, err)
}