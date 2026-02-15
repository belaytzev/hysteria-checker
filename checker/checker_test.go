package checker

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/belaytzev/hysteria-checker/models"
)

// mockProxyClient is a mock implementation of ProxyClient for testing.
type mockProxyClient struct {
	tcpFunc   func(addr string) (net.Conn, error)
	closeFunc func() error
}

func (m *mockProxyClient) TCP(addr string) (net.Conn, error) {
	if m.tcpFunc != nil {
		return m.tcpFunc(addr)
	}
	return nil, fmt.Errorf("TCP not configured")
}

func (m *mockProxyClient) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

// mockConnector is a mock implementation of Connector for testing.
type mockConnector struct {
	connectFunc func(cfg models.ProxyConfig) (ProxyClient, error)
}

func (m *mockConnector) Connect(cfg models.ProxyConfig) (ProxyClient, error) {
	if m.connectFunc != nil {
		return m.connectFunc(cfg)
	}
	return nil, fmt.Errorf("connect not configured")
}

func TestCheckViaProxy_Success(t *testing.T) {
	// Start a test HTTP server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "1.2.3.4")
	}))
	defer ts.Close()

	// Create a mock ProxyClient that dials the test server directly
	mc := &mockProxyClient{
		tcpFunc: func(addr string) (net.Conn, error) {
			// Connect directly to the test server instead of through a proxy
			return net.DialTimeout("tcp", addr, 5*time.Second)
		},
	}

	alive, latency, err := CheckViaProxy(mc, ts.URL, 10*time.Second)
	if err != nil {
		t.Fatalf("CheckViaProxy returned error: %v", err)
	}
	if !alive {
		t.Error("expected alive=true")
	}
	if latency <= 0 {
		t.Error("expected positive latency")
	}
}

func TestCheckViaProxy_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "error")
	}))
	defer ts.Close()

	mc := &mockProxyClient{
		tcpFunc: func(addr string) (net.Conn, error) {
			return net.DialTimeout("tcp", addr, 5*time.Second)
		},
	}

	alive, _, err := CheckViaProxy(mc, ts.URL, 10*time.Second)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if alive {
		t.Error("expected alive=false for 500 response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error to mention status 500, got: %v", err)
	}
}

func TestCheckViaProxy_TCPConnectFailure(t *testing.T) {
	mc := &mockProxyClient{
		tcpFunc: func(addr string) (net.Conn, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}

	alive, _, err := CheckViaProxy(mc, "http://example.com/check", 10*time.Second)
	if err == nil {
		t.Fatal("expected error for TCP connect failure")
	}
	if alive {
		t.Error("expected alive=false for TCP connect failure")
	}
	if !strings.Contains(err.Error(), "TCP connect failed") {
		t.Errorf("expected TCP connect error, got: %v", err)
	}
}

func TestCheckViaProxy_InvalidURL(t *testing.T) {
	mc := &mockProxyClient{}

	alive, _, err := CheckViaProxy(mc, "://invalid", 10*time.Second)
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
	if alive {
		t.Error("expected alive=false for invalid URL")
	}
}

func TestCheckViaProxy_DefaultPorts(t *testing.T) {
	tests := []struct {
		url          string
		expectedAddr string
	}{
		{"http://example.com/check", "example.com:80"},
		{"https://example.com/check", "example.com:443"},
		{"http://example.com:8080/check", "example.com:8080"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			var dialedAddr string
			mc := &mockProxyClient{
				tcpFunc: func(addr string) (net.Conn, error) {
					dialedAddr = addr
					return nil, fmt.Errorf("intentional error")
				},
			}

			CheckViaProxy(mc, tt.url, 10*time.Second)
			if dialedAddr != tt.expectedAddr {
				t.Errorf("expected dial to %s, got %s", tt.expectedAddr, dialedAddr)
			}
		})
	}
}

func TestNormalizeCertHash(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"AA:BB:CC:DD", "aabbccdd"},
		{"aa-bb-cc-dd", "aabbccdd"},
		{"AABBCCDD", "aabbccdd"},
		{"AA:BB-CC:DD", "aabbccdd"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeCertHash(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeCertHash(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestHysteria2Connector_InvalidServer(t *testing.T) {
	connector := &Hysteria2Connector{}

	// Missing port should fail at SplitHostPort
	_, err := connector.Connect(models.ProxyConfig{
		Version: 2,
		Server:  "example.com", // no port
	})
	if err == nil {
		t.Fatal("expected error for server without port")
	}
}

func TestConnectorInterface(t *testing.T) {
	// Verify Hysteria2Connector implements Connector
	var _ Connector = &Hysteria2Connector{}
}

func TestProxyClientInterface(t *testing.T) {
	// Verify mockProxyClient implements ProxyClient
	var _ ProxyClient = &mockProxyClient{}
}

func TestCheckResult_Fields(t *testing.T) {
	now := time.Now()
	cr := CheckResult{
		Alive:     true,
		Latency:   150 * time.Millisecond,
		LastCheck: now,
		Error:     "",
	}

	if !cr.Alive {
		t.Error("expected Alive=true")
	}
	if cr.Latency != 150*time.Millisecond {
		t.Errorf("expected Latency=150ms, got %v", cr.Latency)
	}
	if cr.LastCheck != now {
		t.Error("expected LastCheck to match")
	}
	if cr.Error != "" {
		t.Error("expected empty Error")
	}
}
