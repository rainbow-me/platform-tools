package interceptors

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/rainbow-me/platform-tools/grpc/correlation"
)

// UnaryClientInterceptor generates and attaches correlation headers to outgoing unary gRPC calls.
// This ensures that correlation context is propagated from client to server for request tracing.
func UnaryClientInterceptor(
	ctx context.Context,
	method string,
	req,
	reply interface{},
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	// Generate a correlation header from the current context
	header := correlation.Generate(ctx)

	// Add the correlation header to outgoing metadata if one was generated
	if header != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, correlation.ContextCorrelationHeader, header)
	}

	// Continue with the actual gRPC call using the updated context
	return invoker(ctx, method, req, reply, cc, opts...)
}

// UnaryServerInterceptor extracts correlation headers from incoming unary gRPC requests
// and parses them into the request context for downstream processing.
func UnaryServerInterceptor(
	ctx context.Context,
	req interface{},
	_ *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	// Extract metadata from the incoming request
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		// Get the correlation header value from metadata
		header := strings.Join(md.Get(correlation.ContextCorrelationHeader), ",")

		// Parse the correlation header and update the context
		ctx = correlation.Parse(ctx, header)
	}

	// Continue with the actual request handler using the updated context
	return handler(ctx, req)
}
