package logger

import (
	"context"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"go.uber.org/zap"
)

type loggerKey struct{}

// FromContext extracts a logger from the context or instantiates a new one if none found, and adds custom fields
// for tracing.
func FromContext(ctx context.Context) *Logger {
	var fields []zap.Field
	span, found := tracer.SpanFromContext(ctx)
	if found {
		fields = append(fields, WithTrace(span.Context())...)
	}

	if logger, ok := ctx.Value(loggerKey{}).(*Logger); ok {
		return logger.With(fields...)
	}

	// no logger in context, let's grab the global logger
	log, err := Instance()
	if err != nil {
		// in the unlikely event that the global logger failed to start, try creating a production logger
		z, zErr := zap.NewProduction()
		if zErr != nil {
			// if this fails as well, just use a no-op
			z = zap.NewNop()
		}
		log = NewLogger(z)
	}
	return log.With(fields...)
}

// ContextWithLogger returns a context with the logger stored for later retrieval via FromContext
func ContextWithLogger(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// ContextWithFields returns a context with a logger to which the fields are appended
func ContextWithFields(ctx context.Context, fields []Field) context.Context {
	return ContextWithLogger(ctx, FromContext(ctx).With(fields...))
}
