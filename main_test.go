package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRunStartupSequence(t *testing.T) {
	// Create a fake subscription server that returns a hysteria2 link
	subServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "hysteria2://testauth@127.0.0.1:44556/?insecure=1#test-proxy")
	}))
	defer subServer.Close()

	// Find a free port for the HTTP server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- run(ctx, []string{
			"--subscription-url", subServer.URL,
			"--metrics-host", "127.0.0.1",
			"--metrics-port", fmt.Sprintf("%d", port),
			"--check-interval", "1h",
			"--check-timeout", "1s",
		})
	}()

	// Wait for the HTTP server to be ready
	addr := fmt.Sprintf("http://127.0.0.1:%d", port)
	ready := false
	for i := 0; i < 100; i++ {
		resp, err := http.Get(addr + "/api/v1/status")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				ready = true
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !ready {
		t.Fatal("HTTP server did not become ready")
	}

	// Verify status endpoint returns valid JSON
	resp, err := http.Get(addr + "/api/v1/status")
	if err != nil {
		t.Fatalf("status request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %s", ct)
	}

	// Verify metrics endpoint
	resp, err = http.Get(addr + "/metrics")
	if err != nil {
		t.Fatalf("metrics request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected metrics status 200, got %d", resp.StatusCode)
	}

	// Verify dashboard
	resp, err = http.Get(addr + "/")
	if err != nil {
		t.Fatalf("dashboard request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected dashboard status 200, got %d", resp.StatusCode)
	}

	// Graceful shutdown
	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("run returned error: %v", err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("run did not exit within timeout")
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"debug", "DEBUG"},
		{"info", "INFO"},
		{"warn", "WARN"},
		{"error", "ERROR"},
		{"unknown", "INFO"},
		{"DEBUG", "DEBUG"},
	}
	for _, tt := range tests {
		got := parseLogLevel(tt.input)
		if got.String() != tt.want {
			t.Errorf("parseLogLevel(%q) = %s, want %s", tt.input, got.String(), tt.want)
		}
	}
}
