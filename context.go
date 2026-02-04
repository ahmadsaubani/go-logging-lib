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

type Meta struct {
	RequestID string
	IP        string
	Method    string
	Path      string
	UserAgent string
}

func WithMeta(ctx context.Context, meta Meta) context.Context {
	return context.WithValue(ctx, metaKey, meta)
}

func FromContext(ctx context.Context) (Meta, bool) {
	meta, ok := ctx.Value(metaKey).(Meta)
	return meta, ok
}

func WithError(ctx context.Context, err error) context.Context {
	return context.WithValue(ctx, loggedErrorKey, err)
}

func ErrorFromContext(ctx context.Context) (error, bool) {
	err, ok := ctx.Value(loggedErrorKey).(error)
	return err, ok
}

/**
 * NewRequestContext creates a context with request metadata from http.Request.
 * This is the framework-agnostic alternative to Gin middleware.
 *
 * @param r HTTP request to extract metadata from
 * @return context.Context Context with embedded request metadata
 */
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

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return r.RemoteAddr
}