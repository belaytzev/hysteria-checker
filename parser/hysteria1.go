package parser

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/belaytzev/hysteria-checker/models"
)

// ParseHysteria1 parses a hysteria:// URI into a ProxyConfig.
// Format: hysteria://host:port?protocol=udp&auth=123456&peer=sni.domain&insecure=1&upmbps=100&downmbps=100&alpn=hysteria&obfs=xplus&obfsParam=123456#remarks
func ParseHysteria1(rawURI string) (*models.ProxyConfig, error) {
	u, err := url.Parse(rawURI)
	if err != nil {
		return nil, fmt.Errorf("invalid hysteria v1 URI: %w", err)
	}

	if u.Scheme != "hysteria" {
		return nil, fmt.Errorf("expected scheme 'hysteria', got '%s'", u.Scheme)
	}

	host := u.Hostname()
	if host == "" {
		return nil, fmt.Errorf("missing host in hysteria v1 URI")
	}

	port := u.Port()
	if port == "" {
		return nil, fmt.Errorf("missing port in hysteria v1 URI")
	}

	cfg := &models.ProxyConfig{
		Version: 1,
		Server:  net_JoinHostPort(host, port),
		Name:    u.Fragment,
	}

	q := u.Query()

	cfg.Auth = q.Get("auth")
	cfg.SNI = q.Get("peer")
	cfg.Protocol = q.Get("protocol")
	cfg.ALPN = q.Get("alpn")
	cfg.Obfs = q.Get("obfs")
	cfg.ObfsParam = q.Get("obfsParam")

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

// net_JoinHostPort joins host and port, wrapping IPv6 addresses in brackets.
func net_JoinHostPort(host, port string) string {
	if strings.Contains(host, ":") {
		return "[" + host + "]:" + port
	}
	return host + ":" + port
}
