package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func testTelemetry(t *testing.T) (*Telemetry, *tracetest.InMemoryExporter, *sdkmetric.ManualReader) {
	t.Helper()
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()), sdktrace.WithSyncer(exp))
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	tel := assemble(tp, mp)
	tel.tp = tp
	tel.mp = mp
	t.Cleanup(func() { _ = tel.Shutdown(context.Background()) })
	return tel, exp, reader
}
func TestHTTPTraceParentAndMetrics(t *testing.T) {
	tel, exp, reader := testTelemetry(t)
	const traceID = "4bf92f3577b34da6a3ce929d0e0e4736"
	h := tel.HTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, child := tel.Start(r.Context(), "media.create")
		Finish(child, nil, false)
		w.WriteHeader(201)
	}))
	r := httptest.NewRequest("POST", "/https://graph.facebook.com/v24.0/account-private-identifier/media?access_token=NEVER_EXPORT", strings.NewReader("private caption"))
	r.Header.Set("traceparent", "00-"+traceID+"-00f067aa0ba902b7-01")
	r.Header.Set("Authorization", "Bearer NEVER_EXPORT")
	r.Header.Set("baggage", "password=NEVER_EXPORT")
	h.ServeHTTP(httptest.NewRecorder(), r)
	spans := exp.GetSpans()
	if len(spans) != 3 {
		t.Fatalf("got %d spans", len(spans))
	}
	child, parent := spans[1], spans[2]
	if child.Parent.SpanID() != parent.SpanContext.SpanID() || parent.Parent.SpanID().String() != "00f067aa0ba902b7" {
		t.Fatal("broken parent-child relationship")
	}
	if parent.SpanContext.TraceID().String() != traceID || parent.SpanKind != trace.SpanKindServer {
		t.Fatal("incoming trace was not propagated")
	}
	if parent.Name != "POST /{version}/{id}/media" {
		t.Fatalf("unexpected name %q", parent.Name)
	}
	b, _ := json.Marshal(spans)
	for _, secret := range []string{"NEVER_EXPORT", "private caption", "account-private-identifier", "password"} {
		if strings.Contains(string(b), secret) {
			t.Fatalf("sensitive content exported: %s", secret)
		}
	}
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, scope := range rm.ScopeMetrics {
		for _, m := range scope.Metrics {
			if m.Name == "social_mockery.http.requests" {
				sum := m.Data.(metricdata.Sum[int64])
				if len(sum.DataPoints) != 1 || sum.DataPoints[0].Value != 1 {
					t.Fatal("missing request count")
				}
				attrs := sum.DataPoints[0].Attributes
				value, _ := attrs.Value(attribute.Key("http.response.status_code"))
				if value.AsInt64() != 201 {
					t.Fatal("missing status")
				}
				found = true
			}
		}
	}
	if !found {
		t.Fatal("missing HTTP metric")
	}
}
func TestExpectedFailureVersusInternalError(t *testing.T) {
	for _, expected := range []bool{true, false} {
		t.Run(fmt.Sprint(expected), func(t *testing.T) {
			tel, exp, _ := testTelemetry(t)
			tel.HTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if expected {
					ExpectedFailure(r.Context())
				}
				_, s := tel.Start(r.Context(), "publishing")
				Finish(s, errors.New("secret_token_in_error"), expected)
				w.WriteHeader(500)
			})).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/https://graph.facebook.com/v24.0/123/media_publish", nil))
			for _, s := range exp.GetSpans() {
				if s.Name == "provider.route" {
					continue
				}
				if (s.Status.Code == codes.Error) == expected {
					t.Fatalf("incorrect error classification: %v", s.Status)
				}
				b, _ := json.Marshal(s)
				if strings.Contains(string(b), "secret_token") {
					t.Fatal("error leaks secret")
				}
			}
		})
	}
}
func TestStructuredLogsCorrelateAndRedact(t *testing.T) {
	tel, _, _ := testTelemetry(t)
	var out bytes.Buffer
	logger := tel.Logger(slog.New(slog.NewJSONHandler(&out, nil))).With("access_token", "NEVER_EXPORT", "provider", "meta")
	ctx, span := tel.Start(context.Background(), "publish")
	logger.InfoContext(ctx, "operation completed", "password", "NEVER_EXPORT", "payload", map[string]string{"caption": "NEVER_EXPORT"}, "status", 200)
	span.End()
	var record map[string]any
	if err := json.Unmarshal(out.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	if record["trace_id"] != span.SpanContext().TraceID().String() || record["span_id"] != span.SpanContext().SpanID().String() {
		t.Fatalf("missing log correlation: %s versus %v", out.String(), span.SpanContext())
	}
	if strings.Contains(out.String(), "NEVER_EXPORT") {
		t.Fatal("log leaked secret")
	}
	out.Reset()
	Noop().Logger(slog.New(slog.NewJSONHandler(&out, nil))).Info("server started")
	if strings.Contains(out.String(), "trace_id") || out.Len() == 0 {
		t.Fatal("disabled logs should remain available without trace ID")
	}
}
func TestConfigSignalIndependence(t *testing.T) {
	t.Setenv("OTEL_SDK_DISABLED", "false")
	t.Setenv("OTEL_TRACES_EXPORTER", "otlp")
	t.Setenv("OTEL_METRICS_EXPORTER", "none")
	t.Setenv("OTEL_LOGS_EXPORTER", "none")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "grpc")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_PROTOCOL", "http/protobuf")
	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_PROTOCOL", "")
	c := ConfigFromEnv("1.2.3")
	if !c.Traces || c.Metrics || c.Logs || c.TraceProtocol != "http/protobuf" || c.MetricProtocol != "grpc" {
		t.Fatalf("bad config: %+v", c)
	}
	t.Setenv("OTEL_SDK_DISABLED", "true")
	c = ConfigFromEnv("test")
	if c.Traces || c.Metrics || c.Logs {
		t.Fatal("SDK_DISABLED ignored")
	}
}
func TestDisabledTelemetryMakesNoRequests(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests.Add(1) }))
	defer server.Close()
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", server.URL)
	tel, err := New(context.Background(), Config{}, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	ctx, s := tel.Start(context.Background(), "test")
	Finish(s, nil, false)
	tel.ObserveOperation(ctx, "test", "meta", "success", time.Second)
	if err := tel.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 0 || tel.tp != nil || tel.mp != nil || tel.lp != nil {
		t.Fatal("disabled telemetry created exporters")
	}
}

type blockingExporter struct {
	called  atomic.Bool
	stopped atomic.Bool
}

func (e *blockingExporter) ExportSpans(ctx context.Context, _ []sdktrace.ReadOnlySpan) error {
	e.called.Store(true)
	<-ctx.Done()
	return ctx.Err()
}
func (e *blockingExporter) Shutdown(context.Context) error { e.stopped.Store(true); return nil }
func TestExporterOutageDoesNotBlockRequestsAndShutdownIsBounded(t *testing.T) {
	e := &blockingExporter{}
	tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()), sdktrace.WithBatcher(e, sdktrace.WithMaxQueueSize(2), sdktrace.WithMaxExportBatchSize(1), sdktrace.WithBatchTimeout(time.Millisecond), sdktrace.WithExportTimeout(20*time.Millisecond)))
	tel := Noop()
	tel.tracer = tp.Tracer("test")
	tel.tp = tp
	tel.shutdownTimeout = 100 * time.Millisecond
	start := time.Now()
	for i := 0; i < 50; i++ {
		_, s := tel.Start(context.Background(), "operation")
		s.End()
	}
	if time.Since(start) > time.Second {
		t.Fatal("exporter blocked request path")
	}
	start = time.Now()
	_ = tel.Shutdown(context.Background())
	if time.Since(start) > time.Second {
		t.Fatal("unbounded shutdown")
	}
	if !e.called.Load() {
		t.Fatal("shutdown did not attempt buffered export")
	}
}

type recordingLogExporter struct {
	mu      sync.Mutex
	records []sdklog.Record
	stopped bool
}

func (e *recordingLogExporter) Export(_ context.Context, rs []sdklog.Record) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, r := range rs {
		e.records = append(e.records, r.Clone())
	}
	return nil
}
func (e *recordingLogExporter) Shutdown(context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.stopped = true
	return nil
}
func (e *recordingLogExporter) ForceFlush(context.Context) error { return nil }
func TestShutdownFlushesLogsAndSpans(t *testing.T) {
	spanExp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()), sdktrace.WithBatcher(spanExp, sdktrace.WithBatchTimeout(time.Hour)))
	// Span exporter Reset on Shutdown prevents inspecting afterward; a recorder is
	// used to verify the batch was exported before Shutdown clears it.
	capture := &captureSpanExporter{}
	_ = tp.Shutdown(context.Background())
	tp = sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()), sdktrace.WithBatcher(capture, sdktrace.WithBatchTimeout(time.Hour)))
	logExp := &recordingLogExporter{}
	lp := sdklog.NewLoggerProvider(sdklog.WithProcessor(sdklog.NewBatchProcessor(logExp, sdklog.WithExportInterval(time.Hour))))
	tel := Noop()
	tel.tp = tp
	tel.tracer = tp.Tracer("test")
	tel.lp = lp
	tel.log = lp.Logger("test")
	var out bytes.Buffer
	logger := tel.Logger(slog.New(slog.NewJSONHandler(&out, nil)))
	ctx, s := tel.Start(context.Background(), "test")
	logger.InfoContext(ctx, "operation completed", "access_token", "private")
	s.End()
	if err := tel.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	if capture.count != 1 || !capture.stopped || len(logExp.records) != 1 || !logExp.stopped {
		t.Fatalf("shutdown did not flush and stop providers: spans=%d stopped=%v logs=%d stopped=%v", capture.count, capture.stopped, len(logExp.records), logExp.stopped)
	}
	if logExp.records[0].TraceID() != s.SpanContext().TraceID() {
		t.Fatal("exported log missing trace correlation")
	}
	logExp.records[0].WalkAttributes(func(a attribute.KeyValue) bool {
		if strings.Contains(fmt.Sprint(a), "private") {
			t.Fatal("OTLP log leak")
		}
		return true
	})
	if err := tel.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}

type captureSpanExporter struct {
	count   int
	stopped bool
}

func (e *captureSpanExporter) ExportSpans(_ context.Context, s []sdktrace.ReadOnlySpan) error {
	e.count += len(s)
	return nil
}
func (e *captureSpanExporter) Shutdown(context.Context) error { e.stopped = true; return nil }
func TestRouteBounds(t *testing.T) {
	for _, path := range []string{"/arbitrary/private_id?token=private", "/https://example.com/private", "/https://user:private@graph.facebook.com/v24.0/123/media", "/https://graph.facebook.com/v99999/123/private"} {
		_, version, route := Route(httptest.NewRequest("GET", path, nil))
		if route != "unmatched" || version != "none" {
			t.Fatalf("unsafe route labels %s %s", version, route)
		}
	}
}

func TestLogAttributeTypesCannotBypassRedaction(t *testing.T) {
	var out bytes.Buffer
	logger := Noop().Logger(slog.New(slog.NewJSONHandler(&out, nil)))
	logger.Info("operation completed", slog.Any("provider", map[string]string{"access_token": "PRIVATE_VALUE"}), slog.Group("operation", slog.String("token", "PRIVATE_VALUE")))
	if strings.Contains(out.String(), "PRIVATE_VALUE") {
		t.Fatal("nested attribute leaked")
	}
}
func TestSDKErrorHandlerSanitizesAndRateLimits(t *testing.T) {
	var out bytes.Buffer
	handler := ErrorHandler(slog.New(slog.NewJSONHandler(&out, nil)))
	handler.Handle(errors.New("collector http://host/?token=SECRET returned PRIVATE_BODY"))
	handler.Handle(errors.New("SECRET"))
	if strings.Contains(out.String(), "SECRET") || strings.Contains(out.String(), "PRIVATE_BODY") {
		t.Fatal("exporter failure leak")
	}
	if strings.Count(out.String(), "telemetry export failed") != 1 {
		t.Fatal("export errors were not rate limited")
	}
}
func BenchmarkNoopRequest(b *testing.B) {
	tel := Noop()
	h := tel.HTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	r := httptest.NewRequest("GET", "/healthz", nil)
	b.ReportAllocs()
	for b.Loop() {
		h.ServeHTTP(httptest.NewRecorder(), r)
	}
}

func TestOTLPHTTPExportsAllSignalsAndResourceSettings(t *testing.T) {
	var mu sync.Mutex
	paths := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths[r.URL.Path]++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/x-protobuf")
		w.WriteHeader(200)
	}))
	defer server.Close()
	t.Setenv("OTEL_TRACES_SAMPLER", "always_on")
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "service.version=test-version")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", server.URL)
	t.Setenv("OTEL_EXPORTER_OTLP_HEADERS", "")
	for _, signal := range []string{"TRACES", "METRICS", "LOGS"} {
		t.Setenv("OTEL_EXPORTER_OTLP_"+signal+"_ENDPOINT", server.URL+"/v1/"+strings.ToLower(signal))
		t.Setenv("OTEL_EXPORTER_OTLP_"+signal+"_HEADERS", "")
	}
	var out bytes.Buffer
	tel, err := New(context.Background(), Config{Traces: true, Metrics: true, Logs: true, ServiceName: "test-simulator", TraceProtocol: "http/protobuf", MetricProtocol: "http/protobuf", LogProtocol: "http/protobuf"}, slog.New(slog.NewJSONHandler(&out, nil)))
	if err != nil {
		t.Fatal(err)
	}
	ctx, span := tel.Start(context.Background(), "request")
	tel.Publish(ctx, "instagram", "success")
	tel.Logger(slog.New(slog.NewJSONHandler(&out, nil))).InfoContext(ctx, "operation completed")
	span.End()
	if err := tel.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	for _, signal := range []string{"traces", "metrics", "logs"} {
		if paths["/v1/"+signal] == 0 {
			t.Fatalf("signal %s not exported: %v", signal, paths)
		}
	}
}
