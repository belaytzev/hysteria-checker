package checker

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/belaytzev/hysteria-checker/models"
)

func TestHysteria1Connector_BinaryNotFound(t *testing.T) {
	connector := &Hysteria1Connector{
		BinaryPath: "/nonexistent/hysteria-v1-binary",
	}

	_, err := connector.Connect(models.ProxyConfig{
		Version:  1,
		Server:   "example.com:443",
		Auth:     "test",
		UpMbps:   100,
		DownMbps: 100,
	})
	if err == nil {
		t.Fatal("expected error when binary not found")
	}
	if got := err.Error(); !contains(got, "not found") {
		t.Errorf("expected 'not found' in error, got: %s", got)
	}
}

func TestHysteria1ConnectorInterface(t *testing.T) {
	// Verify Hysteria1Connector implements Connector
	var _ Connector = &Hysteria1Connector{}
}

func TestHysteria1ClientInterface(t *testing.T) {
	// Verify hysteria1Client implements ProxyClient
	var _ ProxyClient = &hysteria1Client{}
}

func TestHysteria1Client_Close_NilProcess(t *testing.T) {
	// Close should not panic with nil cmd/process
	client := &hysteria1Client{}
	err := client.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

func TestWaitForPort_AlreadyListening(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer listener.Close()

	err = waitForPort(listener.Addr().String(), 2*time.Second)
	if err != nil {
		t.Errorf("waitForPort failed for already-listening port: %v", err)
	}
}

func TestWaitForPort_Timeout(t *testing.T) {
	// Use a port that's definitely not listening
	err := waitForPort("127.0.0.1:1", 500*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if got := err.Error(); !contains(got, "timeout") {
		t.Errorf("expected 'timeout' in error, got: %s", got)
	}
}

func TestDialViaSocks5_InvalidProxy(t *testing.T) {
	_, err := dialViaSocks5("127.0.0.1:1", "example.com:80", 1*time.Second)
	if err == nil {
		t.Fatal("expected error when proxy is not reachable")
	}
}

// minimalSocks5Server starts a minimal SOCKS5 server that accepts CONNECT
// requests and connects to the target directly. Returns the listener address.
func minimalSocks5Server(t *testing.T) net.Listener {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start SOCKS5 server: %v", err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go handleSocks5Conn(conn)
		}
	}()

	return listener
}

func handleSocks5Conn(conn net.Conn) {
	defer conn.Close()

	// Read greeting
	buf := make([]byte, 256)
	n, err := conn.Read(buf)
	if err != nil || n < 3 {
		return
	}

	// Respond: version 5, no auth
	conn.Write([]byte{0x05, 0x00})

	// Read connect request
	n, err = conn.Read(buf)
	if err != nil || n < 7 {
		return
	}

	// Parse the address
	var targetAddr string
	addrType := buf[3]
	switch addrType {
	case 0x01: // IPv4
		if n < 10 {
			return
		}
		ip := net.IP(buf[4:8])
		port := int(buf[8])<<8 | int(buf[9])
		targetAddr = fmt.Sprintf("%s:%d", ip.String(), port)
	case 0x03: // Domain
		domainLen := int(buf[4])
		if n < 5+domainLen+2 {
			return
		}
		domain := string(buf[5 : 5+domainLen])
		port := int(buf[5+domainLen])<<8 | int(buf[5+domainLen+1])
		targetAddr = fmt.Sprintf("%s:%d", domain, port)
	default:
		return
	}

	// Connect to target
	targetConn, err := net.DialTimeout("tcp", targetAddr, 5*time.Second)
	if err != nil {
		// Send failure response
		conn.Write([]byte{0x05, 0x01, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}
	defer targetConn.Close()

	// Send success response
	conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})

	// Bidirectional copy
	done := make(chan struct{})
	go func() {
		io.Copy(targetConn, conn)
		done <- struct{}{}
	}()
	go func() {
		io.Copy(conn, targetConn)
		done <- struct{}{}
	}()
	<-done
}

func TestDialViaSocks5_Success(t *testing.T) {
	// Start a test HTTP server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "hello from test")
	}))
	defer ts.Close()

	// Start a minimal SOCKS5 proxy
	socks := minimalSocks5Server(t)
	defer socks.Close()

	// Dial through SOCKS5 to the HTTP server
	conn, err := dialViaSocks5(socks.Addr().String(), ts.Listener.Addr().String(), 5*time.Second)
	if err != nil {
		t.Fatalf("dialViaSocks5 failed: %v", err)
	}
	defer conn.Close()

	// Send an HTTP request
	fmt.Fprintf(conn, "GET / HTTP/1.0\r\nHost: localhost\r\n\r\n")
	resp := make([]byte, 1024)
	n, _ := conn.Read(resp)
	if !contains(string(resp[:n]), "hello from test") {
		t.Errorf("unexpected response: %s", string(resp[:n]))
	}
}

func TestDialViaSocks5_TargetUnreachable(t *testing.T) {
	socks := minimalSocks5Server(t)
	defer socks.Close()

	// Try to connect to a port that's not listening
	_, err := dialViaSocks5(socks.Addr().String(), "127.0.0.1:1", 5*time.Second)
	if err == nil {
		t.Fatal("expected error when target is unreachable")
	}
}

func TestCheckViaProxy_WithSocks5(t *testing.T) {
	// End-to-end test: HTTP server -> SOCKS5 proxy -> CheckViaProxy
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "1.2.3.4")
	}))
	defer ts.Close()

	socks := minimalSocks5Server(t)
	defer socks.Close()

	// Create a ProxyClient backed by our SOCKS5 proxy
	client := &socks5ProxyClient{socksAddr: socks.Addr().String()}

	alive, latency, err := CheckViaProxy(client, ts.URL, 10*time.Second)
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

// socks5ProxyClient is a ProxyClient that dials through a SOCKS5 proxy.
type socks5ProxyClient struct {
	socksAddr string
}

func (s *socks5ProxyClient) TCP(addr string) (net.Conn, error) {
	return dialViaSocks5(s.socksAddr, addr, 10*time.Second)
}

func (s *socks5ProxyClient) Close() error {
	return nil
}

func TestReadFull(t *testing.T) {
	// Create a pipe to test readFull
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	go func() {
		// Write in two chunks
		server.Write([]byte{0x01, 0x02})
		time.Sleep(10 * time.Millisecond)
		server.Write([]byte{0x03, 0x04})
	}()

	buf := make([]byte, 4)
	n, err := readFull(client, buf)
	if err != nil {
		t.Fatalf("readFull returned error: %v", err)
	}
	if n != 4 {
		t.Errorf("expected 4 bytes, got %d", n)
	}
	expected := []byte{0x01, 0x02, 0x03, 0x04}
	for i, b := range buf {
		if b != expected[i] {
			t.Errorf("byte %d: expected %x, got %x", i, expected[i], b)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
