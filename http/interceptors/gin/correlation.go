package gin

import (
	"github.com/gin-gonic/gin"

	"github.com/rainbow-me/platform-tools/common/logger"
	"github.com/rainbow-me/platform-tools/grpc/correlation"
)

// CorrelationMiddleware extracts correlation data from http headers if found and propagates it as Go context.
// If no header is found, it will create a new correlation id.
func CorrelationMiddleware(c *gin.Context) {
	// TODO martin what about request id?
	ctx := correlation.ContextWithCorrelation(c.Request.Context(), getCorrelationHeader(c))
	ctx = logger.ContextWithFields(ctx, correlation.ToZapFields(ctx))
	c.Request = c.Request.WithContext(ctx)
}

func getCorrelationHeader(c *gin.Context) func() string {
	return func() string {
		return c.GetHeader(correlation.ContextCorrelationHeader)
	}
}
