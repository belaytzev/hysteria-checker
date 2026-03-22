package subscription

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const (
	testTimeout = 5 * time.Second

	hy2Link1 = "hysteria2://letmein@example.com:443/?insecure=1&sni=real.example.com#Server1"
	hy2Link2 = "hysteria2://pass@other.com:8443/?insecure=0#Server2"
	hy1Link  = "hysteria://host.com:4433?protocol=udp&auth=secret&upmbps=100&downmbps=50#HY1Server"
)

func TestFetchSubscription_DirectLink(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantVer int
		wantErr bool
	}{
		{"hysteria2 direct link", hy2Link1, 2, false},
		{"hy2 direct link", "hy2://user@host.com:443/#Test", 2, false},
		{"hysteria1 direct link", hy1Link, 1, false},
		{"invalid direct link", "hysteria://", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configs, err := FetchSubscription(tt.url, testTimeout)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(configs) != 1 {
				t.Fatalf("expected 1 config, got %d", len(configs))
			}
			if configs[0].Version != tt.wantVer {
				t.Errorf("expected version %d, got %d", tt.wantVer, configs[0].Version)
			}
		})
	}
}

func TestFetchSubscription_PlainTextSubscription(t *testing.T) {
	links := hy2Link1 + "\n" + hy2Link2
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(links))
	}))
	defer srv.Close()

	configs, err := FetchSubscription(srv.URL, testTimeout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(configs) != 2 {
		t.Fatalf("expected 2 configs, got %d", len(configs))
	}
	if configs[0].Name != "Server1" {
		t.Errorf("expected name Server1, got %q", configs[0].Name)
	}
	if configs[1].Name != "Server2" {
		t.Errorf("expected name Server2, got %q", configs[1].Name)
	}
}

func TestFetchSubscription_Base64Subscription(t *testing.T) {
	links := hy2Link1 + "\n" + hy1Link
	encoded := base64.StdEncoding.EncodeToString([]byte(links))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(encoded))
	}))
	defer srv.Close()

	configs, err := FetchSubscription(srv.URL, testTimeout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(configs) != 2 {
		t.Fatalf("expected 2 configs, got %d", len(configs))
	}
	if configs[0].Version != 2 {
		t.Errorf("expected first config version 2, got %d", configs[0].Version)
	}
	if configs[1].Version != 1 {
		t.Errorf("expected second config version 1, got %d", configs[1].Version)
	}
}

func TestFetchSubscription_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := FetchSubscription(srv.URL, testTimeout)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "status 500") {
		t.Errorf("expected status 500 in error, got: %v", err)
	}
}

func TestFetchSubscription_InvalidContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not a valid link\nanother bad line"))
	}))
	defer srv.Close()

	_, err := FetchSubscription(srv.URL, testTimeout)
	if err == nil {
		t.Fatal("expected error for invalid content")
	}
}

func TestFetchAll_Deduplication(t *testing.T) {
	// Both servers return the same link
	links := hy2Link1
	srv1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(links))
	}))
	defer srv1.Close()
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(links))
	}))
	defer srv2.Close()

	configs, err := FetchAll([]string{srv1.URL, srv2.URL}, testTimeout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(configs) != 1 {
		t.Fatalf("expected 1 config after dedup, got %d", len(configs))
	}
}

func TestFetchAll_MixedDirectAndSubscription(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(hy2Link2))
	}))
	defer srv.Close()

	configs, err := FetchAll([]string{hy2Link1, srv.URL}, testTimeout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(configs) != 2 {
		t.Fatalf("expected 2 configs, got %d", len(configs))
	}
}

func TestFetchAll_AllFail(t *testing.T) {
	_, err := FetchAll([]string{"http://localhost:1/nonexistent"}, testTimeout)
	if err == nil {
		t.Fatal("expected error when all URLs fail")
	}
}

func TestFetchAll_EmptyURLs(t *testing.T) {
	configs, err := FetchAll([]string{"", "  "}, testTimeout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(configs) != 0 {
		t.Fatalf("expected 0 configs for empty URLs, got %d", len(configs))
	}
}

func TestFetchAll_PartialFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(hy2Link1))
	}))
	defer srv.Close()

	// One good URL, one bad URL - should return configs AND an error
	configs, err := FetchAll([]string{srv.URL, "http://localhost:1/nonexistent"}, testTimeout)
	if err == nil {
		t.Fatal("expected error for partial failure")
	}
	if len(configs) != 1 {
		t.Fatalf("expected 1 config from partial success, got %d", len(configs))
	}
}

func TestDecodeContent(t *testing.T) {
	original := "hysteria2://test@host:443/#Name"

	// Create base64 with line breaks (every 20 chars) to simulate MIME-style wrapping
	encoded := base64.StdEncoding.EncodeToString([]byte(original))
	var withLineBreaks string
	for i := 0; i < len(encoded); i += 20 {
		end := i + 20
		if end > len(encoded) {
			end = len(encoded)
		}
		if i > 0 {
			withLineBreaks += "\n"
		}
		withLineBreaks += encoded[i:end]
	}

	tests := []struct {
		name    string
		input   string
		want    string
	}{
		{"plain text passthrough", original, original},
		{"standard base64", encoded, original},
		{"raw base64 (no padding)", base64.RawStdEncoding.EncodeToString([]byte(original)), original},
		{"with whitespace", "  " + encoded + "  ", original},
		{"base64 with line breaks", withLineBreaks, original},
		{"base64 with CRLF", strings.ReplaceAll(withLineBreaks, "\n", "\r\n"), original},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decodeContent(tt.input)
			if got != tt.want {
				t.Errorf("decodeContent(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
