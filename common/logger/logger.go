package logger

import (
	"fmt"
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/rainbow-me/platform-tools/common/env"
)

const (
	MessageKey    = "message"
	StacktraceKey = "stacktrace"
)

type Logger interface {
	Log(lvl zapcore.Level, msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Infof(msg string, v ...any)
	Debug(msg string, fields ...Field)
	Debugf(msg string, v ...any)
	Warn(msg string, fields ...Field)
	Warnf(msg string, v ...any)
	Error(msg string, fields ...Field)
	Errorf(msg string, v ...any)
	With(fields ...Field) Logger
	WithOptions(...zap.Option) Logger
}

type Level zapcore.Level

var (
	zLog *ZapLogger
	once sync.Once
)

type ZapLogger struct {
	zap *zap.Logger
}

func NewZapLogger(zap *zap.Logger) *ZapLogger {
	return &ZapLogger{zap: zap}
}

func (l *ZapLogger) Log(lvl zapcore.Level, msg string, fields ...Field) {
	l.zap.Log(lvl, msg, fields...)
}

func (l *ZapLogger) Info(msg string, fields ...Field) {
	l.zap.Info(msg, fields...)
}

func (l *ZapLogger) Infof(msg string, v ...any) {
	if len(v) > 0 {
		l.zap.Sugar().Infof(msg, v...)
	} else {
		l.zap.Info(msg)
	}
}

func (l *ZapLogger) Debug(msg string, fields ...Field) {
	l.zap.Debug(msg, fields...)
}

func (l *ZapLogger) Debugf(msg string, v ...any) {
	if len(v) > 0 {
		l.zap.Sugar().Debugf(msg, v...)
	} else {
		l.zap.Debug(msg)
	}
}

func (l *ZapLogger) Warn(msg string, fields ...Field) {
	l.zap.Warn(msg, fields...)
}

func (l *ZapLogger) Warnf(msg string, v ...any) {
	if len(v) > 0 {
		l.zap.Sugar().Warnf(msg, v...)
	} else {
		l.zap.Warn(msg)
	}
}

func (l *ZapLogger) Error(msg string, fields ...Field) {
	l.zap.Error(msg, fields...)
}

func (l *ZapLogger) Errorf(msg string, v ...any) {
	if len(v) > 0 {
		l.zap.Sugar().Errorf(msg, v...)
	} else {
		l.zap.Error(msg)
	}
}

func (l *ZapLogger) With(fields ...Field) Logger {
	return &ZapLogger{zap: l.zap.With(fields...)}
}

func (l *ZapLogger) WithOptions(option ...zap.Option) Logger {
	return &ZapLogger{zap: l.zap.WithOptions(option...)}
}

// Init can be called early during application bootstrap to initialize a logger,
// but it's optional as the first call to Instance will trigger this.
// Note that in case of failure to instantiate a Logger, this will panic.
func Init() {
	once.Do(func() {
		currentEnv, err := env.FromString(os.Getenv(env.ApplicationEnvKey))
		if err != nil {
			panic(fmt.Errorf("invalid environment: %w", err))
		}
		zLog, err = newZapLogger(currentEnv)
		if err != nil {
			panic(fmt.Errorf("failed to initialize logger: %w", err))
		}
	})
}

// Instance returns the singleton instance of the application logger, initializing it on the first invocation.
// Note that in case of failure to instantiate a Logger, this will panic.
func Instance() Logger {
	if zLog == nil {
		Init()
	}
	return zLog
}

// NoOp returns a "no operations" logger, which swallows statements without printing anything
func NoOp() Logger {
	return NewZapLogger(zap.NewNop())
}

func newZapLogger(environment env.Environment) (*ZapLogger, error) {
	var (
		config  zap.Config
		options []zap.Option
	)

	// Define a consistent encoder configuration
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey, // Hide function name for brevity
		MessageKey:     MessageKey,
		StacktraceKey:  StacktraceKey,
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeTime:     zapcore.ISO8601TimeEncoder,  // Use human-readable timestamp format
		EncodeLevel:    zapcore.CapitalLevelEncoder, // INFO, WARN, ERROR, etc.
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder, // Short file path
	}

	// Configure logging based on the environment
	switch environment {
	case env.EnvironmentLocal:
		config = zap.NewDevelopmentConfig()
		options = append(options, zap.AddStacktrace(zap.ErrorLevel))

	case env.EnvironmentLocalDocker, env.EnvironmentDevelopment, env.EnvironmentStaging:
		// Development/Staging: JSON logs for Datadog ingestion
		config = zap.NewProductionConfig()
		config.EncoderConfig = encoderConfig
		options = append(options, zap.AddStacktrace(zap.ErrorLevel))

	case env.EnvironmentProduction:
		// Production: JSON logs with structured metadata
		config = zap.NewProductionConfig()
		config.EncoderConfig = encoderConfig
		config.Level.SetLevel(zap.InfoLevel)
		options = append(options, zap.AddStacktrace(zap.ErrorLevel))
	}

	// Build the logger
	z, err := config.Build(options...)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	return NewZapLogger(z), nil
}
