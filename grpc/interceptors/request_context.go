package interceptors

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	internalmetadata "github.com/rainbow-me/platform-tools/grpc/metadata"
)

// Define a custom type for context keys to avoid collisions
type contextKey string

// requestContextKey is the key used to store RequestInfo in context
const requestContextKey contextKey = "request_info"

// RequestContextUnaryServerInterceptor creates a gRPC interceptor that extracts RequestInfo
func RequestContextUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		parser := internalmetadata.NewRequestParser(true, true)
		updatedCtx, requestInfo := parser.ParseMetadata(ctx)

		// Add to context for handlers using custom context key type
		ctxWithInfo := context.WithValue(updatedCtx, requestContextKey, requestInfo)

		// Call handler
		resp, err := handler(ctxWithInfo, req)

		return resp, err
	}
}

// GetRequestInfoFromContext extracts RequestInfo from context
func GetRequestInfoFromContext(ctx context.Context) (*internalmetadata.RequestInfo, bool) {
	requestInfo, ok := ctx.Value(requestContextKey).(*internalmetadata.RequestInfo)
	return requestInfo, ok
}

func UnaryRequestContextClientInterceptor(
	ctx context.Context,
	method string,
	req, reply interface{},
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	// Extract request ID from incoming metadata
	requestID := "unknown"
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get(internalmetadata.HeaderXRequestID); len(values) > 0 && values[0] != "" {
			requestID = values[0]
		}
	}

	// Add request ID to outgoing metadata
	ctx = metadata.AppendToOutgoingContext(ctx, internalmetadata.HeaderXRequestID, requestID)

	// Continue with the actual gRPC call using the updated context
	return invoker(ctx, method, req, reply, cc, opts...)
}
