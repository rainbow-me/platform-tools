package interceptors

import (
	"time"

	grpctrace "github.com/DataDog/dd-trace-go/contrib/google.golang.org/grpc/v2"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc/codes"

	"github.com/rainbow-me/platform-tools/common/logger"
)

const (
	healthCheckMethod = "/grpc.health.v1.Health/Check"
)

// Config holds essential configuration options for the interceptor chain.
// Uses sensible defaults and can be customized with functional options.
type Config struct {
	// Core settings
	RequestTimeout time.Duration
	Environment    string
	ServiceName    string

	// Feature flags
	PanicRecoveryEnabled bool

	// Logging options - uses existing LoggingInterceptorOption functions
	LoggingOptions []LoggingInterceptorOption
}

// ConfigOption is a functional option for configuring the interceptor chain
type ConfigOption func(*Config)

// WithRequestTimeout sets the server-side request timeout duration
func WithRequestTimeout(timeout time.Duration) ConfigOption {
	return func(c *Config) {
		c.RequestTimeout = timeout
	}
}

// WithPanicRecovery enables or disables panic recovery interceptor
func WithPanicRecovery() ConfigOption {
	return func(c *Config) {
		c.PanicRecoveryEnabled = true
	}
}

// WithLoggingOptions sets logging configuration using existing LoggingInterceptorOption functions
func WithLoggingOptions(opts ...LoggingInterceptorOption) ConfigOption {
	return func(c *Config) {
		c.LoggingOptions = append(c.LoggingOptions, opts...)
	}
}

// Convenience logging configuration functions

// WithBasicLogging enables logging with basic configuration
func WithBasicLogging(enabled bool, level zapcore.Level) ConfigOption {
	return WithLoggingOptions(
		LogEnabled(enabled),
		LogLevel(level),
		ErrorLogLevel(zapcore.ErrorLevel),
		LogRequests(false),
		LogResponses(false),
		LogParams(false),
	)
}

// WithDetailedLogging enables detailed logging with request/response payloads
func WithDetailedLogging() ConfigOption {
	return WithLoggingOptions(
		LogEnabled(true),
		LogLevel(zapcore.InfoLevel),
		ErrorLogLevel(zapcore.ErrorLevel),
		LogParams(true),
		LogRequests(true),
		LogResponses(true),
	)
}

// NewConfig creates a new configuration with sensible defaults
func NewConfig(serviceName, environment string, opts ...ConfigOption) *Config {
	// Set sensible defaults
	config := &Config{
		RequestTimeout:       30 * time.Second,
		ServiceName:          serviceName,
		Environment:          environment,
		PanicRecoveryEnabled: true,
		LoggingOptions: []LoggingInterceptorOption{
			LogEnabled(true),
			LogLevel(zapcore.InfoLevel),
			LogRequests(false),
			LogResponses(false),                         // Default to false to avoid logging sensitive data
			WithSkippedLogsByMethods(healthCheckMethod), // Skip health check method by default
			GrpcCodeLogLevel(
				map[codes.Code]zapcore.Level{ //nolint:exhaustive
					codes.Canceled: zapcore.WarnLevel, // Handle cancellations as warnings
				},
			),
		},
	}

	// Apply functional options
	for _, opt := range opts {
		opt(config)
	}

	return config
}

// NewDefaultServerUnaryChain creates a server interceptor chain with sensible defaults.
// Can be customized using functional options.
//
// Example usage:
//
//	// Customized usage
//	chain := NewDefaultServerUnaryChain(logger,
//	    "test-service",
//	    "production",
//	    logger,
//	    WithRequestTimeout(60 * time.Second),
//	    WithDetailedLogging(),
//	    WithPanicRecovery(),
//	)
func NewDefaultServerUnaryChain(
	serviceName,
	environment string,
	logger logger.Logger,
	opts ...ConfigOption,
) *UnaryServerInterceptorChain {
	// Create configuration with provided options
	cfg := NewConfig(serviceName, environment, opts...)

	// Create the interceptor chain
	chain := NewUnaryServerInterceptorChain()

	// Add request timeout interceptor if configured
	if cfg.RequestTimeout > 0 {
		chain.Push("server-deadline", ServerDeadlineInterceptor(cfg.RequestTimeout))
	}

	// Add tracing interceptor
	chain.Push("trace", grpctrace.UnaryServerInterceptor(
		grpctrace.WithService(cfg.ServiceName),
		grpctrace.WithAnalytics(true),
		grpctrace.WithMetadataTags(),
		grpctrace.WithUntracedMethods(healthCheckMethod),
	))

	chain.Push("request-context", RequestContextUnaryServerInterceptor())
	chain.Push("headers", ResponseHeadersInterceptor())

	// Add logging interceptor if logger is provided
	if logger != nil {
		chain.Push("logger", UnaryLoggerServerInterceptor(logger, cfg.LoggingOptions...))
	}

	// add errors handling
	chain.Push("errors", UnaryErrorServerInterceptor)

	// Add panic recovery interceptor if enabled
	if cfg.PanicRecoveryEnabled {
		chain.Push("panic-recovery", UnaryPanicRecoveryServerInterceptor(logger))
	}

	// Add context status interceptor
	chain.Push("context-status", UnaryContextStatusInterceptor())

	return chain
}

func NewDefaultClientUnaryChain(
	serviceName string,
	logger logger.Logger,
	loggerOpts ...LoggingInterceptorOption,
) *UnaryClientInterceptorChain {
	chain := NewUnaryClientInterceptorChain()
	chain.Push("tracer", grpctrace.UnaryClientInterceptor(
		grpctrace.WithService(serviceName),
		grpctrace.WithAnalytics(true),
	))

	// Added after trace so that a current span is active.
	chain.Push("request-context", UnaryRequestContextClientInterceptor)
	chain.Push("correlation-context", UnaryCorrelationClientInterceptor)
	chain.Push("upstream-info", UnaryUpstreamInfoClientInterceptor(serviceName))
	chain.Push("logger", UnaryLoggerClientInterceptor(logger, loggerOpts...))

	return chain
}

// Convenience functions for common configurations

// NewProductionServerChain creates a production-ready server interceptor chain
func NewProductionServerChain(serviceName, environment string, logger logger.Logger) *UnaryServerInterceptorChain {
	return NewDefaultServerUnaryChain(serviceName, environment, logger,
		WithRequestTimeout(30*time.Second),
		WithBasicLogging(true, zapcore.InfoLevel),
		WithPanicRecovery(),
		WithLoggingOptions(WithSkippedLogsByMethods(healthCheckMethod)),
	)
}

// NewDevelopmentServerChain creates a development-friendly server interceptor chain
func NewDevelopmentServerChain(serviceName, environment string, logger logger.Logger) *UnaryServerInterceptorChain {
	return NewDefaultServerUnaryChain(serviceName, environment, logger,
		WithRequestTimeout(60*time.Second),
		WithDetailedLogging(),
		WithPanicRecovery(),
		WithLoggingOptions(LogLevel(zapcore.DebugLevel)),
	)
}

// NewMinimalServerChain creates a minimal server interceptor chain for testing
func NewMinimalServerChain(serviceName, environment string, logger logger.Logger) *UnaryServerInterceptorChain {
	return NewDefaultServerUnaryChain(serviceName, environment, logger,
		WithBasicLogging(false, zapcore.InfoLevel),
		WithPanicRecovery(),
	)
}
