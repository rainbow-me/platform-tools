package logger

import (
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Field = zap.Field

var (
	Any        = zap.Any
	Array      = zap.Array
	Bool       = zap.Bool
	ByteString = zap.ByteString
	Complex128 = zap.Complex128
	Complex64  = zap.Complex64
	Duration   = zap.Duration
	Float64    = zap.Float64
	Float32    = zap.Float32
	Int        = zap.Int
	Int64      = zap.Int64
	Int32      = zap.Int32
	Int16      = zap.Int16
	Int8       = zap.Int8
	Object     = zap.Object
	String     = zap.String
	Stringer   = zap.Stringer
	Uint       = zap.Uint
	Uint64     = zap.Uint64
	Uint32     = zap.Uint32
	Uint16     = zap.Uint16
	Uint8      = zap.Uint8
	Uintptr    = zap.Uintptr
	Error      = zap.Error
	Errors     = zap.Errors
)

type Level zapcore.Level

func (l Level) Enabled(lvl zapcore.Level) bool {
	return Level(lvl) >= l
}

const (
	DebugLevel  = Level(zapcore.DebugLevel)
	InfoLevel   = Level(zapcore.InfoLevel)
	WarnLevel   = Level(zapcore.WarnLevel)
	ErrorLevel  = Level(zapcore.ErrorLevel)
	DPanicLevel = Level(zapcore.DPanicLevel)
	PanicLevel  = Level(zapcore.PanicLevel)
	FatalLevel  = Level(zapcore.FatalLevel)
)

// LevelFromString returns the level associated with the string argument in a case-insensitive manner.
// If the string is not recognised as a valid log level, it defaults to InfoLevel and returns false in the second
// return parameter.
func LevelFromString(s string) (Level, bool) {
	switch strings.ToLower(s) {
	case "debug":
		return DebugLevel, true
	case "info":
		return InfoLevel, true
	case "warn", "warning":
		return WarnLevel, true
	case "error":
		return ErrorLevel, true
	case "dpanic":
		return DPanicLevel, true
	case "panic":
		return PanicLevel, true
	case "fatal":
		return FatalLevel, true
	default:
		return InfoLevel, false
	}
}

var AddStackTrace = zap.AddStacktrace

func WithTrace(ctx *tracer.SpanContext) []Field {
	return []Field{
		String(traceIDKey, ctx.TraceID()),
		String(spanIDKey, strconv.FormatUint(ctx.SpanID(), 10)),
	}
}

func WithPanic(p any) []Field {
	msg := extractPanicMessage(p)
	panicType := extractPanicType(msg)
	return []Field{
		String(PanicValueKey, msg),
		String(PanicTypeKey, panicType),
		ByteString(StacktraceKey, debug.Stack()),
	}
}

// extractPanicMessage safely extracts a string representation of the panic value
func extractPanicMessage(panicValue any) string {
	if panicValue == nil {
		return "unknown panic (nil value)"
	}
	return fmt.Sprintf("%v", panicValue)
}

// extractPanicType safely extracts the type information of the panic value
func extractPanicType(panicValue any) string {
	if panicValue == nil {
		return "nil"
	}
	return fmt.Sprintf("%T", panicValue)
}
