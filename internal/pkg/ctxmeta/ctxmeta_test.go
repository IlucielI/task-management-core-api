package ctxmeta

import (
	"context"
	"testing"
)

func TestContextMeta(t *testing.T) {
	ctx := context.Background()

	// Initial empty
	if ip := GetClientIP(ctx); ip != "" {
		t.Fatalf("expected empty IP, got %s", ip)
	}
	if ua := GetUserAgent(ctx); ua != "" {
		t.Fatalf("expected empty UA, got %s", ua)
	}

	// Injected
	ctx = WithClientMeta(ctx, "192.168.1.100", "Mozilla/5.0 UnitTester")
	if ip := GetClientIP(ctx); ip != "192.168.1.100" {
		t.Fatalf("expected 192.168.1.100, got %s", ip)
	}
	if ua := GetUserAgent(ctx); ua != "Mozilla/5.0 UnitTester" {
		t.Fatalf("expected Mozilla/5.0 UnitTester, got %s", ua)
	}

	// Nil context checks
	if ip := GetClientIP(nil); ip != "" {
		t.Fatalf("expected empty on nil ctx, got %s", ip)
	}
	if ua := GetUserAgent(nil); ua != "" {
		t.Fatalf("expected empty on nil ctx, got %s", ua)
	}
}
