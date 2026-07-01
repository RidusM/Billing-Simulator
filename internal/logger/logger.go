package logger

import (
	"context"
	"time"
)

type Level int

const (
	Debug Level = iota - 4
	Info
	Warn
	Error
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
	case Debug:
		return "DEBUG"
	case Info:
		return "INFO"
	case Warn:
		return "WARN"
	case Error:
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

func Int8(key string, value int8) Attr {
	return Attr{Key: key, Value: value}
}

func Int16(key string, value int16) Attr {
	return Attr{Key: key, Value: value}
}

func Int32(key string, value int32) Attr {
	return Attr{Key: key, Value: value}
}

func Int64(key string, value int64) Attr {
	return Attr{Key: key, Value: value}
}

func Uint(key string, value uint) Attr {
	return Attr{Key: key, Value: value}
}

func Uint8(key string, value uint8) Attr {
	return Attr{Key: key, Value: value}
}

func Uint16(key string, value uint16) Attr {
	return Attr{Key: key, Value: value}
}

func Uint32(key string, value uint32) Attr {
	return Attr{Key: key, Value: value}
}

func Uint64(key string, value uint64) Attr {
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

func Slice[T any](key string, value []T) Attr {
	return Attr{Key: key, Value: value}
}
