package web

import (
	"net/http"

	"github.com/belaytzev/hysteria-checker/checker"
)

// RouterConfig holds configuration for the HTTP router.
type RouterConfig struct {
	Checker        *checker.ProxyChecker
	MetricsHandler http.Handler
	Protected      bool
	Username       string
	Password       string
}

// NewRouter creates an http.ServeMux with all API, metrics, and future dashboard routes.
func NewRouter(cfg RouterConfig) *http.ServeMux {
	mux := http.NewServeMux()
	api := NewAPIHandler(cfg.Checker)

	// Metrics endpoint
	if cfg.MetricsHandler != nil {
		mux.Handle("/metrics", cfg.MetricsHandler)
	}

	// API endpoints with optional Basic Auth
	apiProxies := http.HandlerFunc(api.ListProxies)
	apiProxy := http.HandlerFunc(api.GetProxy)
	apiStatus := http.HandlerFunc(api.Status)

	if cfg.Protected {
		mux.Handle("/api/v1/proxies/", basicAuth(apiProxy, cfg.Username, cfg.Password))
		mux.Handle("/api/v1/proxies", basicAuth(apiProxies, cfg.Username, cfg.Password))
		mux.Handle("/api/v1/status", basicAuth(apiStatus, cfg.Username, cfg.Password))
	} else {
		mux.Handle("/api/v1/proxies/", apiProxy)
		mux.Handle("/api/v1/proxies", apiProxies)
		mux.Handle("/api/v1/status", apiStatus)
	}

	return mux
}

func basicAuth(next http.Handler, username, password string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != username || p != password {
			w.Header().Set("WWW-Authenticate", `Basic realm="api"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
