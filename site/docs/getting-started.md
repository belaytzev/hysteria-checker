# Getting Started

## Prerequisites

You need one of the following:

- **Docker** (recommended) — Docker Engine 20.10+ and Docker Compose v2
- **Go** — Go 1.25 or later (for building from source)

You also need at least one Hysteria subscription URL or direct proxy URI (`hysteria://`, `hysteria2://`, or `hy2://`).

## Installation

### Docker Compose (recommended)

1. Clone the repository:

    ```bash
    git clone https://github.com/belaytzev/hysteria-checker.git
    cd hysteria-checker
    ```

2. Create your configuration file:

    ```bash
    cp config.example.env .env
    ```

3. Edit `.env` and set your subscription URL:

    ```bash
    SUBSCRIPTION_URL=https://example.com/your-subscription
    ```

4. Start the service:

    ```bash
    docker compose up -d
    ```

The full `docker-compose.yml` for reference:

```yaml
services:
  hysteria-checker:
    build: .
    ports:
      - "2112:2112"
    env_file:
      - .env
    environment:
      - LOG_LEVEL=info
    healthcheck:
      test:
        [
          "CMD",
          "wget",
          "--no-verbose",
          "--tries=1",
          "--spider",
          "http://localhost:2112/healthz",
        ]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 10s
    restart: unless-stopped
```

### Docker standalone

1. Build the image:

    ```bash
    docker build -t hysteria-checker .
    ```

2. Run the container:

    ```bash
    docker run -d \
      --name hysteria-checker \
      -p 2112:2112 \
      -e SUBSCRIPTION_URL=https://example.com/your-subscription \
      -e LOG_LEVEL=info \
      --restart unless-stopped \
      hysteria-checker
    ```

### From source

1. Clone and build:

    ```bash
    git clone https://github.com/belaytzev/hysteria-checker.git
    cd hysteria-checker
    go build -o hysteria-checker .
    ```

2. Run with environment variables:

    ```bash
    SUBSCRIPTION_URL=https://example.com/your-subscription ./hysteria-checker
    ```

    Or with CLI flags:

    ```bash
    ./hysteria-checker --subscription-url https://example.com/your-subscription
    ```

## Verification

Once the service is running, verify it's working:

### Dashboard

Open [http://localhost:2112](http://localhost:2112) in your browser. You should see the web dashboard showing your proxy servers and their health status.

### API status endpoint

```bash
curl http://localhost:2112/api/v1/status
```

A healthy response returns JSON with proxy counts:

```json
{
  "total": 5,
  "up": 4,
  "down": 1
}
```

### Prometheus metrics

```bash
curl http://localhost:2112/metrics
```

You should see `hysteria_proxy_status` and `hysteria_proxy_latency_ms` gauges in the output.

### Docker health check

If using Docker Compose, check the container health:

```bash
docker compose ps
```

The `STATUS` column should show `healthy` after the initial 10-second start period.

## Troubleshooting

### Service starts but no proxies appear

- Verify your `SUBSCRIPTION_URL` is correct and accessible from the container
- Check logs: `docker compose logs -f` or look at stdout if running directly
- Ensure the subscription returns valid `hysteria://`, `hysteria2://`, or `hy2://` URIs

### Connection refused on port 2112

- Confirm the port mapping in your `docker-compose.yml` or `docker run` command
- Check if another service is using port 2112: `lsof -i :2112`
- Change the port with the `METRICS_PORT` environment variable

### All proxies show as down

- Increase the check timeout: `CHECK_TIMEOUT=60s`
- Verify the proxies work independently (e.g., with a Hysteria client)
- Check if your network blocks QUIC (UDP) traffic
- Try the status check method instead of IP: `CHECK_METHOD=status`

### Authentication issues

- If you enabled `METRICS_PROTECTED=true`, ensure `METRICS_USERNAME` and `METRICS_PASSWORD` are both set
- Basic Auth credentials are required for the API and metrics endpoints
- The dashboard is also protected when `METRICS_PROTECTED=true`
