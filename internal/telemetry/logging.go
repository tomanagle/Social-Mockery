package telemetry

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/trace"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"
)

// Correlate wraps an existing stdout/stderr handler. Callers must use fixed log
// messages and LogContext/InfoContext: payload text never belongs in a message.
// Only known diagnostic keys survive, including attributes supplied via With.
func Correlate(h slog.Handler) slog.Handler { return &correlationHandler{Handler: h} }

type correlationHandler struct{ slog.Handler }

func safeAttr(a slog.Attr) slog.Attr {
	switch a.Key {
	case "provider", "operation", "outcome", "category", "api.version", "http.route", "method", "status", "duration_ms", "duration_seconds", "clock", "trace_id", "span_id":
		a.Value = a.Value.Resolve()
		switch a.Value.Kind() {
		case slog.KindString, slog.KindInt64, slog.KindFloat64, slog.KindBool:
		default:
			return slog.Attr{}
		}
		if a.Value.Kind() == slog.KindString && len(a.Value.String()) > 128 {
			return slog.String(a.Key, "[REDACTED]")
		}
		return a
	default:
		return slog.Attr{}
	}
}
func (h *correlationHandler) Handle(ctx context.Context, r slog.Record) error {
	clean := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	r.Attrs(func(a slog.Attr) bool {
		if a = safeAttr(a); a.Key != "" {
			clean.AddAttrs(a)
		}
		return true
	})
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		sc := span.SpanContext()
		clean.AddAttrs(slog.String("trace_id", sc.TraceID().String()), slog.String("span_id", sc.SpanID().String()))
	}
	return h.Handler.Handle(ctx, clean)
}
func (h *correlationHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	safe := make([]slog.Attr, 0, len(attrs))
	for _, a := range attrs {
		if a = safeAttr(a); a.Key != "" {
			safe = append(safe, a)
		}
	}
	return &correlationHandler{Handler: h.Handler.WithAttrs(safe)}
}

// Grouped arbitrary values are deliberately not accepted by the allowlist.
func (h *correlationHandler) WithGroup(name string) slog.Handler {
	if strings.TrimSpace(name) == "" {
		return h
	}
	return &correlationHandler{Handler: h.Handler.WithGroup(name)}
}

// Logger mirrors sanitized slog records to an optional non-blocking OTLP batch
// exporter, keeping the original stdout/stderr handler active in every mode.
func (t *Telemetry) Logger(base *slog.Logger) *slog.Logger {
	h := base.Handler()
	if t.log != nil {
		h = &otlpHandler{stdout: h, logger: t.log}
	}
	return slog.New(Correlate(h))
}

type otlpHandler struct {
	stdout slog.Handler
	logger otellog.Logger
	attrs  []slog.Attr
	group  string
}

func (h *otlpHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.stdout.Enabled(ctx, l)
}
func (h *otlpHandler) Handle(ctx context.Context, r slog.Record) error {
	var record otellog.Record
	record.SetTimestamp(r.Time)
	record.SetObservedTimestamp(time.Now())
	record.SetSeverity(otellog.Severity(int(r.Level) + 9))
	record.SetSeverityText(r.Level.String())
	record.SetBody(attribute.StringValue(r.Message))
	add := func(a slog.Attr) bool {
		key := a.Key
		if h.group != "" {
			key = h.group + "." + key
		}
		switch a.Value.Kind() {
		case slog.KindString:
			record.AddAttributes(attribute.String(key, a.Value.String()))
		case slog.KindInt64:
			record.AddAttributes(attribute.Int64(key, a.Value.Int64()))
		case slog.KindFloat64:
			record.AddAttributes(attribute.Float64(key, a.Value.Float64()))
		case slog.KindBool:
			record.AddAttributes(attribute.Bool(key, a.Value.Bool()))
		}
		return true
	}
	for _, a := range h.attrs {
		add(a)
	}
	r.Attrs(add)
	h.logger.Emit(ctx, record)
	return h.stdout.Handle(ctx, r)
}
func (h *otlpHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &otlpHandler{stdout: h.stdout.WithAttrs(attrs), logger: h.logger, attrs: append(append([]slog.Attr{}, h.attrs...), attrs...), group: h.group}
}
func (h *otlpHandler) WithGroup(name string) slog.Handler {
	g := name
	if h.group != "" {
		g = h.group + "." + name
	}
	return &otlpHandler{stdout: h.stdout.WithGroup(name), logger: h.logger, attrs: append([]slog.Attr{}, h.attrs...), group: g}
}

// ErrorHandler sanitizes SDK/exporter failures before they reach a logger. Install
// it with otel.SetErrorHandler once in main; pass a non-OTLP logger to avoid an
// export-error logging loop. Repeated backend warnings are rate limited.
func ErrorHandler(logger *slog.Logger) otel.ErrorHandler {
	var last atomic.Int64
	return otel.ErrorHandlerFunc(func(error) {
		now := time.Now().Unix()
		previous := last.Load()
		if now-previous < 60 || !last.CompareAndSwap(previous, now) {
			return
		}
		logger.Warn("telemetry export failed; simulation continues")
	})
}
