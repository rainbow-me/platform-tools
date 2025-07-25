package logger

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/rainbow-me/platform-tools/common/env"
)

const (
	StringJSONEncoderName = "string_json"
	MessageKey            = "message"
)

type stringJSONEncoder struct {
	zapcore.Encoder
}

func newStringJSONEncoder(cfg zapcore.EncoderConfig) *stringJSONEncoder {
	return &stringJSONEncoder{zapcore.NewJSONEncoder(cfg)}
}

// NewStringJSONEncoder returns an encoder that encodes the JSON log dict as a string
// so the log processing pipeline can correctly process logs with nested JSON.
func NewStringJSONEncoder(cfg zapcore.EncoderConfig) (zapcore.Encoder, error) {
	return newStringJSONEncoder(cfg), nil
}

// InitLogger initializes and returns a configured Zap logger with environment-specific settings. and other
func InitLogger(zapOpts ...zap.Option) (*zap.Logger, error) {
	var (
		config  zap.Config
		options []zap.Option
	)

	// Retrieve current environment
	currentEnv := os.Getenv(env.ApplicationEnvKey)
	if err := env.IsEnvironmentValid(currentEnv); err != nil {
		return nil, fmt.Errorf("invalid environment: %w", err)
	}

	// Register custom JSON encoder
	if err := zap.RegisterEncoder(StringJSONEncoderName, NewStringJSONEncoder); err != nil {
		return nil, fmt.Errorf("failed to register string JSON encoder: %w", err)
	}

	// Define a consistent encoder configuration
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:       "timestamp",
		LevelKey:      "level",
		NameKey:       "logger",
		CallerKey:     "caller",
		FunctionKey:   zapcore.OmitKey, // Hide function name for brevity
		MessageKey:    MessageKey,
		StacktraceKey: "stacktrace",
		EncodeTime:    zapcore.ISO8601TimeEncoder,  // Use human-readable timestamp format
		EncodeLevel:   zapcore.CapitalLevelEncoder, // INFO, WARN, ERROR, etc.
		EncodeCaller:  zapcore.ShortCallerEncoder,  // Short file path
	}

	// Configure logging based on the environment
	switch currentEnv {
	case string(env.EnvironmentLocal):
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.MessageKey = MessageKey
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		options = append(options, zap.AddStacktrace(zap.ErrorLevel))

	case string(env.EnvironmentLocalDocker), string(env.EnvironmentDevelopment), string(env.EnvironmentStaging):
		// Development/Staging: JSON logs for Datadog ingestion
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig = encoderConfig
		config.Encoding = StringJSONEncoderName
		options = append(options, zap.AddStacktrace(zap.ErrorLevel))

	case string(env.EnvironmentProduction):
		// Production: JSON logs with structured metadata
		config = zap.NewProductionConfig()
		config.EncoderConfig = encoderConfig
		config.Encoding = StringJSONEncoderName
		config.Level.SetLevel(zap.InfoLevel)
		options = append(options, zap.AddStacktrace(zap.ErrorLevel))
	}

	// Apply additional logging options if provided
	if len(zapOpts) > 0 {
		options = append(options, zapOpts...)
	}

	// Build the logger
	logger, err := config.Build(options...)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	return logger, nil
}
