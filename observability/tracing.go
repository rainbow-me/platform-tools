package observability

import (
	"context"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"

	"github.com/rainbow-me/platform-tools/common/logger"
)

// StartSpan is a helper function that we should always use instead of tracer.StartSpanFromContext to ensure that our
// context logger gets updated with trace and span ID.
func StartSpan(ctx context.Context, opName string, opts ...tracer.StartSpanOption) (*tracer.Span, context.Context) {
	span, ctx := tracer.StartSpanFromContext(ctx, opName, opts...)
	ctx = logger.ContextWithFields(ctx, logger.WithTrace(span.Context())...)
	return span, ctx
}
