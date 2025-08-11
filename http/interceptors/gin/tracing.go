package gin

import (
	"fmt"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/ext"
	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/gin-gonic/gin"

	"github.com/rainbow-me/platform-tools/common/logger"
)

// TracingMiddleware takes the trace id from http headers, or creates a new one if none found.
// It then creates a new span and tags it appropriately with http route, method, url, response code, error, etc.
// Finally, it injects the trace/span ids in the context log fields.
func TracingMiddleware(c *gin.Context) {
	// We could have just used "github.com/DataDog/dd-trace-go/contrib/gin-gonic/gin/v2",
	// but this middleware adds some extra useful tags
	spanOpts := []tracer.StartSpanOption{
		tracer.Tag(ext.Component, componentName),
		tracer.Tag(ext.SpanType, ext.SpanTypeWeb),
		tracer.Tag(ext.HTTPMethod, c.Request.Method),
		tracer.Tag(ext.HTTPURL, c.Request.URL.String()),
		tracer.Tag(ext.ResourceName, fmt.Sprintf("%s %s", c.Request.Method, c.FullPath())),
		tracer.Tag(ext.HTTPRoute, c.FullPath()),
	}
	ctx := c.Request.Context()

	// Try to capture an existing trace in the http headers
	sCtx, err := tracer.Extract(tracer.HTTPHeadersCarrier(c.Request.Header))
	if err == nil && sCtx != nil {
		spanOpts = append(spanOpts, func(cfg *tracer.StartSpanConfig) {
			cfg.Parent = sCtx
		})
	}

	// Handle span creation and termination
	span := tracer.StartSpan(httpHandlerOp, spanOpts...)
	defer span.Finish()

	// Pass the span through the request context
	ctx = logger.ContextWithFields(ctx, logger.WithTrace(span.Context()))
	c.Request = c.Request.WithContext(ctx)
	c.Next()

	span.SetTag(ext.HTTPCode, c.Writer.Status())
	if c.Writer.Status() >= 500 {
		span.SetTag(ext.Error, true)
	}
}
