package interceptors

import (
	"context"

	"google.golang.org/grpc"

	"github.com/rainbow-me/platform-tools/common/logger"
)

// UnaryLoggerServerInterceptor creates a gRPC unary server interceptor that logs
// incoming requests and outgoing responses with timing and context information.
//
// This interceptor logs:
// - Request and response payloads (based on configuration)
// - Request timing and duration
// - gRPC method and service names
// - Client ID and trace information
// - Error details and status codes
func UnaryLoggerServerInterceptor(log *logger.Logger, opts ...LoggingInterceptorOption) grpc.UnaryServerInterceptor {
	// Build configuration from provided options
	config := interceptorConfig(opts...)

	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Log the request with context and execute the gRPC handler
		return logWithContext(
			ctx,
			"server.request", // Log entry identifier
			info.FullMethod,  // gRPC method being called
			config,           // Logging configuration
			log,              // Logger instance
			req,              // Request payload
			func(ctx context.Context) (interface{}, error) {
				// Execute the actual gRPC handler
				return handler(ctx, req)
			},
		)
	}
}
