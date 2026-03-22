package checker

import (
	"net"
	"time"

	"github.com/belaytzev/hysteria-checker/models"
)

// Connector abstracts connecting to a Hysteria proxy and checking connectivity through it.
type Connector interface {
	// Connect establishes a connection to the Hysteria server described by cfg.
	// Returns a ProxyClient that can be used to make TCP connections through the proxy.
	Connect(cfg models.ProxyConfig) (ProxyClient, error)
}

// ProxyClient abstracts an established Hysteria client connection.
type ProxyClient interface {
	// TCP opens a TCP connection to addr through the proxy.
	TCP(addr string) (net.Conn, error)
	// Close closes the client connection.
	Close() error
}

// CheckResult holds the result of a single proxy health check.
type CheckResult struct {
	Alive     bool
	Latency   time.Duration
	LastCheck time.Time
	Error     string
}
