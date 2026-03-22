# API Reference

Hysteria Checker exposes a JSON REST API for programmatic access to proxy status and health check results. All API endpoints are under the `/api/v1/` prefix.

## Base URL

```
http://<host>:<port>/api/v1
```

The default server address is `0.0.0.0:2112`. Configure with `METRICS_HOST` and `METRICS_PORT`.

## Authentication

API endpoints use HTTP Basic Authentication when `METRICS_PROTECTED=true`.

### Auth Matrix

| Endpoint | `METRICS_PROTECTED=false` | `METRICS_PROTECTED=true` |
|---|---|---|
| Dashboard (`/`) | Public | Basic Auth required |
| API (`/api/v1/*`) | Public | Basic Auth required |
| Metrics (`/metrics`) | Public | Basic Auth required |
| Health check (`/healthz`) | Public | Public |

When `WEB_PUBLIC=true`, sensitive information (server addresses, error details) is redacted in both the dashboard and API responses, but authentication is still required when `METRICS_PROTECTED=true`.

When authentication is required, include credentials in your request:

```bash
curl -u username:password http://localhost:2112/api/v1/status
```

Unauthenticated requests to protected endpoints return `401 Unauthorized`:

```http
HTTP/1.1 401 Unauthorized
WWW-Authenticate: Basic realm="restricted"

Unauthorized
```

---

## Endpoints

### GET /api/v1/status

Returns an overall health summary with proxy counts.

**Example Request:**

```bash
curl http://localhost:2112/api/v1/status
```

**Response: `200 OK`**

```json
{
  "total": 5,
  "up": 3,
  "down": 2
}
```

| Field | Type | Description |
|---|---|---|
| `total` | integer | Total number of configured proxies |
| `up` | integer | Number of proxies that passed the last health check |
| `down` | integer | Number of proxies that failed the last health check |

---

### GET /api/v1/proxies

Returns a list of all proxies with their latest check results.

**Example Request:**

```bash
curl http://localhost:2112/api/v1/proxies
```

**Response: `200 OK`**

```json
[
  {
    "id": "a1b2c3d4",
    "name": "US-West-1",
    "server": "us-west.example.com:443",
    "version": "hy2",
    "alive": true,
    "latency_ms": 45.0,
    "last_check": "2026-03-22T10:30:00Z"
  },
  {
    "id": "e5f6g7h8",
    "name": "EU-Central",
    "server": "eu.example.com:443",
    "version": "hy1",
    "alive": false,
    "latency_ms": 0,
    "last_check": "2026-03-22T10:30:00Z",
    "error": "connection refused"
  }
]
```

| Field | Type | Description |
|---|---|---|
| `id` | string | Stable unique identifier for the proxy |
| `name` | string | Display name from the subscription |
| `server` | string | Server address and port (`"***"` when `WEB_PUBLIC=true`) |
| `version` | string | Hysteria protocol version (`"hy1"` or `"hy2"`) |
| `alive` | boolean | Whether the last health check succeeded |
| `latency_ms` | number | Round-trip latency in milliseconds (0 when not alive) |
| `last_check` | string | ISO 8601 timestamp of the last check (omitted if never checked) |
| `error` | string | Error message from the last check (omitted when alive; `"check failed"` when `WEB_PUBLIC=true`) |

---

### GET /api/v1/proxies/{id}

Returns a single proxy by its stable ID.

**Example Request:**

```bash
curl http://localhost:2112/api/v1/proxies/a1b2c3d4
```

**Response: `200 OK`**

```json
{
  "id": "a1b2c3d4",
  "name": "US-West-1",
  "server": "us-west.example.com:443",
  "version": "hy2",
  "alive": true,
  "latency_ms": 45.0,
  "last_check": "2026-03-22T10:30:00Z"
}
```

The response fields are the same as in the [list endpoint](#get-apiv1proxies).

---

## Error Responses

### 404 Not Found — Proxy not found

```bash
curl http://localhost:2112/api/v1/proxies/nonexistent
```

```json
{
  "error": "proxy not found"
}
```

### 400 Bad Request — Missing proxy ID

```bash
curl http://localhost:2112/api/v1/proxies/
```

```json
{
  "error": "missing proxy id"
}
```

### 405 Method Not Allowed

All API endpoints only accept `GET` requests. Other methods return:

```bash
curl -X POST http://localhost:2112/api/v1/status
```

```json
{
  "error": "Method Not Allowed"
}
```

### 401 Unauthorized

Returned when `METRICS_PROTECTED=true` and no valid credentials are provided:

```
HTTP/1.1 401 Unauthorized
WWW-Authenticate: Basic realm="restricted"

Unauthorized
```

---

## Other Endpoints

### GET /

The web dashboard. Returns an HTML page showing proxy status in a table with auto-refresh.

- **Not authenticated** when `METRICS_PROTECTED=false`
- **Basic Auth required** when `METRICS_PROTECTED=true`

### GET /healthz

Health probe endpoint. Always returns `200 OK` with body `ok`. Never requires authentication, even when `METRICS_PROTECTED=true`. Use this for Docker healthchecks, Kubernetes liveness/readiness probes, and load balancer health checks.

### GET /metrics

Prometheus metrics endpoint. Protected with Basic Auth when `METRICS_PROTECTED=true`. See the Metrics page for details on available metrics.
