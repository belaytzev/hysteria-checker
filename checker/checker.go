package checker

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/belaytzev/hysteria-checker/models"
)

// ProxyChecker orchestrates health checking of all configured proxies.
type ProxyChecker struct {
	mu          sync.RWMutex
	proxies     []models.ProxyConfig
	results     map[string]CheckResult
	v1Connector Connector
	v2Connector Connector
	checkURL    string
	timeout     time.Duration
	hostIP      string
	checkMethod string
	concurrency int
}

// ProxyCheckerOption configures a ProxyChecker.
type ProxyCheckerOption func(*ProxyChecker)

// WithConcurrency sets the maximum number of concurrent checks.
func WithConcurrency(n int) ProxyCheckerOption {
	return func(pc *ProxyChecker) {
		if n > 0 {
			pc.concurrency = n
		}
	}
}

// WithHostIP sets the host's public IP for IP-based checks.
func WithHostIP(ip string) ProxyCheckerOption {
	return func(pc *ProxyChecker) {
		pc.hostIP = ip
	}
}

// WithCheckMethod sets the check method ("ip" or "status").
func WithCheckMethod(method string) ProxyCheckerOption {
	return func(pc *ProxyChecker) {
		pc.checkMethod = method
	}
}

// NewProxyChecker creates a new ProxyChecker.
func NewProxyChecker(
	proxies []models.ProxyConfig,
	v1Connector Connector,
	v2Connector Connector,
	checkURL string,
	timeout time.Duration,
	opts ...ProxyCheckerOption,
) *ProxyChecker {
	pc := &ProxyChecker{
		proxies:     proxies,
		results:     make(map[string]CheckResult),
		v1Connector: v1Connector,
		v2Connector: v2Connector,
		checkURL:    checkURL,
		timeout:     timeout,
		concurrency: 5,
	}
	for _, opt := range opts {
		opt(pc)
	}
	return pc
}

// UpdateProxies replaces the proxy list (e.g., after subscription refresh).
func (pc *ProxyChecker) UpdateProxies(proxies []models.ProxyConfig) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.proxies = proxies
	// Remove results for proxies that no longer exist
	active := make(map[string]bool, len(proxies))
	for _, p := range proxies {
		active[p.StableID] = true
	}
	for id := range pc.results {
		if !active[id] {
			delete(pc.results, id)
		}
	}
}

// Results returns a copy of the current check results.
func (pc *ProxyChecker) Results() map[string]CheckResult {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	out := make(map[string]CheckResult, len(pc.results))
	for k, v := range pc.results {
		out[k] = v
	}
	return out
}

// SetTestResults sets check results directly (for testing only).
func (pc *ProxyChecker) SetTestResults(results map[string]CheckResult) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	for k, v := range results {
		pc.results[k] = v
	}
}

// Proxies returns a copy of the current proxy list.
func (pc *ProxyChecker) Proxies() []models.ProxyConfig {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	out := make([]models.ProxyConfig, len(pc.proxies))
	copy(out, pc.proxies)
	return out
}

// CheckAll checks all proxies concurrently with a semaphore limiting concurrency.
func (pc *ProxyChecker) CheckAll() {
	pc.mu.RLock()
	proxies := make([]models.ProxyConfig, len(pc.proxies))
	copy(proxies, pc.proxies)
	pc.mu.RUnlock()

	sem := make(chan struct{}, pc.concurrency)
	var wg sync.WaitGroup

	for _, proxy := range proxies {
		wg.Add(1)
		sem <- struct{}{}
		go func(p models.ProxyConfig) {
			defer wg.Done()
			defer func() { <-sem }()
			result := pc.checkOne(p)
			pc.mu.Lock()
			pc.results[p.StableID] = result
			pc.mu.Unlock()
		}(proxy)
	}

	wg.Wait()
}

// checkOne performs a health check on a single proxy.
func (pc *ProxyChecker) checkOne(proxy models.ProxyConfig) CheckResult {
	var connector Connector
	switch proxy.Version {
	case 1:
		connector = pc.v1Connector
	case 2:
		connector = pc.v2Connector
	default:
		return CheckResult{
			LastCheck: time.Now(),
			Error:     fmt.Sprintf("unsupported proxy version: %d", proxy.Version),
		}
	}

	if connector == nil {
		return CheckResult{
			LastCheck: time.Now(),
			Error:     fmt.Sprintf("no connector available for v%d", proxy.Version),
		}
	}

	client, err := connector.Connect(proxy)
	if err != nil {
		slog.Warn("failed to connect to proxy", "name", proxy.Name, "server", proxy.Server, "error", err)
		return CheckResult{
			LastCheck: time.Now(),
			Error:     fmt.Sprintf("connect failed: %v", err),
		}
	}
	defer func() { _ = client.Close() }()

	alive, latency, body, err := CheckViaProxy(client, pc.checkURL, pc.timeout)
	result := CheckResult{
		Alive:     alive,
		Latency:   latency,
		LastCheck: time.Now(),
	}
	if err != nil {
		result.Error = err.Error()
		slog.Warn("proxy check failed", "name", proxy.Name, "server", proxy.Server, "error", err)
	} else if pc.checkMethod == "ip" && pc.hostIP != "" {
		// For IP check: verify the proxy's exit IP differs from the host's IP
		if net.ParseIP(body) == nil {
			result.Alive = false
			result.Error = fmt.Sprintf("proxy returned non-IP response: %q", body)
			slog.Warn("proxy returned non-IP response", "name", proxy.Name, "server", proxy.Server, "body", body)
		} else if net.ParseIP(body).Equal(net.ParseIP(pc.hostIP)) {
			result.Alive = false
			result.Error = "proxy exit IP matches host IP"
			slog.Warn("proxy IP matches host", "name", proxy.Name, "server", proxy.Server, "ip", body)
		} else {
			slog.Info("proxy check passed", "name", proxy.Name, "server", proxy.Server, "latency", latency, "ip", body)
		}
	} else {
		slog.Info("proxy check passed", "name", proxy.Name, "server", proxy.Server, "latency", latency)
	}
	return result
}

// DetectHostIP fetches the host's public IP by making a direct HTTP GET to the given URL.
func DetectHostIP(checkURL string, timeout time.Duration) (string, error) {
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(checkURL)
	if err != nil {
		return "", fmt.Errorf("failed to detect host IP: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("host IP detection returned status %d from %s", resp.StatusCode, checkURL)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err != nil {
		return "", fmt.Errorf("failed to read host IP response: %w", err)
	}

	ip := strings.TrimSpace(string(body))
	if ip == "" {
		return "", fmt.Errorf("empty IP response from %s", checkURL)
	}
	if net.ParseIP(ip) == nil {
		return "", fmt.Errorf("invalid IP address %q from %s", ip, checkURL)
	}
	return ip, nil
}
