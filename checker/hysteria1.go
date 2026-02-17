package checker

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/belaytzev/hysteria-checker/models"
)

// Hysteria1Connector implements Connector for Hysteria v1 servers.
// Since the v1 Go library is incompatible with modern Go versions (requires Go <1.21),
// this connector shells out to the `hysteria` v1 binary, starts a local SOCKS5 proxy,
// and returns a ProxyClient that dials through it.
type Hysteria1Connector struct {
	// BinaryPath is the path to the hysteria v1 binary. Defaults to "hysteria".
	BinaryPath string
}

// hysteria1Client wraps a running hysteria v1 process and its SOCKS5 proxy.
type hysteria1Client struct {
	cmd       *exec.Cmd
	socksAddr string
	tmpFile   string
}

func (h *hysteria1Client) TCP(addr string) (net.Conn, error) {
	return dialViaSocks5(h.socksAddr, addr, 10*time.Second)
}

func (h *hysteria1Client) Close() error {
	if h.cmd != nil && h.cmd.Process != nil {
		_ = h.cmd.Process.Kill()
		_ = h.cmd.Wait()
	}
	if h.tmpFile != "" {
		_ = os.Remove(h.tmpFile)
	}
	return nil
}

// hysteria1Config is the JSON config format for the hysteria v1 client binary.
type hysteria1Config struct {
	Server     string            `json:"server"`
	Protocol   string            `json:"protocol,omitempty"`
	UpMbps     int               `json:"up_mbps"`
	DownMbps   int               `json:"down_mbps"`
	Auth       string            `json:"auth_str,omitempty"`
	ALPN       string            `json:"alpn,omitempty"`
	ServerName string            `json:"server_name,omitempty"`
	Insecure   bool              `json:"insecure,omitempty"`
	Obfs       string            `json:"obfs,omitempty"`
	SOCKS5     hysteria1Socks5   `json:"socks5"`
	TLS        *hysteria1TLSConf `json:"tls,omitempty"`
}

type hysteria1Socks5 struct {
	Listen string `json:"listen"`
}

type hysteria1TLSConf struct {
	SNI      string `json:"sni,omitempty"`
	Insecure bool   `json:"insecure,omitempty"`
}

func (c *Hysteria1Connector) Connect(cfg models.ProxyConfig) (ProxyClient, error) {
	binaryPath := c.BinaryPath
	if binaryPath == "" {
		binaryPath = "hysteria"
	}

	// Verify the binary exists
	if _, err := exec.LookPath(binaryPath); err != nil {
		return nil, fmt.Errorf("hysteria v1 binary not found at %q: %w", binaryPath, err)
	}

	// Find a free port for the SOCKS5 listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to find free port: %w", err)
	}
	socksAddr := listener.Addr().String()
	_ = listener.Close()

	// Build the config
	hyCfg := hysteria1Config{
		Server:   cfg.Server,
		UpMbps:   cfg.UpMbps,
		DownMbps: cfg.DownMbps,
		Auth:     cfg.Auth,
		SOCKS5:   hysteria1Socks5{Listen: socksAddr},
	}

	if cfg.Protocol != "" {
		hyCfg.Protocol = cfg.Protocol
	}
	if cfg.ALPN != "" {
		hyCfg.ALPN = cfg.ALPN
	}
	if strings.EqualFold(cfg.Obfs, "xplus") && cfg.ObfsParam != "" {
		hyCfg.Obfs = cfg.ObfsParam
	}
	if cfg.SNI != "" || cfg.Insecure {
		hyCfg.TLS = &hysteria1TLSConf{
			SNI:      cfg.SNI,
			Insecure: cfg.Insecure,
		}
	}

	// Write config to a temp file
	tmpFile, err := os.CreateTemp("", "hysteria1-*.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp config: %w", err)
	}
	if err := json.NewEncoder(tmpFile).Encode(hyCfg); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("failed to write config: %w", err)
	}
	_ = tmpFile.Close()

	// Start the hysteria v1 binary
	cmd := exec.Command(binaryPath, "client", "--config", tmpFile.Name())
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		_ = os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("failed to start hysteria v1 binary: %w", err)
	}

	// Wait for the SOCKS5 proxy to become available
	if err := waitForPort(socksAddr, 10*time.Second); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		_ = os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("hysteria v1 SOCKS5 proxy did not start: %w", err)
	}

	return &hysteria1Client{
		cmd:       cmd,
		socksAddr: socksAddr,
		tmpFile:   tmpFile.Name(),
	}, nil
}

// waitForPort polls until a TCP connection can be made to addr, or timeout expires.
func waitForPort(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s", addr)
}

// dialViaSocks5 establishes a TCP connection to target through a SOCKS5 proxy.
// Implements a minimal SOCKS5 client (RFC 1928) — no auth, CONNECT only.
func dialViaSocks5(proxyAddr, target string, timeout time.Duration) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", proxyAddr, timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SOCKS5 proxy: %w", err)
	}

	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		_ = conn.Close()
		return nil, err
	}

	// Greeting: version 5, 1 auth method (no auth)
	if _, err := conn.Write([]byte{0x05, 0x01, 0x00}); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("SOCKS5 greeting write failed: %w", err)
	}

	// Read greeting response (2 bytes)
	resp := make([]byte, 2)
	if _, err := io.ReadFull(conn, resp); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("SOCKS5 greeting read failed: %w", err)
	}
	if resp[0] != 0x05 || resp[1] != 0x00 {
		_ = conn.Close()
		return nil, fmt.Errorf("SOCKS5 auth not accepted: %x", resp)
	}

	// Parse target host:port
	host, portStr, err := net.SplitHostPort(target)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("invalid target address: %w", err)
	}
	port, err := net.LookupPort("tcp", portStr)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("invalid port: %w", err)
	}

	// CONNECT request
	req := []byte{
		0x05, // version
		0x01, // CONNECT
		0x00, // reserved
		0x03, // domain name
		byte(len(host)),
	}
	req = append(req, []byte(host)...)
	req = append(req, byte(port>>8), byte(port&0xff))

	if _, err := conn.Write(req); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("SOCKS5 connect write failed: %w", err)
	}

	// Read response: at least 4 bytes for header, then address
	header := make([]byte, 4)
	if _, err := io.ReadFull(conn, header); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("SOCKS5 connect response read failed: %w", err)
	}
	if header[1] != 0x00 {
		_ = conn.Close()
		return nil, fmt.Errorf("SOCKS5 connect failed with code: %d", header[1])
	}

	// Skip the bound address in the response
	var skipErr error
	switch header[3] {
	case 0x01: // IPv4
		skip := make([]byte, 4+2)
		_, skipErr = io.ReadFull(conn, skip)
	case 0x04: // IPv6
		skip := make([]byte, 16+2)
		_, skipErr = io.ReadFull(conn, skip)
	case 0x03: // Domain
		lenBuf := make([]byte, 1)
		if _, skipErr = io.ReadFull(conn, lenBuf); skipErr == nil {
			skip := make([]byte, int(lenBuf[0])+2)
			_, skipErr = io.ReadFull(conn, skip)
		}
	}
	if skipErr != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("SOCKS5 failed to read bound address: %w", skipErr)
	}

	// Clear deadline for the caller to manage
	_ = conn.SetDeadline(time.Time{})

	return conn, nil
}
