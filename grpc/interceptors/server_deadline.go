package interceptors

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

// ServerDeadlineInterceptor creates a gRPC unary server interceptor that enforces
// a maximum server-side timeout for all incoming requests.
//
// This interceptor:
// - Sets a maximum timeout for request processing
// - Prevents requests from running indefinitely
// - Respects existing shorter deadlines from clients
// - Automatically cancels context when deadline is exceeded
//
// Timeout behavior:
// - If the incoming request has no deadline, the specified timeout is applied
// - If the incoming request has a deadline shorter than the timeout, the client deadline is used
// - If the incoming request has a deadline longer than the timeout, the timeout is used
// - The earliest (shortest) deadline always takes precedence
//
// Usage:
//
//	interceptor := ServerDeadlineInterceptor(30 * time.Second)
//	server := grpc.NewServer(grpc.UnaryInterceptor(interceptor))
func ServerDeadlineInterceptor(timeout time.Duration) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context, req interface{},
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Create a new context with the specified timeout
		// Note: If the existing context has a deadline that occurs before now + timeout,
		// then that earlier deadline will be used (the earliest timeout wins).
		// Reference: https://golang.org/pkg/context/#WithDeadline
		ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout) //nolint:govet

		// Always cancel the context when the function returns to prevent resource leaks
		defer cancel()

		// Execute the gRPC handler with the timeout-constrained context
		return handler(ctxWithTimeout, req)
	}
}
