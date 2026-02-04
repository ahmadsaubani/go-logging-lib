package main

import (
	"errors"
	"net/http"

	"github.com/ahmadsaubani/go-logging-lib"
	"github.com/ahmadsaubani/go-logging-lib/middleware"
	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize logger
	config := &logging.Config{
		ServiceName:    "gin-example",
		LogPath:        "./logs",
		FilePrefix:     "gin-app",
		EnableStdout:   true,
		EnableFile:     true,
		EnableLoki:     true,
		EnableRotation: false,
	}

	logger, err := logging.New(config)
	if err != nil {
		panic(err)
	}

	// Setup Gin
	r := gin.New()

	// Add logging middleware (order matters!)
	r.Use(middleware.GinMiddleware(logger))      // 1. Setup request context
	r.Use(middleware.GinLogger(logger))          // 2. Log requests (with consistent Loki format)
	r.Use(middleware.GinHTTPErrorLogger(logger)) // 3. Log errors to error.log
	r.Use(middleware.GinRecovery(logger))        // 4. Catch panics last (so logger runs after)

	// Routes
	// Success endpoint - Loki JSON will have errors=null
	r.GET("/", func(c *gin.Context) {
		logger.Info("Home endpoint accessed")
		c.JSON(http.StatusOK, gin.H{"message": "Hello World"})
	})

	// Error endpoint - Loki JSON will have errors object with details
	r.GET("/error", func(c *gin.Context) {
		// Log an error with detailed format and store for Loki
		testErr := errors.New("test error for demonstration")
		logger.LogErrorWithMark(c, testErr)

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong"})
	})

	// Panic endpoint - Loki JSON will have errors from panic
	r.GET("/panic", func(c *gin.Context) {
		// This will trigger the recovery middleware
		panic("test panic")
	})

	logger.Info("Starting server on :8081")
	logger.Info("Try: curl http://localhost:8081/")
	logger.Info("Try: curl http://localhost:8081/error")
	logger.Info("Try: curl http://localhost:8081/panic")
	r.Run(":8081")
}
