package web

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/belaytzev/hysteria-checker/checker"
)

//go:embed templates/index.html
var templateFS embed.FS

var dashboardTmpl = template.Must(template.ParseFS(templateFS, "templates/index.html"))

// DashboardData holds the data passed to the dashboard template.
type DashboardData struct {
	Proxies         []DashboardProxy
	Total           int
	Up              int
	Down            int
	RefreshInterval int
	Public          bool
	LastUpdated     string
}

// DashboardProxy represents a single proxy row in the dashboard.
type DashboardProxy struct {
	Name      string
	Version   string
	Server    string
	Alive     bool
	LatencyMs float64
}

// DashboardHandler serves the HTML dashboard.
type DashboardHandler struct {
	checker         *checker.ProxyChecker
	refreshInterval int
	public          bool
}

// NewDashboardHandler creates a new DashboardHandler.
func NewDashboardHandler(pc *checker.ProxyChecker, refreshIntervalSec int, public bool) *DashboardHandler {
	return &DashboardHandler{
		checker:         pc,
		refreshInterval: refreshIntervalSec,
		public:          public,
	}
}

// ServeHTTP renders the dashboard HTML page.
func (h *DashboardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	proxies := h.checker.Proxies()
	results := h.checker.Results()

	data := DashboardData{
		Proxies:         make([]DashboardProxy, 0, len(proxies)),
		RefreshInterval: h.refreshInterval,
		Public:          h.public,
	}

	var latest time.Time
	for _, p := range proxies {
		result := results[p.StableID]
		server := p.Server
		if h.public {
			server = "***"
		}
		dp := DashboardProxy{
			Name:    p.Name,
			Version: fmt.Sprintf("hy%d", p.Version),
			Server:  server,
			Alive:   result.Alive,
		}
		if result.Alive {
			dp.LatencyMs = float64(result.Latency.Milliseconds())
		}
		if result.LastCheck.After(latest) {
			latest = result.LastCheck
		}
		data.Proxies = append(data.Proxies, dp)
		if result.Alive {
			data.Up++
		}
	}
	data.Total = len(proxies)
	data.Down = data.Total - data.Up
	if !latest.IsZero() {
		data.LastUpdated = latest.Format("15:04:05")
	}

	var buf bytes.Buffer
	if err := dashboardTmpl.Execute(&buf, data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = buf.WriteTo(w)
}
