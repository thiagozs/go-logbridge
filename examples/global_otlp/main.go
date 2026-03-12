package main

import (
	"context"
	"fmt"
	"time"

	"github.com/thiagozs/go-logbridge/logbridge"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func main() {
	traceProvider, err := newTraceProvider(context.Background())
	if err != nil {
		panic(err)
	}

	log, err := logbridge.New(
		logbridge.WithEngine(logbridge.Zap),
		logbridge.WithLevel(logbridge.Debug),
		logbridge.WithJSON(),
		logbridge.WithCaller(),
		logbridge.WithServiceName("go-logbridge-global-otlp"),
		logbridge.WithOTEL(),
		logbridge.WithOTLPLogs("localhost:4317"),
	)
	if err != nil {
		panic(err)
	}

	ctx, span := traceProvider.Tracer("github.com/thiagozs/go-logbridge/examples/global_otlp").Start(
		context.Background(),
		"payment.workflow",
		trace.WithAttributes(
			attribute.String("service.name", "go-logbridge-global-otlp"),
			attribute.String("workflow", "checkout"),
			attribute.String("payment_id", "pay-1001"),
			attribute.String("order_id", "ord-123"),
		),
	)

	simulatePaymentFlow(ctx, log)
	span.End()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := logbridge.Shutdown(shutdownCtx, log); err != nil {
		panic(err)
	}
	if err := traceProvider.Shutdown(shutdownCtx); err != nil {
		panic(err)
	}
}

func simulatePaymentFlow(ctx context.Context, log logbridge.Logger) {
	base := log.With(
		"service", "payments",
		"version", "1.0.0",
		"workflow", "checkout",
		"payment_id", "pay-1001",
		"order_id", "ord-123",
		"customer_id", "cus-789",
	)

	startedAt := time.Now()
	tracer := trace.SpanFromContext(ctx).TracerProvider().Tracer("github.com/thiagozs/go-logbridge/examples/global_otlp")

	base.Info(ctx, "payment workflow received",
		"step", "received",
		"attempt", 1,
		"elapsed_ms", 0,
	)
	time.Sleep(700 * time.Millisecond)

	antiFraudCtx, antiFraudSpan := tracer.Start(ctx, "payment.anti_fraud")
	base.Debug(antiFraudCtx, "validating anti-fraud rules",
		"step", "anti_fraud",
		"attempt", 1,
		"elapsed_ms", time.Since(startedAt).Milliseconds(),
	)
	time.Sleep(900 * time.Millisecond)
	antiFraudSpan.End()

	acquirerCtx, acquirerSpan := tracer.Start(ctx, "payment.acquirer_request")
	base.Info(acquirerCtx, "calling acquirer",
		"step", "acquirer_request",
		"attempt", 1,
		"provider", "mock-acquirer",
		"elapsed_ms", time.Since(startedAt).Milliseconds(),
	)
	time.Sleep(1200 * time.Millisecond)

	base.Warn(acquirerCtx, "acquirer response above threshold",
		"step", "acquirer_timeout",
		"attempt", 1,
		"provider", "mock-acquirer",
		"latency_ms", 1200,
		"elapsed_ms", time.Since(startedAt).Milliseconds(),
	)
	time.Sleep(800 * time.Millisecond)

	acquirerSpan.RecordError(fmt.Errorf("timeout contacting acquirer"))
	base.Error(acquirerCtx, "payment authorization failed",
		"step", "authorization_failed",
		"attempt", 1,
		"provider", "mock-acquirer",
		"error", fmt.Errorf("timeout contacting acquirer\n\tgateway/mock.go:88\n\tretry/runner.go:41"),
		"elapsed_ms", time.Since(startedAt).Milliseconds(),
	)
	acquirerSpan.End()
	time.Sleep(700 * time.Millisecond)

	retry := base.With("attempt", 2)
	retryCtx, retrySpan := tracer.Start(ctx, "payment.retry")

	retry.Info(retryCtx, "retry scheduled",
		"step", "retry_scheduled",
		"backoff_ms", 500,
		"elapsed_ms", time.Since(startedAt).Milliseconds(),
	)
	time.Sleep(500 * time.Millisecond)

	retry.Info(retryCtx, "payment authorized",
		"step", "authorized",
		"provider", "mock-acquirer",
		"authorization_code", "AUTH-9001",
		"elapsed_ms", time.Since(startedAt).Milliseconds(),
	)
	time.Sleep(400 * time.Millisecond)

	retry.Info(retryCtx, "workflow completed",
		"step", "completed",
		"status", "success",
		"elapsed_ms", time.Since(startedAt).Milliseconds(),
	)
	retrySpan.End()
}

func newTraceProvider(ctx context.Context) (*sdktrace.TracerProvider, error) {
	exporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithEndpoint("localhost:4317"),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(
			sdkresource.NewWithAttributes(
				"",
				attribute.String("service.name", "go-logbridge-global-otlp"),
			),
		),
	), nil
}
