package gin

import (
	"github.com/gin-gonic/gin"

	"github.com/rainbow-me/platform-tools/common/correlation"
	"github.com/rainbow-me/platform-tools/common/headers"
	"github.com/rainbow-me/platform-tools/common/metadata"
)

// CorrelationMiddleware extracts correlation data from http headers if found and propagates it as Go context.
// If no header is found, it will create a new correlation id.
func CorrelationMiddleware(c *gin.Context) {
	ctx := c.Request.Context()
	h := c.GetHeader(correlation.ContextCorrelationHeader) // TODO why is this a correlation-id header?
	ctx = correlation.ContextWithCorrelation(ctx, h)
	c.Request = c.Request.WithContext(ctx)
}

// RequestInfoMiddleware propagates a subset of RequestInfo fields to the context.
// We only support RequestID and CorrelationID for now via http, while full support is obtained only
// via grpc interceptors.
func RequestInfoMiddleware(c *gin.Context) {
	requestInfo := metadata.RequestInfo{
		RequestID:     c.GetHeader(headers.HeaderXRequestID),
		CorrelationID: correlation.ID(c.Request.Context()),
	}
	ctx := c.Request.Context()
	ctx = metadata.ContextWithRequestInfo(ctx, requestInfo)
	c.Request = c.Request.WithContext(ctx)
}
