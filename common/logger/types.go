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
	String     = zap.String
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

const (
	DebugLevel  = Level(zapcore.DebugLevel)
	InfoLevel   = Level(zapcore.InfoLevel)
	WarnLevel   = Level(zapcore.WarnLevel)
	ErrorLevel  = Level(zapcore.ErrorLevel)
	DPanicLevel = Level(zapcore.DPanicLevel)
	PanicLevel  = Level(zapcore.PanicLevel)
	FatalLevel  = Level(zapcore.FatalLevel)
)

func LevelFromString(s string) Level {
	switch strings.ToLower(s) {
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn", "warning":
		return WarnLevel
	case "error":
		return ErrorLevel
	case "dpanic":
		return DPanicLevel
	case "panic":
		return PanicLevel
	case "fatal":
		return FatalLevel
	default:
		return InfoLevel // default to info
	}
}

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
