package logger

import (
	"fmt"
	"maps"
	"os"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/rainbow-me/platform-tools/common/env"
)

const (
	MessageKey    = "message"
	StacktraceKey = "stacktrace"
	PanicValueKey = "panic_value"
	PanicTypeKey  = "panic_type"

	traceIDKey = "dd.trace_id"
	spanIDKey  = "dd.span_id"
)

var (
	zLog   *Logger
	errLog error
	once   sync.Once
)

// Logger abstracts away the underlying log implementation, by exposing our own methods, which allow both structured
// and unstructured logging. It is environment aware and customizes options accordingly.
// For example, it will print human-readable, tab-separated log lines for local environment and json logs in non-local.
// By default, the log level is DEBUG in all nonprod environments, and INFO in prod.
// Such behaviour cna be overridden by setting the LOG_LEVEL env var, see LevelFromString.
type Logger struct {
	root    *zap.Logger
	zap     *zap.Logger
	fields  map[string]Field
	options []Option
}

func NewLogger(zap *zap.Logger) *Logger {
	return &Logger{
		root:    zap,
		zap:     zap,
		fields:  make(map[string]Field),
		options: []Option{},
	}
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

func (l *Logger) Fatal(msg string, fields ...Field) {
	l.zap.Fatal(msg, fields...)
}

func (l *Logger) Fatalf(msg string, v ...any) {
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
	return l.clone(fields, nil)
}

type Option zap.Option

func (l *Logger) WithOptions(options ...Option) *Logger {
	return l.clone(nil, options)
}

func (l *Logger) clone(newFields []Field, newOptions []Option) *Logger {
	newLogger := &Logger{
		root:    l.root,
		fields:  maps.Clone(l.fields),
		options: l.options,
	}
	// overwrite fields and transform to list
	for _, field := range newFields {
		newLogger.fields[field.Key] = field
	}
	fields := lo.Values(newLogger.fields)

	// append options
	if len(newOptions) > 0 {
		newLogger.options = append(newLogger.options, newOptions...)
	}
	options := lo.Map(newLogger.options, func(item Option, _ int) zap.Option {
		return item
	})

	// create a new zap with all the fields and options
	newLogger.zap = l.root.With(fields...).WithOptions(options...)

	return newLogger
}

// Init can be called early during application bootstrap to initialize a logger,
// but it's optional as the first call to Instance will trigger this.
// Note that in case of failure to instantiate a Logger, this will panic.
func Init() error {
	once.Do(func() {
		var err error
		currentEnv := env.GetApplicationEnvOrDefault(env.EnvironmentDevelopment)
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
		config.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder // human-readable log lines
		options = append(options, zap.AddStacktrace(zap.PanicLevel))

	case env.EnvironmentLocalDocker, env.EnvironmentDevelopment, env.EnvironmentStaging:
		// Development/Staging: JSON logs with structured metadata
		config = zap.NewProductionConfig()
		config.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
		config.EncoderConfig = encoderConfig
		options = append(options, zap.AddStacktrace(zap.PanicLevel))

	case env.EnvironmentProduction:
		// Production: JSON logs with structured metadata
		config = zap.NewProductionConfig()
		config.EncoderConfig = encoderConfig
		config.Level.SetLevel(zap.InfoLevel)
		options = append(options, zap.AddStacktrace(zap.PanicLevel))
	}

	// Override log level based on env var, if present
	if lvl, ok := LevelFromString(os.Getenv("LOG_LEVEL")); ok {
		config.Level.SetLevel(zapcore.Level(lvl))
	}

	// Build the logger
	z, err := config.Build(options...)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	return NewLogger(z), nil
}
