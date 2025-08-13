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
		PanicRecoveryMiddleware,
		ErrorHandlingMiddleware,
	}
	if cfg.TracingEnabled {
		middlewares = append(middlewares, TracingMiddleware)
	}
	if cfg.CorrelationEnabled {
		middlewares = append(middlewares, CorrelationMiddleware)
	}
	if cfg.CompressionLevel != gzip.NoCompression {
		middlewares = append(middlewares, gzip.Gzip(cfg.CompressionLevel))
	}
	middlewares = append(middlewares, TimeoutMiddleware(cfg.Timeout))

	return middlewares
}
