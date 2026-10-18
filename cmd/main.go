package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/tomanagle/Social-Mockery/internal/app"
	"github.com/tomanagle/Social-Mockery/internal/config"
	"github.com/tomanagle/Social-Mockery/internal/profiles"
	"github.com/tomanagle/Social-Mockery/internal/store/db"
	"github.com/tomanagle/Social-Mockery/internal/telemetry"
	"go.opentelemetry.io/otel"
)

func main() {
	exitCode := 0
	// Run cleanup before reporting a runtime failure to the shell.
	defer func() {
		if exitCode != 0 {
			os.Exit(exitCode)
		}
	}()
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	otel.SetErrorHandler(telemetry.ErrorHandler(logger))
	cfg, err := parseFlags(os.Args[1:])
	if err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			logger.Error("invalid startup flags", "category", "configuration_or_startup")
			exitCode = 1
		}
		return
	}
	cfg = config.Must(cfg)
	observability, err := telemetry.New(context.Background(), telemetry.ConfigFromEnv("dev"), logger)
	if err != nil {
		logger.Warn("telemetry initialization failed; continuing without exporters")
		observability = telemetry.Noop()
	}
	defer func() {
		if err := observability.Shutdown(context.Background()); err != nil {
			logger.Warn("telemetry shutdown incomplete")
		}
	}()
	logger = observability.Logger(logger)
	_, profileSpan := observability.Start(context.Background(), "profile.load")
	profileReady := false
	defer func() {
		if !profileReady {
			telemetry.Finish(profileSpan, errors.New("profile initialization failed"), false)
		}
	}()
	seed := profiles.Must(cfg.SeedPath)
	profileReady = true
	telemetry.Finish(profileSpan, nil, false)

	_, databaseSpan := observability.Start(context.Background(), "db.initialize")
	databaseReady := false
	defer func() {
		if !databaseReady {
			telemetry.Finish(databaseSpan, errors.New("database initialization failed"), false)
		}
	}()
	database := db.Must(cfg.Database)
	databaseReady = true
	telemetry.Finish(databaseSpan, nil, false)
	defer database.Close()
	application := app.Must(cfg, seed, database, observability, logger)
	defer application.Close()
	// Runtime failures are ordinary errors. Required startup work is already done.
	if err := application.Run(); err != nil {
		logger.Error("server stopped", "category", "runtime")
		exitCode = 1
	}
}

func parseFlags(args []string) (config.Config, error) {
	cfg := config.Default()
	flags := flag.NewFlagSet("social-mockery", flag.ContinueOnError)
	flags.StringVar(&cfg.Addr, "addr", cfg.Addr, "HTTP listening address")
	flags.StringVar(&cfg.Database, "db", cfg.Database, "SQLite file path or :memory:")
	flags.StringVar(&cfg.SeedPath, "seed", "", "YAML or JSON seed; replaces existing state when supplied")
	flags.StringVar(&cfg.AdminToken, "admin-token", os.Getenv("SOCIAL_MOCKERY_ADMIN_TOKEN"), "admin bearer token (prefer SOCIAL_MOCKERY_ADMIN_TOKEN environment variable)")
	flags.IntVar(&cfg.MaxRecords, "max-records", cfg.MaxRecords, "maximum combined containers and posts")
	if err := flags.Parse(args); err != nil {
		return cfg, err
	}
	if flags.NArg() != 0 {
		return cfg, fmt.Errorf("unexpected arguments")
	}
	return cfg, nil
}
