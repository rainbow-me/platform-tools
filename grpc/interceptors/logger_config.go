package interceptors

import (
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

const (
	DefaultInterceptorLogLevel      zapcore.Level = zapcore.InfoLevel
	DefaultInterceptorErrorLogLevel zapcore.Level = zapcore.WarnLevel
)

type LoggingInterceptorConfig struct {
	LogEnabled         bool
	LogParams          bool // Logs both request and response.
	LogRequests        bool
	LogResponses       bool
	LogParamsBlocklist []fieldmaskpb.FieldMask
	LogLevel           zapcore.Level
	ErrorLogLevel      zapcore.Level

	// If set, overrides ErrorLogLevel for specified gRPC codes. All other codes will be logged with ErrorLogLevel.
	// Setting code.OK here will have no effect (LogLevel will still be followed)
	GrpcCodeLogLevel map[codes.Code]zapcore.Level

	skipLoggingByMethod map[string]struct{}
}

type LoggingInterceptorOption func(*LoggingInterceptorConfig)

func LogParams(v bool) LoggingInterceptorOption {
	return func(o *LoggingInterceptorConfig) {
		o.LogParams = v
	}
}

func LogEnabled(v bool) LoggingInterceptorOption {
	return func(o *LoggingInterceptorConfig) {
		o.LogEnabled = v
	}
}

func LogRequests(v bool) LoggingInterceptorOption {
	return func(o *LoggingInterceptorConfig) {
		o.LogRequests = v
	}
}

func LogResponses(v bool) LoggingInterceptorOption {
	return func(o *LoggingInterceptorConfig) {
		o.LogResponses = v
	}
}

func LogLevel(level zapcore.Level) LoggingInterceptorOption {
	return func(o *LoggingInterceptorConfig) {
		o.LogLevel = level
	}
}

func GrpcCodeLogLevel(errorCodeLogLevel map[codes.Code]zapcore.Level) LoggingInterceptorOption {
	return func(o *LoggingInterceptorConfig) {
		o.GrpcCodeLogLevel = errorCodeLogLevel
	}
}

func ErrorLogLevel(level zapcore.Level) LoggingInterceptorOption {
	return func(o *LoggingInterceptorConfig) {
		o.ErrorLogLevel = level
	}
}

func WithSkippedLogsByMethods(methods ...string) LoggingInterceptorOption {
	skipLoggingByMethod := make(map[string]struct{}, len(methods))
	for _, method := range methods {
		skipLoggingByMethod[method] = struct{}{}
	}
	return func(o *LoggingInterceptorConfig) {
		o.skipLoggingByMethod = skipLoggingByMethod
	}
}

func interceptorConfig(opts ...LoggingInterceptorOption) *LoggingInterceptorConfig {
	cfg := &LoggingInterceptorConfig{
		LogEnabled:         true,
		LogParams:          false,
		LogRequests:        false,
		LogResponses:       false,
		LogParamsBlocklist: nil,
		LogLevel:           DefaultInterceptorLogLevel,
		ErrorLogLevel:      DefaultInterceptorErrorLogLevel,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}
