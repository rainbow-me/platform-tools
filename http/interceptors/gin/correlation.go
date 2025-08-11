package gin

import (
	"github.com/gin-gonic/gin"
)

// CorrelationMiddleware extracts context metadata from http headers and propagates it as Go context
func CorrelationMiddleware(c *gin.Context) {
	c.Next() // TODO implement
}
