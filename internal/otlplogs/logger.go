package otlplogs

import (
	"context"
	"fmt"
	"maps"
	"time"

	"github.com/thiagozs/go-logbridge/internal/core"
	internalotel "github.com/thiagozs/go-logbridge/internal/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	otellog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
)

type Logger struct {
	cfg      core.Config
	logger   otellog.Logger
	provider *sdklog.LoggerProvider
	fields   map[string]any
}

func New(ctx context.Context, cfg core.Config) (*Logger, error) {
	if cfg.OTELLogProvider != nil {
		return &Logger{
			cfg:    cfg,
			logger: cfg.OTELLogProvider.Logger("github.com/thiagozs/go-logbridge"),
			fields: map[string]any{},
		}, nil
	}

	if cfg.OTLPLogs.Endpoint == "" {
		return nil, nil
	}

	options := []otlploggrpc.Option{
		otlploggrpc.WithEndpoint(cfg.OTLPLogs.Endpoint),
		otlploggrpc.WithTimeout(cfg.OTLPLogs.Timeout),
	}

	if cfg.OTLPLogs.Insecure {
		options = append(options, otlploggrpc.WithInsecure())
	}

	exporter, err := otlploggrpc.New(ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("create OTLP logs exporter: %w", err)
	}

	provider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewSimpleProcessor(exporter)),
		sdklog.WithResource(sdkresource.NewSchemaless(
			attribute.String("service.name", cfg.ServiceName),
			attribute.String("logger.engine", string(cfg.Engine)),
		)),
	)

	return &Logger{
		cfg:      cfg,
		logger:   provider.Logger("github.com/thiagozs/go-logbridge"),
		provider: provider,
		fields:   map[string]any{},
	}, nil
}

func (l *Logger) Debug(ctx context.Context, msg string, args ...any) {
	l.emit(ctx, otellog.SeverityDebug, "DEBUG", msg, args...)
}

func (l *Logger) Info(ctx context.Context, msg string, args ...any) {
	l.emit(ctx, otellog.SeverityInfo, "INFO", msg, args...)
}

func (l *Logger) Infof(ctx context.Context, format string, args ...any) {
	l.Info(ctx, fmt.Sprintf(format, args...))
}

func (l *Logger) Warn(ctx context.Context, msg string, args ...any) {
	l.emit(ctx, otellog.SeverityWarn, "WARN", msg, args...)
}

func (l *Logger) Warnf(ctx context.Context, format string, args ...any) {
	l.Warn(ctx, fmt.Sprintf(format, args...))
}

func (l *Logger) Error(ctx context.Context, msg string, args ...any) {
	l.emit(ctx, otellog.SeverityError, "ERROR", msg, args...)
}

func (l *Logger) Errorf(ctx context.Context, format string, args ...any) {
	l.Error(ctx, fmt.Sprintf(format, args...))
}

func (l *Logger) With(args ...any) core.Logger {
	fields := maps.Clone(l.fields)
	maps.Copy(fields, core.Map(args...))

	return &Logger{
		cfg:      l.cfg,
		logger:   l.logger,
		provider: l.provider,
		fields:   fields,
	}
}

func (l *Logger) Shutdown(ctx context.Context) error {
	if l == nil || l.provider == nil {
		return nil
	}

	return l.provider.Shutdown(ctx)
}

func (l *Logger) emit(ctx context.Context, severity otellog.Severity, severityText, msg string, args ...any) {
	attrs := maps.Clone(l.fields)
	maps.Copy(attrs, core.Map(args...))
	maps.Copy(attrs, core.Map(core.CallerFields(l.cfg.Caller, l.cfg.CallerSkip)...))
	maps.Copy(attrs, core.Map(internalotel.Fields(ctx, l.cfg)...))

	var record otellog.Record
	now := time.Now()

	record.SetTimestamp(now)
	record.SetObservedTimestamp(now)
	record.SetSeverity(severity)
	record.SetSeverityText(severityText)
	record.SetBody(otellog.StringValue(msg))
	record.AddAttributes(toAttributes(attrs)...)

	l.logger.Emit(ctx, record)
}

func toAttributes(fields map[string]any) []otellog.KeyValue {
	attrs := make([]otellog.KeyValue, 0, len(fields))

	for key, value := range fields {
		attrs = append(attrs, otellog.KeyValue{
			Key:   key,
			Value: toValue(value),
		})
	}

	return attrs
}

func toValue(value any) otellog.Value {
	switch v := value.(type) {
	case string:
		return otellog.StringValue(v)
	case bool:
		return otellog.BoolValue(v)
	case int:
		return otellog.IntValue(v)
	case int8:
		return otellog.Int64Value(int64(v))
	case int16:
		return otellog.Int64Value(int64(v))
	case int32:
		return otellog.Int64Value(int64(v))
	case int64:
		return otellog.Int64Value(v)
	case uint:
		return otellog.Int64Value(int64(v))
	case uint8:
		return otellog.Int64Value(int64(v))
	case uint16:
		return otellog.Int64Value(int64(v))
	case uint32:
		return otellog.Int64Value(int64(v))
	case uint64:
		if v > uint64(^uint64(0)>>1) {
			return otellog.StringValue(fmt.Sprint(v))
		}
		return otellog.Int64Value(int64(v))
	case float32:
		return otellog.Float64Value(float64(v))
	case float64:
		return otellog.Float64Value(v)
	case time.Time:
		return otellog.StringValue(v.Format(time.RFC3339Nano))
	case []string:
		values := make([]otellog.Value, 0, len(v))
		for _, item := range v {
			values = append(values, otellog.StringValue(item))
		}
		return otellog.SliceValue(values...)
	case []any:
		values := make([]otellog.Value, 0, len(v))
		for _, item := range v {
			values = append(values, toValue(item))
		}
		return otellog.SliceValue(values...)
	case fmt.Stringer:
		return otellog.StringValue(v.String())
	case error:
		return otellog.StringValue(v.Error())
	default:
		return otellog.StringValue(fmt.Sprint(v))
	}
}
