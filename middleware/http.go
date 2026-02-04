package middleware

import (
	"net/http"
	"time"

	"github.com/ahmadsaubani/go-logging-lib"
	"github.com/google/uuid"
)

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// HTTPMiddleware returns standard http middleware for request logging
// This is the basic/framework-agnostic alternative to GinMiddleware
func HTTPMiddleware(logger *logging.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := r.Header.Get("X-Request-ID")
			if reqID == "" {
				reqID = uuid.NewString()
			}

			meta := logging.Meta{
				RequestID: reqID,
				IP:        getClientIP(r),
				Method:    r.Method,
				Path:      r.URL.Path,
				UserAgent: r.UserAgent(),
			}

			ctx := logging.WithMeta(r.Context(), meta)
			r = r.WithContext(ctx)
			w.Header().Set("X-Request-ID", reqID)

			next.ServeHTTP(w, r)
		})
	}
}

// HTTPLogger returns standard http middleware that logs all requests
// This is the basic/framework-agnostic alternative to GinLogger
func HTTPLogger(logger *logging.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(rw, r)

			latency := time.Since(start)
			statusCode := rw.statusCode

			// Get error from context if exists
			var err error
			if e, ok := logging.ErrorFromContext(r.Context()); ok {
				err = e
			}

			// Log with consistent format (errors=null for success)
			logger.LogRequestWithError(r.Context(), statusCode, latency, err)
		})
	}
}

// HTTPRecovery handles panic recovery for standard http handlers
// This is the basic/framework-agnostic alternative to GinRecovery
func HTTPRecovery(logger *logging.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					// Store panic as error in context for logging
					ctx := logging.WithError(r.Context(), errFromPanic(rec))
					r = r.WithContext(ctx)

					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// getClientIP extracts client IP from request
func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return r.RemoteAddr
}

// errFromPanic converts panic value to error
func errFromPanic(rec interface{}) error {
	switch v := rec.(type) {
	case error:
		return v
	default:
		return &panicError{value: rec}
	}
}

type panicError struct {
	value interface{}
}

func (e *panicError) Error() string {
	return "PANIC: " + stringFromInterface(e.value)
}

func stringFromInterface(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	default:
		return "unknown panic"
	}
}
