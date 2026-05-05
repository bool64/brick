package telemetry

import (
	"context"
	"log/slog"

	"go.uber.org/zap/zapcore"
)

type otelZapCore struct {
	handler slog.Handler
	enabler zapcore.LevelEnabler
	attrs   []slog.Attr
}

// NewZapCore creates a zap core that mirrors zap entries into a slog handler.
func NewZapCore(handler slog.Handler, enabler zapcore.LevelEnabler) zapcore.Core {
	return &otelZapCore{
		handler: handler,
		enabler: enabler,
	}
}

func (o *otelZapCore) Enabled(level zapcore.Level) bool {
	return o.enabler.Enabled(level)
}

func (o *otelZapCore) With(fields []zapcore.Field) zapcore.Core {
	attrs := append([]slog.Attr{}, o.attrs...)
	attrs = append(attrs, zapFieldsToAttrs(fields)...)

	return &otelZapCore{
		handler: o.handler,
		enabler: o.enabler,
		attrs:   attrs,
	}
}

func (o *otelZapCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if o.Enabled(entry.Level) {
		return checked.AddCore(entry, o)
	}

	return checked
}

func (o *otelZapCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	record := slog.NewRecord(entry.Time, zapLevelToSlog(entry.Level), entry.Message, 0)
	record.AddAttrs(o.attrs...)
	record.AddAttrs(zapFieldsToAttrs(fields)...)

	if entry.LoggerName != "" {
		record.AddAttrs(slog.String("logger.name", entry.LoggerName))
	}

	if entry.Caller.Defined {
		record.AddAttrs(slog.String("code.filepath", entry.Caller.TrimmedPath()))
	}

	if entry.Stack != "" {
		record.AddAttrs(slog.String("exception.stacktrace", entry.Stack))
	}

	return o.handler.Handle(context.Background(), record)
}

func (o *otelZapCore) Sync() error {
	return nil
}

func zapLevelToSlog(level zapcore.Level) slog.Level {
	switch {
	case level <= zapcore.DebugLevel:
		return slog.LevelDebug
	case level <= zapcore.InfoLevel:
		return slog.LevelInfo
	case level <= zapcore.WarnLevel:
		return slog.LevelWarn
	default:
		return slog.LevelError
	}
}

func zapFieldsToAttrs(fields []zapcore.Field) []slog.Attr {
	if len(fields) == 0 {
		return nil
	}

	enc := zapcore.NewMapObjectEncoder()
	for _, field := range fields {
		field.AddTo(enc)
	}

	attrs := make([]slog.Attr, 0, len(enc.Fields))
	for key, value := range enc.Fields {
		attrs = append(attrs, slog.Any(key, value))
	}

	return attrs
}
