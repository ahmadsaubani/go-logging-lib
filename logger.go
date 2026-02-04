package logging

import (
	"context"
	"fmt"
	"io"
	"log"
	"path"
	"runtime"
	"time"

	"github.com/ahmadsaubani/go-logging-lib/alerts"
	"github.com/ahmadsaubani/go-logging-lib/alerts/discord"
	"github.com/ahmadsaubani/go-logging-lib/alerts/email"
	"github.com/ahmadsaubani/go-logging-lib/alerts/slack"
	"github.com/ahmadsaubani/go-logging-lib/alerts/telegram"
)

type LogLevel string

const (
	LevelDebug    LogLevel = "DEBUG"
	LevelInfo     LogLevel = "INFO"
	LevelWarn     LogLevel = "WARN"
	LevelError    LogLevel = "ERROR"
	LevelCritical LogLevel = "CRITICAL"
)

type Logger struct {
	accessLogger *log.Logger
	errorLogger  *log.Logger
	lokiWriter   io.Writer
	config       *Config
	alertManager *alerts.Manager
}

type Config struct {
	ServiceName    string        `yaml:"service_name"`
	LogPath        string        `yaml:"log_path"`
	FilePrefix     string        `yaml:"file_prefix"`
	EnableStdout   bool          `yaml:"enable_stdout"`
	EnableFile     bool          `yaml:"enable_file"`
	EnableLoki     bool          `yaml:"enable_loki"`
	EnableRotation bool          `yaml:"enable_rotation"`
	Alerts         *AlertsConfig `yaml:"alerts,omitempty"`
}

type AlertsConfig struct {
	Enabled      bool             `yaml:"enabled"`
	MinLevel     string           `yaml:"min_level"`
	RateLimitSec int              `yaml:"rate_limit_sec"`
	Discord      *discord.Config  `yaml:"discord,omitempty"`
	Slack        *slack.Config    `yaml:"slack,omitempty"`
	Telegram     *telegram.Config `yaml:"telegram,omitempty"`
	Email        *email.Config    `yaml:"email,omitempty"`
}

/**
 * New creates a new logger instance with the provided configuration.
 * Initializes access/error loggers, Loki writer, and alert manager.
 *
 * @param config Logger configuration (uses defaults if nil)
 * @return *Logger Ready-to-use logger instance
 * @return error Error if writer setup fails
 */
func New(config *Config) (*Logger, error) {
	if config == nil {
		config = &Config{
			ServiceName:    "app",
			LogPath:        "./logs",
			FilePrefix:     "app",
			EnableStdout:   true,
			EnableFile:     true,
			EnableLoki:     false,
			EnableRotation: true,
		}
	}

	logger := &Logger{
		config:       config,
		alertManager: setupAlertManager(config.Alerts),
	}

	if err := logger.setupWriters(); err != nil {
		return nil, err
	}

	return logger, nil
}

func setupAlertManager(cfg *AlertsConfig) *alerts.Manager {
	if cfg == nil || !cfg.Enabled {
		return nil
	}

	manager := alerts.NewManager(&alerts.Config{
		Enabled:      cfg.Enabled,
		MinLevel:     alerts.LogLevel(cfg.MinLevel),
		RateLimitSec: cfg.RateLimitSec,
	})

	if cfg.Discord != nil && cfg.Discord.Enabled {
		manager.Register(discord.New(cfg.Discord))
	}

	if cfg.Slack != nil && cfg.Slack.Enabled {
		manager.Register(slack.New(cfg.Slack))
	}

	if cfg.Telegram != nil && cfg.Telegram.Enabled {
		manager.Register(telegram.New(cfg.Telegram))
	}

	if cfg.Email != nil && cfg.Email.Enabled {
		manager.Register(email.New(cfg.Email))
	}

	return manager
}

func (l *Logger) setupWriters() error {
	var accessWriters []io.Writer
	var errorWriters []io.Writer
	var lokiWriters []io.Writer

	filePrefix := l.config.FilePrefix
	if filePrefix == "" {
		filePrefix = "app"
	}

	basePath := l.config.LogPath + "/" + filePrefix

	if l.config.EnableStdout {
		accessWriters = append(accessWriters, log.Writer())
		errorWriters = append(errorWriters, log.Writer())
		lokiWriters = append(lokiWriters, log.Writer())
	}

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

	l.accessLogger = log.New(io.MultiWriter(accessWriters...), "", log.LstdFlags|log.Lshortfile)
	l.errorLogger = log.New(io.MultiWriter(errorWriters...), "", log.LstdFlags|log.Lshortfile)
	l.lokiWriter = io.MultiWriter(lokiWriters...)

	return nil
}

func (l *Logger) GetAccessLogger() *log.Logger {
	return l.accessLogger
}

func (l *Logger) GetErrorLogger() *log.Logger {
	return l.errorLogger
}

func (l *Logger) GetLokiWriter() io.Writer {
	return l.lokiWriter
}

func (l *Logger) GetServiceName() string {
	return l.config.ServiceName
}

func (l *Logger) Info(msg string) {
	l.accessLogger.Printf("[INFO] %s", msg)
}

func (l *Logger) Access(msg string) {
	l.accessLogger.Printf("%s", msg)
}

/**
 * LogRequest logs an HTTP request to access log and Loki for non-Gin usage.
 * Does not include error information.
 *
 * @param ctx Context containing request metadata
 * @param statusCode HTTP response status code
 * @param latency Request processing duration
 */
func (l *Logger) LogRequest(ctx context.Context, statusCode int, latency time.Duration) {
	l.LogRequestWithError(ctx, statusCode, latency, nil)
}

/**
 * LogRequestWithError logs an HTTP request with optional error for non-Gin usage.
 * Automatically determines log level based on status code and triggers alerts.
 *
 * @param ctx Context containing request metadata
 * @param statusCode HTTP response status code
 * @param latency Request processing duration
 * @param err Optional error to include in log
 */
func (l *Logger) LogRequestWithError(ctx context.Context, statusCode int, latency time.Duration, err error) {
	meta, ok := FromContext(ctx)
	if !ok {
		return
	}

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

	level := LevelInfo
	if statusCode >= 500 {
		level = LevelCritical
	} else if statusCode >= 400 {
		level = LevelError
	} else if statusCode >= 300 {
		level = LevelWarn
	}

	LogLoki(ctx, l.config.ServiceName, string(level), statusCode, latency, err, l.lokiWriter)

	if err != nil {
		l.sendAlert(ctx, string(level), err)
	}
}

func (l *Logger) Error(ctx context.Context, err error) {
	LogError(ctx, err, l.errorLogger)
}

/**
 * ErrorLoki logs an error in Loki format and triggers alert notification.
 *
 * @param ctx Context containing request metadata
 * @param level Log severity level (ERROR, CRITICAL)
 * @param err Error to log
 */
func (l *Logger) ErrorLoki(ctx context.Context, level LogLevel, err error) {
	LogErrorLoki(ctx, l.config.ServiceName, string(level), err, l.lokiWriter)

	l.sendAlert(ctx, string(level), err)
}

func (l *Logger) AccessLoki(ctx context.Context, level LogLevel, statusCode int, latency time.Duration) {
	LogAccessLoki(ctx, l.config.ServiceName, string(level), statusCode, latency, l.lokiWriter)
}

/**
 * Loki logs in unified JSON format suitable for Loki/Grafana integration.
 * Includes both access and error information, triggers alerts when error exists.
 *
 * @param ctx Context containing request metadata
 * @param level Log severity level
 * @param statusCode HTTP response status code
 * @param latency Request processing duration
 * @param err Optional error to include
 */
func (l *Logger) Loki(ctx context.Context, level LogLevel, statusCode int, latency time.Duration, err error) {
	LogLoki(ctx, l.config.ServiceName, string(level), statusCode, latency, err, l.lokiWriter)

	if err != nil {
		l.sendAlert(ctx, string(level), err)
	}
}

func (l *Logger) sendAlert(ctx context.Context, level string, err error) {
	if l.alertManager == nil || err == nil {
		return
	}

	meta, _ := FromContext(ctx)

	_, file, line, _ := runtime.Caller(2)

	payload := alerts.Payload{
		ServiceName: l.config.ServiceName,
		Level:       level,
		Error:       err.Error(),
		RequestID:   meta.RequestID,
		Method:      meta.Method,
		Path:        meta.Path,
		IP:          meta.IP,
		UserAgent:   meta.UserAgent,
		File:        path.Base(file),
		Line:        line,
		Stack:       getStackFrames(3, 6),
		Timestamp:   time.Now(),
	}

	l.alertManager.Alert(payload)
}

func getStackFrames(skip, max int) []string {
	var frames []string

	for i := skip; i < skip+max; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		fn := runtime.FuncForPC(pc)
		name := "unknown"
		if fn != nil {
			name = path.Base(fn.Name())
		}

		frames = append(
			frames,
			fmt.Sprintf("%s:%d %s", path.Base(file), line, name),
		)
	}

	return frames
}