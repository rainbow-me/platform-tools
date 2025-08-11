package logger

import (
	"fmt"
	"os"
	"sync"

	"github.com/cockroachdb/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/rainbow-me/platform-tools/common/env"
)

const (
	MessageKey    = "message"
	StacktraceKey = "stacktrace"
	traceIDKey    = "dd.trace_id"
	spanIDKey     = "dd.span_id"
)

var (
	zLog   *Logger
	errLog error
	once   sync.Once
)

type Logger struct {
	zap *zap.Logger
}

func NewLogger(zap *zap.Logger) *Logger {
	return &Logger{zap: zap}
}

func (l *Logger) Log(lvl Level, msg string, fields ...Field) {
	l.zap.Log(zapcore.Level(lvl), msg, fields...)
}

func (l *Logger) Info(msg string, fields ...Field) {
	l.zap.Info(msg, fields...)
}

func (l *Logger) Infof(msg string, v ...any) {
	if len(v) > 0 {
		l.zap.Sugar().Infof(msg, v...)
	} else {
		l.zap.Info(msg)
	}
}

func (l *Logger) Debug(msg string, fields ...Field) {
	l.zap.Debug(msg, fields...)
}

func (l *Logger) Debugf(msg string, v ...any) {
	if len(v) > 0 {
		l.zap.Sugar().Debugf(msg, v...)
	} else {
		l.zap.Debug(msg)
	}
}

func (l *Logger) Warn(msg string, fields ...Field) {
	l.zap.Warn(msg, fields...)
}

func (l *Logger) Warnf(msg string, v ...any) {
	if len(v) > 0 {
		l.zap.Sugar().Warnf(msg, v...)
	} else {
		l.zap.Warn(msg)
	}
}

func (l *Logger) Error(msg string, fields ...Field) {
	l.zap.Error(msg, fields...)
}

func (l *Logger) Errorf(msg string, v ...any) {
	if len(v) > 0 {
		l.zap.Sugar().Errorf(msg, v...)
	} else {
		l.zap.Error(msg)
	}
}

func (l *Logger) Fatal(msg string, v ...any) {
	if len(v) > 0 {
		l.zap.Sugar().Fatalf(msg, v...)
	} else {
		l.zap.Fatal(msg)
	}
}

func (l *Logger) Panic(msg string, v ...any) {
	if len(v) > 0 {
		l.zap.Sugar().Panicf(msg, v...)
	} else {
		l.zap.Panic(msg)
	}
}

func (l *Logger) DPanic(msg string, v ...any) {
	if len(v) > 0 {
		l.zap.Sugar().DPanicf(msg, v...)
	} else {
		l.zap.DPanic(msg)
	}
}

func (l *Logger) With(fields ...Field) *Logger {
	return &Logger{zap: l.zap.With(fields...)}
}

func (l *Logger) WithOptions(option ...zap.Option) *Logger {
	return &Logger{zap: l.zap.WithOptions(option...)}
}

// Init can be called early during application bootstrap to initialize a logger,
// but it's optional as the first call to Instance will trigger this.
// Note that in case of failure to instantiate a Logger, this will panic.
func Init() error {
	once.Do(func() {
		currentEnv, err := env.FromString(os.Getenv(env.ApplicationEnvKey))
		if err != nil {
			currentEnv = env.EnvironmentDevelopment
		}
		zLog, err = newZapLogger(currentEnv)
		if err != nil {
			errLog = errors.Wrap(err, "failed to initialize logger")
		}
	})
	return errLog
}

// Instance returns the singleton instance of the application logger, initializing it on the first invocation.
// Note that in case of failure to instantiate a Logger, this will panic.
func Instance() (*Logger, error) {
	var err error
	if zLog == nil {
		err = Init()
	}
	return zLog, err
}

// NoOp returns a "no operations" logger, which swallows statements without printing anything
func NoOp() *Logger {
	return NewLogger(zap.NewNop())
}

func newZapLogger(environment env.Environment) (*Logger, error) {
	var (
		config zap.Config

		// skip the stack trace top entry since zap logger is wrapped by our own logger
		options = []zap.Option{zap.AddCallerSkip(1)}
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
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		options = append(options, zap.AddStacktrace(zap.PanicLevel))

	case env.EnvironmentLocalDocker, env.EnvironmentDevelopment, env.EnvironmentStaging:
		// Development/Staging: JSON logs for Datadog ingestion
		config = zap.NewProductionConfig()
		config.EncoderConfig = encoderConfig
		options = append(options, zap.AddStacktrace(zap.PanicLevel))

	case env.EnvironmentProduction:
		// Production: JSON logs with structured metadata
		config = zap.NewProductionConfig()
		config.EncoderConfig = encoderConfig
		config.Level.SetLevel(zap.InfoLevel)
		options = append(options, zap.AddStacktrace(zap.PanicLevel))
	}

	// Build the logger
	z, err := config.Build(options...)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	return NewLogger(z), nil
}
