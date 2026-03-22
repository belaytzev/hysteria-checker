# Hysteria Checker

## What is Hysteria?

[Hysteria](https://hysteria.network/) is a censorship-resistant proxy protocol built on QUIC, designed for unreliable and lossy networks. It offers high-speed, encrypted tunneling with features like obfuscation and port hopping that help circumvent network restrictions.

## What is Hysteria Checker?

Hysteria Checker is a monitoring tool that continuously health-checks your Hysteria v1 and v2 proxy servers. Point it at a subscription URL (or individual proxy URIs) and it will periodically verify each proxy is alive, measure latency, and expose the results through a web dashboard, REST API, and Prometheus metrics.

Use it to get alerted when proxies go down, track performance over time, and integrate proxy health into your existing monitoring stack.

## Key Capabilities

- **Hysteria v1 and v2 support** — parses `hysteria://`, `hysteria2://`, and `hy2://` URIs with full parameter handling including obfuscation, port hopping, and TLS options
- **Subscription URL integration** — fetches and re-parses proxy lists automatically, picking up additions and removals
- **Concurrent health checks** — checks all proxies in parallel with configurable intervals and timeouts
- **Two check methods** — verify proxies via IP comparison (confirms traffic exits through the proxy) or HTTP status code
- **Web dashboard** — real-time status overview with auto-refresh, dark theme, and optional sensitive data redaction for public deployments
- **REST API** — JSON endpoints for proxy listing, individual proxy lookup, and aggregate status counts
- **Prometheus metrics** — `hysteria_proxy_status` and `hysteria_proxy_latency_ms` gauges with version, address, name, and ID labels
- **Basic Auth protection** — optional authentication for the dashboard, API, and metrics endpoints
- **Docker-ready** — ships with Dockerfile and Docker Compose configuration for quick deployment
- **Environment and CLI config** — every option configurable via environment variables or command-line flags

## Get Started

Ready to set up monitoring for your Hysteria proxies? Head to the [Getting Started](getting-started.md) guide for installation and configuration instructions.
