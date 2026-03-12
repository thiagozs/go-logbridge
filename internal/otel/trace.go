package otel

import (
	"context"

	"github.com/thiagozs/go-logbridge/internal/core"
	"go.opentelemetry.io/otel/trace"
)

func TraceFields(ctx context.Context) []any {

	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return nil
	}

	sc := span.SpanContext()

	return []any{
		"trace_id", sc.TraceID().String(),
		"span_id", sc.SpanID().String(),
	}
}

func Fields(ctx context.Context, cfg core.Config) []any {
	if !cfg.OTEL {
		return nil
	}

	if cfg.TraceExtractor != nil {
		return cfg.TraceExtractor(ctx)
	}

	return TraceFields(ctx)
}
