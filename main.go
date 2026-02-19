package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/belaytzev/hysteria-checker/checker"
	"github.com/belaytzev/hysteria-checker/config"
	"github.com/belaytzev/hysteria-checker/metrics"
	"github.com/belaytzev/hysteria-checker/subscription"
	"github.com/belaytzev/hysteria-checker/web"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	cfg, err := config.Parse(args)
	if err != nil {
		return err
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

	// Fetch subscriptions and parse proxy configs
	proxies, err := subscription.FetchAll(cfg.SubscriptionURL, cfg.CheckTimeout)
	if err != nil {
		slog.Warn("failed to fetch subscriptions", "error", err)
	}
	slog.Info("loaded proxies", "count", len(proxies))

	// Initialize proxy checker
	var opts []checker.ProxyCheckerOption
	checkMethod := cfg.CheckMethod
	if cfg.CheckMethod == "ip" {
		hostIP, err := checker.DetectHostIP(cfg.CheckURL, cfg.CheckTimeout)
		if err != nil {
			slog.Warn("failed to detect host IP, falling back to status-only checks", "error", err)
			checkMethod = "status"
		} else {
			slog.Info("detected host IP", "ip", hostIP)
			opts = append(opts, checker.WithHostIP(hostIP))
		}
	}
	opts = append(opts, checker.WithCheckMethod(checkMethod))
	pc := checker.NewProxyChecker(
		proxies,
		&checker.Hysteria1Connector{},
		&checker.Hysteria2Connector{},
		cfg.CheckURL,
		cfg.CheckTimeout,
		opts...,
	)

	// Run initial check
	slog.Info("running initial proxy check")
	pc.CheckAll()
	metrics.UpdateMetrics(pc.Results(), pc.Proxies())
	slog.Info("initial check complete", "proxies", len(proxies))

	// Set up HTTP server
	metricsHandler := metrics.Handler(cfg.MetricsProtected, cfg.MetricsUsername, cfg.MetricsPassword)
	router := web.NewRouter(web.RouterConfig{
		Checker:         pc,
		MetricsHandler:  metricsHandler,
		Protected:       cfg.MetricsProtected,
		Username:        cfg.MetricsUsername,
		Password:        cfg.MetricsPassword,
		WebPublic:       cfg.WebPublic,
		RefreshInterval: int(cfg.CheckInterval.Seconds()),
	})

	addr := fmt.Sprintf("%s:%d", cfg.MetricsHost, cfg.MetricsPort)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Set up signal handling for graceful shutdown
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// Start HTTP server
	go func() {
		slog.Info("HTTP server listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", "error", err)
		}
	}()

	// Single ticker for both subscription refresh and health checks
	ticker := time.NewTicker(cfg.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("shutting down")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if err := srv.Shutdown(shutdownCtx); err != nil {
				slog.Error("HTTP server shutdown error", "error", err)
			}
			cancel()
			return nil

		case <-ticker.C:
			// Refresh subscriptions first, then run checks
			slog.Debug("refreshing subscriptions")
			newProxies, err := subscription.FetchAll(cfg.SubscriptionURL, cfg.CheckTimeout)
			if err != nil {
				slog.Warn("subscription refresh failed", "error", err)
			}
			if err == nil && len(newProxies) > 0 {
				pc.UpdateProxies(newProxies)
				slog.Info("subscriptions refreshed", "proxies", len(newProxies))
			} else if err == nil && len(newProxies) == 0 {
				slog.Warn("subscription refresh returned empty list, keeping existing proxies")
			}

			slog.Debug("running scheduled proxy check")
			pc.CheckAll()
			metrics.UpdateMetrics(pc.Results(), pc.Proxies())
			slog.Debug("scheduled check complete")
		}
	}
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
