package logger

import (
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
