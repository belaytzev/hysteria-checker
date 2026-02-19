package parser

import (
	"fmt"
	"strings"

	"github.com/belaytzev/hysteria-checker/models"
)

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
			errs = append(errs, fmt.Sprintf("line %q: %v", line, err))
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
		return nil, fmt.Errorf("unsupported URI scheme in %q", uri)
	}
}
