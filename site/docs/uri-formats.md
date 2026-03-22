# URI Formats

Hysteria Checker parses proxy share links in the Hysteria v1 and v2 URI formats. This page documents the full syntax for each version, including all supported parameters and port hopping notation.

## Supported Schemes

| Scheme | Version | Example |
|--------|---------|---------|
| `hysteria://` | Hysteria v1 | `hysteria://host:port?auth=...#name` |
| `hysteria2://` | Hysteria v2 | `hysteria2://auth@host:port/?sni=...#name` |
| `hy2://` | Hysteria v2 (alias) | `hy2://auth@host:port/?sni=...#name` |

The parser auto-detects the version from the URI scheme. Both `hysteria2://` and `hy2://` are treated identically.

---

## Hysteria v1

### Format

```
hysteria://host:port?protocol=<protocol>&auth=<auth>&peer=<sni>&insecure=<0|1>&upmbps=<n>&downmbps=<n>&alpn=<alpn>&obfs=<type>&obfsParam=<password>#<name>
```

### Parameters

| Parameter | Location | Required | Description |
|-----------|----------|----------|-------------|
| `host` | Authority | Yes | Server hostname or IP address |
| `port` | Authority | Yes | Server port number |
| `auth` | Query | No | Authentication credential |
| `peer` | Query | No | TLS SNI (Server Name Indication) |
| `insecure` | Query | No | Skip TLS verification (`1` or `true`) |
| `protocol` | Query | No | Transport protocol: `udp`, `wechat-video`, `faketcp` |
| `upmbps` | Query | No | Upload speed in Mbps (integer) |
| `downmbps` | Query | No | Download speed in Mbps (integer) |
| `alpn` | Query | No | TLS ALPN protocol (e.g. `hysteria`) |
| `obfs` | Query | No | Obfuscation type (e.g. `xplus`) |
| `obfsParam` | Query | No | Obfuscation password |
| `#fragment` | Fragment | No | Display name / remarks |

### Examples

**Minimal:**

```
hysteria://example.com:36712?auth=mypassword#MyServer
```

**Full parameters:**

```
hysteria://example.com:36712?protocol=udp&auth=secretkey&peer=sni.example.com&insecure=1&upmbps=100&downmbps=200&alpn=hysteria&obfs=xplus&obfsParam=obfskey#US-Server-1
```

---

## Hysteria v2

### Format

```
hysteria2://[auth@]host[:port][/?key=value&...]#name
```

Or equivalently with the `hy2://` alias:

```
hy2://[auth@]host[:port][/?key=value&...]#name
```

### Parameters

| Parameter | Location | Required | Description |
|-----------|----------|----------|-------------|
| `host` | Authority | Yes | Server hostname or IP address |
| `port` | Authority | No | Server port (defaults to `443`) |
| `auth` | Userinfo or Query | No | Authentication — placed before `@` in the authority, or as `?auth=` query parameter |
| `sni` | Query | No | TLS SNI (Server Name Indication) |
| `insecure` | Query | No | Skip TLS verification (`1` or `true`) |
| `pinSHA256` | Query | No | TLS certificate pin (SHA-256 hash) |
| `obfs` | Query | No | Obfuscation type (e.g. `salamander`) |
| `obfs-password` | Query | No | Obfuscation password |
| `upmbps` | Query | No | Upload speed in Mbps (integer) |
| `downmbps` | Query | No | Download speed in Mbps (integer) |
| `#fragment` | Fragment | No | Display name / remarks |

### Authentication Placement

Auth can appear in two positions. If both are present, the userinfo value takes precedence:

```
# Auth in userinfo (preferred)
hysteria2://mypassword@example.com:443/#Server1

# Auth as query parameter
hysteria2://example.com:443/?auth=mypassword#Server1

# Username:password form
hysteria2://user:pass@example.com:443/#Server1
```

### Default Port

If no port is specified, the parser defaults to `443`:

```
# These are equivalent:
hysteria2://auth@example.com/#Server1
hysteria2://auth@example.com:443/#Server1
```

### Examples

**Minimal:**

```
hysteria2://mypassword@example.com/#MyServer
```

**With obfuscation:**

```
hy2://mypassword@example.com:443/?obfs=salamander&obfs-password=gawrgura#JP-Server
```

**With certificate pinning:**

```
hysteria2://mypassword@example.com:443/?pinSHA256=deadbeef1234&insecure=0&sni=real.example.com#Pinned-Server
```

---

## Port Hopping

Hysteria v2 supports port hopping notation, where the server listens on multiple ports or port ranges. The checker parses this syntax and stores it as-is in the server address.

### Syntax

```
hysteria2://auth@host:port1,port2,startPort-endPort/?...#name
```

The port field accepts a comma-separated list of individual ports and/or port ranges (using `-`):

| Notation | Meaning |
|----------|---------|
| `443` | Single port |
| `443,8443` | Two individual ports |
| `5000-6000` | Port range (5000 through 6000) |
| `443,5000-6000` | Port 443 plus range 5000-6000 |
| `443,5000-6000,8000-9000` | Port 443 plus two ranges |

### Examples

```
# Single port with range
hysteria2://auth@example.com:443,5000-6000/?insecure=1#HopServer

# Multiple ranges
hy2://auth@example.com:443,5000-6000,8000-9000/?sni=example.com#MultiHop
```

### IPv6 with Port Hopping

For IPv6 addresses, the host is enclosed in brackets. Port hopping follows the closing bracket:

```
hysteria2://auth@[::1]:443,5000-6000/?insecure=1#IPv6-Hop
```

!!! note
    Port hopping is only supported in Hysteria v2 URIs. The v1 format requires a single port number.

---

## Validation Rules

The parser enforces the following constraints:

- **Scheme** must be `hysteria://` (v1) or `hysteria2://` / `hy2://` (v2). Any other scheme is rejected.
- **Host** is always required and cannot be empty.
- **Port** is required for v1. For v2, it defaults to `443` if omitted.
- **upmbps** and **downmbps** must be valid integers when present.
- **insecure** accepts `1` or `true` (case-insensitive) to enable; any other value or absence means disabled.
- A **stable ID** is generated from the combination of server, auth, version, SNI, obfs, obfs password, and protocol. Two URIs that differ in any of these fields are treated as distinct proxies.

### Error Handling

When parsing a subscription containing multiple URIs (one per line):

- Lines that fail to parse are skipped with a warning.
- If at least one URI parses successfully, the valid configs are returned alongside a partial error.
- If no URIs parse successfully, the entire operation returns an error.

---

## Version Differences

| Feature | v1 | v2 |
|---------|----|----|
| Scheme | `hysteria://` | `hysteria2://` or `hy2://` |
| Default port | None (required) | `443` |
| Auth location | Query parameter (`?auth=`) | Userinfo (`user@`) or query (`?auth=`) |
| SNI parameter | `peer` | `sni` |
| Obfs password parameter | `obfsParam` | `obfs-password` |
| Port hopping | Not supported | Supported |
| Certificate pinning | Not supported | `pinSHA256` |
| ALPN | Supported (`alpn`) | Not supported |
| Protocol selection | Supported (`protocol`) | Not supported |
