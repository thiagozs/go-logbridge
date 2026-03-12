package logbridge

import (
	"context"

	"github.com/thiagozs/go-logbridge/adapter/logrus"
	"github.com/thiagozs/go-logbridge/adapter/slog"
	"github.com/thiagozs/go-logbridge/adapter/zap"
	"github.com/thiagozs/go-logbridge/adapter/zerolog"
	"github.com/thiagozs/go-logbridge/internal/core"
	"github.com/thiagozs/go-logbridge/internal/otlplogs"
)

func New(opts ...Option) (Logger, error) {
	cfg := core.DefaultConfig()

	for _, opt := range opts {
		opt(&cfg)
	}

	base := newEngineLogger(cfg)
	remote, err := otlplogs.New(context.Background(), cfg)
	if err != nil {
		return nil, err
	}
	if remote == nil {
		return base, nil
	}

	return &fanoutLogger{
		local:  base,
		remote: remote,
	}, nil
}

func newEngineLogger(cfg core.Config) Logger {
	switch cfg.Engine {

	case Zap:
		return zap.New(cfg)

	case Zerolog:
		return zerolog.New(cfg)

	case Logrus:
		return logrus.New(cfg)

	default:
		return slog.New(cfg)
	}
}

type fanoutLogger struct {
	local  Logger
	remote Logger
}

func (l *fanoutLogger) Debug(ctx context.Context, msg string, args ...any) {
	l.local.Debug(ctx, msg, args...)
	l.remote.Debug(ctx, msg, args...)
}

func (l *fanoutLogger) Info(ctx context.Context, msg string, args ...any) {
	l.local.Info(ctx, msg, args...)
	l.remote.Info(ctx, msg, args...)
}

func (l *fanoutLogger) Warn(ctx context.Context, msg string, args ...any) {
	l.local.Warn(ctx, msg, args...)
	l.remote.Warn(ctx, msg, args...)
}

func (l *fanoutLogger) Error(ctx context.Context, msg string, args ...any) {
	l.local.Error(ctx, msg, args...)
	l.remote.Error(ctx, msg, args...)
}

func (l *fanoutLogger) With(args ...any) Logger {
	return &fanoutLogger{
		local:  l.local.With(args...),
		remote: l.remote.With(args...),
	}
}

func (l *fanoutLogger) Shutdown(ctx context.Context) error {
	if shutdownable, ok := l.remote.(shutdowner); ok {
		return shutdownable.Shutdown(ctx)
	}

	return nil
}
