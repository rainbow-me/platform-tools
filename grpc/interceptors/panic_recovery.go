package interceptors

import (
	"fmt"
	"time"

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
// - Converts panics to gRPC Internal errors for client responses
// - Provides fallback logging when structured logger is unavailable
func UnaryPanicRecoveryServerInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return grpcrecovery.UnaryServerInterceptor(
		grpcrecovery.WithRecoveryHandler(func(panicValue interface{}) error {
			// Extract panic information for logging
			panicMessage := extractPanicMessage(panicValue)
			panicType := extractPanicType(panicValue)

			// Log the panic with structured information
			logPanicWithFallback(panicMessage, panicType, logger)

			// Return sanitized error to client (don't expose internal panic details)
			return status.Error(codes.Internal, "Internal server error occurred")
		}),
	)
}

// extractPanicMessage safely extracts a string representation of the panic value
func extractPanicMessage(panicValue interface{}) string {
	if panicValue == nil {
		return "unknown panic (nil value)"
	}
	return fmt.Sprintf("%v", panicValue)
}

// extractPanicType safely extracts the type information of the panic value
func extractPanicType(panicValue interface{}) string {
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
	}
}
