package slog

import (
	"context"
	"log/slog"
	"maps"
	"os"
	"time"

	"github.com/thiagozs/go-logbridge/internal/core"
	"github.com/thiagozs/go-logbridge/internal/otel"
)

type Adapter struct {
	log    *slog.Logger
	cfg    core.Config
	fields map[string]any
}

func New(cfg core.Config) core.Logger {

	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: toLevel(cfg.Level),
		ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
			if attr.Key == slog.TimeKey {
				return slog.String("ts", attr.Value.Time().Format(time.RFC3339Nano))
			}

			return attr
		},
	}

	if cfg.JSON {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return &Adapter{
		log:    slog.New(handler),
		cfg:    cfg,
		fields: map[string]any{},
	}
}

func (a *Adapter) Info(ctx context.Context, msg string, args ...any) {
	a.log.InfoContext(ctx, msg, mergedArgs(a.fields, a.cfg, ctx, args...)...)
}

func (a *Adapter) Debug(ctx context.Context, msg string, args ...any) {
	a.log.DebugContext(ctx, msg, mergedArgs(a.fields, a.cfg, ctx, args...)...)
}

func (a *Adapter) Error(ctx context.Context, msg string, args ...any) {
	a.log.ErrorContext(ctx, msg, mergedArgs(a.fields, a.cfg, ctx, args...)...)
}

func (a *Adapter) Warn(ctx context.Context, msg string, args ...any) {
	a.log.WarnContext(ctx, msg, mergedArgs(a.fields, a.cfg, ctx, args...)...)
}

func (a *Adapter) With(args ...any) core.Logger {
	fields := maps.Clone(a.fields)
	maps.Copy(fields, core.Map(args...))

	return &Adapter{
		log:    a.log,
		cfg:    a.cfg,
		fields: fields,
	}
}

func mergedArgs(base map[string]any, cfg core.Config, ctx context.Context, args ...any) []any {
	fields := maps.Clone(base)
	maps.Copy(fields, core.Map(args...))
	maps.Copy(fields, core.Map(core.CallerFields(cfg.Caller)...))
	maps.Copy(fields, core.Map(otel.Fields(ctx, cfg)...))

	return core.KeyValuesFromMap(fields)
}

func toLevel(level core.Level) slog.Level {
	switch level {
	case core.Debug:
		return slog.LevelDebug
	case core.Warn:
		return slog.LevelWarn
	case core.Error:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
