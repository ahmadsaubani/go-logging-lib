package logging

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type ctxKey struct{}
type errorKey struct{}

var metaKey = ctxKey{}
var loggedErrorKey = errorKey{}

// Meta holds request metadata for logging
type Meta struct {
	RequestID string
	IP        string
	Method    string
	Path      string
	UserAgent string
}

// WithMeta adds metadata to context
func WithMeta(ctx context.Context, meta Meta) context.Context {
	return context.WithValue(ctx, metaKey, meta)
}

// FromContext extracts metadata from context
func FromContext(ctx context.Context) (Meta, bool) {
	meta, ok := ctx.Value(metaKey).(Meta)
	return meta, ok
}

// WithError stores error in context for later Loki logging
func WithError(ctx context.Context, err error) context.Context {
	return context.WithValue(ctx, loggedErrorKey, err)
}

// ErrorFromContext extracts stored error from context
func ErrorFromContext(ctx context.Context) (error, bool) {
	err, ok := ctx.Value(loggedErrorKey).(error)
	return err, ok
}

// NewRequestContext creates a context with request metadata from http.Request
// This is the basic/framework-agnostic alternative to Gin middleware
func NewRequestContext(r *http.Request) context.Context {
	reqID := r.Header.Get("X-Request-ID")
	if reqID == "" {
		reqID = uuid.NewString()
	}

	meta := Meta{
		RequestID: reqID,
		IP:        getClientIP(r),
		Method:    r.Method,
		Path:      r.URL.Path,
		UserAgent: r.UserAgent(),
	}

	return WithMeta(r.Context(), meta)
}

// getClientIP extracts client IP from request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fall back to RemoteAddr
	return r.RemoteAddr
}