package checker

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/belaytzev/hysteria-checker/models"
)

func makeProxy(version int, name, server, auth string) models.ProxyConfig {
	p := models.ProxyConfig{
		Version: version,
		Name:    name,
		Server:  server,
		Auth:    auth,
	}
	p.GenerateStableID()
	return p
}

func TestNewProxyChecker_Defaults(t *testing.T) {
	pc := NewProxyChecker(nil, nil, nil, "http://example.com", 10*time.Second)
	if pc.concurrency != 5 {
		t.Errorf("expected default concurrency=5, got %d", pc.concurrency)
	}
	if pc.checkURL != "http://example.com" {
		t.Errorf("unexpected checkURL: %s", pc.checkURL)
	}
}

func TestNewProxyChecker_WithOptions(t *testing.T) {
	pc := NewProxyChecker(nil, nil, nil, "http://example.com", 10*time.Second,
		WithConcurrency(10),
		WithHostIP("1.2.3.4"),
	)
	if pc.concurrency != 10 {
		t.Errorf("expected concurrency=10, got %d", pc.concurrency)
	}
	if pc.hostIP != "1.2.3.4" {
		t.Errorf("expected hostIP=1.2.3.4, got %s", pc.hostIP)
	}
}

func TestWithConcurrency_IgnoresZero(t *testing.T) {
	pc := NewProxyChecker(nil, nil, nil, "http://example.com", 10*time.Second,
		WithConcurrency(0),
	)
	if pc.concurrency != 5 {
		t.Errorf("expected concurrency to remain 5 for zero input, got %d", pc.concurrency)
	}
}

func TestCheckAll_DispatchesV1AndV2(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	}))
	defer ts.Close()

	var v1Calls, v2Calls atomic.Int32

	v1Conn := &mockConnector{
		connectFunc: func(cfg models.ProxyConfig) (ProxyClient, error) {
			v1Calls.Add(1)
			return &mockProxyClient{
				tcpFunc: func(addr string) (net.Conn, error) {
					return net.DialTimeout("tcp", ts.Listener.Addr().String(), 5*time.Second)
				},
			}, nil
		},
	}
	v2Conn := &mockConnector{
		connectFunc: func(cfg models.ProxyConfig) (ProxyClient, error) {
			v2Calls.Add(1)
			return &mockProxyClient{
				tcpFunc: func(addr string) (net.Conn, error) {
					return net.DialTimeout("tcp", ts.Listener.Addr().String(), 5*time.Second)
				},
			}, nil
		},
	}

	proxies := []models.ProxyConfig{
		makeProxy(1, "v1-proxy", "host1:443", "auth1"),
		makeProxy(2, "v2-proxy", "host2:443", "auth2"),
		makeProxy(2, "v2-proxy-2", "host3:443", "auth3"),
	}

	pc := NewProxyChecker(proxies, v1Conn, v2Conn, ts.URL, 10*time.Second)
	pc.CheckAll()

	if v1Calls.Load() != 1 {
		t.Errorf("expected 1 v1 connect call, got %d", v1Calls.Load())
	}
	if v2Calls.Load() != 2 {
		t.Errorf("expected 2 v2 connect calls, got %d", v2Calls.Load())
	}

	results := pc.Results()
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	for _, p := range proxies {
		r, ok := results[p.StableID]
		if !ok {
			t.Errorf("missing result for %s", p.Name)
			continue
		}
		if !r.Alive {
			t.Errorf("expected %s to be alive, error: %s", p.Name, r.Error)
		}
		if r.Latency <= 0 {
			t.Errorf("expected positive latency for %s", p.Name)
		}
	}
}

func TestCheckAll_ConnectFailure(t *testing.T) {
	failConnector := &mockConnector{
		connectFunc: func(cfg models.ProxyConfig) (ProxyClient, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}

	proxies := []models.ProxyConfig{
		makeProxy(2, "fail-proxy", "host:443", "auth"),
	}

	pc := NewProxyChecker(proxies, nil, failConnector, "http://example.com", 10*time.Second)
	pc.CheckAll()

	results := pc.Results()
	r := results[proxies[0].StableID]
	if r.Alive {
		t.Error("expected alive=false for failed connection")
	}
	if r.Error == "" {
		t.Error("expected non-empty error")
	}
}

func TestCheckAll_UnsupportedVersion(t *testing.T) {
	proxies := []models.ProxyConfig{
		makeProxy(99, "bad-version", "host:443", "auth"),
	}

	pc := NewProxyChecker(proxies, nil, nil, "http://example.com", 10*time.Second)
	pc.CheckAll()

	results := pc.Results()
	r := results[proxies[0].StableID]
	if r.Alive {
		t.Error("expected alive=false for unsupported version")
	}
	if r.Error == "" {
		t.Error("expected error for unsupported version")
	}
}

func TestCheckAll_NilConnector(t *testing.T) {
	proxies := []models.ProxyConfig{
		makeProxy(1, "v1-no-connector", "host:443", "auth"),
	}

	pc := NewProxyChecker(proxies, nil, nil, "http://example.com", 10*time.Second)
	pc.CheckAll()

	results := pc.Results()
	r := results[proxies[0].StableID]
	if r.Alive {
		t.Error("expected alive=false when connector is nil")
	}
	if r.Error == "" {
		t.Error("expected error when connector is nil")
	}
}

func TestCheckAll_ConcurrencyLimit(t *testing.T) {
	var concurrent, maxConcurrent atomic.Int32

	slowConnector := &mockConnector{
		connectFunc: func(cfg models.ProxyConfig) (ProxyClient, error) {
			cur := concurrent.Add(1)
			for {
				old := maxConcurrent.Load()
				if cur <= old {
					break
				}
				if maxConcurrent.CompareAndSwap(old, cur) {
					break
				}
			}
			time.Sleep(50 * time.Millisecond)
			concurrent.Add(-1)
			return nil, fmt.Errorf("test error")
		},
	}

	proxies := make([]models.ProxyConfig, 10)
	for i := range proxies {
		proxies[i] = makeProxy(2, fmt.Sprintf("proxy-%d", i), fmt.Sprintf("host%d:443", i), fmt.Sprintf("auth%d", i))
	}

	pc := NewProxyChecker(proxies, nil, slowConnector, "http://example.com", 10*time.Second,
		WithConcurrency(3),
	)
	pc.CheckAll()

	if maxConcurrent.Load() > 3 {
		t.Errorf("expected max concurrency <= 3, got %d", maxConcurrent.Load())
	}
}

func TestUpdateProxies_RemovesStaleResults(t *testing.T) {
	proxies := []models.ProxyConfig{
		makeProxy(2, "proxy-a", "a:443", "auth-a"),
		makeProxy(2, "proxy-b", "b:443", "auth-b"),
	}

	failConnector := &mockConnector{
		connectFunc: func(cfg models.ProxyConfig) (ProxyClient, error) {
			return nil, fmt.Errorf("test")
		},
	}

	pc := NewProxyChecker(proxies, nil, failConnector, "http://example.com", 10*time.Second)
	pc.CheckAll()

	if len(pc.Results()) != 2 {
		t.Fatalf("expected 2 results, got %d", len(pc.Results()))
	}

	// Update to only keep proxy-a
	newProxies := []models.ProxyConfig{proxies[0]}
	pc.UpdateProxies(newProxies)

	results := pc.Results()
	if len(results) != 1 {
		t.Fatalf("expected 1 result after update, got %d", len(results))
	}
	if _, ok := results[proxies[0].StableID]; !ok {
		t.Error("expected proxy-a result to remain")
	}
	if _, ok := results[proxies[1].StableID]; ok {
		t.Error("expected proxy-b result to be removed")
	}
}

func TestProxies_ReturnsCopy(t *testing.T) {
	proxies := []models.ProxyConfig{
		makeProxy(2, "proxy", "host:443", "auth"),
	}
	pc := NewProxyChecker(proxies, nil, nil, "http://example.com", 10*time.Second)

	got := pc.Proxies()
	if len(got) != 1 {
		t.Fatalf("expected 1 proxy, got %d", len(got))
	}

	// Mutating the returned slice should not affect the checker
	got[0].Name = "mutated"
	if pc.Proxies()[0].Name == "mutated" {
		t.Error("Proxies() should return a copy, not a reference")
	}
}

func TestResults_ReturnsCopy(t *testing.T) {
	proxies := []models.ProxyConfig{
		makeProxy(2, "proxy", "host:443", "auth"),
	}
	failConnector := &mockConnector{
		connectFunc: func(cfg models.ProxyConfig) (ProxyClient, error) {
			return nil, fmt.Errorf("test")
		},
	}

	pc := NewProxyChecker(proxies, nil, failConnector, "http://example.com", 10*time.Second)
	pc.CheckAll()

	r1 := pc.Results()
	r1[proxies[0].StableID] = CheckResult{Alive: true}

	r2 := pc.Results()
	if r2[proxies[0].StableID].Alive {
		t.Error("Results() should return a copy, not a reference")
	}
}

func TestCheckAll_EmptyProxies(t *testing.T) {
	pc := NewProxyChecker(nil, nil, nil, "http://example.com", 10*time.Second)
	pc.CheckAll() // should not panic
	if len(pc.Results()) != 0 {
		t.Error("expected empty results for empty proxy list")
	}
}

func TestDetectHostIP_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "  203.0.113.1\n")
	}))
	defer ts.Close()

	ip, err := DetectHostIP(ts.URL, 5*time.Second)
	if err != nil {
		t.Fatalf("DetectHostIP failed: %v", err)
	}
	if ip != "203.0.113.1" {
		t.Errorf("expected '203.0.113.1', got %q", ip)
	}
}

func TestDetectHostIP_EmptyResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "   \n")
	}))
	defer ts.Close()

	_, err := DetectHostIP(ts.URL, 5*time.Second)
	if err == nil {
		t.Fatal("expected error for empty IP response")
	}
}

func TestDetectHostIP_ServerDown(t *testing.T) {
	_, err := DetectHostIP("http://127.0.0.1:1", 1*time.Second)
	if err == nil {
		t.Fatal("expected error when server is down")
	}
}

func TestSetTestResults(t *testing.T) {
	proxies := []models.ProxyConfig{
		makeProxy(2, "proxy", "host:443", "auth"),
	}
	pc := NewProxyChecker(proxies, nil, nil, "http://example.com", 10*time.Second)

	pc.SetTestResults(map[string]CheckResult{
		proxies[0].StableID: {Alive: true, Latency: 42 * time.Millisecond},
	})

	results := pc.Results()
	r, ok := results[proxies[0].StableID]
	if !ok {
		t.Fatal("expected result to be set")
	}
	if !r.Alive || r.Latency != 42*time.Millisecond {
		t.Errorf("unexpected result: %+v", r)
	}
}
