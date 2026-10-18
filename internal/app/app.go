// Package app composes injected dependencies into a ready-to-serve application.
package app

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tomanagle/Social-Mockery/internal/config"
	"github.com/tomanagle/Social-Mockery/internal/handlers"
	"github.com/tomanagle/Social-Mockery/internal/middleware"
	"github.com/tomanagle/Social-Mockery/internal/simulation"
	"github.com/tomanagle/Social-Mockery/internal/store"
	"github.com/tomanagle/Social-Mockery/internal/store/dbstore"
	"github.com/tomanagle/Social-Mockery/internal/telemetry"
)

type App struct {
	server   *http.Server
	listener net.Listener
	logger   *slog.Logger
}

// Must constructs an application with required state and listening socket ready.
// Call it only from main; use New when initialization errors are recoverable.
func Must(cfg config.Config, seed store.Seed, database *sql.DB, observability *telemetry.Telemetry, logger *slog.Logger) *App {
	application, err := New(cfg, seed, database, observability, logger)
	if err != nil {
		panic("app: initialization failed")
	}
	return application
}

// New initializes required state using dependencies owned by the caller.
func New(cfg config.Config, seed store.Seed, database *sql.DB, observability *telemetry.Telemetry, logger *slog.Logger) (_ *App, err error) {
	_, span := observability.Start(context.Background(), "app.initialize")
	defer func() { telemetry.Finish(span, err, false) }()
	repo := dbstore.New(database)
	initialized, err := repo.Initialized(context.Background())
	if err != nil {
		return nil, err
	}
	engine := simulation.New(telemetry.ObserveStore(repo, observability), seed, cfg.MaxRecords)
	if !initialized || cfg.SeedPath != "" {
		if err := engine.Reset(context.Background(), seed); err != nil {
			return nil, err
		}
	}
	if accounts, posts, countErr := repo.Counts(context.Background()); countErr == nil {
		observability.State(context.Background(), accounts, posts)
	}
	server := &http.Server{Addr: cfg.Addr, Handler: observability.HTTP(middleware.Logging(logger, handlers.New(engine, cfg.AdminToken, observability))), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16384}
	listener, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return nil, err
	}
	return &App{server: server, listener: listener, logger: logger}, nil
}

// Close releases the listener without closing caller-owned dependencies.
func (a *App) Close() error {
	serverErr := a.server.Close()
	listenerErr := a.listener.Close()
	if errors.Is(listenerErr, net.ErrClosed) {
		listenerErr = nil
	}
	return errors.Join(serverErr, listenerErr)
}

// Run serves requests until shutdown; runtime failures are returned, never panicked.
func (a *App) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	stopped := make(chan error, 1)
	go func() { stopped <- a.server.Serve(a.listener) }()
	a.logger.Info("Social Mockery started", "addr", a.listener.Addr().String(), "clock", "virtual; use /admin/clock/advance")
	select {
	case err := <-stopped:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := a.server.Shutdown(shutdown); err != nil {
		a.server.Close()
		return err
	}
	return nil
}
