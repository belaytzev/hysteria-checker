# Hysteria Checker

## Overview
Build a Go application that monitors Hysteria proxy servers (v1 and v2) by connecting through them and verifying connectivity. Modeled after xray-checker but using the native Hysteria Go client library instead of Xray Core. Exposes Prometheus metrics, a REST API, and a web dashboard.

## Context
- Fresh Go project, no existing code
- Reference: github.com/kutovoys/xray-checker (architecture model)
- Core dependency: github.com/apernet/hysteria/core/v2/client (Hysteria v2 Go client)
- Hysteria v1 has no maintained Go client library - will use QUIC-based custom connection
- Subscription format: newline-separated or base64-encoded lists of hysteria:// and hysteria2:// URIs

## Architecture Overview

```
hysteria-checker/
├── main.go                    # Entry point, scheduler setup
├── config/
│   └── config.go              # CLI flags + env vars (Kong library)
├── models/
│   └── proxy.go               # ProxyConfig struct for both v1 and v2
├── subscription/
│   ├── subscription.go        # Multi-source loader (HTTP, file, base64)
│   └── parser.go              # Parse hysteria:// and hysteria2:// URIs
├── checker/
│   └── checker.go             # ProxyChecker: connect via hysteria, test HTTP
├── hysteria/
│   ├── client_v2.go           # Hysteria v2 client wrapper using core/v2
│   └── client_v1.go           # Hysteria v1 client wrapper
├── metrics/
│   └── metrics.go             # Prometheus GaugeVec + optional Pushgateway
├── web/
│   ├── api.go                 # REST API handlers
│   ├── handlers.go            # Dashboard handler
│   ├── static/                # CSS, JS
│   └── templates/             # HTML templates
├── logger/
│   └── logger.go              # Structured logging (slog)
├── Dockerfile
├── go.mod
└── .goreleaser.yaml
```

### Key Design Decisions
1. **Native Hysteria client** - Use `github.com/apernet/hysteria/core/v2/client` directly for v2. For v1, implement a minimal QUIC-based client or shell out to the hysteria binary.
2. **SOCKS5 proxy per connection** - Each Hysteria client establishes a tunnel; checker routes HTTP through client.TCP() to verify connectivity.
3. **No embedded binary** - Unlike xray-checker which embeds Xray Core, we use the Hysteria Go library as a dependency.
4. **Stable IDs** - Hash-based proxy identifiers for consistent external monitoring integration.

## Development Approach
- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to the next
- Use Go standard library where possible (slog for logging, net/http for server)
- Follow xray-checker patterns for metrics, API, and web structure
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Implementation Steps

### Task 1: Project Bootstrap and Configuration

**Files:**
- Create: `go.mod`
- Create: `main.go`
- Create: `config/config.go`

- [ ] Initialize Go module: `go mod init github.com/belaytzev/hysteria-checker`
- [ ] Define config struct with Kong tags for CLI/env parsing:
  - `SUBSCRIPTION_URL` (repeatable, required)
  - `SUBSCRIPTION_UPDATE` (bool, default: true)
  - `SUBSCRIPTION_UPDATE_INTERVAL` (seconds, default: 300)
  - `PROXY_CHECK_INTERVAL` (seconds, default: 300)
  - `PROXY_CHECK_METHOD` (ip/status, default: ip)
  - `PROXY_IP_CHECK_URL` (default: https://api.ipify.org?format=text)
  - `PROXY_STATUS_CHECK_URL` (default: http://cp.cloudflare.com/generate_204)
  - `PROXY_TIMEOUT` (seconds, default: 30)
  - `METRICS_HOST` (default: 0.0.0.0)
  - `METRICS_PORT` (default: 2112)
  - `METRICS_PROTECTED` (bool, default: false)
  - `METRICS_USERNAME` / `METRICS_PASSWORD`
  - `METRICS_PUSH_URL` (Pushgateway URL)
  - `METRICS_BASE_PATH` (URL prefix)
  - `WEB_SHOW_DETAILS` (bool, default: false)
  - `LOG_LEVEL` (debug/info/warn/error, default: info)
  - `RUN_ONCE` (bool, default: false)
  - `HYSTERIA_DEFAULT_UP_MBPS` (default: 100, for v1 bandwidth)
  - `HYSTERIA_DEFAULT_DOWN_MBPS` (default: 100, for v1 bandwidth)
- [ ] Implement main.go with config loading and placeholder scheduler
- [ ] Write tests for config parsing (defaults, env var overrides)
- [ ] Run `go test ./...` - must pass before task 2

### Task 2: Models and URI Parsing

**Files:**
- Create: `models/proxy.go`
- Create: `subscription/parser.go`
- Create: `subscription/parser_test.go`

- [ ] Define ProxyConfig struct:
  - Common: Name, Protocol (hysteria/hysteria2), Server, Port, StableID, SubName
  - V2: Auth (password), SNI, Insecure, Obfs (salamander), ObfsPassword, PinSHA256
  - V1: Auth, Peer (SNI), Insecure, UpMbps, DownMbps, ALPN, Obfs (xplus), ObfsParam, Protocol (udp/wechat-video/faketcp)
  - Status: Alive (bool), Latency (time.Duration), IP (string), LastCheck (time.Time)
- [ ] Implement StableID generation: sha256 hash of protocol+server+port+name+subname, truncated to 16 chars
- [ ] Implement `ParseHysteria2URI(uri string) (*ProxyConfig, error)`:
  - Parse `hysteria2://[auth@]host[:port]/?params#name`
  - Handle hy2:// alias
  - Extract: auth, host, port (default 443), sni, insecure, obfs, obfs-password, pinSHA256
  - Handle port ranges in host (e.g., 5000-6000) - pick first port
- [ ] Implement `ParseHysteria1URI(uri string) (*ProxyConfig, error)`:
  - Parse `hysteria://host:port?params#name`
  - Extract: auth, peer, insecure, upmbps, downmbps, alpn, obfs, obfsParam, protocol
- [ ] Write comprehensive tests for URI parsing (valid URIs, edge cases, malformed input)
- [ ] Run `go test ./...` - must pass before task 3

### Task 3: Subscription Loader

**Files:**
- Create: `subscription/subscription.go`
- Create: `subscription/subscription_test.go`

- [ ] Implement `LoadSubscriptions(urls []string) ([]*ProxyConfig, error)`:
  - Fetch each URL concurrently (HTTP GET with timeout)
  - Support `file://` prefix for local files
  - Auto-detect content format: base64-encoded vs plain text
  - Decode base64 if needed, split by newlines
  - Parse each line as hysteria:// or hysteria2:// URI
  - Extract SubName from URL fragment or `profile-title` response header
  - Deduplicate by StableID
- [ ] Write tests with mock HTTP server for subscription fetching
- [ ] Run `go test ./...` - must pass before task 4

### Task 4: Hysteria v2 Client Wrapper

**Files:**
- Create: `hysteria/client_v2.go`
- Create: `hysteria/client_v2_test.go`

- [ ] Implement `NewV2Client(cfg *ProxyConfig) (Client, error)` wrapper:
  - Use `github.com/apernet/hysteria/core/v2/client.NewClient`
  - Map ProxyConfig fields to hysteria Config struct
  - Handle TLS config (SNI, InsecureSkipVerify, PinSHA256)
  - Handle obfuscation (Salamander)
  - Return a Client interface: `TCP(addr string) (net.Conn, error)`, `Close() error`
- [ ] Define common `Client` interface in `hysteria/client.go` for both v1 and v2
- [ ] Write tests (at minimum: config mapping, interface compliance)
- [ ] Run `go test ./...` - must pass before task 5

### Task 5: Hysteria v1 Client Wrapper

**Files:**
- Create: `hysteria/client_v1.go`
- Create: `hysteria/client_v1_test.go`
- Create: `hysteria/client.go` (if not already from task 4)

- [ ] Research v1 Go library availability in the hysteria repo (check `core/client` package for v1 tag)
- [ ] If Go library available: wrap it similar to v2
- [ ] If no library: implement minimal v1 client using `quic-go` directly:
  - QUIC dial with custom ALPN
  - Auth handshake
  - TCP relay request
  - OR: document v1 as experimental/limited and focus v2 as primary
- [ ] Write tests for v1 client
- [ ] Run `go test ./...` - must pass before task 6

### Task 6: Proxy Checker

**Files:**
- Create: `checker/checker.go`
- Create: `checker/checker_test.go`

- [ ] Implement `ProxyChecker` struct:
  - Holds list of `*ProxyConfig`, check method config, generation counter
  - `CheckAllProxies()`: iterate all proxies concurrently with WaitGroup
  - `checkSingle(proxy *ProxyConfig)`: create hysteria client, make HTTP request through tunnel, measure TTFB
- [ ] Implement check methods:
  - `ip`: GET ipify.org through tunnel, compare to host IP
  - `status`: GET cloudflare generate_204 through tunnel, check 2xx
- [ ] Use `net/http/httptrace` for TTFB measurement
- [ ] Implement generation counter for stale result handling
- [ ] Write tests with mock interfaces
- [ ] Run `go test ./...` - must pass before task 7

### Task 7: Prometheus Metrics

**Files:**
- Create: `metrics/metrics.go`
- Create: `metrics/metrics_test.go`

- [ ] Define Prometheus GaugeVec metrics:
  - `hysteria_proxy_status` (labels: protocol, address, name, sub_name) - 1=up, 0=down
  - `hysteria_proxy_latency_ms` (same labels) - TTFB in ms, 0 if down
- [ ] Implement `UpdateMetrics(proxies []*ProxyConfig)` to set gauge values
- [ ] Implement optional Pushgateway push (POST text format to configured URL)
- [ ] Write tests for metric updates
- [ ] Run `go test ./...` - must pass before task 8

### Task 8: REST API

**Files:**
- Create: `web/api.go`
- Create: `web/api_test.go`

- [ ] Implement HTTP handlers:
  - `GET /health` - returns 200 OK
  - `GET /metrics` - Prometheus text format (optional Basic Auth)
  - `GET /config/{stableID}` - 200 if proxy up, 503 if down (Uptime Kuma compatible)
  - `GET /api/v1/proxies` - full proxy list (Basic Auth)
  - `GET /api/v1/proxies/{stableID}` - single proxy details
  - `GET /api/v1/public/proxies` - proxy list without sensitive details
  - `GET /api/v1/status` - aggregate: total/online/offline/avgLatency
  - `GET /api/v1/config` - current checker config
  - `GET /api/v1/system/info` - version, uptime
- [ ] Implement Basic Auth middleware (when METRICS_PROTECTED=true)
- [ ] Write tests for all endpoints
- [ ] Run `go test ./...` - must pass before task 9

### Task 9: Web Dashboard

**Files:**
- Create: `web/handlers.go`
- Create: `web/static/style.css`
- Create: `web/static/app.js`
- Create: `web/templates/index.html`

- [ ] Implement dashboard handler serving embedded HTML
- [ ] Create HTML template showing:
  - Proxy list grouped by subscription name
  - Status (up/down), latency, protocol version, server (if WEB_SHOW_DETAILS)
  - Auto-refresh via JS fetch to /api/v1/public/proxies
  - Dark/light theme toggle
- [ ] Embed static files using Go embed directive
- [ ] Write handler tests
- [ ] Run `go test ./...` - must pass before task 10

### Task 10: Main Application Wiring and Scheduler

**Files:**
- Modify: `main.go`

- [ ] Wire everything together in main.go:
  - Parse config
  - Load initial subscriptions
  - Start proxy checker on interval (using gocron or ticker)
  - Start subscription updater on interval
  - Start HTTP server with all routes
  - Handle graceful shutdown (os.Signal)
- [ ] Implement subscription update logic:
  - Re-fetch subscriptions
  - Diff against current config
  - If changed: update proxy list, increment generation counter
- [ ] Implement RUN_ONCE mode (single check, print results, exit)
- [ ] Write integration test for startup/shutdown
- [ ] Run `go test ./...` - must pass before task 11

### Task 11: Docker and Release

**Files:**
- Create: `Dockerfile`
- Create: `.goreleaser.yaml`
- Update: `.gitignore` (replace Python gitignore with Go gitignore)

- [ ] Create multi-stage Dockerfile:
  - Build stage: golang:1.23-alpine, CGO_ENABLED=0
  - Runtime stage: alpine:3.21, ca-certificates, tzdata, non-root user
- [ ] Create .goreleaser.yaml for multi-platform binary releases
- [ ] Replace .gitignore with Go-appropriate entries
- [ ] Test Docker build: `docker build -t hysteria-checker .`
- [ ] Run `go test ./...` - must pass before task 12

### Task 12: Verify Acceptance Criteria

- [ ] Manual test: run with a real hysteria2:// subscription URL and verify proxies are checked
- [ ] Manual test: verify Prometheus metrics at /metrics show correct proxy_status and latency
- [ ] Manual test: verify web dashboard loads and shows proxy statuses
- [ ] Manual test: verify /config/{stableID} returns correct status codes
- [ ] Run full test suite: `go test ./...`
- [ ] Run linter: `golangci-lint run`
- [ ] Verify test coverage meets 80%+

### Task 13: Update Documentation

- [ ] Create README.md with:
  - Project description
  - Quick start (Docker and binary)
  - Configuration reference (all env vars)
  - API documentation
  - Prometheus metrics reference
  - Comparison with xray-checker
- [ ] Move this plan to `docs/plans/completed/`
