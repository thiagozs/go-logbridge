package logbridge

import (
	"github.com/thiagozs/go-logbridge/internal/core"
	internalotel "github.com/thiagozs/go-logbridge/internal/otel"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
)

type Option = func(*Config)
type TraceExtractor = core.TraceExtractor

func WithEngine(engine Engine) Option {
	return func(c *Config) {
		c.Engine = engine
	}
}

func WithLevel(level Level) Option {
	return func(c *Config) {
		c.Level = level
	}
}

func WithJSON() Option {
	return func(c *Config) {
		c.JSON = true
	}
}

func WithCaller() Option {
	return func(c *Config) {
		c.Caller = true
	}
}

func WithServiceName(name string) Option {
	return func(c *Config) {
		c.ServiceName = name
	}
}

func WithOTEL() Option {
	return func(c *Config) {
		c.OTEL = true
		if c.TraceExtractor == nil {
			c.TraceExtractor = internalotel.TraceFields
		}
	}
}

func WithOTLP(provider otellog.LoggerProvider) Option {
	return func(c *Config) {
		c.OTELLogProvider = provider
	}
}

func WithGlobalOTLP() Option {
	return func(c *Config) {
		c.OTELLogProvider = global.GetLoggerProvider()
	}
}

func WithTraceExtractor(extractor TraceExtractor) Option {
	return func(c *Config) {
		c.OTEL = extractor != nil
		c.TraceExtractor = extractor
	}
}

func WithOTLPLogs(endpoint string) Option {
	return func(c *Config) {
		c.OTLPLogs.Endpoint = endpoint
		c.OTLPLogs.Insecure = true
	}
}

func WithOTLPLogsSecure() Option {
	return func(c *Config) {
		c.OTLPLogs.Insecure = false
	}
}
