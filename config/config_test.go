package config

import (
	"testing"
	"time"
)

func TestParseDefaults(t *testing.T) {
	cfg, err := Parse([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.CheckInterval != 300*time.Second {
		t.Errorf("CheckInterval = %v, want 300s", cfg.CheckInterval)
	}
	if cfg.CheckMethod != "ip" {
		t.Errorf("CheckMethod = %q, want %q", cfg.CheckMethod, "ip")
	}
	if cfg.CheckURL != "https://api.ipify.org" {
		t.Errorf("CheckURL = %q, want %q", cfg.CheckURL, "https://api.ipify.org")
	}
	if cfg.CheckTimeout != 30*time.Second {
		t.Errorf("CheckTimeout = %v, want 30s", cfg.CheckTimeout)
	}
	if cfg.MetricsHost != "0.0.0.0" {
		t.Errorf("MetricsHost = %q, want %q", cfg.MetricsHost, "0.0.0.0")
	}
	if cfg.MetricsPort != 2112 {
		t.Errorf("MetricsPort = %d, want 2112", cfg.MetricsPort)
	}
	if cfg.MetricsProtected {
		t.Error("MetricsProtected should default to false")
	}
	if cfg.WebPublic {
		t.Error("WebPublic should default to false")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
	if len(cfg.SubscriptionURL) != 0 {
		t.Errorf("SubscriptionURL = %v, want empty", cfg.SubscriptionURL)
	}
}

func TestParseCLIFlags(t *testing.T) {
	args := []string{
		"--subscription-url", "hysteria2://auth@example.com:443",
		"--check-interval", "60s",
		"--check-method", "status",
		"--check-url", "https://example.com",
		"--check-timeout", "10s",
		"--metrics-host", "127.0.0.1",
		"--metrics-port", "9090",
		"--metrics-protected",
		"--metrics-username", "admin",
		"--metrics-password", "secret",
		"--web-public",
		"--log-level", "debug",
	}

	cfg, err := Parse(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.SubscriptionURL) != 1 || cfg.SubscriptionURL[0] != "hysteria2://auth@example.com:443" {
		t.Errorf("SubscriptionURL = %v, want [hysteria2://auth@example.com:443]", cfg.SubscriptionURL)
	}
	if cfg.CheckInterval != 60*time.Second {
		t.Errorf("CheckInterval = %v, want 60s", cfg.CheckInterval)
	}
	if cfg.CheckMethod != "status" {
		t.Errorf("CheckMethod = %q, want %q", cfg.CheckMethod, "status")
	}
	if cfg.CheckURL != "https://example.com" {
		t.Errorf("CheckURL = %q, want %q", cfg.CheckURL, "https://example.com")
	}
	if cfg.CheckTimeout != 10*time.Second {
		t.Errorf("CheckTimeout = %v, want 10s", cfg.CheckTimeout)
	}
	if cfg.MetricsHost != "127.0.0.1" {
		t.Errorf("MetricsHost = %q, want %q", cfg.MetricsHost, "127.0.0.1")
	}
	if cfg.MetricsPort != 9090 {
		t.Errorf("MetricsPort = %d, want 9090", cfg.MetricsPort)
	}
	if !cfg.MetricsProtected {
		t.Error("MetricsProtected should be true")
	}
	if cfg.MetricsUsername != "admin" {
		t.Errorf("MetricsUsername = %q, want %q", cfg.MetricsUsername, "admin")
	}
	if cfg.MetricsPassword != "secret" {
		t.Errorf("MetricsPassword = %q, want %q", cfg.MetricsPassword, "secret")
	}
	if !cfg.WebPublic {
		t.Error("WebPublic should be true")
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}
}

func TestParseEnvVars(t *testing.T) {
	t.Setenv("SUBSCRIPTION_URL", "https://example.com/sub")
	t.Setenv("CHECK_INTERVAL", "120s")
	t.Setenv("CHECK_METHOD", "status")
	t.Setenv("LOG_LEVEL", "warn")

	cfg, err := Parse([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.SubscriptionURL) != 1 || cfg.SubscriptionURL[0] != "https://example.com/sub" {
		t.Errorf("SubscriptionURL = %v, want [https://example.com/sub]", cfg.SubscriptionURL)
	}
	if cfg.CheckInterval != 120*time.Second {
		t.Errorf("CheckInterval = %v, want 120s", cfg.CheckInterval)
	}
	if cfg.CheckMethod != "status" {
		t.Errorf("CheckMethod = %q, want %q", cfg.CheckMethod, "status")
	}
	if cfg.LogLevel != "warn" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "warn")
	}
}

func TestParseMultipleSubscriptionURLs(t *testing.T) {
	args := []string{
		"--subscription-url", "https://sub1.example.com",
		"--subscription-url", "hysteria2://auth@host:443",
	}

	cfg, err := Parse(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.SubscriptionURL) != 2 {
		t.Fatalf("SubscriptionURL count = %d, want 2", len(cfg.SubscriptionURL))
	}
	if cfg.SubscriptionURL[0] != "https://sub1.example.com" {
		t.Errorf("SubscriptionURL[0] = %q, want %q", cfg.SubscriptionURL[0], "https://sub1.example.com")
	}
	if cfg.SubscriptionURL[1] != "hysteria2://auth@host:443" {
		t.Errorf("SubscriptionURL[1] = %q, want %q", cfg.SubscriptionURL[1], "hysteria2://auth@host:443")
	}
}

func TestParseInvalidCheckMethod(t *testing.T) {
	args := []string{"--check-method", "invalid"}
	_, err := Parse(args)
	if err == nil {
		t.Error("expected error for invalid check method, got nil")
	}
}

func TestParseInvalidLogLevel(t *testing.T) {
	args := []string{"--log-level", "invalid"}
	_, err := Parse(args)
	if err == nil {
		t.Error("expected error for invalid log level, got nil")
	}
}
