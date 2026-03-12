package zap

import (
	"context"
	"maps"

	"github.com/thiagozs/go-logbridge/internal/core"
	"github.com/thiagozs/go-logbridge/internal/otel"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Adapter struct {
	log    *zap.SugaredLogger
	cfg    core.Config
	fields map[string]any
}

func New(cfg core.Config) core.Logger {

	zapCfg := zap.NewProductionConfig()
	if !cfg.JSON {
		zapCfg = zap.NewDevelopmentConfig()
	}
	zapCfg.Level = zap.NewAtomicLevelAt(toLevel(cfg.Level))
	zapCfg.DisableStacktrace = true
	zapCfg.DisableCaller = true
	zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logger, err := zapCfg.Build()
	if err != nil {
		logger = zap.NewNop()
	}

	return &Adapter{
		log:    logger.Sugar(),
		cfg:    cfg,
		fields: map[string]any{},
	}
}

func (a *Adapter) Info(ctx context.Context, msg string, args ...any) {
	a.log.Infow(msg, mergedArgs(a.fields, a.cfg, ctx, args...)...)
}

func (a *Adapter) Debug(ctx context.Context, msg string, args ...any) {
	a.log.Debugw(msg, mergedArgs(a.fields, a.cfg, ctx, args...)...)
}

func (a *Adapter) Warn(ctx context.Context, msg string, args ...any) {
	a.log.Warnw(msg, mergedArgs(a.fields, a.cfg, ctx, args...)...)
}

func (a *Adapter) Error(ctx context.Context, msg string, args ...any) {
	a.log.Errorw(msg, mergedArgs(a.fields, a.cfg, ctx, args...)...)
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
	maps.Copy(fields, core.Map(core.CallerFields(cfg.Caller, cfg.CallerSkip)...))
	maps.Copy(fields, core.Map(otel.Fields(ctx, cfg)...))

	return core.KeyValuesFromMap(fields)
}

func toLevel(level core.Level) zapcore.Level {
	switch level {
	case core.Debug:
		return zapcore.DebugLevel
	case core.Warn:
		return zapcore.WarnLevel
	case core.Error:
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}
