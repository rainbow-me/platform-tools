package gin

import (
	"time"

	"github.com/gin-gonic/gin"
)

const (
	httpHandlerOp = "http.handler"
	componentName = "gin"
)

type interceptorCfg struct {
	TracingEnabled     bool
	CorrelationEnabled bool
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
		ErrorHandlingMiddleware,
	}
	if cfg.TracingEnabled {
		middlewares = append(middlewares, TracingMiddleware)
	}
	if cfg.CorrelationEnabled {
		middlewares = append(middlewares, CorrelationMiddleware)
	}
	middlewares = append(middlewares, TimeoutMiddleware(cfg.Timeout))

	return middlewares
}
