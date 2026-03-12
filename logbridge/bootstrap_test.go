package logbridge

import (
	"context"
	stderrors "errors"
	"io"
	"os"
	"strings"
	"testing"

	lognoop "go.opentelemetry.io/otel/log/noop"
	"go.opentelemetry.io/otel/trace"
)

func TestAllAdapters(t *testing.T) {
	engines := []Engine{Slog, Zap, Zerolog, Logrus}

	for _, engine := range engines {
		t.Run(string(engine), func(t *testing.T) {
			output := captureOutput(t, func() {
				logger, err := New(
					WithEngine(engine),
					WithLevel(Debug),
					WithJSON(),
				)
				if err != nil {
					t.Fatalf("initialize logger: %v", err)
				}

				logger.With("adapter", string(engine)).Info(
					context.Background(),
					"adapter log entry",
					"request_id", "req-123",
					123, "ignored",
					"dangling",
				)
			})

			assertContains(t, output, "adapter log entry")
			assertContains(t, output, `"adapter":"`+string(engine)+`"`)
			assertContains(t, output, `"request_id":"req-123"`)
			assertNotContains(t, output, "trace_id")
			assertNotContains(t, output, "span_id")
		})
	}
}

func TestAllAdaptersWithOpenTelemetry(t *testing.T) {
	engines := []Engine{Slog, Zap, Zerolog, Logrus}
	ctx := contextWithSpan()

	for _, engine := range engines {
		t.Run(string(engine), func(t *testing.T) {
			output := captureOutput(t, func() {
				logger, err := New(
					WithEngine(engine),
					WithLevel(Debug),
					WithJSON(),
					WithOTEL(),
				)
				if err != nil {
					t.Fatalf("initialize logger: %v", err)
				}

				logger.With("adapter", string(engine)).Info(
					ctx,
					"adapter log entry",
					"request_id", "req-123",
				)
			})

			assertContains(t, output, "adapter log entry")
			assertContains(t, output, `"adapter":"`+string(engine)+`"`)
			assertContains(t, output, `"request_id":"req-123"`)
			assertContains(t, output, `"trace_id":"0102030405060708090a0b0c0d0e0f10"`)
			assertContains(t, output, `"span_id":"0102030405060708"`)
		})
	}
}

func TestAllAdaptersWithExternalLoggerProvider(t *testing.T) {
	engines := []Engine{Slog, Zap, Zerolog, Logrus}
	loggerProvider := lognoop.NewLoggerProvider()

	for _, engine := range engines {
		t.Run(string(engine), func(t *testing.T) {
			output := captureOutput(t, func() {
				logger, err := New(
					WithEngine(engine),
					WithLevel(Debug),
					WithJSON(),
					WithOTLP(loggerProvider),
				)
				if err != nil {
					t.Fatalf("initialize logger: %v", err)
				}

				logger.With("adapter", string(engine)).Info(
					context.Background(),
					"adapter log entry",
					"request_id", "req-123",
				)

				if err := Shutdown(context.Background(), logger); err != nil {
					t.Fatalf("shutdown logger: %v", err)
				}
			})

			assertContains(t, output, "adapter log entry")
			assertContains(t, output, `"adapter":"`+string(engine)+`"`)
			assertContains(t, output, `"request_id":"req-123"`)
		})
	}
}

func TestAllAdaptersFormatErrors(t *testing.T) {
	engines := []Engine{Slog, Zap, Zerolog, Logrus}
	errWithStack := stderrors.New("database timeout\n\tservice/repository.go:42\n\thandler/payment.go:18")

	for _, engine := range engines {
		t.Run(string(engine), func(t *testing.T) {
			output := captureOutput(t, func() {
				logger, err := New(
					WithEngine(engine),
					WithLevel(Debug),
					WithJSON(),
				)
				if err != nil {
					t.Fatalf("initialize logger: %v", err)
				}

				logger.Error(
					context.Background(),
					"payment failed",
					"error", errWithStack,
				)
			})

			assertContains(t, output, `"error":"database timeout"`)
			assertContains(t, output, `"error_type":"*errors.errorString"`)
			assertContains(t, output, `"error_stack":[`)
			assertContains(t, output, `service/repository.go:42`)
			assertContains(t, output, `handler/payment.go:18`)
			assertNotContains(t, output, `\n\t`)
		})
	}
}

func TestAllAdaptersWithCaller(t *testing.T) {
	engines := []Engine{Slog, Zap, Zerolog, Logrus}

	for _, engine := range engines {
		t.Run(string(engine), func(t *testing.T) {
			output := captureOutput(t, func() {
				logger, err := New(
					WithEngine(engine),
					WithLevel(Debug),
					WithJSON(),
					WithCaller(),
				)
				if err != nil {
					t.Fatalf("initialize logger: %v", err)
				}

				logger.Info(context.Background(), "adapter log entry")
			})

			assertContains(t, output, `"caller_file":"bootstrap_test.go"`)
			assertContains(t, output, `"caller_func":"logbridge.TestAllAdaptersWithCaller.func1.1"`)
			assertContains(t, output, `"caller_line":"`)
		})
	}
}

func TestAllAdaptersWithCallerSkip(t *testing.T) {
	engines := []Engine{Slog, Zap, Zerolog, Logrus}

	for _, engine := range engines {
		t.Run(string(engine), func(t *testing.T) {
			output := captureOutput(t, func() {
				logger, err := New(
					WithEngine(engine),
					WithLevel(Debug),
					WithJSON(),
					WithCallerSkip(1),
				)
				if err != nil {
					t.Fatalf("initialize logger: %v", err)
				}

				service := loggerService{logger: logger}
				callServiceInfo(context.Background(), service)
			})

			assertContains(t, output, `"caller_file":"bootstrap_test.go"`)
			assertContains(t, output, `"caller_func":"logbridge.callServiceInfo"`)
			assertContains(t, output, `"caller_line":"`)
			assertNotContains(t, output, `services.(*LoggerService).Info`)
			assertNotContains(t, output, `"caller_func":"logbridge.loggerService.Info"`)
		})
	}
}

func TestAllAdaptersWithFields(t *testing.T) {
	engines := []Engine{Slog, Zap, Zerolog, Logrus}

	for _, engine := range engines {
		t.Run(string(engine), func(t *testing.T) {
			output := captureOutput(t, func() {
				logger, err := New(
					WithEngine(engine),
					WithLevel(Debug),
					WithJSON(),
				)
				if err != nil {
					t.Fatalf("initialize logger: %v", err)
				}

				child := logger.With(
					"service", "payments",
					"env", "test",
				).With(
					"request_id", "req-123",
					"env", "prod",
				)

				child.Info(
					context.Background(),
					"adapter log entry",
					"user_id", "user-42",
				)
			})

			assertContains(t, output, "adapter log entry")
			assertContains(t, output, `"service":"payments"`)
			assertContains(t, output, `"request_id":"req-123"`)
			assertContains(t, output, `"user_id":"user-42"`)
			assertContains(t, output, `"env":"prod"`)
			assertNotContains(t, output, `"env":"test"`)
		})
	}
}

func TestAllAdaptersFormattedMessages(t *testing.T) {
	engines := []Engine{Slog, Zap, Zerolog, Logrus}

	for _, engine := range engines {
		t.Run(string(engine), func(t *testing.T) {
			output := captureOutput(t, func() {
				logger, err := New(
					WithEngine(engine),
					WithLevel(Debug),
					WithJSON(),
				)
				if err != nil {
					t.Fatalf("initialize logger: %v", err)
				}

				logger.Infof(context.Background(), "payment %s received", "pay-123")
				logger.Warnf(context.Background(), "attempt %d delayed", 2)
				logger.Errorf(context.Background(), "payment %s failed", "pay-123")
			})

			assertContains(t, output, "payment pay-123 received")
			assertContains(t, output, "attempt 2 delayed")
			assertContains(t, output, "payment pay-123 failed")
		})
	}
}

func TestFormattedMessagesWithFanout(t *testing.T) {
	output := captureOutput(t, func() {
		logger, err := New(
			WithEngine(Zap),
			WithLevel(Debug),
			WithJSON(),
			WithOTLP(lognoop.NewLoggerProvider()),
		)
		if err != nil {
			t.Fatalf("initialize logger: %v", err)
		}

		logger.Infof(context.Background(), "payment %s received", "pay-123")
		logger.Warnf(context.Background(), "attempt %d delayed", 2)
		logger.Errorf(context.Background(), "payment %s failed", "pay-123")

		if err := Shutdown(context.Background(), logger); err != nil {
			t.Fatalf("shutdown logger: %v", err)
		}
	})

	assertContains(t, output, "payment pay-123 received")
	assertContains(t, output, "attempt 2 delayed")
	assertContains(t, output, "payment pay-123 failed")
}

type loggerService struct {
	logger Logger
}

func (s loggerService) Info(ctx context.Context, msg string) {
	s.logger.Info(ctx, msg)
}

func callServiceInfo(ctx context.Context, service loggerService) {
	service.Info(ctx, "adapter log entry")
}

func TestAllAdaptersFormatTimestamp(t *testing.T) {
	engines := []Engine{Slog, Zap, Zerolog, Logrus}

	for _, engine := range engines {
		t.Run(string(engine), func(t *testing.T) {
			output := captureOutput(t, func() {
				logger, err := New(
					WithEngine(engine),
					WithLevel(Debug),
					WithJSON(),
				)
				if err != nil {
					t.Fatalf("initialize logger: %v", err)
				}

				logger.Info(context.Background(), "adapter log entry")
			})

			assertContains(t, output, `"ts":"20`)
			assertNotContains(t, output, `"time":`)
			assertNotContains(t, output, `"ts":17`)
		})
	}
}

func captureOutput(t *testing.T, fn func()) string {
	t.Helper()

	originalStdout := os.Stdout
	originalStderr := os.Stderr
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}

	os.Stdout = writer
	os.Stderr = writer

	defer func() {
		os.Stdout = originalStdout
		os.Stderr = originalStderr
	}()

	fn()

	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	if err := reader.Close(); err != nil {
		t.Fatalf("close reader: %v", err)
	}

	return string(output)
}

func contextWithSpan() context.Context {
	traceID := trace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	spanID := trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8}

	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})

	return trace.ContextWithSpanContext(context.Background(), spanContext)
}

func assertContains(t *testing.T, output, want string) {
	t.Helper()

	if !strings.Contains(output, want) {
		t.Fatalf("expected output to contain %q, got %q", want, output)
	}
}

func assertNotContains(t *testing.T, output, want string) {
	t.Helper()

	if strings.Contains(output, want) {
		t.Fatalf("expected output not to contain %q, got %q", want, output)
	}
}
