package models

import (
	"testing"
)

func TestGenerateStableID(t *testing.T) {
	p := ProxyConfig{
		Server:  "example.com:443",
		Auth:    "password",
		Version: 2,
	}
	p.GenerateStableID()

	if p.StableID == "" {
		t.Fatal("expected non-empty StableID")
	}
	if len(p.StableID) != 16 { // 8 bytes hex = 16 chars
		t.Errorf("expected StableID length 16, got %d", len(p.StableID))
	}

	// Same inputs should produce same ID
	p2 := ProxyConfig{
		Server:  "example.com:443",
		Auth:    "password",
		Version: 2,
	}
	p2.GenerateStableID()
	if p.StableID != p2.StableID {
		t.Errorf("expected same StableID for same inputs, got %q and %q", p.StableID, p2.StableID)
	}

	// Different inputs should produce different ID
	p3 := ProxyConfig{
		Server:  "other.com:443",
		Auth:    "password",
		Version: 2,
	}
	p3.GenerateStableID()
	if p.StableID == p3.StableID {
		t.Error("expected different StableID for different server")
	}
}
