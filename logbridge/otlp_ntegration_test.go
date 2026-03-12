package logbridge

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace"
)

func TestOTLPLGTMLogExport(t *testing.T) {
	endpoint := envOrDefault("LOGBRIDGE_OTLP_LOGS_ENDPOINT", "localhost:4317")

	if !isReachable(endpoint, 2*time.Second) {
		t.Skipf("OTLP endpoint %q is not reachable", endpoint)
	}

	traceID := trace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	spanID := trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8}
	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})

	runID := fmt.Sprintf("go-logbridge-%d", time.Now().UnixNano())

	logger, err := New(
		WithEngine(Zap),
		WithLevel(Debug),
		WithJSON(),
		WithServiceName("go-logbridge-integration"),
		WithOTEL(),
		WithOTLPLogs(endpoint),
	)
	if err != nil {
		t.Fatalf("initialize logbridge with OTLP logs exporter: %v", err)
	}

	ctx := trace.ContextWithSpanContext(context.Background(), spanContext)
	logger.With(
		"test.name", t.Name(),
		"run.id", runID,
		"stack.target", endpoint,
	).Info(ctx, "integration log delivered to LGTM")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := Shutdown(shutdownCtx, logger); err != nil {
		t.Fatalf("shutdown OTLP logs exporter: %v", err)
	}

	t.Logf("logbridge exported log to %s with run.id=%s trace_id=%s span_id=%s", endpoint, runID, traceID.String(), spanID.String())
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func isReachable(endpoint string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", endpoint, timeout)
	if err != nil {
		return false
	}

	_ = conn.Close()

	return true
}
