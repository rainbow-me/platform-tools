package interceptors

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/rainbow-me/platform-tools/common/logger"
)

const (
	DefaultInterceptorLogLevel      logger.Level = logger.InfoLevel
	DefaultInterceptorErrorLogLevel logger.Level = logger.WarnLevel
)

type LoggingInterceptorConfig struct {
	Environment        string
	LogEnabled         bool
	LogParams          bool // Logs both request and response.
	LogRequests        bool
	LogResponses       bool
	LogParamsBlocklist []fieldmaskpb.FieldMask
	LogLevel           logger.Level
	ErrorLogLevel      logger.Level

	// If set, overrides ErrorLogLevel for specified gRPC codes. All other codes will be logged with ErrorLogLevel.
	// Setting code.OK here will have no effect (LogLevel will still be followed)
	GrpcCodeLogLevel map[codes.Code]logger.Level

	skipLoggingByMethod map[string]struct{}

	// skip logging by environment and code
	skipLoggingByEnvAndCode map[string]map[codes.Code]struct{}
}

type LoggingInterceptorOption func(*LoggingInterceptorConfig)

func LogParams(v bool) LoggingInterceptorOption {
	return func(o *LoggingInterceptorConfig) {
		o.LogParams = v
	}
}

func Environment(e string) LoggingInterceptorOption {
	return func(o *LoggingInterceptorConfig) {
		o.Environment = e
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

func LogLevel(level logger.Level) LoggingInterceptorOption {
	return func(o *LoggingInterceptorConfig) {
		o.LogLevel = level
	}
}

func GrpcCodeLogLevel(errorCodeLogLevel map[codes.Code]logger.Level) LoggingInterceptorOption {
	return func(o *LoggingInterceptorConfig) {
		o.GrpcCodeLogLevel = errorCodeLogLevel
	}
}

func ErrorLogLevel(level logger.Level) LoggingInterceptorOption {
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

func WithSkippedLogsForEnv(env string, grpcCodes ...codes.Code) LoggingInterceptorOption {
	return func(o *LoggingInterceptorConfig) {
		if o.skipLoggingByEnvAndCode == nil {
			o.skipLoggingByEnvAndCode = make(map[string]map[codes.Code]struct{})
		}
		if _, exists := o.skipLoggingByEnvAndCode[env]; !exists {
			o.skipLoggingByEnvAndCode[env] = make(map[codes.Code]struct{})
		}
		for _, code := range grpcCodes {
			o.skipLoggingByEnvAndCode[env][code] = struct{}{}
		}
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
