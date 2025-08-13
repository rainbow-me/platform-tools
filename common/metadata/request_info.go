package metadata

import (
	"context"

	"github.com/rainbow-me/platform-tools/common/logger"
)

// RequestInfo contains extracted information from metadata
type RequestInfo struct {
	RequestTime string `json:"requestTime"`

	// Request Identification
	RequestID     string `json:"requestId"`
	CorrelationID string `json:"correlationId"`
	TraceID       string `json:"traceId"`

	// Authentication
	HasAuth   bool   `json:"hasAuth"`
	AuthType  string `json:"authType,omitempty"`  // e.g., Bearer, Basic, Digest, ApiKey
	AuthToken string `json:"authToken,omitempty"` // Masked if sensitive

	// Raw headers for debugging
	AllHeaders map[string]string `json:"allHeaders,omitempty"`
}

// requestContextKey is the key used to store RequestInfo in context
type requestContextKey struct{}

func ContextWithRequestInfo(ctx context.Context, requestInfo RequestInfo) context.Context {
	ctx = context.WithValue(ctx, requestContextKey{}, requestInfo)
	return logger.ContextWithFields(ctx, requestInfo.ToZapFields())
}

// GetRequestInfoFromContext extracts RequestInfo from context
func GetRequestInfoFromContext(ctx context.Context) (RequestInfo, bool) {
	// Check if context is nil
	if ctx == nil {
		return RequestInfo{}, false
	}

	// Get value from context
	val := ctx.Value(requestContextKey{})
	if val == nil {
		return RequestInfo{}, false
	}

	// Type assert to RequestInfo
	requestInfo, ok := val.(RequestInfo)
	if !ok {
		return RequestInfo{}, false
	}

	return requestInfo, true
}

func (r RequestInfo) ToZapFields() []logger.Field {
	return []logger.Field{
		logger.String("request_id", r.RequestID),
		logger.String("correlation_id", r.CorrelationID),
	}
}
