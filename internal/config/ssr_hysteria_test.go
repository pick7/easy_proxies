package config

import (
	"strings"
	"testing"
)

func TestConvertClashProxy_SSR(t *testing.T) {
	p := clashProxy{
		Name:          "SSR-Test",
		Type:          "ssr",
		Server:        "1.2.3.4",
		Port:          8388,
		Password:      "testpass",
		Cipher:        "aes-256-cfb",
		Protocol:      "auth_aes128_md5",
		ProtocolParam: "12345:user",
		Obfs:          "tls1.2_ticket_auth",
		ObfsParam:     "example.com",
	}
	uri := convertClashProxyToURI(p)
	if !strings.HasPrefix(uri, "ssr://") {
		t.Fatalf("expected ssr:// prefix, got %q", uri)
	}
}

func TestConvertClashProxy_Hysteria(t *testing.T) {
	p := clashProxy{
		Name:           "Hysteria-Test",
		Type:           "hysteria",
		Server:         "5.6.7.8",
		Port:           20088,
		AuthStr:        "secretauth",
		UpMbps:         500,
		DownMbps:       1000,
		SkipCertVerify: true,
		ALPN:           []string{"h3"},
	}
	uri := convertClashProxyToURI(p)
	if !strings.HasPrefix(uri, "hysteria://") {
		t.Fatalf("expected hysteria:// prefix, got %q", uri)
	}
	if !strings.Contains(uri, "auth=secretauth") {
		t.Errorf("URI missing auth param: %q", uri)
	}
	if !strings.Contains(uri, "insecure=1") {
		t.Errorf("URI missing insecure param: %q", uri)
	}
	if !strings.Contains(uri, "upmbps=500") {
		t.Errorf("URI missing upmbps: %q", uri)
	}
}

func TestConvertClashProxy_HysteriaFallbackAuth(t *testing.T) {
	// Some Clash configs use "auth" field, others "auth-str", others "password"
	p := clashProxy{
		Name:     "Hy-Fallback",
		Type:     "hysteria",
		Server:   "1.2.3.4",
		Port:     443,
		Password: "pw-as-auth",
	}
	uri := convertClashProxyToURI(p)
	if !strings.Contains(uri, "auth=pw-as-auth") {
		t.Errorf("URI should fall back to password as auth: %q", uri)
	}
}

func TestParseClashYAML_SSRAndHysteria(t *testing.T) {
	content := `proxies:
  - name: SSR-Node
    type: ssr
    server: 10.0.0.1
    port: 8388
    cipher: aes-256-cfb
    password: mypass
    protocol: auth_aes128_sha1
    protocol-param: "123:user"
    obfs: tls1.2_ticket_auth
    obfs-param: example.com
  - name: Hysteria-Node
    type: hysteria
    server: 10.0.0.2
    port: 20088
    auth-str: myauth
    up: 100
    down: 200
    skip-cert-verify: true
    alpn:
      - h3
  - name: VLESS-Node
    type: vless
    server: 10.0.0.3
    port: 443
    uuid: test-uuid
    network: ws
    tls: true
`
	nodes, err := parseClashYAML(content)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(nodes))
	}

	// SSR
	if !strings.HasPrefix(nodes[0].URI, "ssr://") {
		t.Errorf("node 0 should be ssr://, got %q", nodes[0].URI)
	}

	// Hysteria v1
	if !strings.HasPrefix(nodes[1].URI, "hysteria://") {
		t.Errorf("node 1 should be hysteria://, got %q", nodes[1].URI)
	}
	if !strings.Contains(nodes[1].URI, "auth=myauth") {
		t.Errorf("node 1 missing auth: %q", nodes[1].URI)
	}

	// VLESS (existing)
	if !strings.HasPrefix(nodes[2].URI, "vless://") {
		t.Errorf("node 2 should be vless://, got %q", nodes[2].URI)
	}
}
