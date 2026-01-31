package logging

import (
	"context"
	"io"
	"log"
)

// LogLevel represents the severity of a log entry
type LogLevel string

const (
	LevelDebug    LogLevel = "DEBUG"
	LevelInfo     LogLevel = "INFO"
	LevelWarn     LogLevel = "WARN"
	LevelError    LogLevel = "ERROR"
	LevelCritical LogLevel = "CRITICAL"
)

// Logger represents the main logging interface
type Logger struct {
	accessLogger *log.Logger
	errorLogger  *log.Logger
	lokiWriter   io.Writer
	config       *Config
}

// Config holds logger configuration
type Config struct {
	ServiceName string
	LogPath     string
	EnableStdout bool
	EnableFile   bool
	EnableLoki   bool
}

// New creates a new logger instance
func New(config *Config) (*Logger, error) {
	if config == nil {
		config = &Config{
			ServiceName:  "app",
			LogPath:      "./logs/app",
			EnableStdout: true,
			EnableFile:   true,
			EnableLoki:   false,
		}
	}

	logger := &Logger{
		config: config,
	}

	if err := logger.setupWriters(); err != nil {
		return nil, err
	}

	return logger, nil
}

// setupWriters configures the output writers based on config
func (l *Logger) setupWriters() error {
	var accessWriters []io.Writer
	var errorWriters []io.Writer
	var lokiWriters []io.Writer

	// Stdout writers
	if l.config.EnableStdout {
		accessWriters = append(accessWriters, log.Writer())
		errorWriters = append(errorWriters, log.Writer())
		lokiWriters = append(lokiWriters, log.Writer())
	}

	// File writers
	if l.config.EnableFile {
		accessWriter, err := NewDailyWriter(l.config.LogPath + ".access")
		if err != nil {
			return err
		}
		
		errorWriter, err := NewDailyWriter(l.config.LogPath + ".error")
		if err != nil {
			return err
		}

		errorLokiWriter, err := NewDailyWriter(l.config.LogPath + ".error-loki")
		if err != nil {
			return err
		}

		accessWriters = append(accessWriters, accessWriter)
		errorWriters = append(errorWriters, errorWriter)
		lokiWriters = append(lokiWriters, errorLokiWriter)
	}

	// Setup multi-writers
	l.accessLogger = log.New(io.MultiWriter(accessWriters...), "", log.LstdFlags|log.Lshortfile)
	l.errorLogger = log.New(io.MultiWriter(errorWriters...), "", log.LstdFlags|log.Lshortfile)
	l.lokiWriter = io.MultiWriter(lokiWriters...)

	return nil
}

// GetAccessLogger returns the access logger
func (l *Logger) GetAccessLogger() *log.Logger {
	return l.accessLogger
}

// GetErrorLogger returns the error logger
func (l *Logger) GetErrorLogger() *log.Logger {
	return l.errorLogger
}

// GetLokiWriter returns the Loki writer
func (l *Logger) GetLokiWriter() io.Writer {
	return l.lokiWriter
}

// GetServiceName returns the configured service name
func (l *Logger) GetServiceName() string {
	return l.config.ServiceName
}

// Info logs an info message
func (l *Logger) Info(msg string) {
	l.accessLogger.Printf("[INFO] %s", msg)
}

// Error logs an error with context and marks it as manually logged
func (l *Logger) Error(ctx context.Context, err error) {
	LogError(ctx, err, l.errorLogger)
}

// ErrorLoki logs an error in Loki format
func (l *Logger) ErrorLoki(ctx context.Context, level LogLevel, err error) {
	LogErrorLoki(ctx, l.config.ServiceName, string(level), err, l.lokiWriter)
}