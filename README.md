# Hysteria Checker

Health checker for [Hysteria](https://github.com/apernet/hysteria) proxy servers (v1 and v2). Reads proxy configurations from subscription URLs or direct share links, checks connectivity through each server, and exposes results via Prometheus metrics, a REST API, and a web dashboard.

Inspired by [xray-checker](https://github.com/kutovoys/xray-checker).

## Features

- Hysteria v1 (`hysteria://`) and v2 (`hysteria2://`, `hy2://`) protocol support
- Subscription URL fetching with base64 decoding
- Direct share link input alongside subscription URLs
- Two check methods: IP verification and HTTP status check
- Prometheus metrics with per-proxy status and latency gauges
- REST API for programmatic access
- Web dashboard with auto-refresh, dark/light theme toggle, responsive design
- Optional Basic Auth protection for metrics and API
- Public mode to hide sensitive server details on the dashboard
- Periodic health checks and subscription refresh
- Graceful shutdown on SIGTERM/SIGINT
- Docker-first deployment

## Quick Start

### Docker Compose (recommended)

Create a `docker-compose.yml`:

```yaml
services:
  hysteria-checker:
    image: ghcr.io/belaytzev/hysteria-checker:latest
    # Or build locally:
    # build: .
    ports:
      - "2112:2112"
    environment:
      - SUBSCRIPTION_URL=https://example.com/subscription
      - CHECK_INTERVAL=300s
      - CHECK_METHOD=ip
      - LOG_LEVEL=info
    restart: unless-stopped
```

```sh
docker compose up -d
```

### Docker

```sh
docker build -t hysteria-checker .
docker run -d \
  -p 2112:2112 \
  -e SUBSCRIPTION_URL="https://example.com/sub" \
  hysteria-checker
```

### From Source

Requires Go 1.25+.

```sh
go build -o hysteria-checker .
./hysteria-checker --subscription-url "hysteria2://auth@host:443/?insecure=1&sni=example.com"
```

## Configuration

All options can be set via CLI flags or environment variables.

| Environment Variable | CLI Flag | Default | Description |
|---|---|---|---|
| `SUBSCRIPTION_URL` | `--subscription-url` | _(required)_ | Subscription URLs or direct `hysteria://`/`hysteria2://`/`hy2://` links. Comma-separated or repeated. |
| `CHECK_INTERVAL` | `--check-interval` | `300s` | Interval between health checks. |
| `CHECK_METHOD` | `--check-method` | `ip` | Check method: `ip` (verify proxy changes IP) or `status` (verify HTTP 2xx). |
| `CHECK_URL` | `--check-url` | `https://api.ipify.org` | URL used for the check request through each proxy. |
| `CHECK_TIMEOUT` | `--check-timeout` | `30s` | Timeout for each individual proxy check. |
| `METRICS_HOST` | `--metrics-host` | `0.0.0.0` | Host to bind the HTTP server. |
| `METRICS_PORT` | `--metrics-port` | `2112` | Port for the HTTP server. |
| `METRICS_PROTECTED` | `--metrics-protected` | `false` | Enable Basic Auth for metrics/API endpoints. |
| `METRICS_USERNAME` | `--metrics-username` | _(empty)_ | Basic Auth username. |
| `METRICS_PASSWORD` | `--metrics-password` | _(empty)_ | Basic Auth password. |
| `WEB_PUBLIC` | `--web-public` | `false` | Hide sensitive server details on dashboard for unauthenticated users. |
| `LOG_LEVEL` | `--log-level` | `info` | Log level: `debug`, `info`, `warn`, `error`. |

## Endpoints

| Path | Description |
|---|---|
| `/` | Web dashboard |
| `/metrics` | Prometheus metrics |
| `/api/v1/proxies` | List all proxies with status and latency |
| `/api/v1/proxies/{id}` | Single proxy by stable ID |
| `/api/v1/status` | Summary: total, up, and down counts |

## Prometheus Metrics

| Metric | Labels | Description |
|---|---|---|
| `hysteria_proxy_status` | `version`, `address`, `name`, `id` | Proxy status: 1 = up, 0 = down |
| `hysteria_proxy_latency_ms` | `version`, `address`, `name`, `id` | Latency in milliseconds (0 if down) |

## URI Formats

### Hysteria v1

```
hysteria://host:port?protocol=udp&auth=123456&peer=sni.domain&insecure=1&upmbps=100&downmbps=100&alpn=hysteria&obfs=xplus&obfsParam=123456#remarks
```

### Hysteria v2

```
hysteria2://[auth@]hostname[:port]/?insecure=1&obfs=salamander&obfs-password=gawrgura&pinSHA256=deadbeef&sni=real.example.com
```

The `hy2://` scheme is also supported.

## License

MIT
