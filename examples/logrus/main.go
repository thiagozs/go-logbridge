package main

import (
	"context"

	"github.com/thiagozs/go-logbridge/logbridge"
	"go.opentelemetry.io/otel/trace"
)

func main() {

	log, err := logbridge.New(
		logbridge.WithEngine(logbridge.Logrus),
		logbridge.WithLevel(logbridge.Debug),
		logbridge.WithJSON(),
		logbridge.WithCaller(),
		logbridge.WithServiceName("go-logbridge-logrus"),
		logbridge.WithGlobalOTLP(),
	)
	if err != nil {
		panic(err)
	}

	ctx := GenerateContextWithSpan()

	log.Info(ctx, "application started",
		"service", "payments",
		"version", "1.0.0",
	)

	log.Warn(ctx, "payment processing is slow",
		"service", "payments",
		"version", "1.0.0",
	)

	log.Error(ctx, "payment failed",
		"service", "payments",
		"version", "1.0.0",
		"error", "timeout",
	)

	log.Debug(ctx, "debugging payment process",
		"service", "payments",
		"version", "1.0.0",
	)
}

func GenerateContextWithSpan() context.Context {
	traceID := trace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	spanID := trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8}

	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})

	ctx := trace.ContextWithSpanContext(context.Background(), spanContext)

	return ctx
}
