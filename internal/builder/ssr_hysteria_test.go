package builder

import (
	"net/url"
	"testing"
)

func TestBuildShadowsocksROptions_RealURI(t *testing.T) {
	uri := "ssr://MTA3LjE1MS4xODIuMjUzOjgwODA6b3JpZ2luOnJjNC1tZDU6cGxhaW46TVRSbVJsQnlZbVY2UlROSVJGcDZjMDFQY2pZLz9ncm91cD1VMU5TVUhKdmRtbGtaWEkmcmVtYXJrcz04Si1IdXZDZmg3Z2dVMU5TTGVlLWp1V2J2UzFPUnVpbm8tbVVnZWlIcXVXSXR1V0pweTFEYUdGMFIxQlVMVlJwYTFSdmF5MVpiM1ZVZFdKbExURXdOeTR4TlRFdU1UZ3lMakkxTXpvNE1EZ3cmb2Jmc3BhcmFtPSZwcm90b3BhcmFtPQ"

	opts, err := buildShadowsocksROptions(uri)
	if err != nil {
		t.Fatalf("parse SSR URI: %v", err)
	}
	if opts.Server != "107.151.182.253" {
		t.Errorf("server = %q, want 107.151.182.253", opts.Server)
	}
	if opts.ServerPort != 8080 {
		t.Errorf("port = %d, want 8080", opts.ServerPort)
	}
	if opts.Protocol != "origin" {
		t.Errorf("protocol = %q, want origin", opts.Protocol)
	}
	if opts.Method != "rc4-md5" {
		t.Errorf("method = %q, want rc4-md5", opts.Method)
	}
	if opts.Obfs != "plain" {
		t.Errorf("obfs = %q, want plain", opts.Obfs)
	}
	if opts.Password == "" {
		t.Error("password should not be empty")
	}
}

func TestBuildShadowsocksROptions_AuthAES128(t *testing.T) {
	uri := "ssr://MTE2LjE2Mi4xMjAuMjQ6NTYyOmF1dGhfYWVzMTI4X21kNTpjaGFjaGEyMC1pZXRmOnBsYWluOmJXSnNZVzVyTVhCdmNuUT0vP2dyb3VwPWFIUjBjSE02THk5Mk1uSmhlWE5sTG1OdmJRPT0mb2Jmc3BhcmFtPWRDNXRaUzkyY0c1b1lYUT0mcHJvdG9wYXJhbT1NamMwTkRVNmFtZHFaMnBuYW1kcVoycG4mcmVtYXJrcz04SitIcVBDZmg3TmZRMDVmNUxpdDVadTlMVDd3bjRldDhKK0hzRjlJUzEvcHBwbm11Szg5"

	opts, err := buildShadowsocksROptions(uri)
	if err != nil {
		t.Fatalf("parse SSR URI: %v", err)
	}
	if opts.Server != "116.162.120.24" {
		t.Errorf("server = %q, want 116.162.120.24", opts.Server)
	}
	if opts.ServerPort != 562 {
		t.Errorf("port = %d, want 562", opts.ServerPort)
	}
	if opts.Protocol != "auth_aes128_md5" {
		t.Errorf("protocol = %q, want auth_aes128_md5", opts.Protocol)
	}
	if opts.Method != "chacha20-ietf" {
		t.Errorf("method = %q, want chacha20-ietf", opts.Method)
	}
	if opts.ObfsParam == "" {
		t.Error("obfs_param should not be empty")
	}
	if opts.ProtocolParam == "" {
		t.Error("protocol_param should not be empty")
	}
}

func TestBuildHysteriaOptions_RealURI(t *testing.T) {
	uri := "hysteria://5.83.129.153:20088?alpn=h3&auth=dongtaiwang.com&downmbps=1000&protocol=udp&insecure=1&peer=apple.com&upmbps=500#DE_test"

	parsed, err := url.Parse(uri)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}
	opts, err := buildHysteriaOptions(parsed, false)
	if err != nil {
		t.Fatalf("build hysteria options: %v", err)
	}
	if opts.Server != "5.83.129.153" {
		t.Errorf("server = %q, want 5.83.129.153", opts.Server)
	}
	if opts.ServerPort != 20088 {
		t.Errorf("port = %d, want 20088", opts.ServerPort)
	}
	if opts.AuthString != "dongtaiwang.com" {
		t.Errorf("auth = %q, want dongtaiwang.com", opts.AuthString)
	}
	if opts.UpMbps != 500 {
		t.Errorf("up = %d, want 500", opts.UpMbps)
	}
	if opts.DownMbps != 1000 {
		t.Errorf("down = %d, want 1000", opts.DownMbps)
	}
	if opts.TLS == nil {
		t.Fatal("TLS options should not be nil")
	}
	if opts.TLS.ServerName != "apple.com" {
		t.Errorf("sni = %q, want apple.com", opts.TLS.ServerName)
	}
	if !opts.TLS.Insecure {
		t.Error("insecure should be true")
	}
}

func TestBase64Decode_Variants(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"standard padded", "aGVsbG8=", "hello"},
		{"standard unpadded", "aGVsbG8", "hello"},
		{"url-safe", "aGVsbG8", "hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := base64Decode(tt.input)
			if err != nil {
				t.Fatalf("decode %q: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
