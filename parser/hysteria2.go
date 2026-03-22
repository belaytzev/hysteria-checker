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
// Port hopping notation is also supported: hysteria2://auth@hostname:443,5000-6000/?...
func ParseHysteria2(rawURI string) (*models.ProxyConfig, error) {
	// url.Parse rejects port hopping specs (e.g. "443,5000-6000") as invalid ports.
	// Extract the hop spec before parsing, substitute a valid placeholder, then restore.
	parsedURI, hopPortSpec := extractHopPortSpec(rawURI)

	u, err := url.Parse(parsedURI)
	if err != nil {
		return nil, fmt.Errorf("invalid hysteria v2 URI: %s", redactParseError(err))
	}

	if u.Scheme != "hysteria2" && u.Scheme != "hy2" {
		return nil, fmt.Errorf("expected scheme 'hysteria2' or 'hy2', got '%s'", u.Scheme)
	}

	host := u.Hostname()
	if host == "" {
		return nil, fmt.Errorf("missing host in hysteria v2 URI")
	}

	port := hopPortSpec
	if port == "" {
		port = u.Port()
	}
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

// extractHopPortSpec detects a port hopping spec (e.g. "443,5000-6000") in the URI authority
// and returns a modified URI with a valid placeholder port plus the original hop spec.
// If no hop spec is found, returns the URI unchanged and an empty spec.
func extractHopPortSpec(rawURI string) (string, string) {
	schemeEnd := strings.Index(rawURI, "://")
	if schemeEnd < 0 {
		return rawURI, ""
	}
	rest := rawURI[schemeEnd+3:]

	// Find end of authority (before the first / ? or #)
	authEnd := strings.IndexAny(rest, "/?#")
	var authority, pathAndQuery string
	if authEnd >= 0 {
		authority = rest[:authEnd]
		pathAndQuery = rest[authEnd:]
	} else {
		authority = rest
		pathAndQuery = ""
	}

	// Find the port spec: the segment after the last colon in the authority,
	// unless the authority starts with '[' (IPv6 literal).
	var portSpec string
	var hostPart string
	if strings.HasPrefix(authority, "[") {
		// IPv6 literal: "[::1]:443,5000-6000"
		bracketEnd := strings.Index(authority, "]")
		if bracketEnd >= 0 && bracketEnd+1 < len(authority) && authority[bracketEnd+1] == ':' {
			hostPart = authority[:bracketEnd+2] // includes "]:"
			portSpec = authority[bracketEnd+2:]
		} else {
			return rawURI, ""
		}
	} else {
		// Strip userinfo before looking for the port colon so that colons
		// inside passwords (e.g. "user:pass-word@host") are not mistaken
		// for a port separator.
		hostAuthority := authority
		if at := strings.LastIndex(authority, "@"); at >= 0 {
			hostAuthority = authority[at+1:]
		}
		lastColon := strings.LastIndex(hostAuthority, ":")
		if lastColon < 0 {
			return rawURI, ""
		}
		// Compute offset within the full authority string.
		offset := len(authority) - len(hostAuthority)
		hostPart = authority[:offset+lastColon+1] // includes ":"
		portSpec = authority[offset+lastColon+1:]
	}

	// Only treat as a hop spec if it contains ',' or '-' (not a plain port number).
	if !strings.ContainsAny(portSpec, ",-") {
		return rawURI, ""
	}

	// Substitute the hop spec with a valid placeholder port for url.Parse.
	newURI := rawURI[:schemeEnd+3] + hostPart + "443" + pathAndQuery
	return newURI, portSpec
}
