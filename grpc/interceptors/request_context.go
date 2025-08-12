package interceptors

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	common "github.com/rainbow-me/platform-tools/common/metadata"
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
		req any,
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		parser := internalmetadata.NewRequestParser(internalmetadata.RequestParserOpt{
			IncludeAllHeaders: true,
			MaskSensitive:     true,
		})
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
	// Check if context is nil
	if ctx == nil {
		return &internalmetadata.RequestInfo{}, false
	}

	// Get value from context
	val := ctx.Value(requestContextKey)
	if val == nil {
		return &internalmetadata.RequestInfo{}, false
	}

	// Type assert to RequestInfo
	requestInfo, ok := val.(*internalmetadata.RequestInfo)
	if !ok || requestInfo == nil {
		return &internalmetadata.RequestInfo{}, false
	}

	return requestInfo, true
}

func UnaryRequestContextClientInterceptor(
	ctx context.Context,
	method string,
	req, reply any,
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	// Extract request ID from incoming metadata
	requestID := "unknown"
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get(common.HeaderXRequestID); len(values) > 0 && values[0] != "" {
			requestID = values[0]
		}
	}

	// Add request ID to outgoing metadata
	ctx = metadata.AppendToOutgoingContext(ctx, common.HeaderXRequestID, requestID)

	// Continue with the actual gRPC call using the updated context
	return invoker(ctx, method, req, reply, cc, opts...)
}
