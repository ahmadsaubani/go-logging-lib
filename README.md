# Go Logging Library

A production-ready Go logging library with optional daily rotation, multi-format output, and native Gin framework integration.

## Features

- **Daily Log Rotation**: Optional automatic dated log files with thread-safe operations
- **Multi-format Output**: Console, file, and JSON/Loki formats simultaneously
- **Native Gin Integration**: Complete middleware suite with anti-duplication mechanisms
- **Context-aware Logging**: Request metadata injection for structured logging
- **Advanced Error Handling**: Stack traces, request context, error categorization
- **Thread-safe**: Built for concurrent high-performance applications

## Installation

```bash
go get github.com/ahmadsaubani/go-logging-lib
```

For Gin projects:
```bash
go get github.com/ahmadsaubani/go-logging-lib
go get github.com/gin-gonic/gin
```

## Configuration

```go
type Config struct {
    ServiceName    string `yaml:"service_name"`    // Service name for logs
    LogPath        string `yaml:"log_path"`        // Base path for log files
    EnableStdout   bool   `yaml:"enable_stdout"`   // Output to console
    EnableFile     bool   `yaml:"enable_file"`     // Output to files
    EnableLoki     bool   `yaml:"enable_loki"`     // Output JSON format for Loki
    EnableRotation bool   `yaml:"enable_rotation"` // Enable daily log rotation
}
```

### Log Files Generated

When `EnableRotation: true`:
```
logs/
  app.access-2026-02-04.log      # Access logs with date
  app.error-2026-02-04.log       # Error logs with date
  app.error-loki-2026-02-04.log  # JSON logs with date
```

When `EnableRotation: false`:
```
logs/
  app.access.log       # Single access log file
  app.error.log        # Single error log file
  app.error-loki.log   # Single JSON log file
```

### YAML Configuration Example

```yaml
service_name: "my-api"
log_path: "./logs/app"
enable_stdout: true
enable_file: true
enable_loki: true
enable_rotation: true
```

## Usage Without Gin (Basic/CLI Application)

```go
package main

import (
    "context"
    "errors"
    "time"

    logging "github.com/ahmadsaubani/go-logging-lib"
)

func main() {
    config := &logging.Config{
        ServiceName:    "my-cli-app",
        LogPath:        "./logs/app",
        EnableStdout:   true,
        EnableFile:     true,
        EnableLoki:     true,
        EnableRotation: true,
    }

    logger, err := logging.New(config)
    if err != nil {
        panic(err)
    }

    // Info logging
    logger.Info("Application started")
    logger.Info("Processing data...")

    // Create context with request metadata
    ctx := logging.WithMeta(context.Background(), logging.Meta{
        RequestID: "req-001",
        IP:        "127.0.0.1",
        Method:    "GET",
        Path:      "/api/users",
        UserAgent: "MyApp/1.0",
    })

    // Log HTTP request (simulated)
    logger.LogRequest(ctx, 200, 50*time.Millisecond)

    // Log another request with different status
    ctx2 := logging.WithMeta(context.Background(), logging.Meta{
        RequestID: "req-002",
        IP:        "127.0.0.1",
        Method:    "POST",
        Path:      "/api/data",
        UserAgent: "MyApp/1.0",
    })
    logger.LogRequest(ctx2, 500, 200*time.Millisecond)

    // Error logging with context
    dbErr := errors.New("database connection failed")
    logger.Error(ctx, dbErr)

    // JSON error logging for Loki/monitoring
    logger.ErrorLoki(ctx, logging.LevelCritical, dbErr)

    logger.Info("Application finished")
}
```

### Output Example (Access Log)

```
2026/02/04 12:00:00 logger.go:129: [INFO] Application started
2026/02/04 12:00:00 logger.go:129: [INFO] Processing data...
2026/02/04 12:00:00 logger.go:155: [REQ:req-001] 2026-02-04T12:00:00+07:00 | 200 |          50ms |       127.0.0.1 | GET     /api/users
2026/02/04 12:00:00 logger.go:155: [REQ:req-002] 2026-02-04T12:00:00+07:00 | 500 |         200ms |       127.0.0.1 | POST    /api/data
2026/02/04 12:00:00 logger.go:129: [INFO] Application finished
```

### Output Example (Loki JSON Log)

```json
{"http":{"ip":"127.0.0.1","method":"GET","path":"/api/users","ua":"MyApp/1.0"},"latency_ms":50,"level":"INFO","request_id":"req-001","service":"my-cli-app","status_code":200,"ts":"2026-02-04T12:00:00+07:00"}
{"http":{"ip":"127.0.0.1","method":"POST","path":"/api/data","ua":"MyApp/1.0"},"latency_ms":200,"level":"CRITICAL","request_id":"req-002","service":"my-cli-app","status_code":500,"ts":"2026-02-04T12:00:00+07:00"}
```

## Usage With Gin Framework

```go
package main

import (
    "errors"

    "github.com/gin-gonic/gin"
    logging "github.com/ahmadsaubani/go-logging-lib"
    "github.com/ahmadsaubani/go-logging-lib/middleware"
)

func main() {
    config := &logging.Config{
        ServiceName:    "my-api",
        LogPath:        "./logs/api",
        EnableStdout:   true,
        EnableFile:     true,
        EnableLoki:     true,
        EnableRotation: true,
    }

    logger, err := logging.New(config)
    if err != nil {
        panic(err)
    }

    r := gin.New()

    // Middleware order is important!
    r.Use(middleware.GinMiddleware(logger))       // 1. Request metadata injection
    r.Use(middleware.GinLogger(logger))           // 2. Access logging (all status codes)
    r.Use(middleware.GinHTTPErrorLogger(logger))  // 3. Error logging (4xx, 5xx)
    r.Use(middleware.GinRecovery(logger))         // 4. Panic recovery

    // Routes
    r.GET("/", func(c *gin.Context) {
        logger.Info("Home endpoint accessed")
        c.JSON(200, gin.H{"message": "Hello World"})
    })

    r.GET("/users", func(c *gin.Context) {
        logger.Info("Fetching users")
        c.JSON(200, gin.H{"users": []string{"alice", "bob"}})
    })

    r.POST("/users", func(c *gin.Context) {
        logger.Info("Creating user")
        c.JSON(201, gin.H{"message": "User created"})
    })

    // Manual error logging with anti-duplication
    r.GET("/error", func(c *gin.Context) {
        err := errors.New("something went wrong")
        logger.LogErrorWithMark(c, err) // Prevents duplicate logging
        c.JSON(500, gin.H{"error": "Internal server error"})
    })

    // Auto error logging (handled by GinHTTPErrorLogger)
    r.GET("/not-found", func(c *gin.Context) {
        c.JSON(404, gin.H{"error": "Resource not found"})
    })

    // Panic recovery test
    r.GET("/panic", func(c *gin.Context) {
        panic("unexpected error")
    })

    logger.Info("Starting server on :8080")
    r.Run(":8080")
}
```

### Gin Middleware Functions

| Middleware | Description |
|------------|-------------|
| `GinMiddleware` | Injects request metadata (request ID, IP, method, path) into context |
| `GinLogger` | Logs all HTTP requests to access log and Loki (all status codes) |
| `GinHTTPErrorLogger` | Logs detailed errors for 4xx and 5xx responses |
| `GinRecovery` | Recovers from panics and logs the error |

### Access Log Output (Gin)

```
2026/02/04 12:00:00 logger.go:129: [INFO] Home endpoint accessed
2026/02/04 12:00:00 logger.go:133: [REQ:abc-123] 2026-02-04T12:00:00+07:00 | 200 |      150.25µs |       127.0.0.1 | GET     /
2026/02/04 12:00:00 logger.go:129: [INFO] Fetching users
2026/02/04 12:00:00 logger.go:133: [REQ:def-456] 2026-02-04T12:00:00+07:00 | 200 |       85.50µs |       127.0.0.1 | GET     /users
2026/02/04 12:00:00 logger.go:133: [REQ:ghi-789] 2026-02-04T12:00:00+07:00 | 404 |       25.00µs |       127.0.0.1 | GET     /not-found
2026/02/04 12:00:00 logger.go:133: [REQ:jkl-012] 2026-02-04T12:00:00+07:00 | 500 |      200.00µs |       127.0.0.1 | GET     /error
```

### Error Log Output (Gin)

```
2026/02/04 12:00:00 error_logging.go:37: [ERROR]
==============================CRITICAL[12:00:00]==================================
ERROR  : something went wrong
REQ    : jkl-012
FROM   : handler.go:25
HTTP   : GET /error (127.0.0.1)
UA     : Mozilla/5.0
STACK  :
- handler.go:25                main.errorHandler
- context.go:192               gin.(*Context).Next
- gin.go:114                   middleware.GinRecovery.func1
==============================CRITICAL[12:00:00]==================================
```

### Loki JSON Output (Gin)

```json
{"http":{"ip":"127.0.0.1","method":"GET","path":"/","ua":"Mozilla/5.0"},"latency_ms":0,"level":"INFO","request_id":"abc-123","service":"my-api","status_code":200,"ts":"2026-02-04T12:00:00+07:00"}
{"http":{"ip":"127.0.0.1","method":"GET","path":"/not-found","ua":"Mozilla/5.0"},"latency_ms":0,"level":"ERROR","request_id":"ghi-789","service":"my-api","status_code":404,"ts":"2026-02-04T12:00:00+07:00"}
{"http":{"ip":"127.0.0.1","method":"GET","path":"/error","ua":"Mozilla/5.0"},"latency_ms":0,"level":"CRITICAL","request_id":"jkl-012","service":"my-api","status_code":500,"ts":"2026-02-04T12:00:00+07:00"}
```

## Log Levels

| Level | Status Code | Description |
|-------|-------------|-------------|
| `INFO` | 200-299 | Successful requests |
| `WARN` | 300-399 | Redirects |
| `ERROR` | 400-499 | Client errors |
| `CRITICAL` | 500+ | Server errors |

## API Reference

### Logger Methods

```go
// Info logs an info message to access log
logger.Info(msg string)

// LogRequest logs HTTP request to access log and Loki (for non-Gin usage)
logger.LogRequest(ctx context.Context, statusCode int, latency time.Duration)

// Error logs detailed error to error log
logger.Error(ctx context.Context, err error)

// ErrorLoki logs error in JSON format to Loki log
logger.ErrorLoki(ctx context.Context, level LogLevel, err error)

// LogErrorWithMark logs error and marks it to prevent duplicate logging (Gin only)
logger.LogErrorWithMark(c *gin.Context, err error)
```

### Context Functions

```go
// WithMeta adds request metadata to context
ctx := logging.WithMeta(context.Background(), logging.Meta{
    RequestID: "req-001",
    IP:        "127.0.0.1",
    Method:    "GET",
    Path:      "/api/users",
    UserAgent: "MyApp/1.0",
})

// FromContext retrieves metadata from context
meta, ok := logging.FromContext(ctx)
```

## Best Practices

1. **Always set `EnableRotation: true` in production** to prevent log files from growing indefinitely
2. **Use `LogErrorWithMark` in Gin handlers** to prevent duplicate error logging
3. **Set meaningful `ServiceName`** for easier log filtering in monitoring systems
4. **Use appropriate log levels** via `ErrorLoki` for proper alerting
5. **Middleware order matters in Gin**: GinMiddleware -> GinLogger -> GinHTTPErrorLogger -> GinRecovery

## License

MIT License - see [LICENSE](LICENSE) for details.
