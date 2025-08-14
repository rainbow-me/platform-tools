package interceptors

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/rainbow-me/platform-tools/common/correlation"
	meta "github.com/rainbow-me/platform-tools/grpc/metadata"
)

// UnaryCorrelationServerInterceptor returns a gRPC unary server interceptor that manages correlation data.
//
// This interceptor performs the following operations:
//  1. Extracts the "correlation-context" header from incoming gRPC metadata
//  2. Parses the correlation data and injects it into the request context
//  3. Automatically propagates correlation data to OpenTelemetry baggage on the active span
//  4. Generates a new correlation_id (UUID v4) if one is not present in the incoming request
//
// The correlation data flows through the entire request lifecycle and can be accessed
// by downstream handlers and services.
func UnaryCorrelationServerInterceptor(
	ctx context.Context,
	req interface{},
	_ *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	ctx = correlation.ContextWithCorrelation(ctx, getCorrelationFromMD(ctx))
	return handler(ctx, req)
}

func getCorrelationFromMD(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		// Extract and parse correlation-context header
		return meta.GetFirst(md, correlation.ContextCorrelationHeader)
	}
	return ""
}

// UnaryCorrelationClientInterceptor returns a gRPC unary client interceptor that propagates correlation data.
//
// This interceptor performs the following operations:
//  1. Retrieves all correlation data from the current context
//  2. Serializes the correlation data into a single "correlation-context" header value
//  3. Appends this header to the outgoing gRPC metadata for transmission to the server
//
// This ensures correlation data is automatically propagated across service boundaries,
// maintaining request traceability throughout distributed system calls.
func UnaryCorrelationClientInterceptor(
	ctx context.Context,
	method string,
	req, reply interface{},
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
