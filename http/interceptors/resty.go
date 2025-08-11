package interceptors

import (
	"fmt"
	"net/url"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/ext"
	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/go-resty/resty/v2"

	"github.com/rainbow-me/platform-tools/common/logger"
	"github.com/rainbow-me/platform-tools/common/metadata"
)

const (
	httpRequestOp      = "http.request"
	restyComponentName = "resty"
)

func InjectMiddlewares(client *resty.Client) {
	before, after := RestyTracingMiddleware()
	client.OnBeforeRequest(before)
	client.OnAfterResponse(after)
}

// RestyTracingMiddleware propagates traces from context to http headers.
// Also, creates a new span and tags it with the http method, url, status code etc.
func RestyTracingMiddleware() (resty.RequestMiddleware, resty.ResponseMiddleware) {
	beforeRequest := func(c *resty.Client, req *resty.Request) error {
		opts := []tracer.StartSpanOption{
			tracer.SpanType(ext.SpanTypeHTTP),
			tracer.Tag(ext.HTTPMethod, req.Method),
			tracer.Tag(ext.HTTPURL, req.URL),
			tracer.Tag(ext.Component, restyComponentName),
			tracer.Tag(ext.SpanKind, ext.SpanKindClient),
		}
		if parsedURL, err := url.Parse(req.URL); err == nil {
			opts = append(opts, tracer.Tag(ext.NetworkDestinationName, parsedURL.Hostname()))
			opts = append(opts, tracer.Tag("http.host", parsedURL.Host))
			opts = append(opts, tracer.Tag("http.path", parsedURL.Path))
		}

		span, ctx := tracer.StartSpanFromContext(req.Context(), httpRequestOp, opts...)
		req.SetContext(ctx)

		// propagate trace through Rainbow custom tracing header
		req.SetHeader(metadata.HeaderXTraceID, span.Context().TraceID())

		// also propagate through DataDog's standard headers
		if err := tracer.Inject(span.Context(), tracer.HTTPHeadersCarrier(req.Header)); err != nil {
			// this should never happen
			logger.FromContext(ctx).Warn("failed to inject trace header", logger.Error(err))
		}
		return nil
	}

	afterResponse := func(c *resty.Client, resp *resty.Response) error {
		span, ok := tracer.SpanFromContext(resp.Request.Context())
		if !ok {
			return nil // No span found, skip
		}
		span.SetTag(ext.HTTPCode, resp.StatusCode())
		span.SetTag("http.response_size", len(resp.Body()))

		if resp.StatusCode() >= 400 {
			span.SetTag(ext.Error, true)
			span.SetTag(ext.ErrorMsg, fmt.Sprintf("HTTP %d: %s", resp.StatusCode(), resp.Status()))
		}
		span.Finish()

		return nil
	}

	return beforeRequest, afterResponse
}

//func RestyCorrelationMiddleware() resty.RequestMiddleware {
//	return func(c *resty.Client, req *resty.Request) error {
//		// Generate a correlation header from the current context
//		header := correlation.Generate(req.Context())
//		// Add the correlation header to outgoing metadata if one was generated
//		if header != "" {
//			ctx = metadata.AppendToOutgoingContext(ctx, correlation.ContextCorrelationHeader, header)
//		}
//		// Continue with the actual gRPC call using the updated context
//	}
//}
