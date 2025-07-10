package interceptors

import (
	"context"

	"google.golang.org/grpc"

	"github.com/rainbow-me/platfomt-tools/grpc/metadata"
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
		parser := metadata.NewMetadataParser(true, true)
		updatedCtx, requestInfo := parser.ParseMetadata(ctx)

		// Add to context for handlers using custom context key type
		ctxWithInfo := context.WithValue(updatedCtx, requestContextKey, requestInfo)

		// Call handler
		resp, err := handler(ctxWithInfo, req)

		return resp, err
	}
}

// GetRequestInfoFromContext extracts RequestInfo from context
func GetRequestInfoFromContext(ctx context.Context) (*metadata.RequestInfo, bool) {
	requestInfo, ok := ctx.Value(requestContextKey).(*metadata.RequestInfo)
	return requestInfo, ok
}
