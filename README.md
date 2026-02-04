# Go Logging Library

A production-ready Go logging library with daily rotation, multi-format output, unified Loki/Grafana JSON format, alert notifications, and support for both Gin framework and standard `net/http`.

## Features

- **Daily Log Rotation**: Automatic dated log files with thread-safe operations
- **Multi-format Output**: Console, file, and JSON/Loki formats simultaneously
- **Unified Loki Format**: Consistent JSON structure for Grafana visualization
- **Alert Notifications**: Send errors to Discord, Slack, Telegram, and Email
- **Gin Framework Integration**: Complete middleware suite with anti-duplication
- **Standard HTTP Support**: Middleware for `net/http` applications
- **Context-aware Logging**: Request metadata injection for structured logging
- **Advanced Error Handling**: Stack traces, source file info, error categorization
- **Rate Limiting**: Prevent alert spam with configurable rate limits
- **Thread-safe**: Built for concurrent high-performance applications

## Installation

```bash
go get github.com/ahmadsaubani/go-logging-lib
```

## Quick Start

### Gin Framework

```go
package main

import (
    "github.com/gin-gonic/gin"
    logging "github.com/ahmadsaubani/go-logging-lib"
    "github.com/ahmadsaubani/go-logging-lib/middleware"
)

func main() {
    logger, _ := logging.New(&logging.Config{
        ServiceName:    "my-api",
        LogPath:        "./logs",
        FilePrefix:     "app",
        EnableStdout:   true,
        EnableFile:     true,
        EnableLoki:     true,
        EnableRotation: true,
    })

    r := gin.New()
    
    r.Use(middleware.GinMiddleware(logger))
    r.Use(middleware.GinRecovery(logger))
    r.Use(middleware.GinLogger(logger))

    r.GET("/ping", func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "pong"})
    })

    r.Run(":8080")
}
```

### Standard HTTP (net/http)

```go
package main

import (
    "net/http"
    logging "github.com/ahmadsaubani/go-logging-lib"
    "github.com/ahmadsaubani/go-logging-lib/middleware"
)

func main() {
    logger, _ := logging.New(&logging.Config{
        ServiceName:    "my-api",
        LogPath:        "./logs",
        FilePrefix:     "app",
        EnableStdout:   true,
        EnableFile:     true,
        EnableLoki:     true,
        EnableRotation: true,
    })

    mux := http.NewServeMux()
    mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("pong"))
    })

    handler := middleware.HTTPMiddleware(logger)(
        middleware.HTTPRecovery(logger)(
            middleware.HTTPLogger(logger)(mux)))

    http.ListenAndServe(":8080", handler)
}
```

## Configuration

```go
type Config struct {
    ServiceName    string        // Service identifier in logs
    LogPath        string        // Directory for log files
    FilePrefix     string        // Prefix for log filenames (default: "app")
    EnableStdout   bool          // Output to console
    EnableFile     bool          // Output to log files
    EnableLoki     bool          // Output JSON format for Loki/Grafana
    EnableRotation bool          // Enable daily log rotation
    Alerts         *AlertsConfig // Alert notifications config
}
```

### Log Files Generated

With `EnableRotation: true`:
```
logs/
├── app.access-2026-02-04.log
├── app.error-2026-02-04.log
└── app.loki-2026-02-04.log
```

With `EnableRotation: false`:
```
logs/
├── app.access.log
├── app.error.log
└── app.loki.log
```

## Alert Notifications

Send error alerts to multiple platforms when errors occur.

### Supported Platforms

| Platform | Required Fields |
|----------|-----------------|
| Discord | `webhook_url` |
| Slack | `webhook_url` |
| Telegram | `bot_token`, `chat_id` |
| Email | `smtp_host`, `smtp_port`, `from`, `to` |

### YAML Configuration

```yaml
logging:
  service_name: "my-api"
  log_path: "./logs"
  file_prefix: "app"
  enable_stdout: true
  enable_file: true
  enable_loki: true
  enable_rotation: true
  
  alerts:
    enabled: true
    min_level: "ERROR"           # WARN, ERROR, CRITICAL
    rate_limit_sec: 300          # 5 minutes between same error
    
    discord:
      enabled: true
      webhook_url: "https://discord.com/api/webhooks/xxx/yyy"
      username: "Alert Bot"
      avatar_url: "https://example.com/avatar.png"
    
    slack:
      enabled: true
      webhook_url: "https://hooks.slack.com/services/xxx/yyy/zzz"
      channel: "#alerts"
      username: "Alert Bot"
      icon_emoji: ":rotating_light:"
    
    telegram:
      enabled: true
      bot_token: "123456:ABC-DEF..."
      chat_id: "-1001234567890"
    
    email:
      enabled: true
      smtp_host: "smtp.gmail.com"
      smtp_port: 587
      username: "alerts@example.com"
      password: "app-password"
      from: "alerts@example.com"
      to:
        - "dev@example.com"
        - "ops@example.com"
      use_tls: true
      skip_verify: false
```

### Programmatic Configuration

```go
import (
    logging "github.com/ahmadsaubani/go-logging-lib"
    "github.com/ahmadsaubani/go-logging-lib/alerts/discord"
    "github.com/ahmadsaubani/go-logging-lib/alerts/slack"
    "github.com/ahmadsaubani/go-logging-lib/alerts/telegram"
    "github.com/ahmadsaubani/go-logging-lib/alerts/email"
)

config := &logging.Config{
    ServiceName:    "my-api",
    LogPath:        "./logs",
    FilePrefix:     "app",
    EnableStdout:   true,
    EnableFile:     true,
    EnableLoki:     true,
    EnableRotation: true,
    Alerts: &logging.AlertsConfig{
        Enabled:      true,
        MinLevel:     "ERROR",
        RateLimitSec: 300,
        Discord: &discord.Config{
            Enabled:    true,
            WebhookURL: "https://discord.com/api/webhooks/xxx/yyy",
            Username:   "Alert Bot",
        },
        Slack: &slack.Config{
            Enabled:    true,
            WebhookURL: "https://hooks.slack.com/services/xxx",
            Channel:    "#alerts",
        },
        Telegram: &telegram.Config{
            Enabled:  true,
            BotToken: "123456:ABC-DEF...",
            ChatID:   "-1001234567890",
        },
        Email: &email.Config{
            Enabled:  true,
            SMTPHost: "smtp.gmail.com",
            SMTPPort: 587,
            Username: "alerts@example.com",
            Password: "app-password",
            From:     "alerts@example.com",
            To:       []string{"dev@example.com"},
            UseTLS:   true,
        },
    },
}

logger, _ := logging.New(config)
```

### Alert Levels

| Level | Priority | When Triggered |
|-------|----------|----------------|
| WARN | 1 | Status 300-399 with error |
| ERROR | 2 | Status 400-499 with error |
| CRITICAL | 3 | Status 500+ or explicit critical |

Setting `min_level: "ERROR"` will trigger alerts for ERROR and CRITICAL.

### Rate Limiting

Prevents alert spam by limiting duplicate errors:
- Same error key (service + error + path + method) is rate-limited
- Default: 300 seconds (5 minutes) between same alerts
- Each alert platform receives independently

## Unified Loki JSON Format

Consistent JSON structure for all requests:

### Success Response
```json
{
    "ts": "2026-02-04T22:13:29+07:00",
    "level": "INFO",
    "service": "my-api",
    "request_id": "27fd79fe-1e04-47a9-8c56-683269a4c5f0",
    "status_code": 200,
    "latency_ms": 15,
    "http": {
        "ip": "127.0.0.1",
        "method": "GET",
        "path": "/ping",
        "ua": "Mozilla/5.0"
    },
    "errors": null
}
```

### Error Response
```json
{
    "ts": "2026-02-04T22:13:29+07:00",
    "level": "CRITICAL",
    "service": "my-api",
    "request_id": "27fd79fe-1e04-47a9-8c56-683269a4c5f0",
    "status_code": 500,
    "latency_ms": 0,
    "http": {
        "ip": "127.0.0.1",
        "method": "POST",
        "path": "/users",
        "ua": "Mozilla/5.0"
    },
    "errors": {
        "error": "database connection failed",
        "source": {
            "file": "user_handler.go",
            "line": 45
        },
        "stack": [
            "user_handler.go:45 handlers.CreateUser",
            "context.go:192 gin.(*Context).Next"
        ]
    }
}
```

## Project Structure

```
go-logging-lib/
├── logger.go           # Main Logger struct and config
├── context.go          # Context metadata handling
├── error_logging.go    # Error logging and Loki format
├── writers.go          # Daily rotating file writer
├── gin_helpers.go      # Gin-specific helpers
├── utils.go            # Utility functions
├── alerts/
│   ├── types.go        # Alerter interface, Payload, Config
│   ├── manager.go      # Alert manager with rate limiting
│   ├── discord/
│   │   └── alerter.go  # Discord webhook alerter
│   ├── slack/
│   │   └── alerter.go  # Slack webhook alerter
│   ├── telegram/
│   │   └── alerter.go  # Telegram Bot API alerter
│   └── email/
│       ├── alerter.go  # SMTP email alerter
│       └── template.go # HTML email template
├── middleware/
│   ├── gin.go          # Gin middleware
│   └── http.go         # Standard HTTP middleware
└── tests/
    └── complete_test.go
```

## Middleware Reference

### Gin Middleware

| Middleware | Description |
|------------|-------------|
| `GinMiddleware` | Setup request context with metadata |
| `GinLogger` | Log all requests to access log + Loki |
| `GinHTTPErrorLogger` | Log detailed errors for 4xx/5xx |
| `GinRecovery` | Catch panics and log with stack trace |

```go
r.Use(middleware.GinMiddleware(logger))
r.Use(middleware.GinRecovery(logger))
r.Use(middleware.GinLogger(logger))
```

### HTTP Middleware (net/http)

| Middleware | Description |
|------------|-------------|
| `HTTPMiddleware` | Setup request context with metadata |
| `HTTPLogger` | Log all requests to access log + Loki |
| `HTTPRecovery` | Catch panics and log with stack trace |

```go
handler := middleware.HTTPMiddleware(logger)(
    middleware.HTTPRecovery(logger)(
        middleware.HTTPLogger(logger)(mux)))
```

## API Reference

### Logger Methods

```go
logger.Info(msg string)
logger.Access(msg string)
logger.Error(ctx context.Context, err error)
logger.LogRequest(ctx context.Context, statusCode int, latency time.Duration)
logger.LogRequestWithError(ctx context.Context, statusCode int, latency time.Duration, err error)
logger.ErrorLoki(ctx context.Context, level LogLevel, err error)
logger.Loki(ctx context.Context, level LogLevel, statusCode int, latency time.Duration, err error)
logger.LogErrorWithMark(c *gin.Context, err error)  // Gin only
```

### Context Functions

```go
ctx := logging.WithMeta(ctx, logging.Meta{...})
meta, ok := logging.FromContext(ctx)
ctx := logging.WithError(ctx, err)
err, ok := logging.ErrorFromContext(ctx)
ctx := logging.NewRequestContext(r *http.Request)
```

## Grafana/Loki Integration

### Promtail Configuration

```yaml
scrape_configs:
  - job_name: my-api
    static_configs:
      - targets:
          - localhost
        labels:
          job: my-api
          __path__: /var/log/my-api/*.loki*.log
    pipeline_stages:
      - json:
          expressions:
            level: level
            service: service
            status_code: status_code
            method: http.method
            path: http.path
```

### LogQL Queries

```logql
# All errors
{job="my-api"} | json | level="CRITICAL" or level="ERROR"

# Slow requests (>1000ms)
{job="my-api"} | json | latency_ms > 1000

# Requests with errors
{job="my-api"} | json | errors != "null"
```

## Testing

```bash
go test -v ./tests/...
```

## License

MIT License
