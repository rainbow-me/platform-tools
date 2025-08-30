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
	Any         = zap.Any
	Array       = zap.Array
	Bool        = zap.Bool
	Bools       = zap.Bools
	ByteString  = zap.ByteString
	ByteStrings = zap.ByteStrings
	Complex128  = zap.Complex128
	Complex128s = zap.Complex128s
	Complex64   = zap.Complex64
	Complex64s  = zap.Complex64s
	Duration    = zap.Duration
	Durations   = zap.Durations
	Float64     = zap.Float64
	Float64s    = zap.Float64s
	Float32     = zap.Float32
	Float32s    = zap.Float32s
	Int         = zap.Int
	Ints        = zap.Ints
	Int64       = zap.Int64
	Int64s      = zap.Int64s
	Int32       = zap.Int32
	Int32s      = zap.Int32s
	Int16       = zap.Int16
	Int16s      = zap.Int16s
	Int8        = zap.Int8
	Int8s       = zap.Int8s
	Object      = zap.Object
	String      = zap.String
	Strings     = zap.Strings
	Stringer    = zap.Stringer
	Uint        = zap.Uint
	Uints       = zap.Uints
	Uint64      = zap.Uint64
	Uint64s     = zap.Uint64s
	Uint32      = zap.Uint32
	Uint32s     = zap.Uint32s
	Uint16      = zap.Uint16
	Uint16s     = zap.Uint16s
	Uint8       = zap.Uint8
	Uint8s      = zap.Uint8s
	Uintptr     = zap.Uintptr
	Uintptrs    = zap.Uintptrs
	Error       = zap.Error
	Errors      = zap.Errors
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
