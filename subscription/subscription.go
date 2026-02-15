package subscription

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/belaytzev/hysteria-checker/models"
	"github.com/belaytzev/hysteria-checker/parser"
)

// isDirectLink returns true if the URL is a direct Hysteria share link.
func isDirectLink(url string) bool {
	return strings.HasPrefix(url, "hysteria://") ||
		strings.HasPrefix(url, "hysteria2://") ||
		strings.HasPrefix(url, "hy2://")
}

// FetchSubscription fetches a subscription URL and returns parsed proxy configs.
// If the URL is a direct share link, it parses it directly.
// For subscription URLs, it fetches the content, tries base64 decoding, then parses each line.
func FetchSubscription(url string, timeout time.Duration) ([]models.ProxyConfig, error) {
	if isDirectLink(url) {
		cfg, err := parser.ParseLink(url)
		if err != nil {
			return nil, fmt.Errorf("parsing direct link: %w", err)
		}
		return []models.ProxyConfig{*cfg}, nil
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching subscription %q: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("subscription %q returned status %d", url, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading subscription body: %w", err)
	}

	content := decodeContent(string(body))

	configs, err := parser.ParseLinks(content)
	if err != nil {
		return nil, fmt.Errorf("parsing subscription content: %w", err)
	}

	return configs, nil
}

// decodeContent tries to base64-decode the input. If decoding fails, returns the original string.
func decodeContent(s string) string {
	s = strings.TrimSpace(s)
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		// Try URL-safe base64
		decoded, err = base64.URLEncoding.DecodeString(s)
		if err != nil {
			// Try without padding
			decoded, err = base64.RawStdEncoding.DecodeString(s)
			if err != nil {
				return s
			}
		}
	}
	return string(decoded)
}

// FetchAll fetches multiple subscription URLs, aggregates results, and deduplicates by StableID.
func FetchAll(urls []string, timeout time.Duration) ([]models.ProxyConfig, error) {
	seen := make(map[string]struct{})
	var result []models.ProxyConfig
	var errs []string

	for _, url := range urls {
		url = strings.TrimSpace(url)
		if url == "" {
			continue
		}

		configs, err := FetchSubscription(url, timeout)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}

		for _, cfg := range configs {
			if _, exists := seen[cfg.StableID]; !exists {
				seen[cfg.StableID] = struct{}{}
				result = append(result, cfg)
			}
		}
	}

	if len(result) == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("no proxies found: %s", strings.Join(errs, "; "))
	}

	return result, nil
}
