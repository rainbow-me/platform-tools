package resty

import (
	"fmt"
	"net/url"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/ext"
	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/go-resty/resty/v2"

	"github.com/rainbow-me/platform-tools/common/correlation"
	"github.com/rainbow-me/platform-tools/common/headers"
	"github.com/rainbow-me/platform-tools/common/logger"
)

const (
	httpRequestOp      = "http.request"
	restyComponentName = "resty"
)

type interceptorCfg struct {
	TracingEnabled     bool
	CorrelationEnabled bool
	// no timeout specified, that is handled by the underlying http client config
}

type InterceptorOpt func(*interceptorCfg)

// WithCorrelationEnabled enables/disables correlation. Default is enabled.
func WithCorrelationEnabled(enabled bool) InterceptorOpt {
	return func(cfg *interceptorCfg) {
		cfg.CorrelationEnabled = enabled
	}
}

// WithTracingEnabled enables/disables tracing. Default is enabled.
func WithTracingEnabled(enabled bool) InterceptorOpt {
	return func(cfg *interceptorCfg) {
		cfg.TracingEnabled = enabled
	}
}

// InjectInterceptors injects all interceptors required to get Resty requests to propagate traces and correlation info.
// Default behaviour can be changed by passing any of the WithXXX options.
func InjectInterceptors(client *resty.Client, opts ...InterceptorOpt) {
	cfg := &interceptorCfg{
		TracingEnabled:     true,
		CorrelationEnabled: true,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.TracingEnabled {
		before, after := TracingMiddleware()
		client.OnBeforeRequest(before)
		client.OnAfterResponse(after)
	}
	if cfg.CorrelationEnabled {
		client.OnBeforeRequest(CorrelationMiddleware())
	}
}

// TracingMiddleware propagates traces from context to http headers.
// Also, creates a new span and tags it with the http method, url, status code etc.
func TracingMiddleware() (resty.RequestMiddleware, resty.ResponseMiddleware) {
	beforeRequest := func(_ *resty.Client, req *resty.Request) error {
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
		req.SetHeader(headers.HeaderXTraceID, span.Context().TraceID())

		// also propagate through DataDog's standard headers
		if err := tracer.Inject(span.Context(), tracer.HTTPHeadersCarrier(req.Header)); err != nil {
			// this should never happen
			logger.FromContext(ctx).Warn("failed to inject trace header", logger.Error(err))
		}
		return nil
	}

	afterResponse := func(_ *resty.Client, resp *resty.Response) error {
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

func CorrelationMiddleware() resty.RequestMiddleware {
	return func(_ *resty.Client, req *resty.Request) error {
		req.SetHeader(correlation.ContextCorrelationHeader, correlation.Generate(req.Context()))
		req.SetHeader(correlation.RequestIDHeader, correlation.RequestIDFromContext(req.Context()))
		return nil
	}
}
