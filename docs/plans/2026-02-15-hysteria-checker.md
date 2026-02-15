# Hysteria Checker

## Overview

Build a Go application that checks the health of Hysteria proxy servers (both v1 and v2), inspired by [xray-checker](https://github.com/kutovoys/xray-checker). The app reads proxy configurations from subscription URLs or direct share links, connects to each server using the Hysteria protocol directly via Go libraries, and exposes health status via Prometheus metrics, a REST API, and a web dashboard.

## Context

- **Reference project**: kutovoys/xray-checker (Go, embeds xray-core as library, SOCKS5-based checking)
- **Hysteria project**: apernet/hysteria (Go, QUIC-based proxy, v1 and v2 are incompatible protocols)
- **Key Go libraries**:
  - `github.com/apernet/hysteria/core/v2/client` — Hysteria v2 client (connects, returns `net.Conn` via `.TCP()`)
  - `github.com/apernet/hysteria/extras/v2` — Salamander obfuscation, port hopping
  - Hysteria v1: older module at `github.com/apernet/hysteria` (v1 branch/tag), or shell out to `hysteria` binary
- **Architecture decision**: Unlike xray-checker which runs an in-process xray-core with SOCKS5 inbounds, we connect directly to each Hysteria server using the hysteria client library and make a test TCP connection (e.g., HTTP request to an IP-check service) through each server. No local SOCKS5 ports needed.

## Development Approach

- **Language**: Go
- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to the next
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**
- Docker-first deployment with multi-stage build

## URI Formats

### Hysteria v1
```
hysteria://host:port?protocol=udp&auth=123456&peer=sni.domain&insecure=1&upmbps=100&downmbps=100&alpn=hysteria&obfs=xplus&obfsParam=123456#remarks
```

### Hysteria v2
```
hysteria2://[auth@]hostname[:port]/?insecure=1&obfs=salamander&obfs-password=gawrgura&pinSHA256=deadbeef&sni=real.example.com
```
Also valid: `hy2://...`

## Implementation Steps

### Task 1: Project Scaffold and Configuration

**Files:**
- Create: `go.mod`, `go.sum`
- Create: `main.go`
- Create: `config/config.go`
- Create: `Dockerfile`
- Create: `docker-compose.yml`

- [x] Initialize Go module (`github.com/belaytzev/hysteria-checker`)
- [x] Define config struct using `kong` for env vars / CLI flags:
  - `SUBSCRIPTION_URL` (comma-separated or repeated, supports URLs and direct `hysteria://` / `hysteria2://` / `hy2://` links)
  - `CHECK_INTERVAL` (default: 300s)
  - `CHECK_METHOD` (default: `ip`; options: `ip`, `status`)
  - `CHECK_URL` (URL used for IP-based or status-based check)
  - `CHECK_TIMEOUT` (default: 30s)
  - `METRICS_HOST` (default: `0.0.0.0`)
  - `METRICS_PORT` (default: `2112`)
  - `METRICS_PROTECTED` / `METRICS_USERNAME` / `METRICS_PASSWORD`
  - `WEB_PUBLIC` (default: `false`)
  - `LOG_LEVEL` (default: `info`)
- [x] Create `main.go` skeleton: parse config, log startup
- [x] Create Dockerfile (multi-stage: golang builder + alpine runtime)
- [x] Create docker-compose.yml with basic service definition
- [x] Write tests for config parsing (defaults, env var overrides)
- [x] Run `go test ./...` — must pass before task 2

### Task 2: URI Parsing (Hysteria v1 and v2 share links)

**Files:**
- Create: `models/proxy.go`
- Create: `parser/parser.go`
- Create: `parser/hysteria1.go`
- Create: `parser/hysteria2.go`

- [x] Define `ProxyConfig` model struct:
  ```
  type ProxyConfig struct {
      Version    int    // 1 or 2
      Name       string // from URI fragment (#remarks)
      Server     string // host:port
      Auth       string
      SNI        string
      Insecure   bool
      Obfs       string // obfuscation type (xplus for v1, salamander for v2)
      ObfsParam  string // obfuscation password
      UpMbps     int    // v1 required, v2 optional
      DownMbps   int    // v1 required, v2 optional
      ALPN       string // v1 only
      Protocol   string // v1 only (udp, wechat-video, faketcp)
      PinSHA256  string // v2 only
      StableID   string // deterministic hash for metrics labels
  }
  ```
- [x] Implement `hysteria://` URI parser (v1)
- [x] Implement `hysteria2://` and `hy2://` URI parser (v2)
- [x] Implement `ParseLinks(input string) ([]ProxyConfig, error)` — splits by newline, parses each link
- [x] Generate `StableID` as SHA256 hash of `server+auth+version` (for stable metric labels)
- [x] Write comprehensive tests: valid URIs, edge cases (missing port, special chars, percent-encoding), invalid URIs
- [x] Run `go test ./...` — must pass before task 3

### Task 3: Subscription Fetching

**Files:**
- Create: `subscription/subscription.go`

- [x] Implement `FetchSubscription(url string) ([]ProxyConfig, error)`:
  - HTTP GET the URL with timeout
  - Try base64 decode the response body
  - Split into lines and parse each as a share link via Task 2 parsers
- [x] Support multiple subscription URLs (aggregate results, deduplicate by StableID)
- [x] Support direct share links in the subscription URL list (detect by `hysteria://` or `hysteria2://` or `hy2://` prefix)
- [x] Write tests with httptest server returning base64-encoded link lists
- [x] Run `go test ./...` — must pass before task 4

### Task 4: Hysteria v2 Client Connector

**Files:**
- Create: `checker/connector.go`
- Create: `checker/hysteria2.go`

- [x] Add dependency: `github.com/apernet/hysteria/core/v2`
- [x] Implement `ConnectHysteria2(cfg ProxyConfig) (client.Client, error)`:
  - Resolve server address
  - Build `client.Config` with TLS settings (SNI, insecure, pinSHA256)
  - Handle Salamander obfuscation via extras package if `Obfs == "salamander"`
  - Set bandwidth config if provided
  - Call `client.NewClient(cfg)` — this performs the QUIC handshake and HTTP/3 auth
- [x] Implement `CheckViaHysteria2(c client.Client, checkURL string) (alive bool, latency time.Duration, err error)`:
  - Open TCP connection via `c.TCP("host:port")` to the check URL's host
  - Send HTTP request through the connection
  - Measure round-trip latency
  - For IP check: compare returned IP to host's own IP
  - For status check: verify HTTP status code is 2xx
- [x] Write tests with mocked client interface
- [x] Run `go test ./...` — must pass before task 5

### Task 5: Hysteria v1 Client Connector

**Files:**
- Create: `checker/hysteria1.go`

- [x] Research hysteria v1 Go library availability (older `github.com/apernet/hysteria` module)
- [x] If v1 Go library is usable: implement `ConnectHysteria1(cfg ProxyConfig) (net.Conn, error)` similar to v2
- [x] If v1 Go library is not easily importable: implement v1 checking by shelling out to `hysteria` binary (v1) with a temp config file, starting a local SOCKS5 proxy, checking through it, then killing the process
- [x] Implement `CheckViaHysteria1(...)` with same IP/status check logic
- [x] Write tests
- [x] Run `go test ./...` — must pass before task 6

### Task 6: Proxy Checker Orchestration

**Files:**
- Create: `checker/checker.go`

- [x] Implement `ProxyChecker` struct:
  - Holds list of `ProxyConfig`
  - Stores last check results (status, latency) per proxy
  - Thread-safe with `sync.RWMutex`
- [x] Implement `CheckAll()` — iterates all proxies, checks each concurrently (with semaphore to limit concurrency)
- [x] Dispatch to v1 or v2 connector based on `ProxyConfig.Version`
- [x] Detect host's own public IP at startup (for IP-based check method)
- [x] Store results: `map[string]CheckResult` keyed by StableID
  ```
  type CheckResult struct {
      Alive     bool
      Latency   time.Duration
      LastCheck time.Time
      Error     string
  }
  ```
- [x] Write tests with mock connectors
- [x] Run `go test ./...` — must pass before task 7

### Task 7: Prometheus Metrics

**Files:**
- Create: `metrics/metrics.go`

- [x] Define Prometheus gauges:
  - `hysteria_proxy_status` (labels: `version`, `address`, `name`, `sub_name`) — 1=up, 0=down
  - `hysteria_proxy_latency_ms` (same labels) — latency in ms, 0 if down
- [x] Implement `UpdateMetrics(results map[string]CheckResult, proxies []ProxyConfig)`
- [x] Register metrics and expose via `promhttp.Handler()` at `/metrics`
- [x] Optional Basic Auth protection for `/metrics`
- [x] Write tests verifying metric values after updates
- [x] Run `go test ./...` — must pass before task 8

### Task 8: REST API

**Files:**
- Create: `web/api.go`
- Create: `web/router.go`

- [x] Implement REST API endpoints:
  - `GET /api/v1/proxies` — list all proxies with status, latency, version
  - `GET /api/v1/proxies/{id}` — single proxy by StableID
  - `GET /api/v1/status` — summary (total, up, down counts)
- [x] JSON response format
- [x] Optional Basic Auth
- [x] Write tests with httptest
- [x] Run `go test ./...` — must pass before task 9

### Task 9: Web Dashboard

**Files:**
- Create: `web/dashboard.go`
- Create: `web/templates/index.html`

- [ ] Implement HTML dashboard at `GET /`:
  - Table showing: proxy name, version (hy1/hy2), server address, status (up/down), latency
  - Auto-refresh every check interval
  - Dark/light theme support
  - Responsive design
- [ ] Use Go's `html/template` with embedded templates (`embed` package)
- [ ] Optionally hide server details when `WEB_PUBLIC=true` and not authenticated
- [ ] Write tests for dashboard handler (status code, content type)
- [ ] Run `go test ./...` — must pass before task 10

### Task 10: Scheduler and Main Wiring

**Files:**
- Modify: `main.go`

- [ ] Wire everything together in `main.go`:
  1. Parse config
  2. Fetch subscriptions and parse proxy configs
  3. Initialize ProxyChecker
  4. Run initial check
  5. Set up HTTP server with metrics, API, and dashboard routes
  6. Schedule periodic checks using `gocron` or `time.Ticker`
  7. Schedule periodic subscription refresh (re-fetch and update proxy list)
  8. Handle graceful shutdown (SIGTERM/SIGINT)
- [ ] Write integration-style test for startup sequence
- [ ] Run `go test ./...` — must pass before task 11

### Task 11: Docker and Deployment

**Files:**
- Modify: `Dockerfile`
- Modify: `docker-compose.yml`
- Create: `.dockerignore`

- [ ] Finalize multi-stage Dockerfile:
  - Stage 1: `golang:1.24-alpine`, build with `-ldflags="-s -w"`, optionally UPX compress
  - Stage 2: `alpine`, install `ca-certificates` and `tzdata`, non-root user
- [ ] Add health check in docker-compose (`curl localhost:2112/api/v1/status`)
- [ ] Create `.dockerignore` (exclude `.git`, `docs/`, `*.md`)
- [ ] Test Docker build locally
- [ ] Run `go test ./...` — must pass before task 12

### Task 12: Verify Acceptance Criteria

- [ ] Manual test: add a real hysteria2 share link, verify it shows up in dashboard and metrics
- [ ] Manual test: add an invalid/offline server, verify it shows as down
- [ ] Run full test suite: `go test ./...`
- [ ] Run linter: `golangci-lint run`
- [ ] Verify test coverage meets 80%+

### Task 13: Update Documentation

- [ ] Write README.md: project description, features, configuration reference, Docker usage, screenshots
- [ ] Update CLAUDE.md if internal patterns changed
- [ ] Move this plan to `docs/plans/completed/`
