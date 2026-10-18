package app

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"

	"github.com/tomanagle/Social-Mockery/internal/config"
	"github.com/tomanagle/Social-Mockery/internal/profiles"
	"github.com/tomanagle/Social-Mockery/internal/store/db"
	"github.com/tomanagle/Social-Mockery/internal/store/dbstore"
	"github.com/tomanagle/Social-Mockery/internal/telemetry"
)

func TestMustInitializesInjectedDatabase(t *testing.T) {
	connection, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	cfg := config.Default()
	cfg.Addr = "127.0.0.1:0"
	application := Must(cfg, profiles.Default(), connection, telemetry.Noop(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	initialized, err := dbstore.New(connection).Initialized(context.Background())
	if err != nil || !initialized {
		t.Fatalf("initialized = %v, %v", initialized, err)
	}
	addr := application.listener.Addr().String()
	if err := application.Close(); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("Close did not release listener: %v", err)
	}
	listener.Close()
	if err := connection.Ping(); err != nil {
		t.Fatalf("Close closed caller-owned database: %v", err)
	}
}

func TestMustPanicsOnRequiredStartupFailures(t *testing.T) {
	for _, failure := range []string{"database", "listener", "seed"} {
		t.Run(failure, func(t *testing.T) {
			connection, err := db.Open(":memory:")
			if err != nil {
				t.Fatal(err)
			}
			defer connection.Close()
			cfg := config.Default()
			cfg.Addr = "127.0.0.1:0"
			seed := profiles.Default()
			switch failure {
			case "database":
				connection.Close()
			case "listener":
				cfg.Addr = "invalid-secret-listen-address"
			case "seed":
				seed.Version = 0
			}
			defer func() {
				if got := recover(); got != "app: initialization failed" {
					t.Fatalf("panic = %v", got)
				}
			}()
			application := Must(cfg, seed, connection, telemetry.Noop(), slog.New(slog.NewTextHandler(io.Discard, nil)))
			application.Close()
			t.Fatal("Must did not panic")
		})
	}
}
