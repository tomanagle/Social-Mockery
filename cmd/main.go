package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

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
	args := os.Args[1:]
	if len(args) == 0 {
		commandFailed(fmt.Errorf("missing command"), logger)
		return
	}
	switch args[0] {
	case "start":
		cfg, err := parseStart(args[1:])
		if err != nil {
			commandFailed(err, logger)
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
	case "reset":
		commandFailed(reset(args[1:]), logger)
	case "help", "--help", "-h":
		fmt.Println("Usage: social-mockery start [--addr 127.0.0.1:3000] [--db :memory:] [--seed profile.yaml] [--admin-token token] [--max-records 10000]\n       social-mockery reset [--url http://127.0.0.1:3000] [--seed profile.yaml] [--admin-token token]")
	default:
		commandFailed(fmt.Errorf("unknown command"), logger)
	}
}

func commandFailed(err error, logger *slog.Logger) {
	if err != nil && !errors.Is(err, flag.ErrHelp) {
		logger.Error("command failed", "category", "configuration_or_startup")
		os.Exit(1)
	}
}

func parseStart(args []string) (config.Config, error) {
	cfg := config.Default()
	flags := flag.NewFlagSet("start", flag.ContinueOnError)
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
