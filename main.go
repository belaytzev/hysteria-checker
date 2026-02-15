package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/belaytzev/hysteria-checker/config"
)

func main() {
	cfg, err := config.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	level := parseLogLevel(cfg.LogLevel)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	slog.Info("hysteria-checker starting",
		"check_interval", cfg.CheckInterval,
		"check_method", cfg.CheckMethod,
		"check_url", cfg.CheckURL,
		"metrics_host", cfg.MetricsHost,
		"metrics_port", cfg.MetricsPort,
		"subscriptions", len(cfg.SubscriptionURL),
	)
}

func parseLogLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
