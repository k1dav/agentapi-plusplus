package logctx

import (
	"context"
	"log/slog"
)

type contextKey int

const (
	loggerKey contextKey = iota
	requestIDKey
)

// WithLogger returns a new context with the provided logger
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// From retrieves the logger from the context or panics if no logger is found
func From(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return logger
	}
	panic("no logger found in context")
}

// WithRequestID returns a new context with the provided request ID
// attached. Handlers downstream can call RequestID to retrieve it for
// logging, response headers, or propagation to outbound calls.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestID retrieves the request ID previously attached via
// WithRequestID, returning the empty string if none is present. Callers
// that require a value (e.g. middleware that synthesises a fallback)
// should check the result and supply their own.
func RequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// WithLoggerAndRequestID is a small convenience for HTTP middleware:
// attach a per-request logger (already enriched with request_id) and
// the request ID itself in a single call. Equivalent to calling
// WithLogger and WithRequestID separately.
func WithLoggerAndRequestID(ctx context.Context, logger *slog.Logger, id string) context.Context {
	ctx = WithRequestID(ctx, id)
	ctx = WithLogger(ctx, logger)
	return ctx
}
