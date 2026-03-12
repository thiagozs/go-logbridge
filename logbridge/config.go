package logbridge

import "github.com/thiagozs/go-logbridge/internal/core"

type Engine = core.Engine

const (
	Slog    = core.Slog
	Zap     = core.Zap
	Zerolog = core.Zerolog
	Logrus  = core.Logrus
)

type Level = core.Level

const (
	Debug = core.Debug
	Info  = core.Info
	Warn  = core.Warn
	Error = core.Error
)

type Config = core.Config
