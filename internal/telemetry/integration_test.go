package telemetry_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tomanagle/Social-Mockery/internal/handlers"
	"github.com/tomanagle/Social-Mockery/internal/profiles"
	"github.com/tomanagle/Social-Mockery/internal/simulation"
	"github.com/tomanagle/Social-Mockery/internal/store/db"
	"github.com/tomanagle/Social-Mockery/internal/store/dbstore"
	"github.com/tomanagle/Social-Mockery/internal/telemetry"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestInstagramRequestHasProviderAuthAndDatabaseChildren(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()), sdktrace.WithSyncer(exp))
	defer tp.Shutdown(context.Background())
	tel := telemetry.NewWithProviders(tp, nil)
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	seed := profiles.Default()
	engine := simulation.New(telemetry.ObserveStore(dbstore.New(database), tel), seed, 100)
	if err := engine.Reset(context.Background(), seed); err != nil {
		t.Fatal(err)
	}
	exp.Reset()
	h := tel.HTTP(handlers.New(engine, "", tel))
	request := httptest.NewRequest("POST", "/https://graph.instagram.com/v25.0/17841400000000001/media", strings.NewReader(`{"image_url":"https://example.com/private.jpg","caption":"PRIVATE_CAPTION"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer mock-token")
	request.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	response := httptest.NewRecorder()
	h.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("create failed %d %s", response.Code, response.Body.String())
	}
	spans := exp.GetSpans()
	byName := map[string]tracetest.SpanStub{}
	byID := map[trace.SpanID]tracetest.SpanStub{}
	for _, span := range spans {
		byName[span.Name] = span
		byID[span.SpanContext.SpanID()] = span
	}
	for _, name := range []string{"POST /{version}/{id}/media", "instagram.media.create", "auth.validate", "db.token", "db.createcontainer"} {
		if _, ok := byName[name]; !ok {
			t.Fatalf("missing %s in %v", name, byName)
		}
	}
	root := byName["POST /{version}/{id}/media"]
	for _, span := range spans {
		if span.SpanContext.TraceID() != root.SpanContext.TraceID() {
			t.Fatalf("disconnected span %s", span.Name)
		}
		if span.Name != root.Name {
			if _, ok := byID[span.Parent.SpanID()]; !ok {
				t.Fatalf("parent missing for %s", span.Name)
			}
		}
	}
	token := byName["db.token"]
	if token.Parent.SpanID() != byName["auth.validate"].SpanContext.SpanID() {
		t.Fatal("database auth lookup must belong to authentication span")
	}
	data, _ := json.Marshal(spans)
	for _, secret := range []string{"mock-token", "PRIVATE_CAPTION", "private.jpg", "17841400000000001"} {
		if strings.Contains(string(data), secret) {
			t.Fatalf("sensitive field exported %s", secret)
		}
	}
}
