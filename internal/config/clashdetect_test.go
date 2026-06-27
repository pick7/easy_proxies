package config

import (
	"fmt"
	"strings"
	"testing"
)

// TestParseSubscriptionContent_ClashProxiesAfterLargePreamble proves a Clash
// YAML config is detected even when the "proxies:" key appears far past the old
// 16 KB sniff window (full Clash configs often place proxies after large dns /
// rule / proxy-provider sections). Before the fix these were misdetected as
// base64/plaintext and every node was dropped.
func TestParseSubscriptionContent_ClashProxiesAfterLargePreamble(t *testing.T) {
	var b strings.Builder
	b.WriteString("port: 7890\n")
	b.WriteString("socks-port: 7891\n")
	b.WriteString("mode: rule\n")
	b.WriteString("dns:\n  enable: true\n  nameserver:\n")
	// Pad well beyond the old 16384-char threshold with realistic preamble.
	for i := 0; i < 1200; i++ {
		b.WriteString(fmt.Sprintf("    - https://dns-%d.example.com/dns-query\n", i))
	}
	b.WriteString("proxies:\n")
	b.WriteString("  - {name: NodeA, type: vless, server: a.example.com, port: 443, uuid: uuid-a, network: ws, tls: true}\n")
	b.WriteString("  - {name: NodeB, type: trojan, server: b.example.com, port: 443, password: pw-b}\n")

	content := b.String()
	if len(content) <= 16384 {
		t.Fatalf("test setup error: preamble must exceed 16384 bytes, got %d", len(content))
	}

	nodes, err := parseSubscriptionContent(content)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes parsed from Clash YAML, got %d", len(nodes))
	}
	if !strings.HasPrefix(nodes[0].URI, "vless://") {
		t.Errorf("node A should be vless, got %q", nodes[0].URI)
	}
	if !strings.HasPrefix(nodes[1].URI, "trojan://") {
		t.Errorf("node B should be trojan, got %q", nodes[1].URI)
	}
}

// TestParseSubscriptionContent_Base64NotMisdetectedAsClash ensures the
// line-anchored "proxies:" detection does not misfire on a base64 blob that
// merely contains the substring "proxies:" mid-stream.
func TestParseSubscriptionContent_Base64NotMisdetectedAsClash(t *testing.T) {
	// A plaintext (non-clash) subscription: one URI per line.
	content := "vless://uuid-a@a.example.com:443?type=ws#A\n" +
		"trojan://pw@b.example.com:443#B\n"
	nodes, err := parseSubscriptionContent(content)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("expected 2 plaintext nodes, got %d", len(nodes))
	}
}
