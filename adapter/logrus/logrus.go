package logrus

import (
	"context"
	"maps"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/thiagozs/go-logbridge/internal/core"
	"github.com/thiagozs/go-logbridge/internal/otel"
)

type Adapter struct {
	log    *logrus.Logger
	cfg    core.Config
	fields logrus.Fields
}

func New(cfg core.Config) core.Logger {

	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetLevel(toLevel(cfg.Level))

	if cfg.JSON {
		logger.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	}

	return &Adapter{
		log:    logger,
		cfg:    cfg,
		fields: logrus.Fields{},
	}
}

func (a *Adapter) Info(ctx context.Context, msg string, args ...any) {
	fields := mergedFields(a.fields, a.cfg, ctx, args...)
	a.log.WithFields(fields).Info(msg)
}

func (a *Adapter) Debug(ctx context.Context, msg string, args ...any) {
	fields := mergedFields(a.fields, a.cfg, ctx, args...)
	a.log.WithFields(fields).Debug(msg)
}

func (a *Adapter) Warn(ctx context.Context, msg string, args ...any) {
	fields := mergedFields(a.fields, a.cfg, ctx, args...)
	a.log.WithFields(fields).Warn(msg)
}

func (a *Adapter) Error(ctx context.Context, msg string, args ...any) {
	fields := mergedFields(a.fields, a.cfg, ctx, args...)
	a.log.WithFields(fields).Error(msg)
}

func (a *Adapter) With(args ...any) core.Logger {
	fields := logrus.Fields{}
	maps.Copy(fields, a.fields)
	maps.Copy(fields, toFields(args...))

	return &Adapter{
		log:    a.log,
		cfg:    a.cfg,
		fields: fields,
	}
}

func toFields(args ...any) logrus.Fields {

	fields := logrus.Fields{}

	maps.Copy(fields, core.Map(args...))

	return fields
}

func addTrace(traceFields []any, fields logrus.Fields) {

	for i := 0; i < len(traceFields); i += 2 {

		key := traceFields[i].(string)
		fields[key] = traceFields[i+1]
	}
}

func mergedFields(base logrus.Fields, cfg core.Config, ctx context.Context, args ...any) logrus.Fields {
	fields := logrus.Fields{}
	maps.Copy(fields, base)
	maps.Copy(fields, toFields(args...))
	addTrace(core.CallerFields(cfg.Caller), fields)
	addTrace(otel.Fields(ctx, cfg), fields)

	return fields
}

func toLevel(level core.Level) logrus.Level {
	switch level {
	case core.Debug:
		return logrus.DebugLevel
	case core.Warn:
		return logrus.WarnLevel
	case core.Error:
		return logrus.ErrorLevel
	default:
		return logrus.InfoLevel
	}
}
