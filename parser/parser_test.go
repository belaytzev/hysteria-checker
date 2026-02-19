package parser

import (
	"net/url"
	"testing"
)

// --- Hysteria v1 Tests ---

func TestParseHysteria1_FullURI(t *testing.T) {
	uri := "hysteria://example.com:443?protocol=udp&auth=123456&peer=sni.domain&insecure=1&upmbps=100&downmbps=100&alpn=hysteria&obfs=xplus&obfsParam=123456#my-server"
	cfg, err := ParseHysteria1(uri)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Version != 1 {
		t.Errorf("version = %d, want 1", cfg.Version)
	}
	if cfg.Server != "example.com:443" {
		t.Errorf("server = %q, want %q", cfg.Server, "example.com:443")
	}
	if cfg.Auth != "123456" {
		t.Errorf("auth = %q, want %q", cfg.Auth, "123456")
	}
	if cfg.SNI != "sni.domain" {
		t.Errorf("sni = %q, want %q", cfg.SNI, "sni.domain")
	}
	if !cfg.Insecure {
		t.Error("insecure = false, want true")
	}
	if cfg.UpMbps != 100 {
		t.Errorf("upmbps = %d, want 100", cfg.UpMbps)
	}
	if cfg.DownMbps != 100 {
		t.Errorf("downmbps = %d, want 100", cfg.DownMbps)
	}
	if cfg.ALPN != "hysteria" {
		t.Errorf("alpn = %q, want %q", cfg.ALPN, "hysteria")
	}
	if cfg.Protocol != "udp" {
		t.Errorf("protocol = %q, want %q", cfg.Protocol, "udp")
	}
	if cfg.Obfs != "xplus" {
		t.Errorf("obfs = %q, want %q", cfg.Obfs, "xplus")
	}
	if cfg.ObfsParam != "123456" {
		t.Errorf("obfsParam = %q, want %q", cfg.ObfsParam, "123456")
	}
	if cfg.Name != "my-server" {
		t.Errorf("name = %q, want %q", cfg.Name, "my-server")
	}
	if cfg.StableID == "" {
		t.Error("stableID should not be empty")
	}
}

func TestParseHysteria1_MinimalURI(t *testing.T) {
	uri := "hysteria://10.0.0.1:8443?auth=pass#node1"
	cfg, err := ParseHysteria1(uri)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Server != "10.0.0.1:8443" {
		t.Errorf("server = %q, want %q", cfg.Server, "10.0.0.1:8443")
	}
	if cfg.Auth != "pass" {
		t.Errorf("auth = %q, want %q", cfg.Auth, "pass")
	}
	if cfg.Name != "node1" {
		t.Errorf("name = %q, want %q", cfg.Name, "node1")
	}
	if cfg.Insecure {
		t.Error("insecure should be false by default")
	}
	if cfg.UpMbps != 0 {
		t.Errorf("upmbps = %d, want 0", cfg.UpMbps)
	}
}

func TestParseHysteria1_MissingHost(t *testing.T) {
	_, err := ParseHysteria1("hysteria://:443?auth=x")
	if err == nil {
		t.Error("expected error for missing host")
	}
}

func TestParseHysteria1_MissingPort(t *testing.T) {
	_, err := ParseHysteria1("hysteria://example.com?auth=x")
	if err == nil {
		t.Error("expected error for missing port")
	}
}

func TestParseHysteria1_WrongScheme(t *testing.T) {
	_, err := ParseHysteria1("hysteria2://example.com:443?auth=x")
	if err == nil {
		t.Error("expected error for wrong scheme")
	}
}

func TestParseHysteria1_InvalidUpMbps(t *testing.T) {
	_, err := ParseHysteria1("hysteria://example.com:443?upmbps=abc")
	if err == nil {
		t.Error("expected error for invalid upmbps")
	}
}

func TestParseHysteria1_IPv6(t *testing.T) {
	uri := "hysteria://[2001:db8::1]:443?auth=test#ipv6-node"
	cfg, err := ParseHysteria1(uri)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server != "[2001:db8::1]:443" {
		t.Errorf("server = %q, want %q", cfg.Server, "[2001:db8::1]:443")
	}
}

func TestParseHysteria1_InsecureTrue(t *testing.T) {
	uri := "hysteria://example.com:443?insecure=true"
	cfg, err := ParseHysteria1(uri)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Insecure {
		t.Error("insecure should be true for insecure=true")
	}
}

func TestParseHysteria1_PercentEncodedFragment(t *testing.T) {
	uri := "hysteria://example.com:443?auth=pass#my%20server"
	cfg, err := ParseHysteria1(uri)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Name != "my server" {
		t.Errorf("name = %q, want %q", cfg.Name, "my server")
	}
}

// --- Hysteria v2 Tests ---

func TestParseHysteria2_FullURI(t *testing.T) {
	uri := "hysteria2://myauth@example.com:443/?insecure=1&obfs=salamander&obfs-password=gawrgura&pinSHA256=deadbeef&sni=real.example.com#my-hy2"
	cfg, err := ParseHysteria2(uri)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Version != 2 {
		t.Errorf("version = %d, want 2", cfg.Version)
	}
	if cfg.Server != "example.com:443" {
		t.Errorf("server = %q, want %q", cfg.Server, "example.com:443")
	}
	if cfg.Auth != "myauth" {
		t.Errorf("auth = %q, want %q", cfg.Auth, "myauth")
	}
	if cfg.SNI != "real.example.com" {
		t.Errorf("sni = %q, want %q", cfg.SNI, "real.example.com")
	}
	if !cfg.Insecure {
		t.Error("insecure = false, want true")
	}
	if cfg.Obfs != "salamander" {
		t.Errorf("obfs = %q, want %q", cfg.Obfs, "salamander")
	}
	if cfg.ObfsParam != "gawrgura" {
		t.Errorf("obfsParam = %q, want %q", cfg.ObfsParam, "gawrgura")
	}
	if cfg.PinSHA256 != "deadbeef" {
		t.Errorf("pinSHA256 = %q, want %q", cfg.PinSHA256, "deadbeef")
	}
	if cfg.Name != "my-hy2" {
		t.Errorf("name = %q, want %q", cfg.Name, "my-hy2")
	}
	if cfg.StableID == "" {
		t.Error("stableID should not be empty")
	}
}

func TestParseHysteria2_Hy2Scheme(t *testing.T) {
	uri := "hy2://user@example.com:8443/?sni=sni.com#hy2-node"
	cfg, err := ParseHysteria2(uri)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Version != 2 {
		t.Errorf("version = %d, want 2", cfg.Version)
	}
	if cfg.Auth != "user" {
		t.Errorf("auth = %q, want %q", cfg.Auth, "user")
	}
	if cfg.Server != "example.com:8443" {
		t.Errorf("server = %q, want %q", cfg.Server, "example.com:8443")
	}
}

func TestParseHysteria2_DefaultPort(t *testing.T) {
	uri := "hysteria2://auth@example.com/?sni=x"
	cfg, err := ParseHysteria2(uri)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server != "example.com:443" {
		t.Errorf("server = %q, want %q", cfg.Server, "example.com:443")
	}
}

func TestParseHysteria2_AuthWithPassword(t *testing.T) {
	uri := "hysteria2://user:pass@example.com:443/"
	cfg, err := ParseHysteria2(uri)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Auth != "user:pass" {
		t.Errorf("auth = %q, want %q", cfg.Auth, "user:pass")
	}
}

func TestParseHysteria2_AuthInQuery(t *testing.T) {
	uri := "hysteria2://example.com:443/?auth=queryauth"
	cfg, err := ParseHysteria2(uri)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Auth != "queryauth" {
		t.Errorf("auth = %q, want %q", cfg.Auth, "queryauth")
	}
}

func TestParseHysteria2_MissingHost(t *testing.T) {
	_, err := ParseHysteria2("hysteria2://:443/")
	if err == nil {
		t.Error("expected error for missing host")
	}
}

func TestParseHysteria2_WrongScheme(t *testing.T) {
	_, err := ParseHysteria2("hysteria://example.com:443/")
	if err == nil {
		t.Error("expected error for wrong scheme")
	}
}

func TestParseHysteria2_IPv6(t *testing.T) {
	uri := "hysteria2://auth@[::1]:443/"
	cfg, err := ParseHysteria2(uri)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server != "[::1]:443" {
		t.Errorf("server = %q, want %q", cfg.Server, "[::1]:443")
	}
}

func TestParseHysteria2_Bandwidth(t *testing.T) {
	uri := "hysteria2://auth@example.com:443/?upmbps=50&downmbps=200"
	cfg, err := ParseHysteria2(uri)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.UpMbps != 50 {
		t.Errorf("upmbps = %d, want 50", cfg.UpMbps)
	}
	if cfg.DownMbps != 200 {
		t.Errorf("downmbps = %d, want 200", cfg.DownMbps)
	}
}

func TestParseHysteria2_PortHopping_CommaRange(t *testing.T) {
	uri := "hysteria2://auth@example.com:443,5000-6000/?insecure=1&obfs=salamander&obfs-password=pw#hop-node"
	cfg, err := ParseHysteria2(uri)
	if err != nil {
		t.Fatalf("unexpected error parsing port-hopping URI: %v", err)
	}
	if cfg.Server != "example.com:443,5000-6000" {
		t.Errorf("server = %q, want %q", cfg.Server, "example.com:443,5000-6000")
	}
	if cfg.Auth != "auth" {
		t.Errorf("auth = %q, want %q", cfg.Auth, "auth")
	}
	if !cfg.Insecure {
		t.Error("insecure = false, want true")
	}
	if cfg.Obfs != "salamander" {
		t.Errorf("obfs = %q, want %q", cfg.Obfs, "salamander")
	}
	if cfg.ObfsParam != "pw" {
		t.Errorf("obfsParam = %q, want %q", cfg.ObfsParam, "pw")
	}
	if cfg.Name != "hop-node" {
		t.Errorf("name = %q, want %q", cfg.Name, "hop-node")
	}
}

func TestParseHysteria2_PortHopping_RangeOnly(t *testing.T) {
	uri := "hysteria2://auth@example.com:5000-6000/"
	cfg, err := ParseHysteria2(uri)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server != "example.com:5000-6000" {
		t.Errorf("server = %q, want %q", cfg.Server, "example.com:5000-6000")
	}
}

func TestParseHysteria2_PortHopping_IPv4(t *testing.T) {
	uri := "hysteria2://auth@1.2.3.4:443,5000-6000/"
	cfg, err := ParseHysteria2(uri)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server != "1.2.3.4:443,5000-6000" {
		t.Errorf("server = %q, want %q", cfg.Server, "1.2.3.4:443,5000-6000")
	}
}

func TestParseHysteria2_extractHopPortSpec_NoHop(t *testing.T) {
	uri := "hysteria2://auth@example.com:443/?insecure=1"
	got, hopSpec := extractHopPortSpec(uri)
	if got != uri {
		t.Errorf("expected URI unchanged, got %q", got)
	}
	if hopSpec != "" {
		t.Errorf("expected empty hop spec, got %q", hopSpec)
	}
}

func TestParseHysteria2_extractHopPortSpec_WithHop(t *testing.T) {
	uri := "hysteria2://auth@example.com:443,5000-6000/?insecure=1"
	got, hopSpec := extractHopPortSpec(uri)
	if hopSpec != "443,5000-6000" {
		t.Errorf("expected hop spec '443,5000-6000', got %q", hopSpec)
	}
	if got == uri {
		t.Error("expected modified URI, got original")
	}
	// Modified URI must be parseable by url.Parse (the whole point of extractHopPortSpec).
	if _, err := url.Parse(got); err != nil {
		t.Errorf("modified URI %q not parseable by url.Parse: %v", got, err)
	}
}

func TestParseHysteria2_extractHopPortSpec_IPv6Hop(t *testing.T) {
	uri := "hysteria2://auth@[::1]:443,5000-6000/?insecure=1"
	got, hopSpec := extractHopPortSpec(uri)
	if hopSpec != "443,5000-6000" {
		t.Errorf("expected hop spec '443,5000-6000', got %q", hopSpec)
	}
	if got == uri {
		t.Error("expected modified URI, got original")
	}
	// Modified URI must be parseable and retain the IPv6 host.
	if _, err := url.Parse(got); err != nil {
		t.Errorf("modified URI %q not parseable by url.Parse: %v", got, err)
	}
	cfg, err := ParseHysteria2(uri)
	if err != nil {
		t.Fatalf("ParseHysteria2 failed for IPv6 hop URI: %v", err)
	}
	if cfg.Server != "[::1]:443,5000-6000" {
		t.Errorf("Server = %q, want \"[::1]:443,5000-6000\"", cfg.Server)
	}
}

// --- ParseLink Tests ---

func TestParseLink_Hysteria1(t *testing.T) {
	cfg, err := ParseLink("hysteria://example.com:443?auth=x#test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Version != 1 {
		t.Errorf("version = %d, want 1", cfg.Version)
	}
}

func TestParseLink_Hysteria2(t *testing.T) {
	cfg, err := ParseLink("hysteria2://auth@example.com:443/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Version != 2 {
		t.Errorf("version = %d, want 2", cfg.Version)
	}
}

func TestParseLink_Hy2(t *testing.T) {
	cfg, err := ParseLink("hy2://auth@example.com:443/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Version != 2 {
		t.Errorf("version = %d, want 2", cfg.Version)
	}
}

func TestParseLink_Unsupported(t *testing.T) {
	_, err := ParseLink("https://example.com")
	if err == nil {
		t.Error("expected error for unsupported scheme")
	}
}

func TestParseLink_Whitespace(t *testing.T) {
	cfg, err := ParseLink("  hysteria://example.com:443?auth=x  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Version != 1 {
		t.Errorf("version = %d, want 1", cfg.Version)
	}
}

// --- ParseLinks Tests ---

func TestParseLinks_MultipleLinks(t *testing.T) {
	input := "hysteria://server1.com:443?auth=a#node1\nhysteria2://user@server2.com:443/#node2\nhy2://user@server3.com:443/#node3"
	configs, err := ParseLinks(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(configs) != 3 {
		t.Fatalf("got %d configs, want 3", len(configs))
	}
	if configs[0].Version != 1 {
		t.Errorf("configs[0].version = %d, want 1", configs[0].Version)
	}
	if configs[1].Version != 2 {
		t.Errorf("configs[1].version = %d, want 2", configs[1].Version)
	}
	if configs[2].Version != 2 {
		t.Errorf("configs[2].version = %d, want 2", configs[2].Version)
	}
}

func TestParseLinks_EmptyLines(t *testing.T) {
	input := "\nhysteria://server.com:443?auth=a\n\n\nhysteria2://b@server2.com:443/\n"
	configs, err := ParseLinks(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(configs) != 2 {
		t.Fatalf("got %d configs, want 2", len(configs))
	}
}

func TestParseLinks_AllInvalid(t *testing.T) {
	input := "https://example.com\nftp://bad.link"
	_, err := ParseLinks(input)
	if err == nil {
		t.Error("expected error when all links are invalid")
	}
}

func TestParseLinks_PartiallyValid(t *testing.T) {
	input := "hysteria://server.com:443?auth=a#ok\nhttps://bad.link"
	configs, err := ParseLinks(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(configs) != 1 {
		t.Fatalf("got %d configs, want 1", len(configs))
	}
}

func TestParseLinks_Empty(t *testing.T) {
	configs, err := ParseLinks("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(configs) != 0 {
		t.Errorf("got %d configs, want 0", len(configs))
	}
}

// --- StableID Tests ---

func TestStableID_Deterministic(t *testing.T) {
	uri := "hysteria2://auth@example.com:443/"
	cfg1, _ := ParseHysteria2(uri)
	cfg2, _ := ParseHysteria2(uri)
	if cfg1.StableID != cfg2.StableID {
		t.Errorf("stableID not deterministic: %q != %q", cfg1.StableID, cfg2.StableID)
	}
}

func TestStableID_DifferentForDifferentServers(t *testing.T) {
	cfg1, _ := ParseHysteria2("hysteria2://auth@server1.com:443/")
	cfg2, _ := ParseHysteria2("hysteria2://auth@server2.com:443/")
	if cfg1.StableID == cfg2.StableID {
		t.Error("stableID should differ for different servers")
	}
}

func TestStableID_DifferentForDifferentVersions(t *testing.T) {
	cfg1, _ := ParseHysteria1("hysteria://example.com:443?auth=a")
	cfg2, _ := ParseHysteria2("hysteria2://a@example.com:443/")
	if cfg1.StableID == cfg2.StableID {
		t.Error("stableID should differ for different versions")
	}
}
