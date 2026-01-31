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

// GinLogger returns Gin logger middleware
func GinLogger(logger *logging.Logger) gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		if param.StatusCode >= 400 {
			return ""
		}
		meta, ok := logging.FromContext(param.Request.Context())
		if !ok {
			return ""
		}

		return fmt.Sprintf(
			"[REQ:%s] %s | %3d | %13v | %15s | %-7s %s\n",
			meta.RequestID,
			param.TimeStamp.Format(time.RFC3339),
			param.StatusCode,
			param.Latency,
			meta.IP,
			meta.Method,
			meta.Path,
		)
	})
}

// GinHTTPErrorLogger logs HTTP errors with Loki format
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

		// Log with detailed format
		httpErr := fmt.Errorf("%s (status: %d, latency: %v)", errMsg, status, time.Since(start))
		
		// Use the same detailed error logging as basic
		logger.Error(c.Request.Context(), httpErr)
		
		// Also log in Loki format
		if status >= 500 {
			logger.ErrorLoki(c.Request.Context(), logging.LevelCritical, httpErr)
		} else {
			logger.ErrorLoki(c.Request.Context(), logging.LevelError, httpErr)
		}
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