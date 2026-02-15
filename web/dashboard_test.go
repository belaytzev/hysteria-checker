package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/belaytzev/hysteria-checker/checker"
	"github.com/belaytzev/hysteria-checker/models"
)

func TestDashboardHandler_StatusAndContentType(t *testing.T) {
	proxies := []models.ProxyConfig{
		{Version: 2, Name: "proxy1", Server: "host1:443", StableID: "id1"},
	}
	results := map[string]checker.CheckResult{
		"id1": {Alive: true, Latency: 42 * time.Millisecond, LastCheck: time.Now()},
	}
	pc := newTestChecker(proxies, results)
	handler := NewDashboardHandler(pc, 60, false)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("expected text/html content type, got %s", ct)
	}
}

func TestDashboardHandler_ContainsProxyData(t *testing.T) {
	proxies := []models.ProxyConfig{
		{Version: 2, Name: "test-proxy-alpha", Server: "alpha.example.com:443", StableID: "id1"},
		{Version: 1, Name: "test-proxy-beta", Server: "beta.example.com:443", StableID: "id2"},
	}
	results := map[string]checker.CheckResult{
		"id1": {Alive: true, Latency: 55 * time.Millisecond, LastCheck: time.Now()},
		"id2": {Alive: false, Error: "timeout", LastCheck: time.Now()},
	}
	pc := newTestChecker(proxies, results)
	handler := NewDashboardHandler(pc, 120, false)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body := w.Body.String()

	// Check proxy names appear
	if !strings.Contains(body, "test-proxy-alpha") {
		t.Error("expected body to contain proxy name 'test-proxy-alpha'")
	}
	if !strings.Contains(body, "test-proxy-beta") {
		t.Error("expected body to contain proxy name 'test-proxy-beta'")
	}

	// Check server addresses appear (not public mode)
	if !strings.Contains(body, "alpha.example.com:443") {
		t.Error("expected body to contain server address 'alpha.example.com:443'")
	}

	// Check version badges
	if !strings.Contains(body, "hy2") {
		t.Error("expected body to contain 'hy2'")
	}
	if !strings.Contains(body, "hy1") {
		t.Error("expected body to contain 'hy1'")
	}

	// Check status badges
	if !strings.Contains(body, "UP") {
		t.Error("expected body to contain 'UP' badge")
	}
	if !strings.Contains(body, "DOWN") {
		t.Error("expected body to contain 'DOWN' badge")
	}

	// Check latency
	if !strings.Contains(body, "55 ms") {
		t.Error("expected body to contain '55 ms'")
	}

	// Check summary counts
	if !strings.Contains(body, ">2<") {
		t.Error("expected body to contain total count of 2")
	}
}

func TestDashboardHandler_PublicHidesServer(t *testing.T) {
	proxies := []models.ProxyConfig{
		{Version: 2, Name: "secret-proxy", Server: "secret.example.com:443", StableID: "id1"},
	}
	results := map[string]checker.CheckResult{
		"id1": {Alive: true, Latency: 30 * time.Millisecond, LastCheck: time.Now()},
	}
	pc := newTestChecker(proxies, results)
	handler := NewDashboardHandler(pc, 60, true)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body := w.Body.String()

	// Server address should be hidden
	if strings.Contains(body, "secret.example.com") {
		t.Error("expected server address to be hidden in public mode")
	}
	// Should show masked value
	if !strings.Contains(body, "***") {
		t.Error("expected '***' placeholder in public mode")
	}
	// Proxy name should still be visible
	if !strings.Contains(body, "secret-proxy") {
		t.Error("expected proxy name to be visible in public mode")
	}
}

func TestDashboardHandler_EmptyProxies(t *testing.T) {
	pc := newTestChecker(nil, nil)
	handler := NewDashboardHandler(pc, 60, false)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "No proxies configured") {
		t.Error("expected 'No proxies configured' message for empty proxy list")
	}
}

func TestDashboardHandler_RefreshInterval(t *testing.T) {
	pc := newTestChecker(nil, nil)
	handler := NewDashboardHandler(pc, 45, false)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "Auto-refresh: 45s") {
		t.Error("expected refresh interval of 45s in dashboard")
	}
}

func TestDashboardViaRouter(t *testing.T) {
	proxies := []models.ProxyConfig{
		{Version: 2, Name: "p1", Server: "h1:443", StableID: "id1"},
	}
	results := map[string]checker.CheckResult{
		"id1": {Alive: true, Latency: 10 * time.Millisecond},
	}
	pc := newTestChecker(proxies, results)

	mux := NewRouter(RouterConfig{
		Checker:         pc,
		RefreshInterval: 30,
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/html") {
		t.Errorf("expected text/html, got %s", ct)
	}
}
