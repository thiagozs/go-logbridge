package core

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	otellog "go.opentelemetry.io/otel/log"
)

type Logger interface {
	Debug(ctx context.Context, msg string, args ...any)
	Info(ctx context.Context, msg string, args ...any)
	Warn(ctx context.Context, msg string, args ...any)
	Error(ctx context.Context, msg string, args ...any)

	With(args ...any) Logger
}

type Engine string

const (
	Slog    Engine = "slog"
	Zap     Engine = "zap"
	Zerolog Engine = "zerolog"
	Logrus  Engine = "logrus"
)

type Level string

const (
	Debug Level = "debug"
	Info  Level = "info"
	Warn  Level = "warn"
	Error Level = "error"
)

type Config struct {
	Engine          Engine
	Level           Level
	JSON            bool
	Caller          bool
	CallerSkip      int
	OTEL            bool
	OTELLogProvider otellog.LoggerProvider
	TraceExtractor  TraceExtractor
	ServiceName     string
	OTLPLogs        OTLPLogsConfig
}

type Option func(*Config)

type TraceExtractor func(context.Context) []any

type OTLPLogsConfig struct {
	Endpoint string
	Insecure bool
	Timeout  time.Duration
}

func DefaultConfig() Config {
	return Config{
		Engine:      Slog,
		Level:       Info,
		ServiceName: "go-logbridge",
		OTLPLogs: OTLPLogsConfig{
			Timeout: 5 * time.Second,
		},
	}
}

func KeyValues(args ...any) []any {
	fields := Map(args...)
	return KeyValuesFromMap(fields)
}

func KeyValuesFromMap(fields map[string]any) []any {
	pairs := make([]any, 0, len(fields)*2)

	for key, value := range fields {
		pairs = append(pairs, key, value)
	}

	return pairs
}

func Map(args ...any) map[string]any {
	fields := make(map[string]any)

	for i := 0; i+1 < len(args); i += 2 {
		key, ok := args[i].(string)
		if !ok {
			continue
		}

		addField(fields, key, args[i+1])
	}

	return fields
}

func addField(fields map[string]any, key string, value any) {
	if err, ok := value.(error); ok && err != nil {
		fields[key] = firstErrorLine(err)
		fields[key+"_type"] = reflect.TypeOf(err).String()

		if stack := errorStack(err); len(stack) > 0 {
			fields[key+"_stack"] = stack
		}

		if chain := errorChain(err); len(chain) > 1 {
			fields[key+"_chain"] = chain
		}

		return
	}

	fields[key] = value
}

func CallerFields(enabled bool, skip int) []any {
	if !enabled {
		return nil
	}

	frame, ok := callerFrame(skip)
	if !ok {
		return nil
	}

	return []any{
		"caller_file", filepath.Base(frame.File),
		"caller_func", shortFunctionName(frame.Function),
		"caller_line", strconv.Itoa(frame.Line),
	}
}

func firstErrorLine(err error) string {
	lines := splitErrorLines(err)
	if len(lines) == 0 {
		return ""
	}

	return lines[0]
}

func errorStack(err error) []string {
	lines := splitErrorLines(err)
	if len(lines) <= 1 {
		return nil
	}

	return lines[1:]
}

func splitErrorLines(err error) []string {
	raw := strings.ReplaceAll(err.Error(), "\r\n", "\n")
	parts := strings.Split(raw, "\n")
	lines := make([]string, 0, len(parts))

	for _, part := range parts {
		line := strings.TrimSpace(part)
		if line == "" {
			continue
		}

		lines = append(lines, line)
	}

	return lines
}

func errorChain(err error) []string {
	seen := make(map[string]struct{})
	chain := make([]string, 0, 4)

	var walk func(error)
	walk = func(current error) {
		if current == nil {
			return
		}

		message := firstErrorLine(current)
		if message != "" {
			if _, ok := seen[message]; !ok {
				seen[message] = struct{}{}
				chain = append(chain, message)
			}
		}

		type multiUnwrapper interface {
			Unwrap() []error
		}

		if unwrapped, ok := current.(multiUnwrapper); ok {
			for _, item := range unwrapped.Unwrap() {
				walk(item)
			}
			return
		}

		walk(errors.Unwrap(current))
	}

	walk(err)

	return chain
}

func callerFrame(extraSkip int) (runtime.Frame, bool) {
	pcs := make([]uintptr, 16)
	n := runtime.Callers(3, pcs)
	if n == 0 {
		return runtime.Frame{}, false
	}

	frames := runtime.CallersFrames(pcs[:n])

	for {
		frame, more := frames.Next()
		if shouldSkipCallerFrame(frame) {
			if !more {
				break
			}
			continue
		}

		if extraSkip > 0 {
			extraSkip--
			if !more {
				break
			}
			continue
		}

		return frame, true
	}

	return runtime.Frame{}, false
}

func shouldSkipCallerFrame(frame runtime.Frame) bool {
	fn := frame.Function

	switch {
	case strings.Contains(fn, "github.com/thiagozs/go-logbridge/adapter/"):
		return true
	case strings.Contains(fn, "github.com/thiagozs/go-logbridge/internal/otlplogs."):
		return true
	case strings.Contains(fn, "github.com/thiagozs/go-logbridge/logbridge.(*fanoutLogger)."):
		return true
	case strings.HasPrefix(fn, "runtime."):
		return true
	case strings.HasPrefix(fn, "testing."):
		return true
	default:
		return false
	}
}

func shortFunctionName(function string) string {
	slash := strings.LastIndex(function, "/")
	if slash >= 0 && slash+1 < len(function) {
		return function[slash+1:]
	}

	return function
}
