package metadata

import (
	"context"

	internalmetadata "github.com/rainbow-me/platform-tools/grpc/metadata"
)

// requestContextKey is the key used to store RequestInfo in context
type requestContextKey struct{}

func ContextWithRequestInfo(ctx context.Context, requestInfo *internalmetadata.RequestInfo) context.Context {
	return context.WithValue(ctx, requestContextKey{}, requestInfo)
}

// GetRequestInfoFromContext extracts RequestInfo from context
func GetRequestInfoFromContext(ctx context.Context) (*internalmetadata.RequestInfo, bool) {
	// Check if context is nil
	if ctx == nil {
		return &internalmetadata.RequestInfo{}, false
	}

	// Get value from context
	val := ctx.Value(requestContextKey{})
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

func GetRequestIDFromContext(ctx context.Context) string {
	requestInfo, found := GetRequestInfoFromContext(ctx)
	if found {
		return requestInfo.RequestID
	}
	return ""
}

func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	requestInfo, _ := GetRequestInfoFromContext(ctx)
	requestInfo.RequestID = requestID
	return ContextWithRequestInfo(ctx, requestInfo)
}
