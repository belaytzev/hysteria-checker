# Prometheus Metrics

Hysteria Checker exposes Prometheus-compatible metrics at the `/metrics` endpoint, giving you real-time visibility into proxy health and performance.

## Available Metrics

| Metric | Type | Description | Labels |
|--------|------|-------------|--------|
| `hysteria_proxy_status` | Gauge | Proxy status: `1` = up, `0` = down | `version`, `address`, `name`, `id` |
| `hysteria_proxy_latency_ms` | Gauge | Proxy latency in milliseconds (`0` if down) | `version`, `address`, `name`, `id` |

### Label Descriptions

| Label | Description | Example |
|-------|-------------|---------|
| `version` | Hysteria protocol version | `hy1`, `hy2` |
| `address` | Server address (host:port) | `example.com:443` |
| `name` | Proxy display name from the subscription | `US-Server-1` |
| `id` | Stable unique identifier for the proxy | `a1b2c3d4` |

!!! note
    Metrics are reset and re-populated on each check cycle. Proxies removed from the subscription will no longer appear in metrics after the next check.

## Scrape Configuration

Add hysteria-checker as a Prometheus scrape target:

```yaml
scrape_configs:
  - job_name: hysteria-checker
    scrape_interval: 30s
    static_configs:
      - targets: ["localhost:2112"]
```

If you have authentication enabled (`METRICS_PROTECTED=true`), add basic auth:

```yaml
scrape_configs:
  - job_name: hysteria-checker
    scrape_interval: 30s
    basic_auth:
      username: admin
      password: your-password
    static_configs:
      - targets: ["localhost:2112"]
```

## PromQL Query Examples

### Count of down proxies

```promql
count(hysteria_proxy_status == 0)
```

### Count of up proxies

```promql
count(hysteria_proxy_status == 1)
```

### Availability percentage

```promql
count(hysteria_proxy_status == 1) / count(hysteria_proxy_status) * 100
```

### Average latency of healthy proxies

```promql
avg(hysteria_proxy_latency_ms > 0)
```

### Highest latency proxy

```promql
topk(1, hysteria_proxy_latency_ms)
```

### Proxies with latency above threshold (e.g., 500ms)

```promql
hysteria_proxy_latency_ms > 500
```

### Down proxies by version

```promql
count by (version) (hysteria_proxy_status == 0)
```

## Grafana Integration

### Adding the Data Source

1. Go to **Configuration > Data Sources** in Grafana
2. Click **Add data source** and select **Prometheus**
3. Set the URL to your Prometheus server (e.g., `http://localhost:9090`)
4. Click **Save & Test**

### Suggested Dashboard Panels

**Proxy Status Overview** (Stat panel):

- Query: `count(hysteria_proxy_status == 1)` — show total healthy proxies
- Add a second query: `count(hysteria_proxy_status == 0)` — show total down proxies
- Use value mappings or thresholds to color green/red

**Latency Heatmap** (Time series panel):

- Query: `hysteria_proxy_latency_ms > 0`
- Legend: `{{ "{{name}}" }}` to display proxy names
- Set unit to **milliseconds**

**Availability Over Time** (Gauge panel):

- Query: `count(hysteria_proxy_status == 1) / count(hysteria_proxy_status) * 100`
- Set unit to **percent (0-100)**
- Add thresholds: green > 90, yellow > 70, red below

**Status Table** (Table panel):

- Query: `hysteria_proxy_status`
- Format as table, show `name`, `address`, `version`, and `Value` columns
- Use value mappings: `1` → "Up", `0` → "Down"

## Sample Alerting Rules

Use these Prometheus alerting rules to get notified about proxy issues:

```yaml
groups:
  - name: hysteria-checker
    rules:
      - alert: HysteriaProxyDown
        expr: hysteria_proxy_status == 0
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Hysteria proxy {{ $labels.name }} is down"
          description: "Proxy {{ $labels.name }} ({{ $labels.address }}) has been down for more than 5 minutes."

      - alert: HysteriaAllProxiesDown
        expr: count(hysteria_proxy_status == 1) == 0
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "All Hysteria proxies are down"
          description: "No healthy proxies detected for more than 2 minutes."

      - alert: HysteriaHighLatency
        expr: hysteria_proxy_latency_ms > 1000
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "High latency on proxy {{ $labels.name }}"
          description: "Proxy {{ $labels.name }} ({{ $labels.address }}) latency is {{ $value }}ms (above 1000ms for 10+ minutes)."

      - alert: HysteriaLowAvailability
        expr: (count(hysteria_proxy_status == 1) / count(hysteria_proxy_status)) < 0.5
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Less than 50% of Hysteria proxies are healthy"
          description: "Only {{ $value | humanizePercentage }} of proxies are currently up."
```
