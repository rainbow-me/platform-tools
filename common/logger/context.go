package logger

import (
	"context"
)

type loggerKey struct{}

// FromContext extracts a logger from the context or instantiates a new one if none found
func FromContext(ctx context.Context) Logger {
	if logger, ok := ctx.Value(loggerKey{}).(Logger); ok {
		return logger
	}
	return Instance()
}

// ContextWithLogger returns a context with the logger stored for later retrieval via FromContext
func ContextWithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// ContextWithFields returns a context with a logger to which the fields are appended
func ContextWithFields(ctx context.Context, fields []Field) context.Context {
	return ContextWithLogger(ctx, FromContext(ctx).With(fields...))
}
