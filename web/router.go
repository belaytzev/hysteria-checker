package web

import (
	"net/http"

	"github.com/belaytzev/hysteria-checker/checker"
	"github.com/belaytzev/hysteria-checker/middleware"
)

// RouterConfig holds configuration for the HTTP router.
type RouterConfig struct {
	Checker         *checker.ProxyChecker
	MetricsHandler  http.Handler
	Protected       bool
	Username        string
	Password        string
	RedactSensitive bool
	RefreshInterval int // dashboard auto-refresh in seconds
}

// NewRouter creates an http.ServeMux with all API, metrics, and dashboard routes.
func NewRouter(cfg RouterConfig) *http.ServeMux {
	mux := http.NewServeMux()
	api := NewAPIHandler(cfg.Checker, cfg.RedactSensitive)

	// Dashboard
	refreshSec := cfg.RefreshInterval
	if refreshSec <= 0 {
		refreshSec = 300
	}
	dashboard := NewDashboardHandler(cfg.Checker, refreshSec, cfg.RedactSensitive)
	if cfg.Protected {
		mux.Handle("/", middleware.BasicAuth(dashboard, cfg.Username, cfg.Password))
	} else {
		mux.Handle("/", dashboard)
	}

	// Health endpoint (always unauthenticated)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Metrics endpoint
	if cfg.MetricsHandler != nil {
		mux.Handle("/metrics", cfg.MetricsHandler)
	}

	// API endpoints with optional Basic Auth
	apiProxies := http.HandlerFunc(api.ListProxies)
	apiProxy := http.HandlerFunc(api.GetProxy)
	apiStatus := http.HandlerFunc(api.Status)

	if cfg.Protected {
		mux.Handle("/api/v1/proxies/", middleware.BasicAuth(apiProxy, cfg.Username, cfg.Password))
		mux.Handle("/api/v1/proxies", middleware.BasicAuth(apiProxies, cfg.Username, cfg.Password))
		mux.Handle("/api/v1/status", middleware.BasicAuth(apiStatus, cfg.Username, cfg.Password))
	} else {
		mux.Handle("/api/v1/proxies/", apiProxy)
		mux.Handle("/api/v1/proxies", apiProxies)
		mux.Handle("/api/v1/status", apiStatus)
	}

	return mux
}
