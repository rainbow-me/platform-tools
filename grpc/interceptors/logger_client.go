package interceptors

import (
	"context"

	"google.golang.org/grpc"

	"github.com/rainbow-me/platform-tools/common/logger"
)

// UnaryLoggerClientInterceptor creates a gRPC unary client interceptor that logs
// outgoing requests and incoming responses with timing and context information.
//
// This interceptor logs:
// - Request and response payloads (based on configuration)
// - Request timing and duration
// - gRPC method and service names
// - Client ID and trace information
// - Error details and status codes
func UnaryLoggerClientInterceptor(log logger.Logger, opts ...LoggingInterceptorOption) grpc.UnaryClientInterceptor {
	// Build configuration from provided options
	config := interceptorConfig(opts...)

	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		// Log the request with context and execute the gRPC invoker
		reply, err := logWithContext(
			ctx,
			"client.request", // Log entry identifier
			method,           // gRPC method being called
			config,           // Logging configuration
			log,              // Logger instance
			req,              // Request payload
			func(ctx context.Context) (interface{}, error) {
				// Execute the actual gRPC invoker
				err := invoker(ctx, method, req, reply, cc, opts...)
				return reply, err
			},
		)
		return err
	}
}
