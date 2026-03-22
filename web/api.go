package web

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/belaytzev/hysteria-checker/checker"
	"github.com/belaytzev/hysteria-checker/models"
)

// ProxyResponse is the JSON representation of a proxy and its check result.
type ProxyResponse struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Server    string  `json:"server"`
	Version   string  `json:"version"`
	Alive     bool    `json:"alive"`
	LatencyMs float64 `json:"latency_ms"`
	LastCheck string  `json:"last_check,omitempty"`
	Error     string  `json:"error,omitempty"`
}

// StatusResponse is the JSON representation of the overall status summary.
type StatusResponse struct {
	Total int `json:"total"`
	Up    int `json:"up"`
	Down  int `json:"down"`
}

// APIHandler serves the REST API endpoints.
type APIHandler struct {
	checker *checker.ProxyChecker
	redact  bool
}

// NewAPIHandler creates a new APIHandler.
func NewAPIHandler(pc *checker.ProxyChecker, redact bool) *APIHandler {
	return &APIHandler{checker: pc, redact: redact}
}

// ListProxies handles GET /api/v1/proxies.
func (h *APIHandler) ListProxies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method Not Allowed"})
		return
	}
	proxies := h.checker.Proxies()
	results := h.checker.Results()

	resp := make([]ProxyResponse, 0, len(proxies))
	for _, p := range proxies {
		resp = append(resp, buildProxyResponse(p, results[p.StableID], h.redact))
	}

	writeJSON(w, http.StatusOK, resp)
}

// GetProxy handles GET /api/v1/proxies/{id}.
func (h *APIHandler) GetProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method Not Allowed"})
		return
	}
	id := extractProxyID(r.URL.Path)
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing proxy id"})
		return
	}

	proxies := h.checker.Proxies()
	results := h.checker.Results()

	for _, p := range proxies {
		if p.StableID == id {
			writeJSON(w, http.StatusOK, buildProxyResponse(p, results[p.StableID], h.redact))
			return
		}
	}

	writeJSON(w, http.StatusNotFound, map[string]string{"error": "proxy not found"})
}

// Status handles GET /api/v1/status.
func (h *APIHandler) Status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method Not Allowed"})
		return
	}
	proxies := h.checker.Proxies()
	results := h.checker.Results()

	up := 0
	for _, p := range proxies {
		if result, ok := results[p.StableID]; ok && result.Alive {
			up++
		}
	}

	writeJSON(w, http.StatusOK, StatusResponse{
		Total: len(proxies),
		Up:    up,
		Down:  len(proxies) - up,
	})
}

func buildProxyResponse(p models.ProxyConfig, result checker.CheckResult, redact bool) ProxyResponse {
	resp := ProxyResponse{
		ID:      p.StableID,
		Name:    p.Name,
		Server:  p.Server,
		Version: fmt.Sprintf("hy%d", p.Version),
		Alive:   result.Alive,
	}
	if redact {
		resp.Server = "***"
	}
	if result.Alive {
		resp.LatencyMs = float64(result.Latency.Milliseconds())
	}
	if !result.LastCheck.IsZero() {
		resp.LastCheck = result.LastCheck.Format("2006-01-02T15:04:05Z07:00")
	}
	if result.Error != "" {
		if redact {
			resp.Error = "check failed"
		} else {
			resp.Error = result.Error
		}
	}
	return resp
}

// extractProxyID extracts the proxy ID from a path like /api/v1/proxies/{id}.
func extractProxyID(path string) string {
	const prefix = "/api/v1/proxies/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	id := strings.TrimPrefix(path, prefix)
	// Remove trailing slash if present
	id = strings.TrimSuffix(id, "/")
	return id
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}
