package interceptors

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/rainbow-me/platform-tools/common/headers"
	commonmeta "github.com/rainbow-me/platform-tools/common/metadata"
	internalmetadata "github.com/rainbow-me/platform-tools/grpc/metadata"
	"github.com/rainbow-me/platform-tools/observability"
)

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

		if requestInfo.RequestID != "" {
			observability.SetTag(ctx, "request_id", requestInfo.RequestID)
		}

		// Add to context for handlers using custom context key type
		ctxWithInfo := commonmeta.ContextWithRequestInfo(updatedCtx, *requestInfo)

		// Call handler
		resp, err := handler(ctxWithInfo, req)

		return resp, err
	}
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
		if values := md.Get(headers.HeaderXRequestID); len(values) > 0 && values[0] != "" {
			requestID = values[0]
		}
	}

	// Add request ID to outgoing metadata
	ctx = metadata.AppendToOutgoingContext(ctx, headers.HeaderXRequestID, requestID)

	// Continue with the actual gRPC call using the updated context
	return invoker(ctx, method, req, reply, cc, opts...)
}
