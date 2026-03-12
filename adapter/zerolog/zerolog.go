package zerolog

import (
	"context"
	"fmt"
	"maps"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/thiagozs/go-logbridge/internal/core"
	"github.com/thiagozs/go-logbridge/internal/otel"
)

type Adapter struct {
	log    zerolog.Logger
	cfg    core.Config
	fields map[string]any
}

func New(cfg core.Config) core.Logger {
	zerolog.TimestampFieldName = "ts"
	zerolog.TimeFieldFormat = time.RFC3339Nano

	var logger zerolog.Logger

	if cfg.JSON {
		logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	} else {
		writer := zerolog.ConsoleWriter{Out: os.Stdout}
		logger = zerolog.New(writer).With().Timestamp().Logger()
	}

	logger = logger.Level(toLevel(cfg.Level))

	return &Adapter{
		log:    logger,
		cfg:    cfg,
		fields: map[string]any{},
	}
}

func (a *Adapter) Info(ctx context.Context, msg string, args ...any) {
	event := a.log.Info()
	event.Fields(mergedFields(a.fields, a.cfg, ctx, args...)).Msg(msg)
}

func (a *Adapter) Debug(ctx context.Context, msg string, args ...any) {
	event := a.log.Debug()
	event.Fields(mergedFields(a.fields, a.cfg, ctx, args...)).Msg(msg)
}

func (a *Adapter) Infof(ctx context.Context, format string, args ...any) {
	a.Info(ctx, fmt.Sprintf(format, args...))
}

func (a *Adapter) Warn(ctx context.Context, msg string, args ...any) {
	event := a.log.Warn()
	event.Fields(mergedFields(a.fields, a.cfg, ctx, args...)).Msg(msg)
}

func (a *Adapter) Warnf(ctx context.Context, format string, args ...any) {
	a.Warn(ctx, fmt.Sprintf(format, args...))
}

func (a *Adapter) Error(ctx context.Context, msg string, args ...any) {
	event := a.log.Error()
	event.Fields(mergedFields(a.fields, a.cfg, ctx, args...)).Msg(msg)
}

func (a *Adapter) Errorf(ctx context.Context, format string, args ...any) {
	a.Error(ctx, fmt.Sprintf(format, args...))
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

func mergedFields(base map[string]any, cfg core.Config, ctx context.Context, args ...any) map[string]any {
	fields := maps.Clone(base)
	maps.Copy(fields, core.Map(args...))
	maps.Copy(fields, core.Map(core.CallerFields(cfg.Caller, cfg.CallerSkip)...))
	maps.Copy(fields, core.Map(otel.Fields(ctx, cfg)...))

	return fields
}

func toLevel(level core.Level) zerolog.Level {
	switch level {
	case core.Debug:
		return zerolog.DebugLevel
	case core.Warn:
		return zerolog.WarnLevel
	case core.Error:
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}
