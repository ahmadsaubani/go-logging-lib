package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/ahmadsaubani/go-logging-lib"
	"github.com/google/uuid"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

type requestState struct {
	mu  sync.Mutex
	err error
}

func (s *requestState) SetError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.err = err
}

func (s *requestState) GetError() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

type stateKey struct{}

var reqStateKey = stateKey{}

/**
 * HTTPMiddleware returns standard http middleware for request logging.
 * Framework-agnostic alternative to GinMiddleware.
 *
 * @param logger Logger instance
 * @return func(http.Handler) http.Handler Middleware wrapper
 */
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

			state := &requestState{}
			ctx := logging.WithMeta(r.Context(), meta)
			ctx = context.WithValue(ctx, reqStateKey, state)
			r = r.WithContext(ctx)
			w.Header().Set("X-Request-ID", reqID)

			next.ServeHTTP(w, r)
		})
	}
}

/**
 * HTTPLogger returns standard http middleware that logs all requests.
 * Framework-agnostic alternative to GinLogger.
 *
 * @param logger Logger instance
 * @return func(http.Handler) http.Handler Middleware wrapper
 */
func HTTPLogger(logger *logging.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(rw, r)

			latency := time.Since(start)
			statusCode := rw.statusCode

			var err error
			if state, ok := r.Context().Value(reqStateKey).(*requestState); ok && state != nil {
				err = state.GetError()
			}

			logger.LogRequestWithError(r.Context(), statusCode, latency, err)
		})
	}
}

/**
 * HTTPRecovery handles panic recovery for standard http handlers.
 * Framework-agnostic alternative to GinRecovery.
 *
 * @param logger Logger instance
 * @return func(http.Handler) http.Handler Recovery middleware wrapper
 */
func HTTPRecovery(logger *logging.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					if state, ok := r.Context().Value(reqStateKey).(*requestState); ok && state != nil {
						state.SetError(errFromPanic(rec))
					}
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

/**
 * SetHTTPError stores an error in the request state for logging.
 * Use this in handlers to pass errors to the logging middleware.
 *
 * @param r HTTP request
 * @param err Error to store
 */
func SetHTTPError(r *http.Request, err error) {
	if state, ok := r.Context().Value(reqStateKey).(*requestState); ok && state != nil {
		state.SetError(err)
	}
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return r.RemoteAddr
}

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
