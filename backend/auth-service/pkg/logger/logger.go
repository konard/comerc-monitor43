package logger

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// atomicString provides atomic string operations
type atomicString struct {
	mu  sync.RWMutex
	val string
}

func (a *atomicString) Load() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.val
}

func (a *atomicString) Store(val string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.val = val
}

// Level represents log level
type Level string

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
	LevelFatal Level = "fatal"
)

// Fields represents structured log fields
type Fields struct {
	Message   string         `json:"message"`
	RequestID string         `json:"request_id,omitempty"`
	UserID    string         `json:"user_id,omitempty"`
	Email     string         `json:"email,omitempty"`
	TraceID   string         `json:"trace_id,omitempty"`
	SpanID    string         `json:"span_id,omitempty"`
	Error     string         `json:"error,omitempty"`
	Extra     map[string]any `json:"extra,omitempty"`
}

// Logger interface defines logging contract
type Logger interface {
	Debug(ctx context.Context, message string)
	Info(ctx context.Context, message string)
	Warn(ctx context.Context, message string)
	Error(ctx context.Context, message string, err error)
	Fatal(message string, err error)
	WithRequestID(requestID string) Logger
	WithUserID(userID string) Logger
	WithEmail(email string) Logger
	WithTrace(traceID, spanID string) Logger
}

// logger implements structured JSON logging
type logger struct {
	service   string
	requestID atomicString
	userID    atomicString
	email     atomicString
	traceID   atomicString
	spanID    atomicString
}

// New creates a new structured logger
func New(service string) Logger {
	return &logger{
		service: service,
	}
}

// Debug logs debug level message
func (l *logger) Debug(ctx context.Context, message string) {
	l.log(ctx, LevelDebug, message, nil)
}

// Info logs info level message
func (l *logger) Info(ctx context.Context, message string) {
	l.log(ctx, LevelInfo, message, nil)
}

// Warn logs warning message
func (l *logger) Warn(ctx context.Context, message string) {
	l.log(ctx, LevelWarn, message, nil)
}

// Error logs error message with error details
func (l *logger) Error(ctx context.Context, message string, err error) {
	l.log(ctx, LevelError, message, err)
}

// Fatal logs fatal message and calls os.Exit(1)
func (l *logger) Fatal(message string, err error) {
	l.log(context.Background(), LevelFatal, message, err)
	os.Exit(1)
}

// log is the internal logging method
func (l *logger) log(ctx context.Context, level Level, message string, err error) {
	record := slog.Record{
		Time:    time.Now().UTC(),
		Level:   slogLevel(level),
		Message: message,
	}

	attrs := []slog.Attr{
		slog.String("service", l.service),
	}

	if requestID := l.requestID.Load(); requestID != "" {
		attrs = append(attrs, slog.String("request_id", requestID))
	}
	if userID := l.userID.Load(); userID != "" {
		attrs = append(attrs, slog.String("user_id", userID))
	}
	if email := l.email.Load(); email != "" {
		attrs = append(attrs, slog.String("email", email))
	}
	if traceID := l.traceID.Load(); traceID != "" {
		attrs = append(attrs, slog.String("trace_id", traceID))
	}
	if spanID := l.spanID.Load(); spanID != "" {
		attrs = append(attrs, slog.String("span_id", spanID))
	}
	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
	}

	record.AddAttrs(attrs...)
	if err := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slogLevel(level),
	}).Handle(ctx, record); err != nil {
		return
	}
}

// slogLevel converts our Level to slog.Level
func slogLevel(level Level) slog.Level {
	switch level {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	case LevelFatal:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// WithRequestID returns a new logger with request ID
func (l *logger) WithRequestID(requestID string) Logger {
	newLogger := &logger{
		service: l.service,
	}
	newLogger.requestID.Store(requestID)
	newLogger.userID.Store(l.userID.Load())
	newLogger.email.Store(l.email.Load())
	newLogger.traceID.Store(l.traceID.Load())
	newLogger.spanID.Store(l.spanID.Load())
	return newLogger
}

// WithUserID returns a new logger with user ID
func (l *logger) WithUserID(userID string) Logger {
	newLogger := &logger{
		service: l.service,
	}
	newLogger.requestID.Store(l.requestID.Load())
	newLogger.userID.Store(userID)
	newLogger.email.Store(l.email.Load())
	newLogger.traceID.Store(l.traceID.Load())
	newLogger.spanID.Store(l.spanID.Load())
	return newLogger
}

// WithEmail returns a new logger with email
func (l *logger) WithEmail(email string) Logger {
	newLogger := &logger{
		service: l.service,
	}
	newLogger.requestID.Store(l.requestID.Load())
	newLogger.userID.Store(l.userID.Load())
	newLogger.email.Store(email)
	newLogger.traceID.Store(l.traceID.Load())
	newLogger.spanID.Store(l.spanID.Load())
	return newLogger
}

// WithTrace returns a new logger with trace context
func (l *logger) WithTrace(traceID, spanID string) Logger {
	newLogger := &logger{
		service: l.service,
	}
	newLogger.requestID.Store(l.requestID.Load())
	newLogger.userID.Store(l.userID.Load())
	newLogger.email.Store(l.email.Load())
	newLogger.traceID.Store(traceID)
	newLogger.spanID.Store(spanID)
	return newLogger
}

// ExtractTraceID extracts trace ID from OpenTelemetry context
func ExtractTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return ""
	}
	spanCtx := span.SpanContext()
	return spanCtx.TraceID().String()
}

// ExtractSpanID extracts span ID from OpenTelemetry context
func ExtractSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return ""
	}
	spanCtx := span.SpanContext()
	return spanCtx.SpanID().String()
}
