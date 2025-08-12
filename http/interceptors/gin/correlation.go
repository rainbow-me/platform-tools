package gin

import (
	"github.com/gin-gonic/gin"

	"github.com/rainbow-me/platform-tools/common/correlation"
	"github.com/rainbow-me/platform-tools/common/headers"
	"github.com/rainbow-me/platform-tools/common/logger"
)

// CorrelationMiddleware extracts correlation data from http headers if found and propagates it as Go context.
// If no header is found, it will create a new correlation id.
func CorrelationMiddleware(c *gin.Context) {
	ctx := correlation.ContextWithCorrelation(c.Request.Context(), c.GetHeader(correlation.ContextCorrelationHeader))
	ctx = correlation.ContextWithRequestID(ctx, c.GetHeader(headers.HeaderXRequestID))

	ctx = logger.ContextWithFields(ctx, correlation.ToZapFields(ctx))

	c.Request = c.Request.WithContext(ctx)
}
