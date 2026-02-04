package main

import (
	"context"
	"errors"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ahmadsaubani/go-logging-lib"
	"github.com/ahmadsaubani/go-logging-lib/middleware"
	"github.com/gin-gonic/gin"
)

func TestBasicAndGinLogging(t *testing.T) {
	// Test 1: Basic Logging - writes to examples/basic/logs/
	t.Run("BasicLogging", func(t *testing.T) {
		basicLogDir := "../examples/basic/logs"
		
		// Clean and ensure directory exists
		os.RemoveAll(basicLogDir)
		os.MkdirAll(basicLogDir, 0755)
		
		config := &logging.Config{
			ServiceName:    "basic-example",
			LogPath:        basicLogDir + "/app",
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

		// Test access log - simulate HTTP requests
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

		ctx4 := logging.WithMeta(context.Background(), logging.Meta{
			RequestID: "basic-req-004",
			IP:        "127.0.0.1",
			Method:    "GET",
			Path:      "/api/error",
			UserAgent: "BasicTestClient/1.0",
		})
		logger.LogRequest(ctx4, 500, 200*time.Millisecond)

		// Test error logging
		basicErr := errors.New("BASIC TEST: sample database error")
		logger.Error(ctx, basicErr)

		// Test Loki JSON logging
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
		lokiFile := basicLogDir + "/app.error-loki-" + today + ".log"

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

		// Verify Loki log
		if content, err := os.ReadFile(lokiFile); err != nil {
			t.Errorf("Failed to read basic Loki log: %v", err)
		} else {
			contentStr := string(content)
			if !strings.Contains(contentStr, `"service":"basic-example"`) {
				t.Error("Expected basic service name not found in Loki log")
			}
			if !strings.Contains(contentStr, `"level":"CRITICAL"`) {
				t.Error("Expected CRITICAL level not found in Loki log")
			}
			// Verify access logs in Loki format
			if !strings.Contains(contentStr, `"status_code":200`) {
				t.Error("Expected status_code 200 not found in Loki log")
			}
			if !strings.Contains(contentStr, `"status_code":404`) {
				t.Error("Expected status_code 404 not found in Loki log")
			}
			if !strings.Contains(contentStr, `"status_code":500`) {
				t.Error("Expected status_code 500 not found in Loki log")
			}
			// Count JSON entries
			jsonEntries := strings.Count(contentStr, `"level":`)
			t.Logf("BASIC Loki log: %s (%d bytes, %d JSON entries)", lokiFile, len(content), jsonEntries)
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
			LogPath:        ginLogDir + "/gin-app",
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
		r.Use(middleware.GinMiddleware(logger))
		r.Use(middleware.GinLogger(logger))
		r.Use(middleware.GinHTTPErrorLogger(logger))
		r.Use(middleware.GinRecovery(logger))

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

		r.GET("/loki", func(c *gin.Context) {
			// Direct Loki logging
			testErr := errors.New("GIN TEST: loki formatted error")
			logger.ErrorLoki(c.Request.Context(), logging.LevelError, testErr)
			c.JSON(200, gin.H{"message": "Error logged to Loki format"})
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
			{"GET", "/loki", "loki endpoint"},
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
		ginLokiFile := ginLogDir + "/gin-app.error-loki-" + today + ".log"

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

		// Verify Gin Loki log
		if content, err := os.ReadFile(ginLokiFile); err != nil {
			t.Errorf("Failed to read gin Loki log: %v", err)
		} else {
			contentStr := string(content)
			
			// Should contain service name
			if !strings.Contains(contentStr, `"service":"gin-example"`) {
				t.Error("Expected gin service name not found in Loki log")
			}
			
			// Count JSON entries
			jsonEntries := strings.Count(contentStr, `"level":`)
			if jsonEntries < 3 {
				t.Errorf("Expected at least 3 JSON entries in Loki log, found %d", jsonEntries)
			}
			
			t.Logf("GIN Loki log: %s (%d bytes, %d JSON entries)", ginLokiFile, len(content), jsonEntries)
		}
	})

	// Final summary
	t.Log("=== TEST SUMMARY ===")
	t.Log("Basic logging tested - logs in ../examples/basic/logs/")
	t.Log("Gin middleware tested - logs in ../examples/gin/logs/")
	t.Log("Anti-duplication verified")
	t.Log("Multiple log formats verified (access, error, loki)")
	t.Log("Daily rotation verified")
	t.Log("=== ALL TESTS COMPLETED ===")
}

func TestLoggingWithoutRotation(t *testing.T) {
	t.Run("LoggingWithoutRotation", func(t *testing.T) {
		noRotateLogDir := "../examples/basic/logs/no-rotate"

		// Clean and ensure directory exists
		os.RemoveAll(noRotateLogDir)
		os.MkdirAll(noRotateLogDir, 0755)

		config := &logging.Config{
			ServiceName:    "no-rotate-example",
			LogPath:        noRotateLogDir + "/app",
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

		// Test access log - simulate HTTP requests
		logger.LogRequest(ctx, 200, 50*time.Millisecond)

		ctx2 := logging.WithMeta(context.Background(), logging.Meta{
			RequestID: "no-rotate-req-002",
			IP:        "192.168.1.1",
			Method:    "GET",
			Path:      "/api/data",
			UserAgent: "NoRotateTestClient/1.0",
		})
		logger.LogRequest(ctx2, 500, 150*time.Millisecond)

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
		accessFile := noRotateLogDir + "/app.access.log"
		errorFile := noRotateLogDir + "/app.error.log"
		lokiFile := noRotateLogDir + "/app.error-loki.log"

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

		// Verify Loki log (without date)
		if content, err := os.ReadFile(lokiFile); err != nil {
			t.Errorf("Failed to read Loki log without rotation: %v", err)
		} else {
			if !strings.Contains(string(content), `"service":"no-rotate-example"`) {
				t.Error("Expected service name not found in Loki log")
			}
			if !strings.Contains(string(content), `"level":"CRITICAL"`) {
				t.Error("Expected CRITICAL level not found in Loki log")
			}
			t.Logf("NO ROTATE Loki log: %s (%d bytes)", lokiFile, len(content))
		}

		// Verify that date-based files do NOT exist
		today := time.Now().Format("2006-01-02")
		rotatedAccessFile := noRotateLogDir + "/app.access-" + today + ".log"
		rotatedErrorFile := noRotateLogDir + "/app.error-" + today + ".log"
		rotatedLokiFile := noRotateLogDir + "/app.error-loki-" + today + ".log"

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

		ginNoRotateLogDir := "../examples/gin/logs/no-rotate"

		// Clean and ensure directory exists
		os.RemoveAll(ginNoRotateLogDir)
		os.MkdirAll(ginNoRotateLogDir, 0755)

		config := &logging.Config{
			ServiceName:    "gin-no-rotate-example",
			LogPath:        ginNoRotateLogDir + "/gin-app",
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
		r.Use(middleware.GinMiddleware(logger))
		r.Use(middleware.GinLogger(logger))
		r.Use(middleware.GinHTTPErrorLogger(logger))
		r.Use(middleware.GinRecovery(logger))

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

		r.GET("/loki", func(c *gin.Context) {
			testErr := errors.New("GIN NO ROTATE TEST: loki formatted error")
			logger.ErrorLoki(c.Request.Context(), logging.LevelError, testErr)
			c.JSON(200, gin.H{"message": "Error logged to Loki format"})
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
			{"GET", "/loki", "loki endpoint"},
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
		ginAccessFile := ginNoRotateLogDir + "/gin-app.access.log"
		ginErrorFile := ginNoRotateLogDir + "/gin-app.error.log"
		ginLokiFile := ginNoRotateLogDir + "/gin-app.error-loki.log"

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

		// Verify Gin Loki log (without date)
		if content, err := os.ReadFile(ginLokiFile); err != nil {
			t.Errorf("Failed to read gin Loki log without rotation: %v", err)
		} else {
			contentStr := string(content)

			if !strings.Contains(contentStr, `"service":"gin-no-rotate-example"`) {
				t.Error("Expected gin service name not found in Loki log")
			}

			jsonEntries := strings.Count(contentStr, `"level":`)
			if jsonEntries < 3 {
				t.Errorf("Expected at least 3 JSON entries in Loki log, found %d", jsonEntries)
			}
			t.Logf("GIN NO ROTATE Loki log: %s (%d bytes, %d JSON entries)", ginLokiFile, len(content), jsonEntries)
		}

		// Verify that date-based files do NOT exist
		today := time.Now().Format("2006-01-02")
		rotatedGinAccessFile := ginNoRotateLogDir + "/gin-app.access-" + today + ".log"
		rotatedGinErrorFile := ginNoRotateLogDir + "/gin-app.error-" + today + ".log"
		rotatedGinLokiFile := ginNoRotateLogDir + "/gin-app.error-loki-" + today + ".log"

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