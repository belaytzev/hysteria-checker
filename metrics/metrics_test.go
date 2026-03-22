package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/belaytzev/hysteria-checker/checker"
	"github.com/belaytzev/hysteria-checker/models"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func getGaugeValue(gauge *prometheus.GaugeVec, labels prometheus.Labels) float64 {
	m := &dto.Metric{}
	g, err := gauge.GetMetricWith(labels)
	if err != nil {
		return -1
	}
	g.(prometheus.Metric).Write(m)
	return m.GetGauge().GetValue()
}

func TestUpdateMetrics_AliveProxy(t *testing.T) {
	proxies := []models.ProxyConfig{
		{Version: 2, Name: "test-proxy", Server: "1.2.3.4:443", StableID: "abc123"},
	}
	results := map[string]checker.CheckResult{
		"abc123": {Alive: true, Latency: 150 * time.Millisecond, LastCheck: time.Now()},
	}

	UpdateMetrics(results, proxies)

	labels := prometheus.Labels{"version": "hy2", "address": "1.2.3.4:443", "name": "test-proxy", "id": "abc123"}

	status := getGaugeValue(proxyStatus, labels)
	if status != 1 {
		t.Errorf("expected status=1 for alive proxy, got %v", status)
	}

	latency := getGaugeValue(proxyLatency, labels)
	if latency != 150 {
		t.Errorf("expected latency=150ms, got %v", latency)
	}
}

func TestUpdateMetrics_DownProxy(t *testing.T) {
	proxies := []models.ProxyConfig{
		{Version: 2, Name: "down-proxy", Server: "5.6.7.8:443", StableID: "def456"},
	}
	results := map[string]checker.CheckResult{
		"def456": {Alive: false, Latency: 0, LastCheck: time.Now(), Error: "connection refused"},
	}

	UpdateMetrics(results, proxies)

	labels := prometheus.Labels{"version": "hy2", "address": "5.6.7.8:443", "name": "down-proxy", "id": "def456"}

	status := getGaugeValue(proxyStatus, labels)
	if status != 0 {
		t.Errorf("expected status=0 for down proxy, got %v", status)
	}

	latency := getGaugeValue(proxyLatency, labels)
	if latency != 0 {
		t.Errorf("expected latency=0 for down proxy, got %v", latency)
	}
}

func TestUpdateMetrics_NoResult(t *testing.T) {
	proxies := []models.ProxyConfig{
		{Version: 1, Name: "unknown", Server: "9.9.9.9:443", StableID: "ghi789"},
	}
	results := map[string]checker.CheckResult{} // no results

	UpdateMetrics(results, proxies)

	labels := prometheus.Labels{"version": "hy1", "address": "9.9.9.9:443", "name": "unknown", "id": "ghi789"}

	status := getGaugeValue(proxyStatus, labels)
	if status != 0 {
		t.Errorf("expected status=0 for proxy with no result, got %v", status)
	}
}

func TestUpdateMetrics_MixedProxies(t *testing.T) {
	proxies := []models.ProxyConfig{
		{Version: 2, Name: "alive", Server: "1.1.1.1:443", StableID: "a1"},
		{Version: 1, Name: "dead", Server: "2.2.2.2:443", StableID: "a2"},
	}
	results := map[string]checker.CheckResult{
		"a1": {Alive: true, Latency: 50 * time.Millisecond},
		"a2": {Alive: false, Error: "timeout"},
	}

	UpdateMetrics(results, proxies)

	aliveLabels := prometheus.Labels{"version": "hy2", "address": "1.1.1.1:443", "name": "alive", "id": "a1"}
	deadLabels := prometheus.Labels{"version": "hy1", "address": "2.2.2.2:443", "name": "dead", "id": "a2"}

	if v := getGaugeValue(proxyStatus, aliveLabels); v != 1 {
		t.Errorf("expected alive proxy status=1, got %v", v)
	}
	if v := getGaugeValue(proxyLatency, aliveLabels); v != 50 {
		t.Errorf("expected alive proxy latency=50, got %v", v)
	}
	if v := getGaugeValue(proxyStatus, deadLabels); v != 0 {
		t.Errorf("expected dead proxy status=0, got %v", v)
	}
	if v := getGaugeValue(proxyLatency, deadLabels); v != 0 {
		t.Errorf("expected dead proxy latency=0, got %v", v)
	}
}

func TestHandler_Unprotected(t *testing.T) {
	h := Handler(false, "", "")

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "hysteria_proxy_status") {
		t.Error("expected metrics output to contain hysteria_proxy_status")
	}
}

func TestHandler_ProtectedNoAuth(t *testing.T) {
	h := Handler(true, "admin", "secret")

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without auth, got %d", w.Code)
	}
}

func TestHandler_ProtectedWrongAuth(t *testing.T) {
	h := Handler(true, "admin", "secret")

	req := httptest.NewRequest("GET", "/metrics", nil)
	req.SetBasicAuth("admin", "wrong")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 with wrong password, got %d", w.Code)
	}
}

func TestHandler_ProtectedCorrectAuth(t *testing.T) {
	h := Handler(true, "admin", "secret")

	req := httptest.NewRequest("GET", "/metrics", nil)
	req.SetBasicAuth("admin", "secret")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 with correct auth, got %d", w.Code)
	}
}
