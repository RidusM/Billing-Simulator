package logger

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ZapAdapter struct {
	logger *zap.Logger
	sugar  *zap.SugaredLogger
	level  zapcore.Level
}

func NewZapAdapter(appName, env string, opts ...Option) (*ZapAdapter, error) {
	cfg := defaultConfigs()
	for _, opt := range opts {
		opt(cfg)
	}

	logger := newZapLogger(appName, env, cfg)
	return &ZapAdapter{
		logger: logger,
		sugar:  logger.Sugar(),
		level:  toZapLevel(cfg.Level),
	}, nil
}

func newZapLogger(appName, env string, cfg *Config) *zap.Logger {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:       "ts",
		LevelKey:      "level",
		NameKey:       "logger",
		CallerKey:     "caller",
		FunctionKey:   zapcore.OmitKey,
		MessageKey:    "msg",
		StacktraceKey: "stacktrace",
		LineEnding:    zapcore.DefaultLineEnding,
		EncodeLevel:   zapcore.LowercaseLevelEncoder,
		EncodeTime:    zapcore.ISO8601TimeEncoder,
		EncodeCaller:  zapcore.ShortCallerEncoder,
	}

	zapLevel := toZapLevel(cfg.Level)
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(cfg.GetWriter()),
		zapLevel,
	)

	return zap.New(core,
		zap.Fields(
			zap.String("service", appName),
			zap.String("env", env),
		),
		zap.AddCaller(),
		zap.AddStacktrace(zap.ErrorLevel),
	)
}

func (a *ZapAdapter) Debug(msg string, args ...any) {
	a.sugar.WithOptions(zap.AddCallerSkip(1)).Debugw(msg, args...)
}

func (a *ZapAdapter) Info(msg string, args ...any) {
	a.sugar.WithOptions(zap.AddCallerSkip(1)).Infow(msg, args...)
}

func (a *ZapAdapter) Warn(msg string, args ...any) {
	a.sugar.WithOptions(zap.AddCallerSkip(1)).Warnw(msg, args...)
}

func (a *ZapAdapter) Error(msg string, args ...any) {
	a.sugar.WithOptions(zap.AddCallerSkip(1)).Errorw(msg, args...)
}

func (a *ZapAdapter) Ctx(ctx context.Context) Logger {
	requestID := GetRequestID(ctx)
	if requestID == "" {
		return a
	}

	newLogger := a.logger.With(zap.String("request_id", requestID))
	return &ZapAdapter{
		logger: newLogger,
		sugar:  a.sugar.With(zap.String("request_id", requestID)),
		level:  a.level,
	}
}

func (a *ZapAdapter) With(args ...any) Logger {
	if len(args) == 0 {
		return a
	}
	cleanArgs := sanitizeArgs(args)
	fields := toZapFields(cleanArgs)

	return &ZapAdapter{
		logger: a.logger.With(fields...),
		sugar:  a.sugar.With(cleanArgs...),
		level:  a.level,
	}
}

func (a *ZapAdapter) WithGroup(name string) Logger {
	return &ZapAdapter{
		logger: a.logger.With(zap.Namespace(name)),
		sugar:  a.sugar.With(zap.Namespace(name)),
		level:  a.level,
	}
}

func (a *ZapAdapter) Log(level Level, msg string, attrs ...Attr) {
	zapLevel := toZapLevel(level)
	if ce := a.logger.Check(zapLevel, msg); ce != nil {
		ce.Write(toZapFieldsFromAttrs(attrs)...)
	}
}

func (a *ZapAdapter) LogAttrs(ctx context.Context, level Level, msg string, attrs ...Attr) {
	zapLevel := toZapLevel(level)
	ce := a.logger.Check(zapLevel, msg)
	if ce == nil {
		return
	}

	fields := toZapFieldsFromAttrs(attrs)
	if requestID := GetRequestID(ctx); requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}

	ce.Write(fields...)
}

func (a *ZapAdapter) LogRequest(ctx context.Context, method, path string, status int, duration time.Duration) {
	a.LogAttrs(ctx, Info, "request",
		String("method", method),
		String("path", path),
		Int("status", status),
		Duration("duration", duration),
	)
}

func sanitizeArgs(args []any) []any {
	if len(args) == 0 {
		return args
	}

	if len(args)%2 != 0 {
		newArgs := make([]any, len(args), len(args)+1)
		copy(newArgs, args)
		args = append(newArgs, "<missing_value>")
	}

	var copied bool
	for i := 0; i < len(args); i += 2 {
		if _, ok := args[i].(string); !ok {
			if !copied {
				newArgs := make([]any, len(args))
				copy(newArgs, args)
				args = newArgs
				copied = true
			}
			args[i] = fmt.Sprintf("INVALID_KEY_%v", args[i])
		}
	}
	return args
}

func toZapLevel(level Level) zapcore.Level {
	switch level {
	case Debug:
		return zapcore.DebugLevel
	case Info:
		return zapcore.InfoLevel
	case Warn:
		return zapcore.WarnLevel
	case Error:
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func toZapFields(args []any) []zap.Field {
	fields := make([]zap.Field, 0, len(args)/2)
	for i := 0; i < len(args); i += 2 {
		key, _ := args[i].(string)

		switch val := args[i+1].(type) {
		case string:
			fields = append(fields, zap.String(key, val))
		case int:
			fields = append(fields, zap.Int(key, val))
		case bool:
			fields = append(fields, zap.Bool(key, val))
		case error:
			fields = append(fields, zap.Error(val))
		case time.Duration:
			fields = append(fields, zap.Duration(key, val))
		case time.Time:
			fields = append(fields, zap.Time(key, val))
		default:
			fields = append(fields, zap.Any(key, val))
		}
	}
	return fields
}

func toZapFieldsFromAttrs(attrs []Attr) []zap.Field {
	fields := make([]zap.Field, 0, len(attrs))
	for _, a := range attrs {
		switch val := a.Value.(type) {
		case string:
			fields = append(fields, zap.String(a.Key, val))
		case int:
			fields = append(fields, zap.Int(a.Key, val))
		case int64:
			fields = append(fields, zap.Int64(a.Key, val))
		case bool:
			fields = append(fields, zap.Bool(a.Key, val))
		case time.Time:
			fields = append(fields, zap.Time(a.Key, val))
		case time.Duration:
			fields = append(fields, zap.Duration(a.Key, val))
		case int32:
			fields = append(fields, zap.Int32(a.Key, val))
		case int16:
			fields = append(fields, zap.Int16(a.Key, val))
		case int8:
			fields = append(fields, zap.Int8(a.Key, val))
		case uint:
			fields = append(fields, zap.Uint(a.Key, val))
		case uint64:
			fields = append(fields, zap.Uint64(a.Key, val))
		case uint32:
			fields = append(fields, zap.Uint32(a.Key, val))
		case uint16:
			fields = append(fields, zap.Uint16(a.Key, val))
		case uint8:
			fields = append(fields, zap.Uint8(a.Key, val))
		default:
			fields = append(fields, zap.Any(a.Key, val))
		}
	}
	return fields
}
