package logger

import (
	"context"

	"go.uber.org/zap"
)

type loggerKey struct{}

// FromContext extracts a logger from the context or instantiates a new one if none found
func FromContext(ctx context.Context) *Logger {
	if logger, ok := ctx.Value(loggerKey{}).(*Logger); ok {
		return logger
	}
	log, err := Instance()
	if err != nil {
		z, zErr := zap.NewProduction()
		if zErr != nil {
			z = zap.NewNop()
		}
		return NewLogger(z)
	}
	return log
}

// ContextWithLogger returns a context with the logger stored for later retrieval via FromContext
func ContextWithLogger(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// ContextWithFields returns a context with a logger to which the fields are appended
func ContextWithFields(ctx context.Context, fields []Field) context.Context {
	return ContextWithLogger(ctx, FromContext(ctx).With(fields...))
}
