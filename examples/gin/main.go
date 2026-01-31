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
		ServiceName:  "gin-example",
		LogPath:      "./logs/gin-app",
		EnableStdout: true,
		EnableFile:   true,
		EnableLoki:   true,
	}

	logger, err := logging.New(config)
	if err != nil {
		panic(err)
	}

	// Setup Gin
	r := gin.New()

	// Add logging middleware
	r.Use(middleware.GinMiddleware(logger))
	r.Use(middleware.GinLogger(logger))
	r.Use(middleware.GinHTTPErrorLogger(logger))
	r.Use(middleware.GinRecovery(logger))

	// Routes
	r.GET("/", func(c *gin.Context) {
		logger.Info("Home endpoint accessed")
		c.JSON(http.StatusOK, gin.H{"message": "Hello World"})
	})

	r.GET("/error", func(c *gin.Context) {
		// Log an error with detailed format and mark as logged
		testErr := errors.New("test error for demonstration")
		logger.LogErrorWithMark(c, testErr)
		
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong"})
	})

	r.GET("/panic", func(c *gin.Context) {
		// This will trigger the recovery middleware
		panic("test panic")
	})

	r.GET("/loki", func(c *gin.Context) {
		// Log with Loki format
		testErr := errors.New("loki formatted error")
		logger.ErrorLoki(c.Request.Context(), logging.LevelError, testErr)
		c.JSON(http.StatusOK, gin.H{"message": "Error logged to Loki format"})
	})

	logger.Info("Starting server on :8081")
	r.Run(":8081")
}
