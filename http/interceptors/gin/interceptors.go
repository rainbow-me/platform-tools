package gin

import (
	"time"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

const (
	httpHandlerOp = "http.handler"
	componentName = "gin"
)

type interceptorCfg struct {
	TracingEnabled     bool
	CorrelationEnabled bool
	CompressionLevel   int
	HTTPDebug          bool
	HTTPTrace          bool
	Timeout            time.Duration
}

type InterceptorOpt func(cfg *interceptorCfg)

// WithCorrelationEnabled enables/disables correlation. Default is enabled.
func WithCorrelationEnabled(enabled bool) InterceptorOpt {
	return func(cfg *interceptorCfg) {
		cfg.CorrelationEnabled = enabled
	}
}

// WithTimeout sets the http handler timeout. Default is 1 minute.
func WithTimeout(timeout time.Duration) InterceptorOpt {
	return func(cfg *interceptorCfg) {
		cfg.Timeout = timeout
	}
}

// WithTracingEnabled enables/disables tracing. Default is enabled.
func WithTracingEnabled(enabled bool) InterceptorOpt {
	return func(cfg *interceptorCfg) {
		cfg.TracingEnabled = enabled
	}
}

// WithHTTPDebug enables printing log line with request info and duration for every request
func WithHTTPDebug() InterceptorOpt {
	return func(cfg *interceptorCfg) {
		cfg.HTTPDebug = true
	}
}

// WithHTTPTrace enables deeper http debugging by also printing the whole request and response body
func WithHTTPTrace() InterceptorOpt {
	return func(cfg *interceptorCfg) {
		cfg.HTTPDebug = true
		cfg.HTTPTrace = true
	}
}

// WithCompressionLevel specifies the gzip compression level, default is gzip.DefaultCompression.
// Disable by using gzip.NoCompression.
func WithCompressionLevel(level int) InterceptorOpt {
	return func(cfg *interceptorCfg) {
		cfg.CompressionLevel = level
	}
}

// DefaultInterceptors returns all our default interceptors for Gin servers.
// Defaults can be changed by passing any of the WithXXX options.
func DefaultInterceptors(opts ...InterceptorOpt) []gin.HandlerFunc {
	cfg := &interceptorCfg{
		TracingEnabled:     true,
		CorrelationEnabled: true,
		Timeout:            time.Minute,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	// Start at top level with panic interceptor to avoid crashing the process
	middlewares := []gin.HandlerFunc{PanicRecoveryMiddleware}

	// Add tracing/correlation interceptors first so any further interceptor prints logs with trace/correlation IDs
	if cfg.TracingEnabled {
		middlewares = append(middlewares, TracingMiddleware)
	}
	if cfg.CorrelationEnabled {
		middlewares = append(middlewares, CorrelationMiddleware)
		middlewares = append(middlewares, RequestInfoMiddleware)
	}
	middlewares = append(middlewares, RequestLogging(loggingCfg{
		debug: cfg.HTTPDebug,
		trace: cfg.HTTPTrace,
	}))
	middlewares = append(middlewares, ErrorHandlingMiddleware)

	if cfg.CompressionLevel != gzip.NoCompression {
		middlewares = append(middlewares, gzip.Gzip(cfg.CompressionLevel))
	}
	middlewares = append(middlewares, TimeoutMiddleware(cfg.Timeout))

	return middlewares
}
