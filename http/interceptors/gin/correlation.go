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
	ctx := metadata.ContextWithRequestInfo(c.Request.Context(), metadata.RequestInfo{
		RequestID:     c.GetHeader(headers.HeaderXRequestID),
		CorrelationID: c.GetHeader(correlation.ContextCorrelationHeader),
	})
	c.Request = c.Request.WithContext(ctx)
}
