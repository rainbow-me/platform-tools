package correlation

import (
	"context"
	"strings"

	grpcmiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
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
	header := Generate(ctx)

	// Add the correlation header to outgoing metadata if one was generated
	if header != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, ContextCorrelationHeader, header)
	}

	// Continue with the actual gRPC call using the updated context
	return invoker(ctx, method, req, reply, cc, opts...)
}

// StreamClientInterceptor generates and attaches correlation headers to outgoing streaming gRPC calls.
// This ensures that correlation context is propagated from client to server for streaming request tracing.
func StreamClientInterceptor(
	ctx context.Context,
	desc *grpc.StreamDesc,
	cc *grpc.ClientConn,
	method string,
	streamer grpc.Streamer,
	opts ...grpc.CallOption,
) (grpc.ClientStream, error) {
	// Generate a correlation header from the current context
	header := Generate(ctx)

	// Add the correlation header to outgoing metadata if one was generated
	if header != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, ContextCorrelationHeader, header)
	}

	// Continue with the actual gRPC stream call using the updated context
	return streamer(ctx, desc, cc, method, opts...)
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
		header := strings.Join(md.Get(ContextCorrelationHeader), ",")

		// Parse the correlation header and update the context
		ctx = Parse(ctx, header)
	}

	// Continue with the actual request handler using the updated context
	return handler(ctx, req)
}

// StreamServerInterceptor extracts correlation headers from incoming streaming gRPC requests
// and parses them into the request context for downstream processing.
func StreamServerInterceptor(
	srv interface{},
	ss grpc.ServerStream,
	_ *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	// Get the context from the server stream
	ctx := ss.Context()

	// Extract metadata from the incoming request
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		// Get the correlation header value from metadata
		header := strings.Join(md.Get(ContextCorrelationHeader), ",")

		// Parse the correlation header and update the context
		ctx = Parse(ctx, header)
	}

	// Wrap the server stream with the updated context
	wrapped := grpcmiddleware.WrapServerStream(ss)
	wrapped.WrappedContext = ctx

	// Continue with the actual stream handler using the wrapped stream
	return handler(srv, wrapped)
}
