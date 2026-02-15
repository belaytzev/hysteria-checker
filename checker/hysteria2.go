package checker

import (
	"bufio"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/apernet/hysteria/core/v2/client"
	"github.com/apernet/hysteria/extras/v2/obfs"
	"github.com/belaytzev/hysteria-checker/models"
)

// Hysteria2Connector implements Connector for Hysteria v2 servers.
type Hysteria2Connector struct{}

// obfsConnFactory wraps a ConnFactory to apply Salamander obfuscation.
type obfsConnFactory struct {
	obfuscator obfs.Obfuscator
}

func (f *obfsConnFactory) New(addr net.Addr) (net.PacketConn, error) {
	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return nil, err
	}
	return obfs.WrapPacketConn(conn, f.obfuscator), nil
}

// hysteria2Client wraps client.Client to implement ProxyClient.
type hysteria2Client struct {
	c client.Client
}

func (h *hysteria2Client) TCP(addr string) (net.Conn, error) {
	return h.c.TCP(addr)
}

func (h *hysteria2Client) Close() error {
	return h.c.Close()
}

func (c *Hysteria2Connector) Connect(cfg models.ProxyConfig) (ProxyClient, error) {
	host, _, err := net.SplitHostPort(cfg.Server)
	if err != nil {
		return nil, fmt.Errorf("invalid server address %q: %w", cfg.Server, err)
	}

	serverAddr, err := net.ResolveUDPAddr("udp", cfg.Server)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve server address: %w", err)
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

	// Salamander obfuscation
	if strings.EqualFold(cfg.Obfs, "salamander") && cfg.ObfsParam != "" {
		obfuscator, err := obfs.NewSalamanderObfuscator([]byte(cfg.ObfsParam))
		if err != nil {
			return nil, fmt.Errorf("failed to create salamander obfuscator: %w", err)
		}
		clientCfg.ConnFactory = &obfsConnFactory{obfuscator: obfuscator}
	}

	hyClient, _, err := client.NewClient(clientCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to hysteria2 server: %w", err)
	}

	return &hysteria2Client{c: hyClient}, nil
}

// CheckViaProxy performs a health check by making an HTTP request through the proxy.
// It returns whether the proxy is alive, the round-trip latency, and any error.
func CheckViaProxy(pc ProxyClient, checkURL string, timeout time.Duration) (alive bool, latency time.Duration, err error) {
	parsed, err := url.Parse(checkURL)
	if err != nil {
		return false, 0, fmt.Errorf("invalid check URL: %w", err)
	}

	host := parsed.Hostname()
	port := parsed.Port()
	if port == "" {
		if parsed.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}

	start := time.Now()

	conn, err := pc.TCP(net.JoinHostPort(host, port))
	if err != nil {
		return false, 0, fmt.Errorf("TCP connect failed: %w", err)
	}
	defer func() { _ = conn.Close() }()

	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return false, 0, fmt.Errorf("failed to set deadline: %w", err)
	}

	// Build and send raw HTTP request through the proxied connection
	reqStr := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\nConnection: close\r\nUser-Agent: hysteria-checker/1.0\r\n\r\n",
		parsed.RequestURI(), parsed.Host)
	if _, err := conn.Write([]byte(reqStr)); err != nil {
		return false, 0, fmt.Errorf("write request failed: %w", err)
	}

	// Read and parse the HTTP response
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		return false, 0, fmt.Errorf("read response failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read body (needed for IP check method; also ensures full round trip)
	_, _ = io.ReadAll(resp.Body)

	latency = time.Since(start)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true, latency, nil
	}

	return false, latency, fmt.Errorf("HTTP status %d", resp.StatusCode)
}

// normalizeCertHash removes colons and hyphens from a certificate hash string
// and converts it to lowercase.
func normalizeCertHash(hash string) string {
	hash = strings.ReplaceAll(hash, ":", "")
	hash = strings.ReplaceAll(hash, "-", "")
	return strings.ToLower(hash)
}
