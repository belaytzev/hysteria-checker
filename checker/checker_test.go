package checker

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/apernet/hysteria/extras/v2/obfs"
	"github.com/apernet/hysteria/extras/v2/transport/udphop"
	"github.com/belaytzev/hysteria-checker/models"
)

// resolveTestHopAddr is a test helper that resolves a UDP hop address string.
func resolveTestHopAddr(addr string) (*udphop.UDPHopAddr, error) {
	return udphop.ResolveUDPHopAddr(addr)
}

// newTestSalamanderObfuscator is a test helper that creates a Salamander obfuscator.
func newTestSalamanderObfuscator(password string) (obfs.Obfuscator, error) {
	return obfs.NewSalamanderObfuscator([]byte(password))
}

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

	alive, latency, _, err := CheckViaProxy(mc, ts.URL, 10*time.Second)
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

	alive, _, _, err := CheckViaProxy(mc, ts.URL, 10*time.Second)
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

	alive, _, _, err := CheckViaProxy(mc, "http://example.com/check", 10*time.Second)
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

	alive, _, _, err := CheckViaProxy(mc, "://invalid", 10*time.Second)
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

			_, _, _, _ = CheckViaProxy(mc, tt.url, 10*time.Second)
			if dialedAddr != tt.expectedAddr {
				t.Errorf("expected dial to %s, got %s", tt.expectedAddr, dialedAddr)
			}
		})
	}
}

func TestIsPlainPort(t *testing.T) {
	tests := []struct {
		input string
		plain bool
	}{
		{"443", true},
		{"8080", true},
		{"0", true},
		{"65535", true},
		{"443,8000-9000", false},
		{"5000-6000", false},
		{"443,444", false},
		{"", false},
		{"abc", false},
		{"65536", false}, // out of uint16 range
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isPlainPort(tt.input)
			if got != tt.plain {
				t.Errorf("isPlainPort(%q) = %v, want %v", tt.input, got, tt.plain)
			}
		})
	}
}

func TestPortHopConnFactory_New_PlainUDP(t *testing.T) {
	addr, err := resolveTestHopAddr("127.0.0.1:10000,10001")
	if err != nil {
		t.Skipf("resolving hop addr: %v", err)
	}
	f := &portHopConnFactory{addr: addr}
	conn, err := f.New(nil)
	if err != nil {
		t.Fatalf("portHopConnFactory.New returned error: %v", err)
	}
	defer conn.Close()

	// Verify the connection is writable (functional, not just non-nil).
	_, writeErr := conn.WriteTo([]byte("test"), addr)
	if writeErr != nil {
		t.Errorf("expected writable PacketConn, got write error: %v", writeErr)
	}
}

func TestPortHopConnFactory_New_WithObfs(t *testing.T) {
	// Verify that f.New() succeeds when an obfuscator is set and the returned
	// conn is writable. The obfuscation wrapping cannot be inspected directly
	// because udpHopPacketConn hides the inner conn; the write assertion
	// confirms the factory wiring does not panic or error.
	addr, err := resolveTestHopAddr("127.0.0.1:10000,10001")
	if err != nil {
		t.Skipf("resolving hop addr: %v", err)
	}
	obfuscator, err := newTestSalamanderObfuscator("testpassword")
	if err != nil {
		t.Fatalf("creating obfuscator: %v", err)
	}

	f := &portHopConnFactory{addr: addr, obfuscator: obfuscator}
	conn, err := f.New(nil)
	if err != nil {
		t.Fatalf("portHopConnFactory.New returned error: %v", err)
	}
	defer conn.Close()

	// The udpHopPacketConn wraps the inner conn returned by listenFn.
	// We cannot inspect the inner conn directly, but we can verify f.New()
	// succeeded and the conn is usable — a write attempt should not error.
	_, writeErr := conn.WriteTo([]byte("test"), addr)
	if writeErr != nil {
		t.Errorf("expected writable PacketConn with obfs, got write error: %v", writeErr)
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

func TestHysteria2Connector_PortHoppingAddress(t *testing.T) {
	// Verify that Connect() with a port-hopping server address fails at the
	// network level (client.NewClient), not at address parsing. This confirms
	// that isPlainPort detection and udphop.ResolveUDPHopAddr are used instead
	// of net.ResolveUDPAddr (which would fail immediately with "invalid port").
	connector := &Hysteria2Connector{}

	_, err := connector.Connect(models.ProxyConfig{
		Version:  2,
		Server:   "127.0.0.1:443,5000-6000",
		Auth:     "testauth",
		Insecure: true,
	})
	// We expect an error (no server is running). Verify it comes from the
	// hysteria client connection attempt, not from address parsing/resolution.
	// Errors at resolution indicate the wrong code path was taken.
	if err == nil {
		t.Fatal("expected error connecting to non-existent server")
	}
	if strings.Contains(err.Error(), "invalid port") {
		t.Errorf("Connect() used net.ResolveUDPAddr for port-hopping address; got: %v", err)
	}
	if strings.Contains(err.Error(), "failed to resolve server address") {
		t.Errorf("Connect() failed at address resolution instead of using udphop; got: %v", err)
	}
	// Positive assertion: error must come from the hysteria2 connection phase.
	if !strings.Contains(err.Error(), "failed to connect to hysteria2 server") {
		t.Errorf("expected connection-phase error, got: %v", err)
	}
}

func TestHysteria2Connector_HopAddressResolutionFailure(t *testing.T) {
	// Verify that a malformed hop spec (SplitHostPort succeeds but ResolveUDPHopAddr fails)
	// returns the expected "failed to resolve hop server address" error.
	connector := &Hysteria2Connector{}

	_, err := connector.Connect(models.ProxyConfig{
		Version: 2,
		Server:  "127.0.0.1:abc-xyz", // isPlainPort("abc-xyz")==false, udphop resolve will fail
	})
	if err == nil {
		t.Fatal("expected error for malformed hop address")
	}
	if !strings.Contains(err.Error(), "failed to resolve hop server address") {
		t.Errorf("expected hop resolution error, got: %v", err)
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
