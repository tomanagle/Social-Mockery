package telemetry

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// Route returns a bounded, parameter-free route name. It does not export raw
// input even for malformed URLs, unknown providers, or unsupported endpoints.
func Route(r *http.Request) (provider, version, route string) {
	provider, version, route = "none", "none", "unmatched"
	switch r.URL.Path {
	case "/healthz", "/admin/clock", "/admin/clock/advance", "/admin/reset":
		return provider, version, r.URL.Path
	}
	if !strings.HasPrefix(r.RequestURI, "/https://") {
		return
	}
	u, err := url.Parse(strings.TrimPrefix(r.RequestURI, "/"))
	if err != nil || u.User != nil {
		return
	}
	if u.Host != "graph.facebook.com" && u.Host != "graph.instagram.com" {
		return
	}
	provider = "meta"
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 || (parts[0] != "v24.0" && parts[0] != "v25.0") {
		return
	}
	version = parts[0]
	switch {
	case len(parts) == 2:
		route = "/{version}/{id}"
	case len(parts) == 3 && parts[2] == "media":
		route = "/{version}/{id}/media"
	case len(parts) == 3 && parts[2] == "media_publish":
		route = "/{version}/{id}/media_publish"
	}
	return
}
func method(m string) string {
	switch m {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
		return m
	default:
		return "OTHER"
	}
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(status int) {
	if status >= 100 && status < 200 {
		w.ResponseWriter.WriteHeader(status)
		return
	}
	if w.status == 0 {
		w.status = status
		w.ResponseWriter.WriteHeader(status)
	}
}
func (w *responseWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(p)
}
func (w *responseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

type requestMetadataKey struct{}
type requestMetadata struct{ provider, version string }

func (t *Telemetry) HTTP(next http.Handler) http.Handler {
	if !t.enabled {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		ctx, span := t.tracer.Start(ctx, "HTTP request", trace.WithSpanKind(trace.SpanKindServer))
		_, routeSpan := t.Start(ctx, "provider.route")
		provider, version, route := Route(r)
		m := method(r.Method)
		attrs := []attribute.KeyValue{attribute.String("http.request.method", m), attribute.String("http.route", route), attribute.String("provider", provider), attribute.String("api.version", version)}
		routeSpan.SetAttributes(attrs...)
		routeSpan.End()
		span.SetName(m + " " + route)
		span.SetAttributes(attrs...)
		ctx = context.WithValue(ctx, requestMetadataKey{}, requestMetadata{provider, version})
		rw := &responseWriter{ResponseWriter: w}
		defer func() {
			recovered := recover()
			if recovered != nil {
				span.SetStatus(codes.Error, "handler panic")
				span.AddEvent("handler panic")
				if rw.status == 0 {
					rw.WriteHeader(500)
				}
			}
			status := rw.status
			if status == 0 {
				status = 200
			}
			attrs = append(attrs, attribute.Int("http.response.status_code", status))
			span.SetAttributes(attribute.Int("http.response.status_code", status))
			// An injected provider 500 can be expected: provider code marks that outcome.
			if status >= 500 && !expectedFailure(ctx) {
				span.SetStatus(codes.Error, "internal request failure")
			}
			t.requests.Add(ctx, 1, metric.WithAttributes(attrs...))
			t.requestDuration.Record(ctx, time.Since(started).Seconds(), metric.WithAttributes(attrs...))
			span.End()
			if recovered != nil {
				panic(recovered)
			}
		}()
		ctx = withOutcome(ctx)
		next.ServeHTTP(rw, r.WithContext(ctx))
	})
}
