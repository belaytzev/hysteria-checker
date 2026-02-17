package parser

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/belaytzev/hysteria-checker/models"
)

// ParseHysteria2 parses a hysteria2:// or hy2:// URI into a ProxyConfig.
// Format: hysteria2://[auth@]hostname[:port]/?insecure=1&obfs=salamander&obfs-password=gawrgura&pinSHA256=deadbeef&sni=real.example.com
func ParseHysteria2(rawURI string) (*models.ProxyConfig, error) {
	u, err := url.Parse(rawURI)
	if err != nil {
		return nil, fmt.Errorf("invalid hysteria v2 URI: %w", err)
	}

	if u.Scheme != "hysteria2" && u.Scheme != "hy2" {
		return nil, fmt.Errorf("expected scheme 'hysteria2' or 'hy2', got '%s'", u.Scheme)
	}

	host := u.Hostname()
	if host == "" {
		return nil, fmt.Errorf("missing host in hysteria v2 URI")
	}

	port := u.Port()
	if port == "" {
		port = "443"
	}

	cfg := &models.ProxyConfig{
		Version: 2,
		Server:  net.JoinHostPort(host, port),
		Name:    u.Fragment,
	}

	// Auth can be in userinfo (before @) or in query params
	if u.User != nil {
		cfg.Auth = u.User.Username()
		if pwd, ok := u.User.Password(); ok {
			cfg.Auth = cfg.Auth + ":" + pwd
		}
	}

	q := u.Query()

	if cfg.Auth == "" {
		cfg.Auth = q.Get("auth")
	}

	cfg.SNI = q.Get("sni")
	cfg.PinSHA256 = q.Get("pinSHA256")
	cfg.Obfs = q.Get("obfs")
	cfg.ObfsParam = q.Get("obfs-password")

	if v := q.Get("insecure"); v == "1" || strings.EqualFold(v, "true") {
		cfg.Insecure = true
	}

	if v := q.Get("upmbps"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid upmbps value '%s': %w", v, err)
		}
		cfg.UpMbps = n
	}

	if v := q.Get("downmbps"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid downmbps value '%s': %w", v, err)
		}
		cfg.DownMbps = n
	}

	cfg.GenerateStableID()
	return cfg, nil
}
