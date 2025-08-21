package interceptors

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/ext"
	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	grpcrecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rainbow-me/platform-tools/common/env"
	"github.com/rainbow-me/platform-tools/common/logger"
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
func UnaryPanicRecoveryServerInterceptor(logger *logger.Logger) grpc.UnaryServerInterceptor {
	return grpcrecovery.UnaryServerInterceptor(
		grpcrecovery.WithRecoveryHandlerContext(func(ctx context.Context, panicValue any) error {
			// Log the panic with structured information
			logPanicWithFallback(panicValue, logger)

			// Mark the span as failed if tracing is available
			span, ok := tracer.SpanFromContext(ctx)
			if ok {
				span.SetTag(ext.Error, true)
				span.SetTag(ext.ErrorType, "panic")
				span.SetTag(ext.ErrorMsg, codes.Internal.String())
				// span.SetTag(ext.ErrorStack, fmt.Sprintf("%+v", panicValue))
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
func StreamPanicRecoveryServerInterceptor(logger *logger.Logger) grpc.StreamServerInterceptor {
	return grpcrecovery.StreamServerInterceptor(
		grpcrecovery.WithRecoveryHandlerContext(func(ctx context.Context, panicValue any) error {
			// Log the panic with structured information
			logPanicWithFallback(panicValue, logger)

			// Mark the span as failed if tracing is available
			span, ok := tracer.SpanFromContext(ctx)
			if ok {
				span.SetTag(ext.Error, true)
				span.SetTag(ext.ErrorType, "panic")
				span.SetTag(ext.ErrorMsg, codes.Internal.String())
				// span.SetTag(ext.ErrorStack, fmt.Sprintf("%+v", panicValue))
			}

			// Return sanitized error to client (don't expose internal panic details)
			return status.Error(codes.Internal, "Internal server error occurred")
		}),
	)
}

// logPanicWithFallback logs panic information using structured logging when available,
// or falls back to standard output if the logger is unavailable.
// This ensures panic information is never lost, even if the logging system fails.
func logPanicWithFallback(panicValue any, log *logger.Logger) {
	if log != nil {
		// Use structured logging with additional context
		log.Error("Recovered from panic in gRPC handler", logger.WithPanic(panicValue)...)
	} else {
		// Fallback to standard error using slog
		slog.New(slog.NewTextHandler(os.Stderr, nil)).
			Error("Recovered from panic", logger.PanicValueKey, fmt.Sprintf("%+v", panicValue))
	}
	if env.IsLocalApplicationEnv() {
		// pretty print the stack trace to the local console to make it human-readable
		_, _ = fmt.Fprintf(os.Stderr, "%s\n", debug.Stack())
	}
}
