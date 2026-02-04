# Go Logging Library

A production-ready Go logging library with optional daily rotation, multi-format output, unified Loki/Grafana JSON format, and support for both Gin framework and standard `net/http`.

## Features

- **Daily Log Rotation**: Optional automatic dated log files with thread-safe operations
- **Multi-format Output**: Console, file, and JSON/Loki formats simultaneously
- **Unified Loki Format**: Consistent JSON structure for Grafana visualization
- **Gin Framework Integration**: Complete middleware suite with anti-duplication mechanisms
- **Standard HTTP Support**: Middleware for `net/http` (non-Gin) applications
- **Context-aware Logging**: Request metadata injection for structured logging
- **Advanced Error Handling**: Stack traces, source file info, error categorization
- **Thread-safe**: Built for concurrent high-performance applications

## Installation

```bash
go get github.com/ahmadsaubani/go-logging-lib
```

For Gin framework projects:
```bash
go get github.com/ahmadsaubani/go-logging-lib
go get github.com/gin-gonic/gin
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
    
    // Middleware order is important!
    r.Use(middleware.GinMiddleware(logger))       // 1. Setup request context
    r.Use(middleware.GinLogger(logger))           // 2. Log requests + Loki
    r.Use(middleware.GinHTTPErrorLogger(logger))  // 3. Error details (4xx, 5xx)
    r.Use(middleware.GinRecovery(logger))         // 4. Panic recovery (LAST)

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

    // Wrap with middleware (order: Recovery -> Logger -> Middleware -> Handler)
    handler := middleware.HTTPMiddleware(logger,
        middleware.HTTPLogger(logger,
            middleware.HTTPRecovery(logger, mux)))

    http.ListenAndServe(":8080", handler)
}
```

## Configuration

```go
type Config struct {
    ServiceName    string `yaml:"service_name"`    // Service identifier in logs
    LogPath        string `yaml:"log_path"`        // Directory for log files
    FilePrefix     string `yaml:"file_prefix"`     // Prefix for log filenames (default: "app")
    EnableStdout   bool   `yaml:"enable_stdout"`   // Output to console
    EnableFile     bool   `yaml:"enable_file"`     // Output to log files
    EnableLoki     bool   `yaml:"enable_loki"`     // Output JSON format for Loki/Grafana
    EnableRotation bool   `yaml:"enable_rotation"` // Enable daily log rotation
}
```

### Log Files Generated

With `EnableRotation: true`:
```
logs/
├── app.access-2026-02-04.log    # Access logs (daily rotation)
├── app.error-2026-02-04.log     # Error logs (daily rotation)
└── app.loki-2026-02-04.log      # JSON logs for Loki (daily rotation)
```

With `EnableRotation: false`:
```
logs/
├── app.access.log    # Single access log file
├── app.error.log     # Single error log file
└── app.loki.log      # Single JSON log file
```

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
```

## Unified Loki JSON Format

The library outputs a **consistent JSON format** for all requests, making it easy to visualize in Grafana:

### Success Response (2xx, 3xx, 4xx without error)
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
        "ua": "PostmanRuntime/7.51.1"
    },
    "errors": null
}
```

### Error Response (with explicit error)
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
        "method": "GET",
        "path": "/ping",
        "ua": "PostmanRuntime/7.51.1"
    },
    "errors": {
        "error": "database connection failed",
        "source": {
            "file": "user_handler.go",
            "line": 45
        },
        "stack": [
            "user_handler.go:45 handlers.GetUser",
            "context.go:192 gin.(*Context).Next",
            "recovery.go:101 middleware.GinRecovery.func1"
        ]
    }
}
```

### Log Levels

| Level | Status Code | Description |
|-------|-------------|-------------|
| `INFO` | 200-299 | Successful requests |
| `WARN` | 300-399 | Redirects |
| `ERROR` | 400-499 | Client errors |
| `CRITICAL` | 500+ or explicit error | Server errors |

## Usage Examples

### Gin Framework (Complete Example)

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
        LogPath:        "./logs",
        FilePrefix:     "app",
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

    // Middleware order is CRITICAL!
    r.Use(middleware.GinMiddleware(logger))       // 1. Setup request context
    r.Use(middleware.GinLogger(logger))           // 2. Log requests + Loki
    r.Use(middleware.GinHTTPErrorLogger(logger))  // 3. Error details
    r.Use(middleware.GinRecovery(logger))         // 4. Panic recovery (LAST!)

    // Success endpoint
    r.GET("/", func(c *gin.Context) {
        logger.Info("Home endpoint accessed")
        c.JSON(200, gin.H{"message": "Hello World"})
    })

    // Error with manual logging (anti-duplication)
    r.GET("/error", func(c *gin.Context) {
        err := errors.New("database connection failed")
        logger.LogErrorWithMark(c, err) // Mark to prevent duplicate
        c.JSON(500, gin.H{"error": "Internal server error"})
    })

    // Auto error logging (handled by middleware)
    r.GET("/not-found", func(c *gin.Context) {
        c.JSON(404, gin.H{"error": "Resource not found"})
    })

    // Panic recovery test
    r.GET("/panic", func(c *gin.Context) {
        panic("unexpected error!")
    })

    logger.Info("Server starting on :8080")
    r.Run(":8080")
}
```

### Standard HTTP (Complete Example)

```go
package main

import (
    "encoding/json"
    "errors"
    "net/http"

    logging "github.com/ahmadsaubani/go-logging-lib"
    "github.com/ahmadsaubani/go-logging-lib/middleware"
)

func main() {
    config := &logging.Config{
        ServiceName:    "my-api",
        LogPath:        "./logs",
        FilePrefix:     "app",
        EnableStdout:   true,
        EnableFile:     true,
        EnableLoki:     true,
        EnableRotation: true,
    }

    logger, err := logging.New(config)
    if err != nil {
        panic(err)
    }

    mux := http.NewServeMux()

    // Success endpoint
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        logger.Info("Home endpoint accessed")
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]string{"message": "Hello World"})
    })

    // Error endpoint with explicit error logging
    mux.HandleFunc("/error", func(w http.ResponseWriter, r *http.Request) {
        err := errors.New("database connection failed")
        
        // Store error in context for Loki logging
        ctx := logging.WithError(r.Context(), err)
        r = r.WithContext(ctx)
        
        logger.Error(r.Context(), err)
        
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(500)
        json.NewEncoder(w).Encode(map[string]string{"error": "Internal server error"})
    })

    // Panic test
    mux.HandleFunc("/panic", func(w http.ResponseWriter, r *http.Request) {
        panic("unexpected error!")
    })

    // Wrap with middleware (outermost runs first)
    handler := middleware.HTTPMiddleware(logger,
        middleware.HTTPLogger(logger,
            middleware.HTTPRecovery(logger, mux)))

    logger.Info("Server starting on :8080")
    http.ListenAndServe(":8080", handler)
}
```

### CLI/Background Worker (Without HTTP)

```go
package main

import (
    "context"
    "errors"
    "time"

    logging "github.com/ahmadsaubani/go-logging-lib"
)

func main() {
    logger, _ := logging.New(&logging.Config{
        ServiceName:    "my-worker",
        LogPath:        "./logs",
        FilePrefix:     "worker",
        EnableStdout:   true,
        EnableFile:     true,
        EnableLoki:     true,
        EnableRotation: true,
    })

    logger.Info("Worker started")

    // Create context with metadata (for tracking)
    ctx := logging.WithMeta(context.Background(), logging.Meta{
        RequestID: "job-001",
        IP:        "127.0.0.1",
        Method:    "WORKER",
        Path:      "/process-queue",
        UserAgent: "Worker/1.0",
    })

    // Simulate job processing
    startTime := time.Now()
    
    // ... do work ...
    
    // Log success
    logger.LogRequest(ctx, 200, time.Since(startTime))

    // Log error if job fails
    if err := processJob(); err != nil {
        logger.LogRequestWithError(ctx, 500, time.Since(startTime), err)
    }

    logger.Info("Worker finished")
}

func processJob() error {
    return errors.New("job processing failed")
}
```

## Middleware Reference

### Gin Middleware

| Middleware | Description |
|------------|-------------|
| `GinMiddleware(logger)` | Setup request context with metadata (request ID, IP, method, path, user agent) |
| `GinLogger(logger)` | Log all requests to access log + Loki with unified format |
| `GinHTTPErrorLogger(logger)` | Log detailed errors for 4xx/5xx to error log |
| `GinRecovery(logger)` | Catch panics and log with stack trace |

**⚠️ Middleware Order (Critical!):**
```go
r.Use(middleware.GinMiddleware(logger))       // 1. FIRST - Setup context
r.Use(middleware.GinLogger(logger))           // 2. Log requests
r.Use(middleware.GinHTTPErrorLogger(logger))  // 3. Log errors
r.Use(middleware.GinRecovery(logger))         // 4. LAST - Catch panics
```

### HTTP Middleware (net/http)

| Middleware | Description |
|------------|-------------|
| `HTTPMiddleware(logger, next)` | Setup request context with metadata |
| `HTTPLogger(logger, next)` | Log all requests to access log + Loki |
| `HTTPRecovery(logger, next)` | Catch panics and log with stack trace |

**Middleware Wrapping Order:**
```go
// Outermost runs first, innermost runs last
handler := HTTPMiddleware(logger,      // 1. FIRST
    HTTPLogger(logger,                  // 2. Log requests
        HTTPRecovery(logger, mux)))     // 3. LAST - Catch panics
```

## API Reference

### Logger Methods

```go
// Create new logger instance
logger, err := logging.New(config *Config) (*Logger, error)

// Info logging (access log)
logger.Info(msg string)

// Log HTTP request (access log + Loki)
logger.LogRequest(ctx context.Context, statusCode int, latency time.Duration)

// Log HTTP request with error (access log + Loki with error details)
logger.LogRequestWithError(ctx context.Context, statusCode int, latency time.Duration, err error)

// Log error details (error log)
logger.Error(ctx context.Context, err error)

// Log error in Loki format (Loki log)
logger.ErrorLoki(ctx context.Context, level LogLevel, err error)

// Direct Loki logging with all parameters
logger.Loki(ctx context.Context, statusCode int, latency time.Duration, err error)

// Gin-specific: Log error and mark to prevent duplicate (error log)
logger.LogErrorWithMark(c *gin.Context, err error)
```

### Context Functions

```go
// Add request metadata to context
ctx := logging.WithMeta(ctx context.Context, meta logging.Meta) context.Context

// Get metadata from context
meta, ok := logging.FromContext(ctx context.Context) (Meta, bool)

// Add error to context (for HTTP middleware)
ctx := logging.WithError(ctx context.Context, err error) context.Context

// Get error from context
err := logging.ErrorFromContext(ctx context.Context) error

// Create new request context with auto-generated metadata (for net/http)
ctx := logging.NewRequestContext(r *http.Request) context.Context
```

### Gin Helper Functions

```go
// Set error in Gin context (marks for Loki logging)
middleware.SetLoggedError(c *gin.Context, err error)
```

### Meta Struct

```go
type Meta struct {
    RequestID string // Unique request identifier
    IP        string // Client IP address
    Method    string // HTTP method (GET, POST, etc.)
    Path      string // Request path
    UserAgent string // User agent string
}
```

## Output Examples

### Access Log (Console/File)
```
2026/02/04 12:00:00 logger.go:129: [INFO] Server starting
2026/02/04 12:00:00 logger.go:155: [REQ:abc-123] 2026-02-04T12:00:00+07:00 | 200 |       15.25ms |       127.0.0.1 | GET     /ping
2026/02/04 12:00:00 logger.go:155: [REQ:def-456] 2026-02-04T12:00:00+07:00 | 500 |      200.00ms |       127.0.0.1 | POST    /users
```

### Error Log (File)
```
==============================CRITICAL[12:00:00]==================================
ERROR  : database connection failed
REQ    : def-456
FROM   : user_handler.go:45
HTTP   : POST /users (127.0.0.1)
UA     : Mozilla/5.0
STACK  :
- user_handler.go:45           handlers.CreateUser
- context.go:192               gin.(*Context).Next
- recovery.go:101              middleware.GinRecovery.func1
==============================CRITICAL[12:00:00]==================================
```

### Loki JSON Log (File)
```json
{"ts":"2026-02-04T12:00:00+07:00","level":"INFO","service":"my-api","request_id":"abc-123","status_code":200,"latency_ms":15,"http":{"ip":"127.0.0.1","method":"GET","path":"/ping","ua":"Mozilla/5.0"},"errors":null}
{"ts":"2026-02-04T12:00:00+07:00","level":"CRITICAL","service":"my-api","request_id":"def-456","status_code":500,"latency_ms":200,"http":{"ip":"127.0.0.1","method":"POST","path":"/users","ua":"Mozilla/5.0"},"errors":{"error":"database connection failed","source":{"file":"user_handler.go","line":45},"stack":["user_handler.go:45 handlers.CreateUser","context.go:192 gin.(*Context).Next"]}}
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
            request_id: request_id
            status_code: status_code
            latency_ms: latency_ms
            method: http.method
            path: http.path
            ip: http.ip
            error: errors.error
      - labels:
          level:
          service:
          status_code:
          method:
```

### Useful Grafana Queries

```logql
# All errors
{job="my-api"} | json | level="CRITICAL" or level="ERROR"

# Slow requests (>1000ms)
{job="my-api"} | json | latency_ms > 1000

# Specific endpoint
{job="my-api"} | json | path="/api/users"

# Error rate by status code
sum by (status_code) (count_over_time({job="my-api"} | json | status_code >= 400 [5m]))

# Requests with errors
{job="my-api"} | json | errors != "null"
```

## Best Practices

### 1. Always Enable Rotation in Production
```go
config := &logging.Config{
    EnableRotation: true,  // Prevents log files from growing indefinitely
}
```

### 2. Use Meaningful Service Names
```go
config := &logging.Config{
    ServiceName: "user-api-v2",  // Easy to filter in Grafana
}
```

### 3. Use LogErrorWithMark in Gin Handlers
```go
// ✅ Good - prevents duplicate logging
logger.LogErrorWithMark(c, err)

// ❌ Bad - may log twice if middleware also logs
logger.Error(c.Request.Context(), err)
```

### 4. Store Errors in Context for HTTP Middleware
```go
// In your handler
ctx := logging.WithError(r.Context(), err)
r = r.WithContext(ctx)
// Middleware will automatically include error in Loki log
```

### 5. Middleware Order Matters!
```go
// Gin: Recovery LAST so it catches panics from all handlers
r.Use(middleware.GinRecovery(logger))  // LAST!

// HTTP: Recovery innermost so it wraps the handler
handler := HTTPMiddleware(logger,
    HTTPLogger(logger,
        HTTPRecovery(logger, mux)))  // Innermost!
```

### 6. Use File Prefix for Multiple Services
```go
// API service
apiLogger, _ := logging.New(&logging.Config{
    FilePrefix: "api",
    // Creates: api.access.log, api.error.log, api.loki.log
})

// Worker service  
workerLogger, _ := logging.New(&logging.Config{
    FilePrefix: "worker",
    // Creates: worker.access.log, worker.error.log, worker.loki.log
})
```

## Testing

Run tests:
```bash
cd /path/to/go-logging-lib
go test -v ./tests/...
```

## License

MIT License - see [LICENSE](LICENSE) for details.

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request
