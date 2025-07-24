package interceptors

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"time"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/ext"
	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	grpcrecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Log field keys for structured logging
const (
	StackTraceKey = "stack_trace"
	PanicValueKey = "panic_value"
	PanicTypeKey  = "panic_type"
)

// UnaryPanicRecoveryServerInterceptor creates a gRPC unary server interceptor that recovers
// from panics in gRPC handlers, logs them with structured information, and returns
// appropriate gRPC error responses to clients.
//
// This interceptor:
// - Catches panics that occur during gRPC request processing
// - Logs panic details with stack traces for debugging
// - Marks tracing spans as errored if available
// - Converts panics to gRPC Internal errors for client responses
// - Provides fallback logging when structured logger is unavailable
func UnaryPanicRecoveryServerInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return grpcrecovery.UnaryServerInterceptor(
		grpcrecovery.WithRecoveryHandlerContext(func(ctx context.Context, panicValue interface{}) error {
			// Extract panic information for logging
			panicMessage := extractPanicMessage(panicValue)
			panicType := extractPanicType(panicValue)

			// Log the panic with structured information
			logPanicWithFallback(panicMessage, panicType, logger)

			// Mark the span as failed if tracing is available
			span, ok := tracer.SpanFromContext(ctx)
			if ok {
				span.SetTag(ext.Error, true)
				span.SetTag(ext.ErrorType, "panic")
				span.SetTag(ext.ErrorMsg, codes.Internal.String())
				span.SetTag(ext.ErrorStack, fmt.Sprintf("%+v", panicValue))
			}

			// Return sanitized error to client (don't expose internal panic details)
			return status.Error(codes.Internal, "Internal server error occurred")
		}),
	)
}

// StreamPanicRecoveryServerInterceptor creates a gRPC stream server interceptor that recovers
// from panics in gRPC handlers, logs them with structured information, and returns
// appropriate gRPC error responses to clients.
//
// This interceptor:
// - Catches panics that occur during gRPC request processing
// - Logs panic details with stack traces for debugging
// - Marks tracing spans as errored if available
// - Converts panics to gRPC Internal errors for client responses
// - Provides fallback logging when structured logger is unavailable
func StreamPanicRecoveryServerInterceptor(logger *zap.Logger) grpc.StreamServerInterceptor {
	return grpcrecovery.StreamServerInterceptor(
		grpcrecovery.WithRecoveryHandlerContext(func(ctx context.Context, panicValue interface{}) error {
			// Extract panic information for logging
			panicMessage := extractPanicMessage(panicValue)
			panicType := extractPanicType(panicValue)

			// Log the panic with structured information
			logPanicWithFallback(panicMessage, panicType, logger)

			// Mark the span as failed if tracing is available
			span, ok := tracer.SpanFromContext(ctx)
			if ok {
				span.SetTag(ext.Error, true)
				span.SetTag(ext.ErrorType, "panic")
				span.SetTag(ext.ErrorMsg, codes.Internal.String())
				span.SetTag(ext.ErrorStack, fmt.Sprintf("%+v", panicValue))
			}

			// Return sanitized error to client (don't expose internal panic details)
			return status.Error(codes.Internal, "Internal server error occurred")
		}),
	)
}

// extractPanicMessage safely extracts a string representation of the panic value
func extractPanicMessage(panicValue any) string {
	if panicValue == nil {
		return "unknown panic (nil value)"
	}
	return fmt.Sprintf("%v", panicValue)
}

// extractPanicType safely extracts the type information of the panic value
func extractPanicType(panicValue any) string {
	if panicValue == nil {
		return "nil"
	}
	return fmt.Sprintf("%T", panicValue)
}

// logPanicWithFallback logs panic information using structured logging when available,
// or falls back to standard output if the logger is unavailable.
// This ensures panic information is never lost, even if the logging system fails.
func logPanicWithFallback(panicMessage, panicType string, logger *zap.Logger) {
	if logger != nil {
		// Use structured logging with additional context
		logger.Error("Recovered from panic in gRPC handler",
			zap.String(PanicValueKey, panicMessage),
			zap.String(PanicTypeKey, panicType),
			zap.Time("recovery_time", time.Now()),
			zap.Stack(StackTraceKey),
		)
	} else {
		// Fallback to standard error using slog
		l := slog.New(slog.NewTextHandler(os.Stderr, nil))
		l.Error("Recovered from panic",
			"time", time.Now().Format(time.RFC3339),
			"panic_value", panicMessage,
			"panic_type", panicType,
			"stack_trace", string(debug.Stack()),
		)
	}
}
