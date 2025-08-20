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
	HttpDebug          bool //
	HttpTrace          bool // set to true as well to print every http request and response to logs
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

// WithHttpDebug enables printing log line with request info and duration for every request
func WithHttpDebug() InterceptorOpt {
	return func(cfg *interceptorCfg) {
		cfg.HttpDebug = true
	}
}

// WithHttpTrace enables deeper http debugging by also printing the whole request and response body
func WithHttpTrace() InterceptorOpt {
	return func(cfg *interceptorCfg) {
		cfg.HttpDebug = true
		cfg.HttpTrace = true
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
	middlewares := []gin.HandlerFunc{
		RequestLogging(loggingCfg{
			debug: cfg.HttpDebug,
			trace: cfg.HttpTrace,
		}),
		PanicRecoveryMiddleware,
		ErrorHandlingMiddleware,
	}
	if cfg.TracingEnabled {
		middlewares = append(middlewares, TracingMiddleware)
	}
	if cfg.CorrelationEnabled {
		middlewares = append(middlewares, CorrelationMiddleware)
		middlewares = append(middlewares, RequestInfoMiddleware)
	}
	if cfg.CompressionLevel != gzip.NoCompression {
		middlewares = append(middlewares, gzip.Gzip(cfg.CompressionLevel))
	}
	middlewares = append(middlewares, TimeoutMiddleware(cfg.Timeout))

	return middlewares
}
