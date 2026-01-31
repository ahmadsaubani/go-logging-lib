package main

import (
	"context"
	"errors"

	"github.com/ahmadsaubani/go-logging-lib"
)

func main() {
	// Initialize logger with configuration
	config := &logging.Config{
		ServiceName:  "example-app",
		LogPath:      "./logs/example",
		EnableStdout: true,
		EnableFile:   true,
		EnableLoki:   true,
	}

	logger, err := logging.New(config)
	if err != nil {
		panic(err)
	}

	// Create context with metadata
	ctx := logging.WithMeta(context.Background(), logging.Meta{
		RequestID: "req-123",
		IP:        "127.0.0.1",
		Method:    "GET",
		Path:      "/api/test",
		UserAgent: "Example/1.0",
	})

	// Log different types of messages
	logger.Info("Application started successfully")

	// Log an error
	testErr := errors.New("something went wrong")
	logger.Error(ctx, testErr)

	// Log with Loki format
	logger.ErrorLoki(ctx, logging.LevelError, testErr)

	logger.Info("Application finished")
}