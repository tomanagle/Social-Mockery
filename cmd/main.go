package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/tomanagle/Social-Mockery/internal/config"
	"github.com/tomanagle/Social-Mockery/internal/handlers"
	"github.com/tomanagle/Social-Mockery/internal/middleware"
	"github.com/tomanagle/Social-Mockery/internal/profiles"
	"github.com/tomanagle/Social-Mockery/internal/simulation"
	"github.com/tomanagle/Social-Mockery/internal/store/db"
	"github.com/tomanagle/Social-Mockery/internal/store/dbstore"
	"github.com/tomanagle/Social-Mockery/internal/telemetry"
	"go.opentelemetry.io/otel"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	otel.SetErrorHandler(telemetry.ErrorHandler(logger))
	if err := run(os.Args[1:], logger); err != nil && !errors.Is(err, flag.ErrHelp) {
		logger.Error("command failed", "category", "configuration_or_startup")
		os.Exit(1)
	}
}
func run(args []string, logger *slog.Logger) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: social-mockery start [flags] | reset [flags]")
	}
	switch args[0] {
	case "start":
		return start(args[1:], logger)
	case "reset":
		return reset(args[1:])
	case "help", "--help", "-h":
		fmt.Println("Usage: social-mockery start [--addr 127.0.0.1:3000] [--db :memory:] [--seed profile.yaml] [--admin-token token] [--max-records 10000]\n       social-mockery reset [--url http://127.0.0.1:3000] [--seed profile.yaml] [--admin-token token]")
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}
func start(args []string, logger *slog.Logger) error {
	cfg := config.Default()
	flags := flag.NewFlagSet("start", flag.ContinueOnError)
	flags.StringVar(&cfg.Addr, "addr", cfg.Addr, "HTTP listening address")
	flags.StringVar(&cfg.Database, "db", cfg.Database, "SQLite file path or :memory:")
	flags.StringVar(&cfg.SeedPath, "seed", "", "YAML or JSON seed; replaces existing state when supplied")
	flags.StringVar(&cfg.AdminToken, "admin-token", os.Getenv("SOCIAL_MOCKERY_ADMIN_TOKEN"), "admin bearer token (prefer SOCIAL_MOCKERY_ADMIN_TOKEN environment variable)")
	flags.IntVar(&cfg.MaxRecords, "max-records", cfg.MaxRecords, "maximum combined containers and posts")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments")
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	observability, telemetryErr := telemetry.New(context.Background(), telemetry.ConfigFromEnv("dev"), logger)
	if telemetryErr != nil {
		logger.Warn("telemetry initialization failed; continuing without exporters")
		observability = telemetry.Noop()
	}
	logger = observability.Logger(logger)
	defer func() {
		if err := observability.Shutdown(context.Background()); err != nil {
			logger.Warn("telemetry shutdown incomplete")
		}
	}()
	_, profileSpan := observability.Start(context.Background(), "profile.load")
	seed := profiles.Default()
	var err error
	if cfg.SeedPath != "" {
		seed, err = profiles.Load(cfg.SeedPath)
		if err != nil {
			telemetry.Finish(profileSpan, err, true)
			return err
		}
	}
	telemetry.Finish(profileSpan, nil, false)
	_, databaseSpan := observability.Start(context.Background(), "db.initialize")
	database, err := db.Open(cfg.Database)
	telemetry.Finish(databaseSpan, err, false)
	if err != nil {
		return err
	}
	defer database.Close()
	repo := dbstore.New(database)
	initialized, err := repo.Initialized(context.Background())
	if err != nil {
		return err
	}
	engine := simulation.New(telemetry.ObserveStore(repo, observability), seed, cfg.MaxRecords)
	if accounts, posts, countErr := repo.Counts(context.Background()); countErr == nil {
		observability.State(context.Background(), accounts, posts)
	}
	if !initialized || cfg.SeedPath != "" {
		if err = engine.Reset(context.Background(), seed); err != nil {
			return err
		}
	}
	server := &http.Server{Addr: cfg.Addr, Handler: observability.HTTP(middleware.Logging(logger, handlers.New(engine, cfg.AdminToken, observability))), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16384}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	stopped := make(chan error, 1)
	go func() { stopped <- server.ListenAndServe() }()
	logger.Info("Social Mockery started", "addr", cfg.Addr, "database", cfg.Database, "clock", "virtual; use /admin/clock/advance")
	select {
	case err = <-stopped:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = server.Shutdown(shutdown); err != nil {
		server.Close()
		return err
	}
	return nil
}
func reset(args []string) error {
	flags := flag.NewFlagSet("reset", flag.ContinueOnError)
	endpoint := flags.String("url", "http://127.0.0.1:3000", "running simulator base URL")
	seedPath := flags.String("seed", "", "optional replacement seed profile")
	token := flags.String("admin-token", os.Getenv("SOCIAL_MOCKERY_ADMIN_TOKEN"), "admin token")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments")
	}
	var body []byte
	var err error
	if *seedPath != "" {
		if _, err = profiles.Load(*seedPath); err != nil {
			return err
		}
		body, err = os.ReadFile(*seedPath)
		if err != nil {
			return err
		}
	}
	req, err := http.NewRequest(http.MethodPost, *endpoint+"/admin/reset", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/yaml")
	if *token != "" {
		req.Header.Set("Authorization", "Bearer "+*token)
	}
	client := &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	result, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("reset failed (%d): %s", resp.StatusCode, result)
	}
	fmt.Print(string(result))
	return nil
}
