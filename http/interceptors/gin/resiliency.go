package gin

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

// ErrorHandlingMiddleware handles errors and panics and logs them appropriately with our logging framework
func ErrorHandlingMiddleware(c *gin.Context) {
	c.Next() // TODO implement
}

// TimeoutMiddleware sets a timeout on the request context
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
