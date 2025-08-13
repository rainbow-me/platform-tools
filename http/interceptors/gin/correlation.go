package gin

import (
	"github.com/gin-gonic/gin"

	"github.com/rainbow-me/platform-tools/common/correlation"
	"github.com/rainbow-me/platform-tools/common/headers"
	"github.com/rainbow-me/platform-tools/common/logger"
	"github.com/rainbow-me/platform-tools/common/metadata"
	internalmetadata "github.com/rainbow-me/platform-tools/grpc/metadata"
)

// CorrelationMiddleware extracts correlation data from http headers if found and propagates it as Go context.
// If no header is found, it will create a new correlation id.
func CorrelationMiddleware(c *gin.Context) {
	reqInfo := &internalmetadata.RequestInfo{
		RequestID:     c.GetHeader(headers.HeaderXRequestID),
		CorrelationID: c.GetHeader(correlation.ContextCorrelationHeader),
	}
	ctx := metadata.ContextWithRequestInfo(c.Request.Context(), reqInfo)

	fields := correlation.ToZapFields(ctx)
	ctx = logger.ContextWithFields(ctx, fields)

	c.Request = c.Request.WithContext(ctx)
}
