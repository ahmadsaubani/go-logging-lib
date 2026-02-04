package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ahmadsaubani/go-logging-lib"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GinMiddleware returns Gin middleware for request logging
func GinMiddleware(logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader("X-Request-ID")
		if reqID == "" {
			reqID = uuid.NewString()
		}

		meta := logging.Meta{
			RequestID: reqID,
			IP:        c.ClientIP(),
			Method:    c.Request.Method,
			Path:      c.Request.URL.Path,
			UserAgent: c.Request.UserAgent(),
		}

		ctx := logging.WithMeta(c.Request.Context(), meta)
		c.Request = c.Request.WithContext(ctx)
		c.Header("X-Request-ID", reqID)
		c.Next()
	}
}

// GinLogger returns Gin logger middleware that logs all requests to access log
func GinLogger(logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)

		meta, ok := logging.FromContext(c.Request.Context())
		if !ok {
			return
		}

		statusCode := c.Writer.Status()

		// Log ke access log untuk semua request
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
		logger.Access(logLine)

		// Determine log level based on status code
		level := logging.LevelInfo
		if statusCode >= 500 {
			level = logging.LevelCritical
		} else if statusCode >= 400 {
			level = logging.LevelError
		} else if statusCode >= 300 {
			level = logging.LevelWarn
		}

		// Get error if exists
		var err error
		if statusCode >= 400 {
			if panicInfo, exists := c.Get("panic_info"); exists {
				err = fmt.Errorf("%s", panicInfo.(string))
			} else if len(c.Errors) > 0 {
				err = fmt.Errorf("%s", c.Errors.String())
			} else if errVal, exists := c.Get("logged_error"); exists {
				if e, ok := errVal.(error); ok {
					err = e
				}
			}
		}

		// Log ke Loki dengan format konsisten
		logger.Loki(c.Request.Context(), level, statusCode, latency, err)
	}
}

// GinHTTPErrorLogger logs HTTP errors to error log
func GinHTTPErrorLogger(logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		status := c.Writer.Status()
		if status < 400 {
			return
		}

		// Skip logging if error was already manually logged
		if logging.IsErrorLogged(c) {
			return
		}

		errMsg := "HTTP Error"

		// Check if this is from a panic
		if panicInfo, exists := c.Get("panic_info"); exists {
			errMsg = panicInfo.(string)
		} else if len(c.Errors) > 0 {
			errMsg = c.Errors.String()
		}

		// Log with detailed format to error log only
		httpErr := fmt.Errorf("%s (status: %d, latency: %v)", errMsg, status, time.Since(start))
		logger.Error(c.Request.Context(), httpErr)
	}
}

// GinRecovery handles panic recovery without logging (let HTTPErrorLogger handle it)
func GinRecovery(logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				// Set panic info in context for HTTPErrorLogger to use
				c.Set("panic_info", fmt.Sprintf("PANIC: %v", r))
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()

		c.Next()
	}
}