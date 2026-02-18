# Port Hopping Support

## Context

The Hysteria v2 URI scheme supports port hopping via comma/range notation in the port position:

```
hysteria2://auth@hostname:443,5000-6000/?insecure=1&obfs=salamander&obfs-password=pw
```

This is a documented anti-censorship feature. Real-world subscription URLs frequently include servers with port hopping. The current implementation calls `net.ResolveUDPAddr("udp", cfg.Server)` which fails for port hopping addresses because `443,5000-6000` is not a valid port number, causing every port-hopping server from a subscription to show as "connect failed."

## What Changes

**Only `checker/hysteria2.go` changes.**

The parser (`parser/hysteria2.go`) already extracts the full port string correctly via `u.Port()` — for `hostname:443,5000-6000` the port is `"443,5000-6000"`, stored as-is in `cfg.Server`. No model or parser changes needed.

## Design

### Detection

Detect port hopping by checking whether the port portion of `cfg.Server` is a plain integer. If not, it is a port hopping spec.

```go
func isPlainPort(s string) bool {
    _, err := strconv.ParseUint(s, 10, 16)
    return err == nil
}
```

### Address resolution

```
plain port → net.ResolveUDPAddr("udp", cfg.Server)   → *net.UDPAddr
hop spec   → udphop.ResolveUDPHopAddr(cfg.Server)    → *udphop.UDPHopAddr
```

Both implement `net.Addr` and are accepted by `client.Config.ServerAddr`.

### ConnFactory matrix

| Port hopping | Obfuscation | ConnFactory |
|---|---|---|
| No  | No          | nil (library default) |
| No  | Salamander  | existing `obfsConnFactory` |
| Yes | No          | new `portHopConnFactory` (plain `ListenUDPFunc`) |
| Yes | Salamander  | new `portHopConnFactory` (obfuscated `ListenUDPFunc`) |

### New `portHopConnFactory`

```go
type portHopConnFactory struct {
    addr        *udphop.UDPHopAddr
    hopInterval time.Duration   // default: 30s (udphop package default)
    obfuscator  obfs.Obfuscator // nil if no obfuscation
}

func (f *portHopConnFactory) New(_ net.Addr) (net.PacketConn, error) {
    listenFn := func() (net.PacketConn, error) {
        conn, err := net.ListenUDP("udp", nil)
        if err != nil {
            return nil, err
        }
        if f.obfuscator != nil {
            return obfs.WrapPacketConn(conn, f.obfuscator), nil
        }
        return conn, nil
    }
    return udphop.NewUDPHopPacketConn(f.addr, f.hopInterval, listenFn)
}
```

The `portHopConnFactory` ignores the `addr` argument passed by the library (it uses `f.addr` directly). The `udpHopPacketConn.WriteTo` also ignores the destination addr passed by quic-go and routes all packets to the current hop address internally.

### Updated `Connect()` flow

```
1. SplitHostPort(cfg.Server) → host, portStr
2. if isPlainPort(portStr):
       serverAddr = net.ResolveUDPAddr("udp", cfg.Server)
   else:
       serverAddr = udphop.ResolveUDPHopAddr(cfg.Server)
3. Build clientCfg (Auth, TLS, Bandwidth — unchanged)
4. Build obfuscator (if cfg.Obfs == "salamander" && cfg.ObfsParam != "")
5. Set ConnFactory:
       if isHopping:   portHopConnFactory{addr, 0, obfuscator}
       elif obfuscator: obfsConnFactory{obfuscator}
       else: nil (default)
6. client.NewClient(clientCfg)
```

## Imports Added

- `strconv` (stdlib)
- `github.com/apernet/hysteria/extras/v2/transport/udphop` (already in `go.mod`)

## Testing

- Unit test for `isPlainPort` helper with plain ports, ranges, and comma specs.
- Unit/mock test verifying `Connect()` selects the correct ConnFactory for each of the four combinations.
- Existing tests must continue to pass.
