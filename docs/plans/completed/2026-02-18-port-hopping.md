# Port Hopping Support Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add Hysteria v2 port hopping support to the connector so subscription URIs with addresses like `hostname:443,5000-6000` connect correctly instead of failing with "invalid port".

**Architecture:** Detect port hopping by checking whether the port portion of `cfg.Server` is a plain integer. If not, use `udphop.ResolveUDPHopAddr` for the server address and a new `portHopConnFactory` (which wraps `udphop.NewUDPHopPacketConn`) as the `ConnFactory`. Salamander obfuscation composes cleanly: the hop factory accepts an optional `obfs.Obfuscator` and applies it per-connection inside its `ListenUDPFunc`.

**Tech Stack:** Go stdlib (`strconv`, `net`), `github.com/apernet/hysteria/core/v2/client`, `github.com/apernet/hysteria/extras/v2/obfs`, `github.com/apernet/hysteria/extras/v2/transport/udphop` (already in go.mod).

---

### Task 1: `isPlainPort` helper

**Files:**
- Modify: `checker/hysteria2.go`
- Test: `checker/checker_test.go`

**Step 1: Write the failing test**

Add to `checker/checker_test.go`:

```go
func TestIsPlainPort(t *testing.T) {
	tests := []struct {
		input string
		plain bool
	}{
		{"443", true},
		{"8080", true},
		{"0", true},
		{"65535", true},
		{"443,8000-9000", false},
		{"5000-6000", false},
		{"443,444", false},
		{"", false},
		{"abc", false},
		{"65536", false}, // out of uint16 range
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isPlainPort(tt.input)
			if got != tt.plain {
				t.Errorf("isPlainPort(%q) = %v, want %v", tt.input, got, tt.plain)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

```bash
export GOROOT=/opt/homebrew/Cellar/go/1.25.5/libexec && export GOPATH=$HOME/go && export GOBIN="" && $GOROOT/bin/go test -C /Users/belaytzev/Documents/Sync/Personal/hysteria-checker ./checker/ -run TestIsPlainPort -v
```

Expected: `FAIL — undefined: isPlainPort`

**Step 3: Add `isPlainPort` to `checker/hysteria2.go`**

Add `"strconv"` to the import block, then add before `normalizeCertHash`:

```go
// isPlainPort reports whether s is a plain decimal port number in [0, 65535].
func isPlainPort(s string) bool {
	_, err := strconv.ParseUint(s, 10, 16)
	return err == nil
}
```

**Step 4: Run test to verify it passes**

```bash
export GOROOT=/opt/homebrew/Cellar/go/1.25.5/libexec && export GOPATH=$HOME/go && export GOBIN="" && $GOROOT/bin/go test -C /Users/belaytzev/Documents/Sync/Personal/hysteria-checker ./checker/ -run TestIsPlainPort -v
```

Expected: `PASS`

**Step 5: Run all tests to check for regressions**

```bash
export GOROOT=/opt/homebrew/Cellar/go/1.25.5/libexec && export GOPATH=$HOME/go && export GOBIN="" && $GOROOT/bin/go test -C /Users/belaytzev/Documents/Sync/Personal/hysteria-checker ./...
```

Expected: all green.

**Step 6: Commit**

```bash
git -C /Users/belaytzev/Documents/Sync/Personal/hysteria-checker add checker/hysteria2.go checker/checker_test.go
git -C /Users/belaytzev/Documents/Sync/Personal/hysteria-checker commit -m "feat: add isPlainPort helper for port hopping detection"
```

---

### Task 2: `portHopConnFactory`

**Files:**
- Modify: `checker/hysteria2.go`
- Test: `checker/checker_test.go`

**Step 1: Write the failing tests**

Add to `checker/checker_test.go`. These tests verify the struct compiles, the interface is satisfied, and `New()` returns a usable `net.PacketConn`.

```go
func TestPortHopConnFactory_ImplementsConnFactory(t *testing.T) {
	// Compilation test — if portHopConnFactory doesn't implement client.ConnFactory
	// this will not compile.
	hopAddr, err := udphop.ResolveUDPHopAddr("127.0.0.1:8000-9000")
	if err != nil {
		t.Fatalf("ResolveUDPHopAddr: %v", err)
	}
	var _ client.ConnFactory = &portHopConnFactory{addr: hopAddr}
}

func TestPortHopConnFactory_New_PlainUDP(t *testing.T) {
	hopAddr, err := udphop.ResolveUDPHopAddr("127.0.0.1:8000-9000")
	if err != nil {
		t.Fatalf("ResolveUDPHopAddr: %v", err)
	}
	factory := &portHopConnFactory{addr: hopAddr}
	conn, err := factory.New(hopAddr)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	if conn == nil {
		t.Fatal("New() returned nil conn")
	}
	_ = conn.Close()
}

func TestPortHopConnFactory_New_WithObfs(t *testing.T) {
	hopAddr, err := udphop.ResolveUDPHopAddr("127.0.0.1:8000-9000")
	if err != nil {
		t.Fatalf("ResolveUDPHopAddr: %v", err)
	}
	obfuscator, err := obfs.NewSalamanderObfuscator([]byte("test-password"))
	if err != nil {
		t.Fatalf("NewSalamanderObfuscator: %v", err)
	}
	factory := &portHopConnFactory{addr: hopAddr, obfuscator: obfuscator}
	conn, err := factory.New(hopAddr)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	if conn == nil {
		t.Fatal("New() returned nil conn")
	}
	_ = conn.Close()
}
```

You also need to add these imports to `checker_test.go` (add to the existing import block):

```go
"github.com/apernet/hysteria/core/v2/client"
"github.com/apernet/hysteria/extras/v2/obfs"
"github.com/apernet/hysteria/extras/v2/transport/udphop"
```

**Step 2: Run tests to verify they fail**

```bash
export GOROOT=/opt/homebrew/Cellar/go/1.25.5/libexec && export GOPATH=$HOME/go && export GOBIN="" && $GOROOT/bin/go test -C /Users/belaytzev/Documents/Sync/Personal/hysteria-checker ./checker/ -run TestPortHopConnFactory -v
```

Expected: `FAIL — undefined: portHopConnFactory`

**Step 3: Add `portHopConnFactory` to `checker/hysteria2.go`**

Add `"github.com/apernet/hysteria/extras/v2/transport/udphop"` to the import block.

Add the struct and method after the existing `obfsConnFactory` block (around line 36):

```go
// portHopConnFactory implements ConnFactory for port-hopping connections.
// It creates a udpHopPacketConn that rotates between ports on a 30-second interval.
// If obfuscator is non-nil, each underlying UDP connection is wrapped with
// Salamander obfuscation before being handed to the hop conn.
type portHopConnFactory struct {
	addr       *udphop.UDPHopAddr
	obfuscator obfs.Obfuscator // nil if no obfuscation
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
	return udphop.NewUDPHopPacketConn(f.addr, 0, listenFn) // 0 = default 30s hop interval
}
```

**Step 4: Run tests to verify they pass**

```bash
export GOROOT=/opt/homebrew/Cellar/go/1.25.5/libexec && export GOPATH=$HOME/go && export GOBIN="" && $GOROOT/bin/go test -C /Users/belaytzev/Documents/Sync/Personal/hysteria-checker ./checker/ -run TestPortHopConnFactory -v
```

Expected: `PASS`

**Step 5: Run all tests to check for regressions**

```bash
export GOROOT=/opt/homebrew/Cellar/go/1.25.5/libexec && export GOPATH=$HOME/go && export GOBIN="" && $GOROOT/bin/go test -C /Users/belaytzev/Documents/Sync/Personal/hysteria-checker ./...
```

Expected: all green.

**Step 6: Commit**

```bash
git -C /Users/belaytzev/Documents/Sync/Personal/hysteria-checker add checker/hysteria2.go checker/checker_test.go
git -C /Users/belaytzev/Documents/Sync/Personal/hysteria-checker commit -m "feat: add portHopConnFactory for Hysteria v2 port hopping"
```

---

### Task 3: Update `Connect()` to use port hopping

**Files:**
- Modify: `checker/hysteria2.go` (the `Connect` method only)
- Test: `checker/checker_test.go`

**Step 1: Write the failing test**

This test verifies that a port-hopping server address gets past the parse/resolve stage (rather than failing with "invalid port"). It will fail at the QUIC handshake (no real server), but the error must come from the connection layer, not from address parsing.

Add to `checker/checker_test.go`:

```go
func TestHysteria2Connector_PortHoppingAddress(t *testing.T) {
	// Before the fix, Connect() fails with "invalid port" at net.ResolveUDPAddr.
	// After the fix, it should fail at the QUIC/connection level instead.
	connector := &Hysteria2Connector{}
	_, err := connector.Connect(models.ProxyConfig{
		Version: 2,
		Server:  "127.0.0.1:8000-9000",
		Auth:    "test",
	})
	if err == nil {
		t.Fatal("expected error for unreachable server")
	}
	// Must NOT be an address-parse error — it must have reached the QUIC layer.
	errStr := err.Error()
	if strings.Contains(errStr, "invalid port") || strings.Contains(errStr, "invalid syntax") {
		t.Errorf("expected connection-level error, not parse error: %v", err)
	}
}
```

**Step 2: Run test to verify it fails (currently)**

```bash
export GOROOT=/opt/homebrew/Cellar/go/1.25.5/libexec && export GOPATH=$HOME/go && export GOBIN="" && $GOROOT/bin/go test -C /Users/belaytzev/Documents/Sync/Personal/hysteria-checker ./checker/ -run TestHysteria2Connector_PortHoppingAddress -v -timeout 60s
```

Expected: `FAIL` — error contains "invalid port" or "invalid syntax" (current behaviour).

**Step 3: Rewrite `Connect()` in `checker/hysteria2.go`**

Replace the entire `Connect` method body. The signature stays the same. The changes are:

1. Use `net.SplitHostPort` (instead of `net.SplitHostPort` + immediate `ResolveUDPAddr`) to get `host` and `portStr` separately.
2. Call `isPlainPort(portStr)` to decide which resolver to use.
3. Build the obfuscator before setting `ConnFactory` (was at the bottom, now in the middle).
4. Use a `switch` to pick the right factory.

Full replacement:

```go
func (c *Hysteria2Connector) Connect(cfg models.ProxyConfig) (ProxyClient, error) {
	host, portStr, err := net.SplitHostPort(cfg.Server)
	if err != nil {
		return nil, fmt.Errorf("invalid server address %q: %w", cfg.Server, err)
	}

	// Resolve the server address. Port hopping uses a comma/range port spec
	// (e.g. "443,5000-6000"); a plain port is just a decimal number.
	isHopping := !isPlainPort(portStr)
	var serverAddr net.Addr
	var hopAddr *udphop.UDPHopAddr
	if isHopping {
		hopAddr, err = udphop.ResolveUDPHopAddr(cfg.Server)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve port hopping address: %w", err)
		}
		serverAddr = hopAddr
	} else {
		serverAddr, err = net.ResolveUDPAddr("udp", cfg.Server)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve server address: %w", err)
		}
	}

	clientCfg := &client.Config{
		ServerAddr: serverAddr,
		Auth:       cfg.Auth,
		TLSConfig: client.TLSConfig{
			ServerName:         host,
			InsecureSkipVerify: cfg.Insecure,
		},
	}

	// Override SNI if explicitly set
	if cfg.SNI != "" {
		clientCfg.TLSConfig.ServerName = cfg.SNI
	}

	// Certificate pinning
	if cfg.PinSHA256 != "" {
		pinHash := normalizeCertHash(cfg.PinSHA256)
		clientCfg.TLSConfig.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return fmt.Errorf("no certificates presented")
			}
			hash := sha256.Sum256(rawCerts[0])
			hashHex := hex.EncodeToString(hash[:])
			if hashHex == pinHash {
				return nil
			}
			return fmt.Errorf("certificate fingerprint mismatch: got %s, want %s", hashHex, pinHash)
		}
	}

	// Bandwidth config
	if cfg.UpMbps > 0 {
		clientCfg.BandwidthConfig.MaxTx = uint64(cfg.UpMbps) * 1_000_000 / 8
	}
	if cfg.DownMbps > 0 {
		clientCfg.BandwidthConfig.MaxRx = uint64(cfg.DownMbps) * 1_000_000 / 8
	}

	// Build obfuscator (shared by both factory types)
	var obfuscator obfs.Obfuscator
	if strings.EqualFold(cfg.Obfs, "salamander") && cfg.ObfsParam != "" {
		obfuscator, err = obfs.NewSalamanderObfuscator([]byte(cfg.ObfsParam))
		if err != nil {
			return nil, fmt.Errorf("failed to create salamander obfuscator: %w", err)
		}
	}

	// Set ConnFactory based on the feature combination:
	//   hopping + obfs  → portHopConnFactory with obfuscated ListenUDPFunc
	//   hopping only    → portHopConnFactory with plain ListenUDPFunc
	//   obfs only       → obfsConnFactory (existing)
	//   neither         → nil (library uses default udpConnFactory)
	switch {
	case isHopping:
		clientCfg.ConnFactory = &portHopConnFactory{addr: hopAddr, obfuscator: obfuscator}
	case obfuscator != nil:
		clientCfg.ConnFactory = &obfsConnFactory{obfuscator: obfuscator}
	}

	hyClient, _, err := client.NewClient(clientCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to hysteria2 server: %w", err)
	}

	return &hysteria2Client{c: hyClient}, nil
}
```

**Step 4: Run the new test to verify it passes**

```bash
export GOROOT=/opt/homebrew/Cellar/go/1.25.5/libexec && export GOPATH=$HOME/go && export GOBIN="" && $GOROOT/bin/go test -C /Users/belaytzev/Documents/Sync/Personal/hysteria-checker ./checker/ -run TestHysteria2Connector_PortHoppingAddress -v -timeout 60s
```

Expected: `PASS` — error is from the QUIC connection layer, not from parsing.

**Step 5: Run all tests to check for regressions**

```bash
export GOROOT=/opt/homebrew/Cellar/go/1.25.5/libexec && export GOPATH=$HOME/go && export GOBIN="" && $GOROOT/bin/go test -C /Users/belaytzev/Documents/Sync/Personal/hysteria-checker ./... -timeout 120s
```

Expected: all green.

**Step 6: Commit**

```bash
git -C /Users/belaytzev/Documents/Sync/Personal/hysteria-checker add checker/hysteria2.go checker/checker_test.go
git -C /Users/belaytzev/Documents/Sync/Personal/hysteria-checker commit -m "feat: support port hopping URIs in Hysteria v2 connector"
```

---

## Notes

- `udphop.NewUDPHopPacketConn` with `hopInterval=0` uses the package default of 30 seconds.
- The `portHopConnFactory.New` ignores its `addr` argument — the hop conn always routes to `f.addr` internally.
- The `TestHysteria2Connector_PortHoppingAddress` test may take a few seconds while the QUIC handshake times out against `127.0.0.1:8000-9000`. Run with `-timeout 60s` or `-timeout 120s` when running the full suite to be safe.
- No changes needed to `parser/`, `models/`, `subscription/`, or any other package.
