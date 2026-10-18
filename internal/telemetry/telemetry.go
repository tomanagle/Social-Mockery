// Package telemetry owns optional, bounded OpenTelemetry providers. It never
// exports request URLs, bodies, headers, credentials, or database statements.
package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

// Config is explicit wiring; endpoints, TLS, headers, sampler and resource
// attributes are read by the official SDK from standard OTEL_* variables.
type Config struct {
	Traces, Metrics, Logs                      bool
	ServiceName, Version                       string
	TraceProtocol, MetricProtocol, LogProtocol string
	ShutdownTimeout                            time.Duration
}

func ConfigFromEnv(version string) Config {
	disabled := strings.EqualFold(os.Getenv("OTEL_SDK_DISABLED"), "true")
	name := os.Getenv("OTEL_SERVICE_NAME")
	if name == "" {
		name = "social-mockery"
	}
	return Config{Traces: !disabled && os.Getenv("OTEL_TRACES_EXPORTER") == "otlp", Metrics: !disabled && os.Getenv("OTEL_METRICS_EXPORTER") == "otlp", Logs: !disabled && os.Getenv("OTEL_LOGS_EXPORTER") == "otlp", ServiceName: name, Version: version, LogProtocol: protocol("LOGS"), TraceProtocol: protocol("TRACES"), MetricProtocol: protocol("METRICS"), ShutdownTimeout: 5 * time.Second}
}
func protocol(signal string) string {
	if v := os.Getenv("OTEL_EXPORTER_OTLP_" + signal + "_PROTOCOL"); v != "" {
		return v
	}
	if v := os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL"); v != "" {
		return v
	}
	return "http/protobuf"
}

type Telemetry struct {
	enabled            bool
	tracer             trace.Tracer
	logger             *slog.Logger
	meter              metric.Meter
	tp                 *sdktrace.TracerProvider
	mp                 *sdkmetric.MeterProvider
	lp                 *sdklog.LoggerProvider
	log                otellog.Logger
	shutdownTimeout    time.Duration
	once               sync.Once
	shutdownErr        error
	requests           metric.Int64Counter
	requestDuration    metric.Float64Histogram
	operationDuration  metric.Float64Histogram
	dbDuration         metric.Float64Histogram
	publishes          metric.Int64Counter
	simulatedErrors    metric.Int64Counter
	processingDuration metric.Float64Histogram
	accounts           metric.Int64Gauge
	posts              metric.Int64Gauge
}

func Noop() *Telemetry {
	t := assemble(tracenoop.NewTracerProvider(), metricnoop.NewMeterProvider())
	t.enabled = false
	return t
}
func assemble(tp trace.TracerProvider, mp metric.MeterProvider) *Telemetry {
	t := &Telemetry{enabled: true, tracer: tp.Tracer("social-mockery"), meter: mp.Meter("social-mockery"), shutdownTimeout: 5 * time.Second}
	t.requests, _ = t.meter.Int64Counter("social_mockery.http.requests")
	t.requestDuration, _ = t.meter.Float64Histogram("http.server.request.duration", metric.WithUnit("s"))
	t.operationDuration, _ = t.meter.Float64Histogram("social_mockery.simulation.duration", metric.WithUnit("s"))
	t.dbDuration, _ = t.meter.Float64Histogram("social_mockery.db.duration", metric.WithUnit("s"))
	t.publishes, _ = t.meter.Int64Counter("social_mockery.publishing.operations")
	t.simulatedErrors, _ = t.meter.Int64Counter("social_mockery.simulated.errors")
	t.processingDuration, _ = t.meter.Float64Histogram("social_mockery.media.processing.duration", metric.WithUnit("s"))
	t.accounts, _ = t.meter.Int64Gauge("social_mockery.accounts.active")
	t.posts, _ = t.meter.Int64Gauge("social_mockery.posts.active")
	return t
}

// New uses asynchronous, non-blocking batching. Backend outages never enter the
// request path. Constructor errors indicate local configuration errors only.
func New(ctx context.Context, cfg Config, logger *slog.Logger) (*Telemetry, error) {
	if !cfg.Traces && !cfg.Metrics && !cfg.Logs {
		t := Noop()
		t.logger = t.Logger(logger)
		return t, nil
	}
	if cfg.ServiceName == "" {
		cfg.ServiceName = "social-mockery"
	}
	res, err := resource.New(ctx, resource.WithAttributes(attribute.String("service.name", cfg.ServiceName), attribute.String("service.version", cfg.Version)), resource.WithFromEnv())
	if err != nil {
		return nil, errors.New("invalid OpenTelemetry resource configuration")
	}
	var tp *sdktrace.TracerProvider
	var mp *sdkmetric.MeterProvider
	if cfg.Traces {
		var exp sdktrace.SpanExporter
		switch cfg.TraceProtocol {
		case "grpc":
			exp, err = otlptracegrpc.New(ctx)
		case "", "http/protobuf":
			exp, err = otlptracehttp.New(ctx)
		default:
			return nil, fmt.Errorf("unsupported trace protocol (use grpc or http/protobuf)")
		}
		if err != nil {
			return nil, errors.New("cannot initialize trace exporter")
		}
		// Never use WithBlocking: full queues drop telemetry rather than simulations.
		tp = sdktrace.NewTracerProvider(sdktrace.WithResource(res), sdktrace.WithSpanLimits(sdktrace.SpanLimits{AttributeValueLengthLimit: 128, AttributeCountLimit: 24, EventCountLimit: 16, LinkCountLimit: 8, AttributePerEventCountLimit: 8, AttributePerLinkCountLimit: 8}), sdktrace.WithBatcher(exp, sdktrace.WithMaxQueueSize(1024), sdktrace.WithMaxExportBatchSize(256), sdktrace.WithExportTimeout(3*time.Second), sdktrace.WithBatchTimeout(time.Second)))
	}
	if cfg.Metrics {
		var exp sdkmetric.Exporter
		switch cfg.MetricProtocol {
		case "grpc":
			exp, err = otlpmetricgrpc.New(ctx)
		case "", "http/protobuf":
			exp, err = otlpmetrichttp.New(ctx)
		default:
			err = errors.New("unsupported metric protocol")
		}
		if err != nil {
			if tp != nil {
				shutdown, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				_ = tp.Shutdown(shutdown)
			}
			return nil, errors.New("cannot initialize metric exporter")
		}
		mp = sdkmetric.NewMeterProvider(sdkmetric.WithResource(res), sdkmetric.WithCardinalityLimit(128), sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exp, sdkmetric.WithTimeout(3*time.Second))))
	}
	var lp *sdklog.LoggerProvider
	if cfg.Logs {
		var exp sdklog.Exporter
		switch cfg.LogProtocol {
		case "grpc":
			exp, err = otlploggrpc.New(ctx)
		case "", "http/protobuf":
			exp, err = otlploghttp.New(ctx)
		default:
			err = errors.New("unsupported log protocol")
		}
		if err != nil {
			shutdown, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if tp != nil {
				_ = tp.Shutdown(shutdown)
			}
			if mp != nil {
				_ = mp.Shutdown(shutdown)
			}
			return nil, errors.New("cannot initialize log exporter")
		}
		lp = sdklog.NewLoggerProvider(sdklog.WithResource(res), sdklog.WithAttributeCountLimit(24), sdklog.WithAttributeValueLengthLimit(128), sdklog.WithProcessor(sdklog.NewBatchProcessor(exp, sdklog.WithMaxQueueSize(1024), sdklog.WithExportMaxBatchSize(256), sdklog.WithExportTimeout(3*time.Second))))
	}
	var traceProvider trace.TracerProvider = tracenoop.NewTracerProvider()
	var meterProvider metric.MeterProvider = metricnoop.NewMeterProvider()
	if tp != nil {
		traceProvider = tp
	}
	if mp != nil {
		meterProvider = mp
	}
	t := assemble(traceProvider, meterProvider)
	t.tp = tp
	t.mp = mp
	t.lp = lp
	if lp != nil {
		t.log = lp.Logger("social-mockery")
	}
	t.logger = t.Logger(logger)
	if cfg.ShutdownTimeout > 0 {
		t.shutdownTimeout = cfg.ShutdownTimeout
	}
	return t, nil
}

func (t *Telemetry) Shutdown(ctx context.Context) error {
	t.once.Do(func() {
		// Each signal gets a share so one failing backend cannot prevent cleanup of another.
		timeout := t.shutdownTimeout
		if timeout <= 0 {
			timeout = 5 * time.Second
		}
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		var wg sync.WaitGroup
		var mu sync.Mutex
		for _, close := range []func(context.Context) error{func(c context.Context) error {
			if t.tp != nil {
				return t.tp.Shutdown(c)
			}
			return nil
		}, func(c context.Context) error {
			if t.mp != nil {
				return t.mp.Shutdown(c)
			}
			return nil
		}, func(c context.Context) error {
			if t.lp != nil {
				return t.lp.Shutdown(c)
			}
			return nil
		}} {
			wg.Add(1)
			go func(close func(context.Context) error) {
				defer wg.Done()
				err := close(ctx)
				mu.Lock()
				t.shutdownErr = errors.Join(t.shutdownErr, err)
				mu.Unlock()
			}(close)
		}
		wg.Wait()
	})
	return t.shutdownErr
}

// Start accepts only developer-owned operation names, never request paths.
func (t *Telemetry) Start(ctx context.Context, operation string) (context.Context, trace.Span) {
	attrs := []attribute.KeyValue{attribute.String("operation", operation)}
	if m, ok := ctx.Value(requestMetadataKey{}).(requestMetadata); ok {
		attrs = append(attrs, attribute.String("provider", m.provider), attribute.String("api.version", m.version))
	}
	return t.tracer.Start(ctx, operation, trace.WithAttributes(attrs...))
}

// Finish never copies raw error strings (which can contain SQL parameters or
// tokens) into telemetry. Expected provider rejections remain non-error spans.
func Finish(span trace.Span, err error, expected bool) {
	if err != nil {
		if expected {
			span.SetAttributes(attribute.String("social_mockery.outcome", "simulated_error"))
		} else {
			span.RecordError(errors.New("internal operation failure"))
			span.SetStatus(codes.Error, "internal operation failure")
			span.SetAttributes(attribute.String("social_mockery.outcome", "internal_error"))
		}
	} else {
		span.SetAttributes(attribute.String("social_mockery.outcome", "success"))
	}
	span.End()
}
func (t *Telemetry) ObserveOperation(ctx context.Context, operation, provider, outcome string, elapsed time.Duration) {
	if t.logger != nil {
		t.logger.DebugContext(ctx, "simulation operation completed", "operation", operation, "provider", provider, "outcome", outcome, "duration_seconds", elapsed.Seconds())
	}
	t.operationDuration.Record(ctx, elapsed.Seconds(), metric.WithAttributes(attribute.String("operation", operation), attribute.String("provider", provider), attribute.String("outcome", outcome)))
}
func (t *Telemetry) Publish(ctx context.Context, provider, outcome string) {
	t.publishes.Add(ctx, 1, metric.WithAttributes(attribute.String("provider", provider), attribute.String("outcome", outcome)))
}
func (t *Telemetry) SimulatedError(ctx context.Context, provider, category string) {
	t.simulatedErrors.Add(ctx, 1, metric.WithAttributes(attribute.String("provider", provider), attribute.String("category", category)))
}
func (t *Telemetry) Processing(ctx context.Context, seconds float64) {
	t.processingDuration.Record(ctx, seconds, metric.WithAttributes(attribute.String("provider", "meta")))
}
func (t *Telemetry) State(ctx context.Context, accounts, posts int64) {
	t.accounts.Record(ctx, accounts)
	t.posts.Record(ctx, posts)
}

// W3C trace context only: baggage can contain private application data.
var propagator = propagation.TraceContext{}

// NewWithProviders supports embedding and in-memory testing. The caller retains
// ownership of the supplied providers and must flush/shut them down itself.
func NewWithProviders(tp trace.TracerProvider, mp metric.MeterProvider) *Telemetry {
	if tp == nil {
		tp = tracenoop.NewTracerProvider()
	}
	if mp == nil {
		mp = metricnoop.NewMeterProvider()
	}
	return assemble(tp, mp)
}
