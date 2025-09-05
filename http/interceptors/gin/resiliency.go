package gin

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/ext"
	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/gin-gonic/gin"

	"github.com/rainbow-me/platform-tools/common/env"
	"github.com/rainbow-me/platform-tools/common/logger"
)

// ErrorHandlingMiddleware handles errors, logs them appropriately with our logging framework
// and tags the span with the error
func ErrorHandlingMiddleware(c *gin.Context) {
	c.Next()
	if len(c.Errors) == 0 {
		return
	}
	err := c.Errors.Last().Err
	logger.FromContext(c).Error("Error in gin http handler",
		logger.String("path", c.FullPath()),
		logger.Error(err),
	)
	if env.IsLocalApplicationEnv() {
		// pretty print the error to the local console to make it human-readable in case it has a stack trace
		_, _ = fmt.Fprintf(os.Stderr, "Error in gin http handler: %+v\n", err)
	}
	tagSpanAsError(c.Request.Context(), "internal", err.Error())
	c.JSON(500, gin.H{
		"message": "internal server error",
	})
}

// PanicRecoveryMiddleware handles panics, logs them appropriately with our logging framework
// and tags the span with the error
func PanicRecoveryMiddleware(c *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			logger.FromContext(c).Error("Recovered from panic in gin http handler", logger.WithPanic(r)...)
			if env.IsLocalApplicationEnv() {
				// pretty print the stack trace to the local console to make it human-readable
				_, _ = fmt.Fprintf(os.Stderr, "%s\n", debug.Stack())
			}
			tagSpanAsError(c.Request.Context(), "panic", fmt.Sprintf("%v", r))
			c.JSON(500, gin.H{
				"message": "internal server error",
			})
		}
	}()
	c.Next()
}

func tagSpanAsError(ctx context.Context, errorType string, errorMsg string) {
	// Mark the span as failed if tracing is available
	span, ok := tracer.SpanFromContext(ctx)
	if ok {
		span.SetTag(ext.Error, true)
		span.SetTag(ext.ErrorType, errorType)
		span.SetTag(ext.ErrorMsg, errorMsg)
	}
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
