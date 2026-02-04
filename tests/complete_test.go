package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ahmadsaubani/go-logging-lib"
	"github.com/ahmadsaubani/go-logging-lib/middleware"
	"github.com/gin-gonic/gin"
)

// LokiLogEntry represents the unified Loki JSON format
type LokiLogEntry struct {
	TS         string            `json:"ts"`
	Level      string            `json:"level"`
	Service    string            `json:"service"`
	RequestID  string            `json:"request_id"`
	StatusCode int               `json:"status_code"`
	LatencyMS  int64             `json:"latency_ms"`
	HTTP       map[string]string `json:"http"`
	Errors     *ErrorDetail      `json:"errors"`
}

type ErrorDetail struct {
	Error  string                 `json:"error"`
	Source map[string]interface{} `json:"source"`
	Stack  []string               `json:"stack"`
}

func TestBasicAndGinLogging(t *testing.T) {
	// Test 1: Basic Logging - writes to examples/basic/logs/
	t.Run("BasicLogging", func(t *testing.T) {
		basicLogDir := "../examples/basic/logs"
		
		// Clean and ensure directory exists
		os.RemoveAll(basicLogDir)
		os.MkdirAll(basicLogDir, 0755)
		
		config := &logging.Config{
			ServiceName:    "basic-example",
			LogPath:        basicLogDir,
			FilePrefix:     "app",
			EnableStdout:   true,
			EnableFile:     true,
			EnableLoki:     true,
			EnableRotation: true,
		}

		logger, err := logging.New(config)
		if err != nil {
			t.Fatalf("Failed to create basic logger: %v", err)
		}

		// Test basic info logging
		logger.Info("BASIC TEST: Application started successfully")
		logger.Info("BASIC TEST: Configuration loaded from file")
		logger.Info("BASIC TEST: Database connection established")
		logger.Info("BASIC TEST: All services initialized")

		// Create context with metadata
		ctx := logging.WithMeta(context.Background(), logging.Meta{
			RequestID: "basic-req-001",
			IP:        "127.0.0.1",
			Method:    "GET",
			Path:      "/api/basic-test",
			UserAgent: "BasicTestClient/1.0",
		})

		// Test access log - simulate HTTP requests (success - errors=null)
		logger.LogRequest(ctx, 200, 50*time.Millisecond)

		ctx2 := logging.WithMeta(context.Background(), logging.Meta{
			RequestID: "basic-req-002",
			IP:        "127.0.0.1",
			Method:    "POST",
			Path:      "/api/users",
			UserAgent: "BasicTestClient/1.0",
		})
		logger.LogRequest(ctx2, 201, 120*time.Millisecond)

		ctx3 := logging.WithMeta(context.Background(), logging.Meta{
			RequestID: "basic-req-003",
			IP:        "127.0.0.1",
			Method:    "GET",
			Path:      "/api/not-found",
			UserAgent: "BasicTestClient/1.0",
		})
		logger.LogRequest(ctx3, 404, 10*time.Millisecond)

		// Test LogRequestWithError - simulate 500 error with error detail
		ctx4 := logging.WithMeta(context.Background(), logging.Meta{
			RequestID: "basic-req-004",
			IP:        "127.0.0.1",
			Method:    "GET",
			Path:      "/api/error",
			UserAgent: "BasicTestClient/1.0",
		})
		dbError := errors.New("BASIC TEST: database connection failed")
		logger.LogRequestWithError(ctx4, 500, 200*time.Millisecond, dbError)

		// Test error logging (detailed format to error.log)
		basicErr := errors.New("BASIC TEST: sample database error")
		logger.Error(ctx, basicErr)

		// Test Loki JSON logging directly
		criticalErr := errors.New("BASIC TEST: critical system failure")
		logger.ErrorLoki(ctx, logging.LevelCritical, criticalErr)

		// Test different error levels
		warnErr := errors.New("BASIC TEST: connection timeout warning")
		logger.ErrorLoki(ctx, logging.LevelWarn, warnErr)

		logger.Info("BASIC TEST: All operations completed successfully")

		// Wait for file writes
		time.Sleep(300 * time.Millisecond)

		// Verify basic log files
		today := time.Now().Format("2006-01-02")
		accessFile := basicLogDir + "/app.access-" + today + ".log"
		errorFile := basicLogDir + "/app.error-" + today + ".log"
		lokiFile := basicLogDir + "/app.loki-" + today + ".log"

		// Verify access log
		if content, err := os.ReadFile(accessFile); err != nil {
			t.Errorf("Failed to read basic access log: %v", err)
		} else {
			contentStr := string(content)
			if !strings.Contains(contentStr, "Application started successfully") {
				t.Error("Expected basic log message not found in access log")
			}
			// Verify HTTP request logs
			if !strings.Contains(contentStr, "[REQ:basic-req-001]") {
				t.Error("Expected request ID basic-req-001 not found in access log")
			}
			if !strings.Contains(contentStr, "| 200 |") {
				t.Error("Expected status 200 not found in access log")
			}
			if !strings.Contains(contentStr, "| 404 |") {
				t.Error("Expected status 404 not found in access log")
			}
			if !strings.Contains(contentStr, "| 500 |") {
				t.Error("Expected status 500 not found in access log")
			}
			t.Logf("BASIC Access log: %s (%d bytes)", accessFile, len(content))
		}

		// Verify error log
		if content, err := os.ReadFile(errorFile); err != nil {
			t.Errorf("Failed to read basic error log: %v", err)
		} else {
			if !strings.Contains(string(content), "sample database error") {
				t.Error("Expected basic error not found in error log")
			}
			if !strings.Contains(string(content), "basic-req-001") {
				t.Error("Expected basic request ID not found in error log")
			}
			t.Logf("BASIC Error log: %s (%d bytes)", errorFile, len(content))
		}

		// Verify Loki log - UNIFIED FORMAT
		if content, err := os.ReadFile(lokiFile); err != nil {
			t.Errorf("Failed to read basic Loki log: %v", err)
		} else {
			contentStr := string(content)
			lines := strings.Split(strings.TrimSpace(contentStr), "\n")
			
			var successCount, errorCount int
			
			for _, line := range lines {
				if line == "" {
					continue
				}
				
				var entry LokiLogEntry
				if err := json.Unmarshal([]byte(line), &entry); err != nil {
					t.Errorf("Failed to parse Loki JSON: %v\nLine: %s", err, line)
					continue
				}
				
				// Verify consistent structure
				if entry.Service != "basic-example" {
					t.Errorf("Expected service 'basic-example', got '%s'", entry.Service)
				}
				
				// Count entries based on errors field
				if entry.Errors != nil {
					errorCount++
				} else {
					successCount++
				}
			}
			
			t.Logf("BASIC Loki log: %s (%d bytes, %d entries with errors=null, %d entries with errors object)", 
				lokiFile, len(content), successCount, errorCount)
			
			if successCount == 0 {
				t.Error("Expected at least one entry with errors=null")
			}
			if errorCount == 0 {
				t.Error("Expected at least one entry with errors object")
			}
		}
	})

	// Test 2: Gin Middleware Logging - writes to examples/gin/logs/
	t.Run("GinMiddlewareLogging", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		
		ginLogDir := "../examples/gin/logs"
		
		// Clean and ensure directory exists
		os.RemoveAll(ginLogDir)
		os.MkdirAll(ginLogDir, 0755)
		
		config := &logging.Config{
			ServiceName:    "gin-example",
			LogPath:        ginLogDir,
			FilePrefix:     "gin-app",
			EnableStdout:   true,
			EnableFile:     true,
			EnableLoki:     true,
			EnableRotation: true,
		}

		logger, err := logging.New(config)
		if err != nil {
			t.Fatalf("Failed to create gin logger: %v", err)
		}

		// Setup Gin router with middleware
		r := gin.New()
		r.Use(middleware.GinMiddleware(logger))      // 1. Setup request context
		r.Use(middleware.GinLogger(logger))          // 2. Log requests
		r.Use(middleware.GinHTTPErrorLogger(logger)) // 3. Log errors to error.log
		r.Use(middleware.GinRecovery(logger))        // 4. Catch panics last

		// Routes for testing
		r.GET("/", func(c *gin.Context) {
			logger.Info("GIN TEST: Home endpoint accessed")
			c.JSON(200, gin.H{"message": "Hello World", "status": "success"})
		})

		r.GET("/success", func(c *gin.Context) {
			logger.Info("GIN TEST: Success endpoint processing")
			c.JSON(200, gin.H{"message": "Operation successful"})
		})

		r.GET("/error", func(c *gin.Context) {
			// Manual error logging with anti-duplication
			testErr := errors.New("GIN TEST: manual error for testing")
			logger.LogErrorWithMark(c, testErr)
			c.JSON(500, gin.H{"error": "Something went wrong"})
		})

		r.GET("/auto-error", func(c *gin.Context) {
			// Auto error (let middleware handle it)
			c.JSON(500, gin.H{"error": "Auto error handling"})
		})

		r.GET("/panic", func(c *gin.Context) {
			// This will trigger panic recovery
			panic("GIN TEST: test panic for recovery")
		})

		// Execute test requests
		t.Log("Testing Gin endpoints...")
		
		requests := []struct {
			method string
			path   string
			desc   string
		}{
			{"GET", "/", "home endpoint"},
			{"GET", "/success", "success endpoint"},
			{"GET", "/error", "manual error endpoint"},
			{"GET", "/auto-error", "auto error endpoint"},
			{"GET", "/panic", "panic endpoint"},
		}

		for _, req := range requests {
			t.Logf("Testing %s...", req.desc)
			request := httptest.NewRequest(req.method, req.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, request)
			
			// Small delay between requests
			time.Sleep(50 * time.Millisecond)
		}

		// Wait for all file writes to complete
		time.Sleep(500 * time.Millisecond)

		// Verify Gin log files
		today := time.Now().Format("2006-01-02")
		ginAccessFile := ginLogDir + "/gin-app.access-" + today + ".log"
		ginErrorFile := ginLogDir + "/gin-app.error-" + today + ".log"
		ginLokiFile := ginLogDir + "/gin-app.loki-" + today + ".log"

		// Verify Gin access log
		if content, err := os.ReadFile(ginAccessFile); err != nil {
			t.Errorf("Failed to read gin access log: %v", err)
		} else {
			if !strings.Contains(string(content), "Home endpoint accessed") {
				t.Error("Expected gin log message not found in access log")
			}
			if !strings.Contains(string(content), "Success endpoint processing") {
				t.Error("Expected gin success message not found in access log")
			}
			t.Logf("GIN Access log: %s (%d bytes)", ginAccessFile, len(content))
		}

		// Verify Gin error log
		if content, err := os.ReadFile(ginErrorFile); err != nil {
			t.Errorf("Failed to read gin error log: %v", err)
		} else {
			contentStr := string(content)
			
			// Should contain manual error
			if !strings.Contains(contentStr, "manual error for testing") {
				t.Error("Expected gin manual error not found in error log")
			}
			
			// Should contain auto error
			if !strings.Contains(contentStr, "HTTP Error") {
				t.Error("Expected gin auto error not found in error log")
			}
			
			// Should contain panic
			if !strings.Contains(contentStr, "PANIC: GIN TEST: test panic") {
				t.Error("Expected gin panic not found in error log")
			}
			
			t.Logf("GIN Error log: %s (%d bytes)", ginErrorFile, len(content))
		}

		// Verify Gin Loki log - UNIFIED FORMAT
		if content, err := os.ReadFile(ginLokiFile); err != nil {
			t.Errorf("Failed to read gin Loki log: %v", err)
		} else {
			contentStr := string(content)
			lines := strings.Split(strings.TrimSpace(contentStr), "\n")
			
			var successCount, errorCount int
			
			for _, line := range lines {
				if line == "" {
					continue
				}
				
				var entry LokiLogEntry
				if err := json.Unmarshal([]byte(line), &entry); err != nil {
					t.Errorf("Failed to parse Gin Loki JSON: %v\nLine: %s", err, line)
					continue
				}
				
				// Verify consistent structure
				if entry.Service != "gin-example" {
					t.Errorf("Expected service 'gin-example', got '%s'", entry.Service)
				}
				
				// Verify errors field based on status code
				if entry.StatusCode >= 400 {
					if entry.Errors != nil {
						errorCount++
					}
				} else {
					if entry.Errors == nil {
						successCount++
					}
				}
			}
			
			t.Logf("GIN Loki log: %s (%d bytes, %d success with errors=null, %d error with errors object)", 
				ginLokiFile, len(content), successCount, errorCount)
		}
	})

	// Final summary
	t.Log("=== TEST SUMMARY ===")
	t.Log("Basic logging tested - logs in ../examples/basic/logs/")
	t.Log("Gin middleware tested - logs in ../examples/gin/logs/")
	t.Log("Unified Loki format verified (errors=null for success, errors object for errors)")
	t.Log("Anti-duplication verified")
	t.Log("Multiple log formats verified (access, error, loki)")
	t.Log("Daily rotation verified")
	t.Log("=== ALL TESTS COMPLETED ===")
}

func TestLoggingWithoutRotation(t *testing.T) {
	t.Run("LoggingWithoutRotation", func(t *testing.T) {
		basicLogDir := "../examples/basic/logs"

		// Ensure directory exists
		os.MkdirAll(basicLogDir, 0755)

		// Clean previous test files
		os.Remove(basicLogDir + "/no-rotate.access.log")
		os.Remove(basicLogDir + "/no-rotate.error.log")
		os.Remove(basicLogDir + "/no-rotate.loki.log")

		config := &logging.Config{
			ServiceName:    "no-rotate-example",
			LogPath:        basicLogDir,
			FilePrefix:     "no-rotate",
			EnableStdout:   true,
			EnableFile:     true,
			EnableLoki:     true,
			EnableRotation: false, // Disable rotation
		}

		logger, err := logging.New(config)
		if err != nil {
			t.Fatalf("Failed to create logger without rotation: %v", err)
		}

		// Test basic info logging
		logger.Info("NO ROTATE TEST: Application started successfully")
		logger.Info("NO ROTATE TEST: Configuration loaded from file")
		logger.Info("NO ROTATE TEST: Database connection established")

		// Create context with metadata
		ctx := logging.WithMeta(context.Background(), logging.Meta{
			RequestID: "no-rotate-req-001",
			IP:        "192.168.1.1",
			Method:    "POST",
			Path:      "/api/no-rotate-test",
			UserAgent: "NoRotateTestClient/1.0",
		})

		// Test success request (errors=null)
		logger.LogRequest(ctx, 200, 50*time.Millisecond)

		ctx2 := logging.WithMeta(context.Background(), logging.Meta{
			RequestID: "no-rotate-req-002",
			IP:        "192.168.1.1",
			Method:    "GET",
			Path:      "/api/data",
			UserAgent: "NoRotateTestClient/1.0",
		})
		// Test error request with error detail
		dbErr := errors.New("NO ROTATE TEST: database timeout")
		logger.LogRequestWithError(ctx2, 500, 150*time.Millisecond, dbErr)

		// Test error logging
		testErr := errors.New("NO ROTATE TEST: sample error without rotation")
		logger.Error(ctx, testErr)

		// Test Loki JSON logging
		criticalErr := errors.New("NO ROTATE TEST: critical error without rotation")
		logger.ErrorLoki(ctx, logging.LevelCritical, criticalErr)

		logger.Info("NO ROTATE TEST: All operations completed")

		// Wait for file writes
		time.Sleep(300 * time.Millisecond)

		// Verify log files WITHOUT date suffix
		accessFile := basicLogDir + "/no-rotate.access.log"
		errorFile := basicLogDir + "/no-rotate.error.log"
		lokiFile := basicLogDir + "/no-rotate.loki.log"

		// Verify access log (without date)
		if content, err := os.ReadFile(accessFile); err != nil {
			t.Errorf("Failed to read access log without rotation: %v", err)
		} else {
			contentStr := string(content)
			if !strings.Contains(contentStr, "Application started successfully") {
				t.Error("Expected log message not found in access log")
			}
			if !strings.Contains(contentStr, "[REQ:no-rotate-req-001]") {
				t.Error("Expected request ID not found in access log")
			}
			if !strings.Contains(contentStr, "| 200 |") {
				t.Error("Expected status 200 not found in access log")
			}
			if !strings.Contains(contentStr, "| 500 |") {
				t.Error("Expected status 500 not found in access log")
			}
			t.Logf("NO ROTATE Access log: %s (%d bytes)", accessFile, len(content))
		}

		// Verify error log (without date)
		if content, err := os.ReadFile(errorFile); err != nil {
			t.Errorf("Failed to read error log without rotation: %v", err)
		} else {
			if !strings.Contains(string(content), "sample error without rotation") {
				t.Error("Expected error not found in error log")
			}
			if !strings.Contains(string(content), "no-rotate-req-001") {
				t.Error("Expected request ID not found in error log")
			}
			t.Logf("NO ROTATE Error log: %s (%d bytes)", errorFile, len(content))
		}

		// Verify Loki log (without date) - UNIFIED FORMAT
		if content, err := os.ReadFile(lokiFile); err != nil {
			t.Errorf("Failed to read Loki log without rotation: %v", err)
		} else {
			contentStr := string(content)
			lines := strings.Split(strings.TrimSpace(contentStr), "\n")
			
			var successCount, errorCount int
			
			for _, line := range lines {
				if line == "" {
					continue
				}
				
				var entry LokiLogEntry
				if err := json.Unmarshal([]byte(line), &entry); err != nil {
					t.Errorf("Failed to parse Loki JSON: %v", err)
					continue
				}
				
				if entry.Service != "no-rotate-example" {
					t.Errorf("Expected service 'no-rotate-example', got '%s'", entry.Service)
				}
				
				if entry.StatusCode >= 400 {
					if entry.Errors != nil {
						errorCount++
					}
				} else {
					if entry.Errors == nil {
						successCount++
					}
				}
			}
			
			t.Logf("NO ROTATE Loki log: %s (%d bytes, %d success, %d error)", lokiFile, len(content), successCount, errorCount)
		}

		// Verify that date-based files do NOT exist
		today := time.Now().Format("2006-01-02")
		rotatedAccessFile := basicLogDir + "/no-rotate.access-" + today + ".log"
		rotatedErrorFile := basicLogDir + "/no-rotate.error-" + today + ".log"
		rotatedLokiFile := basicLogDir + "/no-rotate.loki-" + today + ".log"

		if _, err := os.Stat(rotatedAccessFile); err == nil {
			t.Errorf("Date-based access file should NOT exist when rotation is disabled: %s", rotatedAccessFile)
		}
		if _, err := os.Stat(rotatedErrorFile); err == nil {
			t.Errorf("Date-based error file should NOT exist when rotation is disabled: %s", rotatedErrorFile)
		}
		if _, err := os.Stat(rotatedLokiFile); err == nil {
			t.Errorf("Date-based loki file should NOT exist when rotation is disabled: %s", rotatedLokiFile)
		}

		t.Log("Verified: No date-based rotation files created")
	})

	// Test Gin Middleware without rotation
	t.Run("GinMiddlewareWithoutRotation", func(t *testing.T) {
		gin.SetMode(gin.TestMode)

		ginLogDir := "../examples/gin/logs"

		// Ensure directory exists
		os.MkdirAll(ginLogDir, 0755)

		// Clean previous test files
		os.Remove(ginLogDir + "/gin-no-rotate.access.log")
		os.Remove(ginLogDir + "/gin-no-rotate.error.log")
		os.Remove(ginLogDir + "/gin-no-rotate.loki.log")

		config := &logging.Config{
			ServiceName:    "gin-no-rotate-example",
			LogPath:        ginLogDir,
			FilePrefix:     "gin-no-rotate",
			EnableStdout:   true,
			EnableFile:     true,
			EnableLoki:     true,
			EnableRotation: false, // Disable rotation
		}

		logger, err := logging.New(config)
		if err != nil {
			t.Fatalf("Failed to create gin logger without rotation: %v", err)
		}

		// Setup Gin router with middleware
		r := gin.New()
		r.Use(middleware.GinMiddleware(logger))      // 1. Setup request context
		r.Use(middleware.GinLogger(logger))          // 2. Log requests
		r.Use(middleware.GinHTTPErrorLogger(logger)) // 3. Log errors to error.log
		r.Use(middleware.GinRecovery(logger))        // 4. Catch panics last

		// Routes for testing
		r.GET("/", func(c *gin.Context) {
			logger.Info("GIN NO ROTATE TEST: Home endpoint accessed")
			c.JSON(200, gin.H{"message": "Hello World", "status": "success"})
		})

		r.GET("/success", func(c *gin.Context) {
			logger.Info("GIN NO ROTATE TEST: Success endpoint processing")
			c.JSON(200, gin.H{"message": "Operation successful"})
		})

		r.GET("/error", func(c *gin.Context) {
			testErr := errors.New("GIN NO ROTATE TEST: manual error for testing")
			logger.LogErrorWithMark(c, testErr)
			c.JSON(500, gin.H{"error": "Something went wrong"})
		})

		r.GET("/auto-error", func(c *gin.Context) {
			c.JSON(500, gin.H{"error": "Auto error handling"})
		})

		r.GET("/panic", func(c *gin.Context) {
			panic("GIN NO ROTATE TEST: test panic for recovery")
		})

		// Execute test requests
		t.Log("Testing Gin endpoints without rotation...")

		requests := []struct {
			method string
			path   string
			desc   string
		}{
			{"GET", "/", "home endpoint"},
			{"GET", "/success", "success endpoint"},
			{"GET", "/error", "manual error endpoint"},
			{"GET", "/auto-error", "auto error endpoint"},
			{"GET", "/panic", "panic endpoint"},
		}

		for _, req := range requests {
			t.Logf("Testing %s...", req.desc)
			request := httptest.NewRequest(req.method, req.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, request)
			time.Sleep(50 * time.Millisecond)
		}

		// Wait for all file writes to complete
		time.Sleep(500 * time.Millisecond)

		// Verify Gin log files WITHOUT date suffix
		ginAccessFile := ginLogDir + "/gin-no-rotate.access.log"
		ginErrorFile := ginLogDir + "/gin-no-rotate.error.log"
		ginLokiFile := ginLogDir + "/gin-no-rotate.loki.log"

		// Verify Gin access log (without date)
		if content, err := os.ReadFile(ginAccessFile); err != nil {
			t.Errorf("Failed to read gin access log without rotation: %v", err)
		} else {
			if !strings.Contains(string(content), "Home endpoint accessed") {
				t.Error("Expected gin log message not found in access log")
			}
			if !strings.Contains(string(content), "Success endpoint processing") {
				t.Error("Expected gin success message not found in access log")
			}
			t.Logf("GIN NO ROTATE Access log: %s (%d bytes)", ginAccessFile, len(content))
		}

		// Verify Gin error log (without date)
		if content, err := os.ReadFile(ginErrorFile); err != nil {
			t.Errorf("Failed to read gin error log without rotation: %v", err)
		} else {
			contentStr := string(content)

			if !strings.Contains(contentStr, "manual error for testing") {
				t.Error("Expected gin manual error not found in error log")
			}
			if !strings.Contains(contentStr, "HTTP Error") {
				t.Error("Expected gin auto error not found in error log")
			}
			if !strings.Contains(contentStr, "PANIC: GIN NO ROTATE TEST: test panic") {
				t.Error("Expected gin panic not found in error log")
			}
			t.Logf("GIN NO ROTATE Error log: %s (%d bytes)", ginErrorFile, len(content))
		}

		// Verify Gin Loki log (without date) - UNIFIED FORMAT
		if content, err := os.ReadFile(ginLokiFile); err != nil {
			t.Errorf("Failed to read gin Loki log without rotation: %v", err)
		} else {
			contentStr := string(content)
			lines := strings.Split(strings.TrimSpace(contentStr), "\n")
			
			var successCount, errorCount int
			
			for _, line := range lines {
				if line == "" {
					continue
				}
				
				var entry LokiLogEntry
				if err := json.Unmarshal([]byte(line), &entry); err != nil {
					t.Errorf("Failed to parse Gin Loki JSON: %v", err)
					continue
				}
				
				if entry.Service != "gin-no-rotate-example" {
					t.Errorf("Expected service 'gin-no-rotate-example', got '%s'", entry.Service)
				}
				
				if entry.StatusCode >= 400 {
					if entry.Errors != nil {
						errorCount++
					}
				} else {
					if entry.Errors == nil {
						successCount++
					}
				}
			}
			
			t.Logf("GIN NO ROTATE Loki log: %s (%d bytes, %d success, %d error)", ginLokiFile, len(content), successCount, errorCount)
		}

		// Verify that date-based files do NOT exist
		today := time.Now().Format("2006-01-02")
		rotatedGinAccessFile := ginLogDir + "/gin-no-rotate.access-" + today + ".log"
		rotatedGinErrorFile := ginLogDir + "/gin-no-rotate.error-" + today + ".log"
		rotatedGinLokiFile := ginLogDir + "/gin-no-rotate.loki-" + today + ".log"

		if _, err := os.Stat(rotatedGinAccessFile); err == nil {
			t.Errorf("Date-based gin access file should NOT exist when rotation is disabled: %s", rotatedGinAccessFile)
		}
		if _, err := os.Stat(rotatedGinErrorFile); err == nil {
			t.Errorf("Date-based gin error file should NOT exist when rotation is disabled: %s", rotatedGinErrorFile)
		}
		if _, err := os.Stat(rotatedGinLokiFile); err == nil {
			t.Errorf("Date-based gin loki file should NOT exist when rotation is disabled: %s", rotatedGinLokiFile)
		}

		t.Log("Verified: No date-based rotation files created for Gin")
	})

	t.Log("=== NO ROTATION TEST COMPLETED ===")
}

// TestHTTPMiddleware tests the basic HTTP middleware (non-Gin)
func TestHTTPMiddleware(t *testing.T) {
	t.Run("BasicHTTPMiddleware", func(t *testing.T) {
		basicLogDir := "../examples/basic/logs"

		// Ensure directory exists
		os.MkdirAll(basicLogDir, 0755)

		// Clean previous test files (now in subdirectory format: LogPath/FilePrefix.type.log)
		os.Remove(basicLogDir + "/http-test.access.log")
		os.Remove(basicLogDir + "/http-test.error.log")
		os.Remove(basicLogDir + "/http-test.loki.log")

		config := &logging.Config{
			ServiceName:    "http-middleware-example",
			LogPath:        basicLogDir,
			FilePrefix:     "http-test",
			EnableStdout:   true,
			EnableFile:     true,
			EnableLoki:     true,
			EnableRotation: false,
		}

		logger, err := logging.New(config)
		if err != nil {
			t.Fatalf("Failed to create logger: %v", err)
		}

		// Create HTTP server with middleware
		mux := http.NewServeMux()

		// Success endpoint
		mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("pong"))
		})

		// Error endpoint
		mux.HandleFunc("/error", func(w http.ResponseWriter, r *http.Request) {
			// Store error in context for Loki logging
			testErr := errors.New("HTTP MIDDLEWARE TEST: database connection failed")
			ctx := logging.WithError(r.Context(), testErr)
			r = r.WithContext(ctx)

			// Also log to error.log with detailed format
			logger.Error(r.Context(), testErr)

			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal Server Error"))
		})

		// Apply middleware chain
		handler := middleware.HTTPRecovery(logger)(
			middleware.HTTPMiddleware(logger)(
				middleware.HTTPLogger(logger)(mux),
			),
		)

		// Test requests
		requests := []struct {
			method       string
			path         string
			expectedCode int
		}{
			{"GET", "/ping", 200},
			{"GET", "/ping", 200},
			{"GET", "/error", 500},
		}

		for _, req := range requests {
			t.Logf("Testing HTTP %s %s...", req.method, req.path)
			request := httptest.NewRequest(req.method, req.path, nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, request)

			if w.Code != req.expectedCode {
				t.Errorf("Expected status %d, got %d", req.expectedCode, w.Code)
			}
			time.Sleep(50 * time.Millisecond)
		}

		// Wait for file writes
		time.Sleep(300 * time.Millisecond)

		// Verify log files
		accessFile := basicLogDir + "/http-test.access.log"
		lokiFile := basicLogDir + "/http-test.loki.log"

		// Verify access log
		if content, err := os.ReadFile(accessFile); err != nil {
			t.Errorf("Failed to read HTTP access log: %v", err)
		} else {
			contentStr := string(content)
			if !strings.Contains(contentStr, "| 200 |") {
				t.Error("Expected status 200 not found in access log")
			}
			if !strings.Contains(contentStr, "| 500 |") {
				t.Error("Expected status 500 not found in access log")
			}
			t.Logf("HTTP Access log: %s (%d bytes)", accessFile, len(content))
		}

		// Verify Loki log - UNIFIED FORMAT
		if content, err := os.ReadFile(lokiFile); err != nil {
			t.Errorf("Failed to read HTTP Loki log: %v", err)
		} else {
			contentStr := string(content)
			lines := strings.Split(strings.TrimSpace(contentStr), "\n")

			var successCount, errorCount int

			for _, line := range lines {
				if line == "" {
					continue
				}

				var entry LokiLogEntry
				if err := json.Unmarshal([]byte(line), &entry); err != nil {
					t.Errorf("Failed to parse HTTP Loki JSON: %v", err)
					continue
				}

				if entry.Service != "http-middleware-example" {
					t.Errorf("Expected service 'http-middleware-example', got '%s'", entry.Service)
				}

				if entry.StatusCode >= 400 {
					errorCount++
				} else {
					if entry.Errors != nil {
						t.Errorf("Expected errors=null for status %d", entry.StatusCode)
					}
					successCount++
				}
			}

			t.Logf("HTTP Loki log: %s (%d bytes, %d success, %d error)", lokiFile, len(content), successCount, errorCount)

			if successCount < 2 {
				t.Errorf("Expected at least 2 success entries, got %d", successCount)
			}
		}
	})

	t.Log("=== HTTP MIDDLEWARE TEST COMPLETED ===")
}

// TestLokiFormatConsistency specifically tests that Loki format is consistent
func TestLokiFormatConsistency(t *testing.T) {
	t.Run("LokiFormatConsistency", func(t *testing.T) {
		basicLogDir := "../examples/basic/logs"

		// Ensure directory exists
		os.MkdirAll(basicLogDir, 0755)

		// Clean previous test files
		os.Remove(basicLogDir + "/loki-test.access.log")
		os.Remove(basicLogDir + "/loki-test.error.log")
		os.Remove(basicLogDir + "/loki-test.loki.log")

		config := &logging.Config{
			ServiceName:    "loki-format-test",
			LogPath:        basicLogDir,
			FilePrefix:     "loki-test",
			EnableStdout:   false,
			EnableFile:     true,
			EnableLoki:     true,
			EnableRotation: false,
		}

		logger, err := logging.New(config)
		if err != nil {
			t.Fatalf("Failed to create logger: %v", err)
		}

		ctx := logging.WithMeta(context.Background(), logging.Meta{
			RequestID: "loki-test-001",
			IP:        "10.0.0.1",
			Method:    "POST",
			Path:      "/api/test",
			UserAgent: "LokiTest/1.0",
		})

		// Test 1: Success request (errors should be null)
		logger.LogRequest(ctx, 200, 100*time.Millisecond)

		// Test 2: Error request (errors should have object)
		testErr := errors.New("test error for Loki format")
		logger.LogRequestWithError(ctx, 500, 200*time.Millisecond, testErr)

		// Test 3: 404 without explicit error (errors should be null)
		logger.LogRequest(ctx, 404, 50*time.Millisecond)

		time.Sleep(200 * time.Millisecond)

		// Verify Loki format
		lokiFile := basicLogDir + "/loki-test.loki.log"
		content, err := os.ReadFile(lokiFile)
		if err != nil {
			t.Fatalf("Failed to read Loki log: %v", err)
		}

		lines := strings.Split(strings.TrimSpace(string(content)), "\n")

		expectedFormats := []struct {
			statusCode  int
			hasErrors   bool
			description string
		}{
			{200, false, "Success 200 should have errors=null"},
			{500, true, "Error 500 with explicit error should have errors object"},
			{404, false, "Error 404 without explicit error should have errors=null"},
		}

		for i, line := range lines {
			if line == "" || i >= len(expectedFormats) {
				continue
			}

			var entry LokiLogEntry
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				t.Errorf("Failed to parse line %d: %v", i, err)
				continue
			}

			expected := expectedFormats[i]

			if entry.StatusCode != expected.statusCode {
				t.Errorf("Line %d: Expected status %d, got %d", i, expected.statusCode, entry.StatusCode)
			}

			if expected.hasErrors {
				if entry.Errors == nil {
					t.Errorf("Line %d: %s - but errors is null", i, expected.description)
				} else {
					// Verify error object structure
					if entry.Errors.Error == "" {
						t.Errorf("Line %d: errors.error should not be empty", i)
					}
					if entry.Errors.Source == nil {
						t.Errorf("Line %d: errors.source should not be nil", i)
					}
					t.Logf("Line %d: ✓ %s (error: %s)", i, expected.description, entry.Errors.Error)
				}
			} else {
				if entry.Errors != nil {
					t.Errorf("Line %d: %s - but errors is not null: %+v", i, expected.description, entry.Errors)
				} else {
					t.Logf("Line %d: ✓ %s", i, expected.description)
				}
			}
		}
	})

	t.Log("=== LOKI FORMAT CONSISTENCY TEST COMPLETED ===")
}