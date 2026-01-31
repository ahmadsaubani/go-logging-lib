# Go Logging Library 🚀

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue.svg)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A comprehensive, production-ready Go logging library with daily rotation, multi-format output, and **Gin framework integration**.

## 🌟 Features

- **📅 Daily Log Rotation**: Automatic dated log files with thread-safe operations
- **🎯 Multi-format Output**: Console, file, and JSON/Loki formats simultaneously  
- **🚀 Gin Integration**: Complete middleware suite with anti-duplication mechanisms
- **📡 Context-aware Logging**: Request metadata injection for structured logging
- **🔍 Advanced Error Handling**: Stack traces, request context, error categorization
- **⚡ Thread-safe**: Built for concurrent high-performance applications
- **🧪 Battle-tested**: Comprehensive test suite with actual file verification

---

## 📦 Installation

```bash
go get github.com/ahmadsaubani/go-logging-lib
```

For Gin projects (recommended):
```bash
go get github.com/ahmadsaubani/go-logging-lib
go get github.com/gin-gonic/gin
```

---

## 🚀 Quick Start Examples

### 1. Basic Console/CLI Application

```go
package main

import (
    "context"
    "errors"
    "fmt"
    
    logging "github.com/ahmadsaubani/go-logging-lib"
)

func main() {
    // Simple configuration
    config := &logging.Config{
        ServiceName:  "my-cli-app",
        LogPath:      "./logs/app",
        EnableStdout: true,  // Show in console
        EnableFile:   true,  // Save to files
        EnableLoki:   false, // Skip JSON format for simple apps
    }

    logger, err := logging.New(config)
    if err != nil {
        panic(fmt.Sprintf("Failed to create logger: %v", err))
    }

    // Basic logging
    logger.Info("🚀 Application started")
    logger.Info("📊 Loading configuration...")
    logger.Info("✅ Ready to process requests")

    // Error logging with context metadata
    ctx := logging.WithMeta(context.Background(), logging.Meta{
        RequestID: "cli-001",
        IP:        "localhost",
        UserAgent: "CLI-App/1.0",
    })

    // Simulate an error
    if err := processData(); err != nil {
        logger.Error(ctx, err)
    }

    // JSON formatted error for monitoring
    criticalErr := errors.New("database connection lost")
    logger.ErrorLoki(ctx, logging.LevelCritical, criticalErr)

    logger.Info("🏁 Application finished")
}

func processData() error {
    return errors.New("failed to connect to database")
}
```

### 2. Web API with Gin Framework (Complete Example)

```go
package main

import (
    "errors"
    "fmt"
    "time"
    
    "github.com/gin-gonic/gin"
    logging "github.com/ahmadsaubani/go-logging-lib"
    "github.com/ahmadsaubani/go-logging-lib/middleware"
)

func main() {
    // Production-ready configuration
    config := &logging.Config{
        ServiceName:  "user-api",
        LogPath:      "./logs/api",
        EnableStdout: true,
        EnableFile:   true,
        EnableLoki:   true, // Enable for monitoring
    }

    logger, err := logging.New(config)
    if err != nil {
        panic(err)
    }

    r := gin.New()
    
    // 🔥 CRITICAL: Middleware order matters!
    r.Use(middleware.GinMiddleware(logger))      // 1. Request metadata injection
    r.Use(middleware.GinLogger(logger))         // 2. Request/response logging
    r.Use(middleware.GinHTTPErrorLogger(logger)) // 3. Auto-error logging
    r.Use(middleware.GinRecovery(logger))       // 4. Panic recovery

    // API Routes
    api := r.Group("/api/v1")
    {
        api.GET("/health", healthCheck(logger))
        api.GET("/users", getUsers(logger))
        api.POST("/users", createUser(logger))
        api.GET("/users/:id", getUser(logger))
        api.DELETE("/users/:id", deleteUser(logger))
    }

    // Error simulation endpoints for testing
    r.GET("/error", func(c *gin.Context) {
        err := errors.New("simulated business logic error")
        logger.LogErrorWithMark(c, err) // Prevents duplicate logging
        c.JSON(500, gin.H{"error": "Internal server error"})
    })

    r.GET("/panic", func(c *gin.Context) {
        panic("simulated server panic") // Will be caught by recovery middleware
    })

    logger.Info("🌐 Starting server on :8080")
    r.Run(":8080")
}

func healthCheck(logger *logging.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        logger.Info("🏥 Health check requested")
        c.JSON(200, gin.H{
            "status":    "healthy",
            "service":   "user-api",
            "version":   "1.0.0",
            "timestamp": time.Now().Format(time.RFC3339),
        })
    }
}

func getUsers(logger *logging.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        logger.Info("👥 Fetching users list")
        
        // Simulate database call
        users := []map[string]interface{}{
            {"id": 1, "name": "Alice Johnson", "email": "alice@example.com"},
            {"id": 2, "name": "Bob Smith", "email": "bob@example.com"},
            {"id": 3, "name": "Charlie Brown", "email": "charlie@example.com"},
        }
        
        logger.Info(fmt.Sprintf("📊 Retrieved %d users", len(users)))
        c.JSON(200, gin.H{"users": users, "count": len(users)})
    }
}

func createUser(logger *logging.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        var user struct {
            Name  string `json:"name" binding:"required"`
            Email string `json:"email" binding:"required,email"`
        }

        if err := c.ShouldBindJSON(&user); err != nil {
            logger.LogErrorWithMark(c, err)
            c.JSON(400, gin.H{"error": "Invalid input", "details": err.Error()})
            return
        }

        logger.Info(fmt.Sprintf("➕ Creating user: %s <%s>", user.Name, user.Email))
        
        // Simulate user creation
        newUser := map[string]interface{}{
            "id":    4,
            "name":  user.Name,
            "email": user.Email,
        }

        c.JSON(201, gin.H{"user": newUser, "message": "User created successfully"})
    }
}

func getUser(logger *logging.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID := c.Param("id")
        logger.Info(fmt.Sprintf("🔍 Fetching user ID: %s", userID))
        
        // Simulate user lookup
        if userID == "999" {
            err := errors.New("user not found in database")
            logger.LogErrorWithMark(c, err)
            c.JSON(404, gin.H{"error": "User not found"})
            return
        }
        
        user := map[string]interface{}{
            "id":    userID,
            "name":  "Sample User",
            "email": "sample@example.com",
        }
        
        c.JSON(200, gin.H{"user": user})
    }
}

func deleteUser(logger *logging.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID := c.Param("id")
        logger.Info(fmt.Sprintf("🗑️ Deleting user ID: %s", userID))
        
        // Simulate critical operation with monitoring
        if userID == "admin" {
            err := errors.New("attempt to delete admin user blocked")
            logger.ErrorLoki(c.Request.Context(), logging.LevelCritical, err)
            c.JSON(403, gin.H{"error": "Cannot delete admin user"})
            return
        }
        
        c.JSON(200, gin.H{"message": "User deleted successfully"})
    }
}
```

---

## 🏗️ Non-Gin Framework Integration

**Note**: This library has **native Gin middleware**, but can be used with other frameworks using standard Go patterns:

### Chi Router Example

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "time"
    
    "github.com/go-chi/chi/v5"
    logging "github.com/ahmadsaubani/go-logging-lib"
)

func main() {
    config := &logging.Config{
        ServiceName:  "chi-api",
        LogPath:      "./logs/chi-app",
        EnableStdout: true,
        EnableFile:   true,
        EnableLoki:   true,
    }

    logger, _ := logging.New(config)
    
    r := chi.NewRouter()
    
    // Custom Chi middleware (manual implementation)
    r.Use(func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Manually inject context metadata
            ctx := logging.WithMeta(r.Context(), logging.Meta{
                RequestID: generateRequestID(),
                IP:        r.RemoteAddr,
                Method:    r.Method,
                Path:      r.URL.Path,
                UserAgent: r.UserAgent(),
            })
            
            logger.Info(fmt.Sprintf("📨 %s %s", r.Method, r.URL.Path))
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    })
    
    r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
        logger.Info("🏥 Health endpoint accessed")
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"status": "healthy", "service": "chi-api"}`))
    })
    
    logger.Info("🚀 Chi server starting on :8080")
    http.ListenAndServe(":8080", r)
}

func generateRequestID() string {
    return fmt.Sprintf("req-%d", time.Now().UnixNano())
}
```

### Standard HTTP Handler Example

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "net/http"
    "time"
    
    logging "github.com/ahmadsaubani/go-logging-lib"
)

func main() {
    config := &logging.Config{
        ServiceName:  "http-server",
        LogPath:      "./logs/http-app",
        EnableStdout: true,
        EnableFile:   true,
        EnableLoki:   true,
    }

    logger, _ := logging.New(config)
    
    // Wrapper untuk inject context
    loggedHandler := func(handler http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            ctx := logging.WithMeta(r.Context(), logging.Meta{
                RequestID: generateRequestID(),
                IP:        r.RemoteAddr,
                Method:    r.Method,
                Path:      r.URL.Path,
                UserAgent: r.UserAgent(),
            })
            
            logger.Info(fmt.Sprintf("%s %s", r.Method, r.URL.Path))
            
            // Pass context to handler
            handler(w, r.WithContext(ctx))
        }
    }
    
    http.HandleFunc("/health", loggedHandler(func(w http.ResponseWriter, r *http.Request) {
        logger.Info("Health check")
        w.Write([]byte("OK"))
    }))
    
    http.HandleFunc("/error", loggedHandler(func(w http.ResponseWriter, r *http.Request) {
        err := errors.New("sample error")
        logger.Error(r.Context(), err)
        logger.ErrorLoki(r.Context(), logging.LevelError, err)
        http.Error(w, "Internal Server Error", 500)
    }))
    
    logger.Info("🚀 HTTP server starting on :8080")
    http.ListenAndServe(":8080", nil)
}
```

---

## ⚙️ Configuration

### Configuration Structure

```go
type Config struct {
    ServiceName  string // Service identifier (appears in logs)
    LogPath      string // Base path for log files (without extension)
    EnableStdout bool   // Enable console output
    EnableFile   bool   // Enable file output  
    EnableLoki   bool   // Enable JSON/structured output
}
```

### Environment-based Configuration

```go
package main

import (
    "os"
    logging "github.com/ahmadsaubani/go-logging-lib"
)

func createLoggerConfig() *logging.Config {
    env := os.Getenv("APP_ENV")
    serviceName := os.Getenv("SERVICE_NAME")
    
    if serviceName == "" {
        serviceName = "my-app"
    }
    
    switch env {
    case "production":
        return &logging.Config{
            ServiceName:  serviceName,
            LogPath:      "/var/log/" + serviceName + "/app",
            EnableStdout: false, // Reduce console noise in production
            EnableFile:   true,  // Essential for production debugging
            EnableLoki:   true,  // Enable monitoring integration
        }
        
    case "staging":
        return &logging.Config{
            ServiceName:  serviceName + "-staging",
            LogPath:      "./logs/staging-app",
            EnableStdout: true,  // Helpful for staging debugging
            EnableFile:   true,  // Keep logs for analysis
            EnableLoki:   true,  // Test monitoring integration
        }
        
    case "testing":
        return &logging.Config{
            ServiceName:  serviceName + "-test",
            LogPath:      "./test-logs/app",
            EnableStdout: false, // Reduce test noise
            EnableFile:   true,  // Verify test log output
            EnableLoki:   true,  // Test JSON format
        }
        
    default: // development
        return &logging.Config{
            ServiceName:  serviceName + "-dev",
            LogPath:      "./logs/dev-app", 
            EnableStdout: true,  // Show logs in terminal
            EnableFile:   false, // Skip file creation for speed
            EnableLoki:   false, // Not needed in development
        }
    }
}

func main() {
    config := createLoggerConfig()
    logger, err := logging.New(config)
    if err != nil {
        panic(err)
    }
    
    logger.Info(fmt.Sprintf("🚀 Starting %s in %s mode", 
        config.ServiceName, os.Getenv("APP_ENV")))
}
```

---

## 📁 Log File Structure & Examples

### File Naming Convention

```
{LogPath}.access-{YYYY-MM-DD}.log      # Info/access logs
{LogPath}.error-{YYYY-MM-DD}.log       # Human-readable error logs  
{LogPath}.error-loki-{YYYY-MM-DD}.log  # JSON structured logs
```

### Real Log File Examples

#### Access Log (`app.access-2026-01-31.log`)
```
2026/01/31 13:20:15 logger.go:125: [INFO] 🚀 Application started
2026/01/31 13:20:15 logger.go:125: [INFO] 📊 Loading configuration...
2026/01/31 13:20:16 logger.go:125: [INFO] 🌐 Starting server on :8080
2026/01/31 13:20:45 logger.go:125: [INFO] 👥 Fetching users list  
2026/01/31 13:20:45 logger.go:125: [INFO] 📊 Retrieved 15 users
2026/01/31 13:21:10 logger.go:125: [INFO] ➕ Creating user: John Doe <john@example.com>
2026/01/31 13:21:30 logger.go:125: [INFO] 🔍 Fetching user ID: 123
```

#### Error Log (`app.error-2026-01-31.log`)
```
2026/01/31 13:22:15 error_logging.go:37: [ERROR]
==============================CRITICAL[13:22:15]==================================
ERROR  : failed to connect to database: connection timeout
REQ    : req-abc123-def456
FROM   : main.go:87
HTTP   : POST /api/users (192.168.1.100)
UA     : PostmanRuntime/7.29.2
STACK  :
- main.go:87              main.createUser
- gin.go:192              gin.(*Context).Next  
- middleware.go:45        middleware.AuthMiddleware

==============================CRITICAL[13:22:15]==================================
```

#### Loki JSON Log (`app.error-loki-2026-01-31.log`)
```json
{"error":"failed to connect to database: connection timeout","http":{"ip":"192.168.1.100","method":"POST","path":"/api/users","ua":"PostmanRuntime/7.29.2"},"level":"CRITICAL","request_id":"req-abc123-def456","service":"user-api","source":{"file":"main.go","line":87},"stack":["main.go:87 main.createUser","gin.go:192 gin.(*Context).Next"],"ts":"2026-01-31T13:22:15+07:00"}

{"error":"validation failed: email is required","http":{"ip":"10.0.0.25","method":"PUT","path":"/api/users/456","ua":"axios/0.21.1"},"level":"ERROR","request_id":"req-validation-001","service":"user-api","source":{"file":"handlers.go","line":234},"stack":["handlers.go:234 handlers.updateUser","validator.go:45 validation.ValidateUser"],"ts":"2026-01-31T13:22:48+07:00"}
```

---

## 📚 Complete API Reference

### Core Logger Methods

#### `New(config *Config) (*Logger, error)`
Creates a new logger instance.

```go
config := &logging.Config{
    ServiceName:  "my-service",
    LogPath:      "./logs/app",
    EnableStdout: true,
    EnableFile:   true,
    EnableLoki:   true,
}

logger, err := logging.New(config)
if err != nil {
    log.Fatal("Failed to create logger:", err)
}
```

#### `Info(message string)`
Logs informational messages to access log and console.

```go
logger.Info("Service started successfully")
logger.Info(fmt.Sprintf("Processing %d items", itemCount))
logger.Info("🚀 Application ready to serve requests")
```

#### `Error(ctx context.Context, err error)`
Logs errors with full context, metadata, and stack trace.

```go
if err := database.Connect(); err != nil {
    logger.Error(ctx, err)
}

// With custom context
ctx := logging.WithMeta(context.Background(), logging.Meta{
    RequestID: "custom-req-001",
    IP:        "127.0.0.1",
    Method:    "POST",
    Path:      "/api/custom",
    UserAgent: "CustomClient/1.0",
})
logger.Error(ctx, errors.New("custom error with context"))
```

#### `ErrorLoki(ctx context.Context, level LogLevel, err error)`
Logs errors in JSON format with specified severity level.

```go
// Available levels: LevelInfo, LevelWarn, LevelError, LevelCritical
logger.ErrorLoki(ctx, logging.LevelInfo, infoErr)       // General info
logger.ErrorLoki(ctx, logging.LevelWarn, warningErr)    // Warning condition
logger.ErrorLoki(ctx, logging.LevelError, standardErr)  // Error condition  
logger.ErrorLoki(ctx, logging.LevelCritical, criticalErr) // Critical failure

// Practical examples
if memUsage > 90 {
    err := errors.New("high memory usage detected")
    logger.ErrorLoki(ctx, logging.LevelWarn, err)
}

if paymentFailed {
    err := errors.New("payment processing failed")
    logger.ErrorLoki(ctx, logging.LevelCritical, err)
}
```

### Context Management

#### `WithMeta(ctx context.Context, meta Meta) context.Context`
Injects request metadata into context for structured logging.

```go
// Complete metadata
meta := logging.Meta{
    RequestID: generateUniqueID(),
    IP:        extractClientIP(),
    Method:    "POST",
    Path:      "/api/v1/users",
    UserAgent: "MyApp/2.1.0",
}
ctx := logging.WithMeta(context.Background(), meta)

// Minimal metadata
meta := logging.Meta{
    RequestID: "batch-job-001",
}
ctx := logging.WithMeta(context.Background(), meta)
```

#### `FromContext(ctx context.Context) (Meta, bool)`
Retrieves metadata from context.

```go
meta, exists := logging.FromContext(ctx)
if exists {
    fmt.Printf("Processing request %s from %s", meta.RequestID, meta.IP)
} else {
    fmt.Println("No metadata in context")
}
```

### Gin Framework Helpers (Only Available for Gin)

#### `LogErrorWithMark(c *gin.Context, err error)`
Logs error with Gin context and prevents duplicate logging by middleware.

```go
func createUser(logger *logging.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        var user User
        if err := c.ShouldBindJSON(&user); err != nil {
            // This prevents the HTTP error middleware from logging again
            logger.LogErrorWithMark(c, err)
            c.JSON(400, gin.H{"error": "Invalid request format"})
            return
        }
        
        // Success case
        c.JSON(201, gin.H{"user": user, "status": "created"})
    }
}
```

#### `MarkErrorLogged(c *gin.Context)` / `IsErrorLogged(c *gin.Context) bool`
Manual control over error logging duplication (Gin only).

```go
func customErrorHandler(logger *logging.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next() // Process request
        
        // Check for errors after processing
        if len(c.Errors) > 0 && !logging.IsErrorLogged(c) {
            for _, err := range c.Errors {
                logger.Error(c.Request.Context(), err.Err)
            }
            logging.MarkErrorLogged(c)
        }
    }
}
```

### Gin Middleware Functions

#### `middleware.GinMiddleware(logger *logging.Logger) gin.HandlerFunc`
Injects request metadata into Gin context.

#### `middleware.GinLogger(logger *logging.Logger) gin.HandlerFunc`
Logs HTTP requests and responses.

#### `middleware.GinHTTPErrorLogger(logger *logging.Logger) gin.HandlerFunc`
Automatically logs HTTP errors with anti-duplication.

#### `middleware.GinRecovery(logger *logging.Logger) gin.HandlerFunc`
Recovers from panics and logs them.

---

## 🧪 Testing Guide

### 1. Testing the Library Itself

#### Running Library Tests

```bash
# Navigate to library directory
cd go-logging-lib

# Run comprehensive tests that create actual log files
cd tests && go test -v .

# Expected output:
# ✅ BASIC Access log: ../examples/basic/logs/app.access-2026-01-31.log (429 bytes)
# ✅ BASIC Error log: ../examples/basic/logs/app.error-2026-01-31.log (549 bytes)  
# ✅ BASIC Loki log: ../examples/basic/logs/app.error-loki-2026-01-31.log (859 bytes)
# ✅ GIN Access log: ../examples/gin/logs/gin-app.access-2026-01-31.log (155 bytes)
# ✅ GIN Error log: ../examples/gin/logs/gin-app.error-2026-01-31.log (2361 bytes)
# ✅ GIN Loki log: ../examples/gin/logs/gin-app.error-loki-2026-01-31.log (2408 bytes)
```

### 2. Testing in Your Projects

#### Unit Testing Example

```go
// user_service_test.go
package main

import (
    "context"
    "errors"
    "os"
    "testing"
    "time"
    
    logging "github.com/ahmadsaubani/go-logging-lib"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestUserService(t *testing.T) {
    // Setup test logger
    testLogDir := "./test-logs"
    os.RemoveAll(testLogDir) // Clean previous test logs
    
    config := &logging.Config{
        ServiceName:  "user-service-test",
        LogPath:      testLogDir + "/app",
        EnableStdout: false, // Reduce test noise
        EnableFile:   true,  // Verify file output
        EnableLoki:   true,  // Test JSON format
    }
    
    logger, err := logging.New(config)
    require.NoError(t, err)
    
    t.Run("successful user creation", func(t *testing.T) {
        ctx := logging.WithMeta(context.Background(), logging.Meta{
            RequestID: "test-create-001",
            IP:        "127.0.0.1",
            Method:    "POST",
            Path:      "/api/users",
        })
        
        // Test the service
        service := NewUserService(logger)
        user, err := service.CreateUser(ctx, "John Doe", "john@example.com")
        
        assert.NoError(t, err)
        assert.Equal(t, "John Doe", user.Name)
        assert.Equal(t, "john@example.com", user.Email)
        
        // Verify logs were created
        verifyLogFiles(t, testLogDir)
    })
    
    // Cleanup
    t.Cleanup(func() {
        os.RemoveAll(testLogDir)
    })
}

func verifyLogFiles(t *testing.T, logDir string) {
    today := time.Now().Format("2006-01-02")
    
    accessFile := logDir + "/app.access-" + today + ".log"
    errorFile := logDir + "/app.error-" + today + ".log"  
    lokiFile := logDir + "/app.error-loki-" + today + ".log"
    
    // Check files exist
    assert.FileExists(t, accessFile, "Access log should exist")
    assert.FileExists(t, errorFile, "Error log should exist")
    assert.FileExists(t, lokiFile, "Loki log should exist")
    
    // Check file sizes
    accessStat, _ := os.Stat(accessFile)
    assert.Greater(t, accessStat.Size(), int64(0), "Access log should not be empty")
}
```

#### Integration Testing with Gin

```go
// gin_api_test.go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "os"
    "strings"
    "testing"
    "time"
    
    "github.com/gin-gonic/gin"
    logging "github.com/ahmadsaubani/go-logging-lib"
    "github.com/ahmadsaubani/go-logging-lib/middleware"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestGinAPIIntegration(t *testing.T) {
    gin.SetMode(gin.TestMode)
    
    // Setup test logger
    testLogDir := "./integration-test-logs"
    os.RemoveAll(testLogDir)
    
    config := &logging.Config{
        ServiceName:  "gin-integration-test",
        LogPath:      testLogDir + "/api",
        EnableStdout: false,
        EnableFile:   true,
        EnableLoki:   true,
    }
    
    logger, err := logging.New(config)
    require.NoError(t, err)
    
    // Setup router with middleware
    r := gin.New()
    r.Use(middleware.GinMiddleware(logger))
    r.Use(middleware.GinLogger(logger))
    r.Use(middleware.GinHTTPErrorLogger(logger))
    r.Use(middleware.GinRecovery(logger))
    
    // Add routes
    r.POST("/api/users", createUserHandler(logger))
    r.GET("/api/error", errorHandler(logger))
    r.GET("/api/panic", panicHandler())
    
    t.Run("successful request", func(t *testing.T) {
        user := map[string]string{
            "name":  "Test User",
            "email": "test@example.com",
        }
        
        body, _ := json.Marshal(user)
        req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(body))
        req.Header.Set("Content-Type", "application/json")
        req.Header.Set("User-Agent", "TestClient/1.0")
        
        w := httptest.NewRecorder()
        r.ServeHTTP(w, req)
        
        assert.Equal(t, 201, w.Code)
    })
    
    t.Run("error handling with anti-duplication", func(t *testing.T) {
        req := httptest.NewRequest("GET", "/api/error", nil)
        w := httptest.NewRecorder()
        r.ServeHTTP(w, req)
        
        assert.Equal(t, 500, w.Code)
        
        // Verify no duplicate error logs
        verifyNoDuplicateErrors(t, testLogDir)
    })
    
    // Cleanup
    t.Cleanup(func() {
        os.RemoveAll(testLogDir)
    })
}

func createUserHandler(logger *logging.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        var user struct {
            Name  string `json:"name" binding:"required"`
            Email string `json:"email" binding:"required,email"`
        }
        
        if err := c.ShouldBindJSON(&user); err != nil {
            logger.LogErrorWithMark(c, err)
            c.JSON(400, gin.H{"error": "Invalid input"})
            return
        }
        
        c.JSON(201, gin.H{"user": user})
    }
}

func errorHandler(logger *logging.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        err := errors.New("manual error for testing")
        logger.LogErrorWithMark(c, err) // This should prevent HTTP middleware duplication
        c.JSON(500, gin.H{"error": "Something went wrong"})
    }
}

func panicHandler() gin.HandlerFunc {
    return func(c *gin.Context) {
        panic("test panic for recovery")
    }
}

func verifyNoDuplicateErrors(t *testing.T, logDir string) {
    today := time.Now().Format("2006-01-02")
    errorFile := logDir + "/api.error-" + today + ".log"
    
    content, err := os.ReadFile(errorFile)
    require.NoError(t, err)
    
    errorContent := string(content)
    manualErrors := strings.Count(errorContent, "manual error for testing")
    httpErrors := strings.Count(errorContent, "HTTP Error")
    
    // Should only have manual error, not HTTP error (due to anti-duplication)
    assert.Equal(t, 1, manualErrors, "Should have exactly one manual error log")
    assert.Equal(t, 0, httpErrors, "Should not have HTTP error log due to anti-duplication")
}
```

---

## 🚀 Using in Your Projects

### Step 1: Install
```bash
go get github.com/ahmadsaubani/go-logging-lib
```

### Step 2: Initialize (once per application)
```go
package main

import (
    "fmt"
    logging "github.com/ahmadsaubani/go-logging-lib"
)

var AppLogger *logging.Logger

func init() {
    config := &logging.Config{
        ServiceName:  "your-app-name", 
        LogPath:      "./logs/app",
        EnableStdout: true,
        EnableFile:   true, 
        EnableLoki:   true,
    }
    
    var err error
    AppLogger, err = logging.New(config)
    if err != nil {
        panic(fmt.Sprintf("Failed to initialize logger: %v", err))
    }
}
```

### Step 3: Use in Business Logic
```go
// In handlers, services, business logic, etc.
func ProcessOrder(ctx context.Context, orderID string) error {
    AppLogger.Info(fmt.Sprintf("Processing order %s", orderID))
    
    if err := validateOrder(orderID); err != nil {
        AppLogger.Error(ctx, err)
        return err
    }
    
    AppLogger.Info(fmt.Sprintf("Order %s processed successfully", orderID))
    return nil
}
```

### Step 4: For Gin Projects, Add Middleware
```go
r := gin.New()
r.Use(middleware.GinMiddleware(AppLogger))
r.Use(middleware.GinLogger(AppLogger))
r.Use(middleware.GinHTTPErrorLogger(AppLogger))
r.Use(middleware.GinRecovery(AppLogger))
```

---

## 🔧 Production Deployment

### Docker Configuration

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .

# Create log directory
RUN mkdir -p /var/log/myapp

# Set environment variables
ENV APP_ENV=production
ENV SERVICE_NAME=my-app

CMD ["./main"]
```

```yaml
# docker-compose.yml
version: '3.8'
services:
  app:
    build: .
    environment:
      - APP_ENV=production
      - SERVICE_NAME=my-app
    volumes:
      - ./logs:/var/log/myapp  # Mount log directory
    ports:
      - "8080:8080"
```

### Monitoring Integration

The JSON logs (`*.error-loki-*.log`) can be directly ingested by:
- **Grafana/Loki**: Configure Promtail to watch log files
- **ELK Stack**: Logstash can parse the JSON format directly
- **Datadog/New Relic**: Configure agents to monitor JSON log files

---

## 📊 Performance Considerations

- **File Output**: Disable in development for faster iteration
- **Buffer Size**: Logs written immediately for reliability
- **Concurrent Safety**: Built-in mutex protection
- **Memory Usage**: Minimal overhead, stack traces only on errors
- **Disk Space**: Daily rotation prevents unlimited growth

---

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

- 📚 [Documentation](https://github.com/ahmadsaubani/go-logging-lib#readme)
- 🐛 [Report Issues](https://github.com/ahmadsaubani/go-logging-lib/issues)
- 💬 [Discussions](https://github.com/ahmadsaubani/go-logging-lib/discussions)

---

**Made with ❤️ for the Go community** 

⭐ **If this library helps your project, please give it a star!** ⭐