package handlers

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"github.com/tomanagle/Social-Mockery/internal/profiles"
	"github.com/tomanagle/Social-Mockery/internal/providers/meta"
	"github.com/tomanagle/Social-Mockery/internal/simulation"
	"github.com/tomanagle/Social-Mockery/internal/telemetry"
	"io"
	"net/http"
	"strings"
	"time"
)

func New(engine *simulation.Engine, adminToken string, instrumentation ...*telemetry.Telemetry) http.Handler {
	t := telemetry.Noop()
	if len(instrumentation) > 0 && instrumentation[0] != nil {
		t = instrumentation[0]
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { meta.JSON(w, 200, map[string]string{"status": "ok"}) })
	admin := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Origin") != "" {
				meta.JSON(w, 403, map[string]string{"error": "browser-origin administrative requests are disabled"})
				return
			}
			if adminToken != "" && subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+adminToken)) != 1 {
				meta.JSON(w, 401, map[string]string{"error": "admin token required"})
				return
			}
			name := "clock.read"
			switch r.URL.Path {
			case "/admin/reset":
				name = "simulation.reset"
			case "/admin/clock/advance":
				name = "clock.advance"
			}
			started := time.Now()
			ctx, span := t.Start(r.Context(), name)
			recorded := &adminStatus{ResponseWriter: w, status: 200}
			defer func() {
				outcome := "success"
				var err error
				if recorded.status >= 400 {
					outcome = "error"
					err = errors.New("administrative operation failed")
				}
				telemetry.Finish(span, err, recorded.status < 500)
				t.ObserveOperation(ctx, name, "simulator", outcome, time.Since(started))
			}()
			engine.Mu.Lock()
			defer engine.Mu.Unlock()
			next(recorded, r.WithContext(ctx))
		}
	}
	mux.HandleFunc("GET /admin/clock", admin(func(w http.ResponseWriter, r *http.Request) {
		now, delay, err := engine.Store.Clock(r.Context())
		if err != nil {
			meta.JSON(w, 500, map[string]string{"error": "storage failure"})
			return
		}
		meta.JSON(w, 200, map[string]any{"now": time.Unix(now, 0).UTC().Format(time.RFC3339), "unix": now, "processing_seconds": delay})
	}))
	mux.HandleFunc("POST /admin/clock/advance", admin(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Seconds *int64 `json:"seconds"`
		}
		d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024))
		d.DisallowUnknownFields()
		if err := d.Decode(&body); err != nil || body.Seconds == nil {
			meta.JSON(w, 400, map[string]string{"error": "JSON integer seconds is required"})
			return
		}
		var extra any
		if d.Decode(&extra) != io.EOF {
			meta.JSON(w, 400, map[string]string{"error": "one JSON object is required"})
			return
		}
		if *body.Seconds < 0 || *body.Seconds > 31536000 {
			meta.JSON(w, 400, map[string]string{"error": "seconds must be 0..31536000"})
			return
		}
		now, err := engine.Store.Advance(r.Context(), *body.Seconds)
		if err != nil {
			meta.JSON(w, 500, map[string]string{"error": "clock update failed"})
			return
		}
		meta.JSON(w, 200, map[string]any{"now": time.Unix(now, 0).UTC().Format(time.RFC3339), "unix": now})
	}))
	mux.HandleFunc("POST /admin/reset", admin(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, profiles.MaxBytes))
		if err != nil {
			meta.JSON(w, 400, map[string]string{"error": "profile exceeds body limit"})
			return
		}
		seed := engine.Seed
		if len(strings.TrimSpace(string(body))) > 0 {
			seed, err = profiles.Decode(strings.NewReader(string(body)))
			if err != nil {
				meta.JSON(w, 400, map[string]string{"error": err.Error()})
				return
			}
		}
		if err = engine.Reset(r.Context(), seed); err != nil {
			meta.JSON(w, 500, map[string]string{"error": "reset failed"})
			return
		}
		meta.JSON(w, 200, map[string]bool{"reset": true})
	}))
	provider := meta.New(engine, t)
	slots := make(chan struct{}, 128)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case slots <- struct{}{}:
			defer func() { <-slots }()
		default:
			meta.JSON(w, 503, map[string]string{"error": "simulator busy"})
			return
		}
		if len(r.RequestURI) > 16384 {
			meta.JSON(w, 414, map[string]string{"error": "request URI too long"})
			return
		}
		if strings.HasPrefix(r.RequestURI, "/https:") || strings.HasPrefix(r.RequestURI, "/http:") {
			provider.ServeHTTP(w, r)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

type adminStatus struct {
	http.ResponseWriter
	status int
}

func (w *adminStatus) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
