package logging

import "context"

type ctxKey struct{}

var metaKey = ctxKey{}

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