package parser

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/belaytzev/hysteria-checker/models"
)

// redactURI returns a URI with auth credentials removed for safe logging.
func redactURI(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		return "<invalid-uri>"
	}
	u.User = nil
	q := u.Query()
	for _, key := range []string{"auth", "auth_str", "obfsParam", "obfs-password"} {
		if q.Has(key) {
			q.Set(key, "REDACTED")
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// redactParseError extracts the reason from a url.Parse error without
// including the raw URL (which may contain credentials).
func redactParseError(err error) string {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return urlErr.Err.Error()
	}
	return "malformed URI"
}

// ParseLinks splits the input by newlines and parses each non-empty line as a Hysteria share link.
// It returns all successfully parsed configs and any errors encountered.
func ParseLinks(input string) ([]models.ProxyConfig, error) {
	lines := strings.Split(input, "\n")
	var configs []models.ProxyConfig
	var errs []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		cfg, err := ParseLink(line)
		if err != nil {
			errs = append(errs, fmt.Sprintf("line %q: %v", redactURI(line), err))
			continue
		}
		configs = append(configs, *cfg)
	}

	if len(configs) == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("no valid links found: %s", strings.Join(errs, "; "))
	}

	if len(errs) > 0 {
		return configs, fmt.Errorf("parse errors: %s", strings.Join(errs, "; "))
	}

	return configs, nil
}

// ParseLink parses a single Hysteria share link URI.
func ParseLink(uri string) (*models.ProxyConfig, error) {
	uri = strings.TrimSpace(uri)

	switch {
	case strings.HasPrefix(uri, "hysteria2://"), strings.HasPrefix(uri, "hy2://"):
		return ParseHysteria2(uri)
	case strings.HasPrefix(uri, "hysteria://"):
		return ParseHysteria1(uri)
	default:
		return nil, fmt.Errorf("unsupported URI scheme in %q", redactURI(uri))
	}
}
