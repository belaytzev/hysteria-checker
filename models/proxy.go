package models

import (
	"crypto/sha256"
	"fmt"
)

// ProxyConfig holds the parsed configuration for a single Hysteria proxy server.
type ProxyConfig struct {
	Version   int    // 1 or 2
	Name      string // from URI fragment (#remarks)
	Server    string // host:port
	Auth      string
	SNI       string
	Insecure  bool
	Obfs      string // obfuscation type (xplus for v1, salamander for v2)
	ObfsParam string // obfuscation password
	UpMbps    int    // v1 required, v2 optional
	DownMbps  int    // v1 required, v2 optional
	ALPN      string // v1 only
	Protocol  string // v1 only (udp, wechat-video, faketcp)
	PinSHA256 string // v2 only
	StableID  string // deterministic hash for metrics labels
}

// GenerateStableID computes a deterministic SHA256 hash from key proxy fields
// (Server, Auth, Version, SNI, Obfs, ObfsParam, Protocol)
// for use as a stable metric label and deduplication key.
// Two configs that differ in any of these fields are considered distinct proxies.
func (p *ProxyConfig) GenerateStableID() {
	// Use length-prefixed fields to avoid delimiter collisions when field values
	// contain the separator character.
	fields := []string{p.Server, p.Auth, fmt.Sprintf("%d", p.Version), p.SNI, p.Obfs, p.ObfsParam, p.Protocol}
	var data string
	for _, f := range fields {
		data += fmt.Sprintf("%d:%s,", len(f), f)
	}
	hash := sha256.Sum256([]byte(data))
	p.StableID = fmt.Sprintf("%x", hash[:8])
}
