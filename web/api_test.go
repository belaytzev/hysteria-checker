package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/belaytzev/hysteria-checker/checker"
	"github.com/belaytzev/hysteria-checker/models"
)

// newTestChecker creates a ProxyChecker with pre-set proxies and results for testing.
func newTestChecker(proxies []models.ProxyConfig, results map[string]checker.CheckResult) *checker.ProxyChecker {
	pc := checker.NewProxyChecker(proxies, nil, nil, "https://example.com", 10*time.Second)
	// Inject results by running through the exported API
	// We need to set results via the checker's internal state.
	// Since ProxyChecker doesn't expose a SetResults method, we use SetTestResults.
	pc.SetTestResults(results)
	return pc
}

func TestListProxies(t *testing.T) {
	proxies := []models.ProxyConfig{
		{Version: 2, Name: "proxy1", Server: "host1:443", StableID: "id1"},
		{Version: 1, Name: "proxy2", Server: "host2:443", StableID: "id2"},
	}
	results := map[string]checker.CheckResult{
		"id1": {Alive: true, Latency: 50 * time.Millisecond, LastCheck: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		"id2": {Alive: false, Error: "connect failed", LastCheck: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
	}

	pc := newTestChecker(proxies, results)
	api := NewAPIHandler(pc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/proxies", nil)
	w := httptest.NewRecorder()
	api.ListProxies(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %s", ct)
	}

	var resp []ProxyResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp) != 2 {
		t.Fatalf("expected 2 proxies, got %d", len(resp))
	}

	// Check first proxy (alive)
	if resp[0].ID != "id1" || resp[0].Version != "hy2" || !resp[0].Alive || resp[0].LatencyMs != 50 {
		t.Errorf("unexpected first proxy: %+v", resp[0])
	}
	// Check second proxy (down)
	if resp[1].ID != "id2" || resp[1].Version != "hy1" || resp[1].Alive || resp[1].Error != "connect failed" {
		t.Errorf("unexpected second proxy: %+v", resp[1])
	}
}

func TestGetProxy(t *testing.T) {
	proxies := []models.ProxyConfig{
		{Version: 2, Name: "proxy1", Server: "host1:443", StableID: "abc123"},
	}
	results := map[string]checker.CheckResult{
		"abc123": {Alive: true, Latency: 100 * time.Millisecond, LastCheck: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
	}

	pc := newTestChecker(proxies, results)
	api := NewAPIHandler(pc)

	t.Run("found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/proxies/abc123", nil)
		w := httptest.NewRecorder()
		api.GetProxy(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		var resp ProxyResponse
		json.NewDecoder(w.Body).Decode(&resp)
		if resp.ID != "abc123" || !resp.Alive {
			t.Errorf("unexpected response: %+v", resp)
		}
	})

	t.Run("not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/proxies/nonexistent", nil)
		w := httptest.NewRecorder()
		api.GetProxy(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("missing id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/proxies/", nil)
		w := httptest.NewRecorder()
		api.GetProxy(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})
}

func TestStatus(t *testing.T) {
	proxies := []models.ProxyConfig{
		{Version: 2, Name: "p1", Server: "h1:443", StableID: "id1"},
		{Version: 2, Name: "p2", Server: "h2:443", StableID: "id2"},
		{Version: 1, Name: "p3", Server: "h3:443", StableID: "id3"},
	}
	results := map[string]checker.CheckResult{
		"id1": {Alive: true, Latency: 50 * time.Millisecond},
		"id2": {Alive: false, Error: "timeout"},
		// id3 has no result yet
	}

	pc := newTestChecker(proxies, results)
	api := NewAPIHandler(pc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	w := httptest.NewRecorder()
	api.Status(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp StatusResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Total != 3 {
		t.Errorf("expected total 3, got %d", resp.Total)
	}
	if resp.Up != 1 {
		t.Errorf("expected up 1, got %d", resp.Up)
	}
	if resp.Down != 2 {
		t.Errorf("expected down 2, got %d", resp.Down)
	}
}

func TestRouterBasicAuth(t *testing.T) {
	proxies := []models.ProxyConfig{
		{Version: 2, Name: "p1", Server: "h1:443", StableID: "id1"},
	}
	pc := newTestChecker(proxies, nil)

	mux := NewRouter(RouterConfig{
		Checker:   pc,
		Protected: true,
		Username:  "admin",
		Password:  "secret",
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	t.Run("unauthorized", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/status")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("wrong credentials", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/status", nil)
		req.SetBasicAuth("admin", "wrong")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("authorized", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/status", nil)
		req.SetBasicAuth("admin", "secret")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
	})
}

func TestRouterNoAuth(t *testing.T) {
	proxies := []models.ProxyConfig{
		{Version: 2, Name: "p1", Server: "h1:443", StableID: "id1"},
	}
	results := map[string]checker.CheckResult{
		"id1": {Alive: true, Latency: 25 * time.Millisecond},
	}
	pc := newTestChecker(proxies, results)

	mux := NewRouter(RouterConfig{
		Checker:   pc,
		Protected: false,
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	// All endpoints should be accessible without auth
	for _, path := range []string{"/api/v1/proxies", "/api/v1/proxies/id1", "/api/v1/status"} {
		resp, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET %s: expected 200, got %d", path, resp.StatusCode)
		}
	}
}

func TestEmptyProxies(t *testing.T) {
	pc := newTestChecker(nil, nil)
	api := NewAPIHandler(pc)

	t.Run("list empty", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/proxies", nil)
		w := httptest.NewRecorder()
		api.ListProxies(w, req)

		var resp []ProxyResponse
		json.NewDecoder(w.Body).Decode(&resp)
		if len(resp) != 0 {
			t.Errorf("expected empty list, got %d items", len(resp))
		}
	})

	t.Run("status empty", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
		w := httptest.NewRecorder()
		api.Status(w, req)

		var resp StatusResponse
		json.NewDecoder(w.Body).Decode(&resp)
		if resp.Total != 0 || resp.Up != 0 || resp.Down != 0 {
			t.Errorf("expected all zeros, got %+v", resp)
		}
	})
}
