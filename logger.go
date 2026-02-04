package logging

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"
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
	ServiceName    string `yaml:"service_name"`
	LogPath        string `yaml:"log_path"`
	FilePrefix     string `yaml:"file_prefix"`
	EnableStdout   bool   `yaml:"enable_stdout"`
	EnableFile     bool   `yaml:"enable_file"`
	EnableLoki     bool   `yaml:"enable_loki"`
	EnableRotation bool   `yaml:"enable_rotation"`
}

// New creates a new logger instance
func New(config *Config) (*Logger, error) {
	if config == nil {
		config = &Config{
			ServiceName:    "app",
			LogPath:        "./logs/app",
			EnableStdout:   true,
			EnableFile:     true,
			EnableLoki:     false,
			EnableRotation: true,
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

	// Set default file prefix if not specified
	filePrefix := l.config.FilePrefix
	if filePrefix == "" {
		filePrefix = "app"
	}

	// Build base path: LogPath/FilePrefix
	basePath := l.config.LogPath + "/" + filePrefix

	// Stdout writers
	if l.config.EnableStdout {
		accessWriters = append(accessWriters, log.Writer())
		errorWriters = append(errorWriters, log.Writer())
		lokiWriters = append(lokiWriters, log.Writer())
	}

	// File writers
	if l.config.EnableFile {
		accessWriter, err := NewDailyWriter(basePath+".access", l.config.EnableRotation)
		if err != nil {
			return err
		}

		errorWriter, err := NewDailyWriter(basePath+".error", l.config.EnableRotation)
		if err != nil {
			return err
		}

		errorLokiWriter, err := NewDailyWriter(basePath+".loki", l.config.EnableRotation)
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

// Access logs an access request message
func (l *Logger) Access(msg string) {
	l.accessLogger.Printf("%s", msg)
}

// LogRequest logs an HTTP request to access log and Loki (for non-Gin usage)
func (l *Logger) LogRequest(ctx context.Context, statusCode int, latency time.Duration) {
	l.LogRequestWithError(ctx, statusCode, latency, nil)
}

// LogRequestWithError logs an HTTP request with optional error to access log and Loki (for non-Gin usage)
func (l *Logger) LogRequestWithError(ctx context.Context, statusCode int, latency time.Duration, err error) {
	meta, ok := FromContext(ctx)
	if !ok {
		return
	}

	// Log ke access log
	logLine := fmt.Sprintf(
		"[REQ:%s] %s | %3d | %13v | %15s | %-7s %s",
		meta.RequestID,
		time.Now().Format(time.RFC3339),
		statusCode,
		latency,
		meta.IP,
		meta.Method,
		meta.Path,
	)
	l.accessLogger.Printf("%s", logLine)

	// Determine log level based on status code
	level := LevelInfo
	if statusCode >= 500 {
		level = LevelCritical
	} else if statusCode >= 400 {
		level = LevelError
	} else if statusCode >= 300 {
		level = LevelWarn
	}

	// Log ke Loki dengan format konsisten (errors=null jika tidak ada error)
	LogLoki(ctx, l.config.ServiceName, string(level), statusCode, latency, err, l.lokiWriter)
}

// Error logs an error with context and marks it as manually logged
func (l *Logger) Error(ctx context.Context, err error) {
	LogError(ctx, err, l.errorLogger)
}

// ErrorLoki logs an error in Loki format
func (l *Logger) ErrorLoki(ctx context.Context, level LogLevel, err error) {
	LogErrorLoki(ctx, l.config.ServiceName, string(level), err, l.lokiWriter)
}

// AccessLoki logs access request in Loki format for all status codes
func (l *Logger) AccessLoki(ctx context.Context, level LogLevel, statusCode int, latency time.Duration) {
	LogAccessLoki(ctx, l.config.ServiceName, string(level), statusCode, latency, l.lokiWriter)
}

// Loki logs in unified JSON format suitable for Loki/Grafana integration
func (l *Logger) Loki(ctx context.Context, level LogLevel, statusCode int, latency time.Duration, err error) {
	LogLoki(ctx, l.config.ServiceName, string(level), statusCode, latency, err, l.lokiWriter)
}