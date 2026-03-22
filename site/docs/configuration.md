# Configuration

Hysteria Checker is configured through environment variables or CLI flags. Every option can be set either way — environment variables are recommended for Docker deployments, while CLI flags work well when running from source.

A complete example environment file is provided at `config.example.env` in the repository root.

## Subscription

| Env Var | CLI Flag | Type | Default | Description |
|---------|----------|------|---------|-------------|
| `SUBSCRIPTION_URL` | `--subscription-url` | string (repeatable) | *(none)* | Subscription URLs or direct `hysteria://` / `hysteria2://` / `hy2://` links. Comma-separated or specify multiple times. The application starts without this but monitors no proxies. |

**Example — single subscription:**

```bash
SUBSCRIPTION_URL=https://provider.example.com/sub/abc123
```

**Example — multiple sources:**

```bash
# Comma-separated
SUBSCRIPTION_URL=https://provider1.example.com/sub,hysteria2://server.example.com:443

# Or repeated (CLI)
./hysteria-checker \
  --subscription-url https://provider1.example.com/sub \
  --subscription-url hysteria2://server.example.com:443
```

## Check Settings

| Env Var | CLI Flag | Type | Default | Description |
|---------|----------|------|---------|-------------|
| `CHECK_INTERVAL` | `--check-interval` | duration | `300s` | Interval between health checks. Accepts Go duration strings like `5m`, `300s`, `1h`. |
| `CHECK_METHOD` | `--check-method` | string | `ip` | Check method. `ip` — connect through the proxy and verify an IP is returned. `status` — check HTTP status code through the proxy. |
| `CHECK_URL` | `--check-url` | string | `https://api.ipify.org` | URL used for both `ip` and `status` check methods. |
| `CHECK_TIMEOUT` | `--check-timeout` | duration | `30s` | Timeout for each individual proxy check. |

### Check methods explained

**`ip` method** (default): Connects through each proxy to the check URL and verifies that a valid IP address is returned, then confirms the exit IP differs from the host's own public IP. This ensures the proxy is actually routing traffic through a different endpoint. Best used with an IP echo service like the default `https://api.ipify.org`. If host IP detection fails at startup (e.g., the check URL is unreachable), the application automatically falls back to the `status` method and logs a warning.

**`status` method**: Connects through each proxy to the check URL and checks for a successful HTTP status code. Useful when you want to verify connectivity to a specific endpoint rather than general internet access.

**Example — faster checks with shorter interval:**

```bash
CHECK_INTERVAL=60s
CHECK_TIMEOUT=15s
CHECK_METHOD=ip
```

**Example — check access to a specific service:**

```bash
CHECK_METHOD=status
CHECK_URL=https://example.com/health
CHECK_TIMEOUT=10s
```

## Server / Metrics

| Env Var | CLI Flag | Type | Default | Description |
|---------|----------|------|---------|-------------|
| `METRICS_HOST` | `--metrics-host` | string | `0.0.0.0` | Host to bind the metrics/web server. |
| `METRICS_PORT` | `--metrics-port` | int | `2112` | Port for the metrics/web server (1–65535). |
| `WEB_PUBLIC` | `--web-public` | bool | `false` | If `true`, redact sensitive details (server addresses, errors) in the dashboard and API responses. Use this when the dashboard is publicly accessible. |

**Example — custom port with public-safe dashboard:**

```bash
METRICS_HOST=0.0.0.0
METRICS_PORT=8080
WEB_PUBLIC=true
```

## Authentication

| Env Var | CLI Flag | Type | Default | Description |
|---------|----------|------|---------|-------------|
| `METRICS_PROTECTED` | `--metrics-protected` | bool | `false` | Enable Basic Auth for API and metrics endpoints. |
| `METRICS_USERNAME` | `--metrics-username` | string | *(empty)* | Basic Auth username. Required when `METRICS_PROTECTED=true`. |
| `METRICS_PASSWORD` | `--metrics-password` | string | *(empty)* | Basic Auth password. Required when `METRICS_PROTECTED=true`. |

When `METRICS_PROTECTED` is enabled:

- **API endpoints** (`/api/v1/*`) require Basic Auth
- **Metrics endpoint** (`/metrics`) requires Basic Auth
- **Dashboard** (`/`) is also protected

When `WEB_PUBLIC=true`, sensitive information (server addresses, error details) is redacted in both the dashboard and API responses, but authentication is still required if `METRICS_PROTECTED=true`.

**Example — protected setup with redacted data:**

```bash
METRICS_PROTECTED=true
METRICS_USERNAME=admin
METRICS_PASSWORD=your-secure-password
WEB_PUBLIC=true
```

**Example — fully protected (private deployment):**

```bash
METRICS_PROTECTED=true
METRICS_USERNAME=admin
METRICS_PASSWORD=your-secure-password
WEB_PUBLIC=false
```

## Logging

| Env Var | CLI Flag | Type | Default | Description |
|---------|----------|------|---------|-------------|
| `LOG_LEVEL` | `--log-level` | string | `info` | Log level. One of: `debug`, `info`, `warn`, `error`. |

**Example:**

```bash
LOG_LEVEL=debug
```

## Complete Configuration Examples

### Minimal setup

```bash
SUBSCRIPTION_URL=https://provider.example.com/sub/abc123
```

Everything else uses sensible defaults: checks every 5 minutes using the IP method, serves on port 2112, no authentication.

### Production with authentication

```bash
SUBSCRIPTION_URL=https://provider.example.com/sub/abc123
CHECK_INTERVAL=120s
CHECK_TIMEOUT=15s
METRICS_PROTECTED=true
METRICS_USERNAME=admin
METRICS_PASSWORD=a-strong-random-password
WEB_PUBLIC=false
LOG_LEVEL=info
```

### Public monitoring dashboard

```bash
SUBSCRIPTION_URL=https://provider.example.com/sub/abc123
CHECK_INTERVAL=60s
METRICS_PROTECTED=true
METRICS_USERNAME=prometheus
METRICS_PASSWORD=scrape-password
WEB_PUBLIC=true
METRICS_PORT=8080
LOG_LEVEL=warn
```

The dashboard is publicly visible with sensitive details redacted. API and metrics endpoints are protected for Prometheus scraping.

### CLI flags equivalent

All environment variables have corresponding CLI flags:

```bash
./hysteria-checker \
  --subscription-url https://provider.example.com/sub/abc123 \
  --check-interval 120s \
  --check-timeout 15s \
  --check-method ip \
  --metrics-port 2112 \
  --metrics-protected \
  --metrics-username admin \
  --metrics-password secret \
  --web-public \
  --log-level info
```
