package logger

import (
	"context"
	"time"
)

type Level int

const (
	DebugLevel Level = iota - 4
	InfoLevel
	WarnLevel
	ErrorLevel
)

type Attr struct {
	Key   string
	Value any
}

type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)

	Ctx(ctx context.Context) Logger

	With(args ...any) Logger
	WithGroup(name string) Logger

	LogRequest(ctx context.Context, method, path string, status int, duration time.Duration)
	Log(level Level, msg string, attrs ...Attr)
	LogAttrs(ctx context.Context, level Level, msg string, attrs ...Attr)
}

func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

func String(key string, value string) Attr {
	return Attr{Key: key, Value: value}
}

func Int(key string, value int) Attr {
	return Attr{Key: key, Value: value}
}

func Int64(key string, value int64) Attr {
	return Attr{Key: key, Value: value}
}

func Uint(key string, value uint) Attr {
	return Attr{Key: key, Value: value}
}

func Bool(key string, value bool) Attr {
	return Attr{Key: key, Value: value}
}

func Time(key string, value time.Time) Attr {
	return Attr{Key: key, Value: value}
}

func Duration(key string, value time.Duration) Attr {
	return Attr{Key: key, Value: value}
}

func Any(key string, value any) Attr {
	return Attr{Key: key, Value: value}
}

func Error(value error) Attr {
    return Attr{Key: "error", Value: value}
}
