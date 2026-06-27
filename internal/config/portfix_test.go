package config

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

// mustJSON marshals v or fails the test.
func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return data
}

// TestNormalizeWithPortMap_DuplicateIdentityNoPortCollision proves that two
// distinct nodes which collapse to the same stable key (e.g. a subscription
// listing the same server twice under different display names) do NOT both get
// assigned the same preserved port — which would bind the same proxy port twice
// (EADDRINUSE). The first keeps the preserved port; the second gets a fresh one.
func TestNormalizeWithPortMap_DuplicateIdentityNoPortCollision(t *testing.T) {
	const dupURI = "vless://uuid-a@a.example.com:443?type=ws&security=tls"

	cfg := &Config{
		Mode:      "multi-port",
		MultiPort: MultiPortConfig{Address: "127.0.0.1", BasePort: 24000},
		Nodes: []NodeConfig{
			{URI: dupURI + "#NodeA-1"},
			{URI: dupURI + "#NodeA-2"}, // same stable key as the first
		},
	}

	// Pretend the first node already had port 24000 preserved.
	portMap := map[string]uint16{stableNodeKey(dupURI): 24000}

	if err := cfg.NormalizeWithPortMap(portMap); err != nil {
		t.Fatalf("normalize: %v", err)
	}

	p0, p1 := cfg.Nodes[0].Port, cfg.Nodes[1].Port
	if p0 == 0 || p1 == 0 {
		t.Fatalf("expected non-zero ports, got %d and %d", p0, p1)
	}
	if p0 == p1 {
		t.Fatalf("duplicate-identity nodes must not share a port, both got %d", p0)
	}
	if p0 != 24000 {
		t.Errorf("first node should keep preserved port 24000, got %d", p0)
	}
}

// TestApplyPersistedPorts_BridgesLegacyRawURIKeys proves that a sidecar written
// by an older version (keyed by the raw full URI, including #fragment) still
// preserves ports after upgrading to stable-key identity — no one-time reset.
func TestApplyPersistedPorts_BridgesLegacyRawURIKeys(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	nodesPath := filepath.Join(dir, "nodes.txt")

	writeFile(t, cfgPath, `mode: multi-port
multi_port:
  address: 127.0.0.1
  base_port: 24000
nodes_file: nodes.txt
management:
  enabled: false
`)

	const (
		uriA = "vless://uuid-a@a.example.com:443?type=ws&security=tls#NodeA"
		uriB = "vless://uuid-b@b.example.com:443?type=ws&security=tls#NodeB"
	)
	writeFile(t, nodesPath, uriA+"\n"+uriB+"\n")

	// Simulate a legacy sidecar: keyed by the RAW full URI (old NodeKey()).
	legacy := map[string]uint16{uriA: 24050, uriB: 24051}
	cfgForSave := &Config{
		Mode:      "multi-port",
		MultiPort: MultiPortConfig{Address: "127.0.0.1", BasePort: 24000},
		Nodes: []NodeConfig{
			{URI: uriA, Port: legacy[uriA]},
			{URI: uriB, Port: legacy[uriB]},
		},
	}
	cfgForSave.SetFilePath(cfgPath)
	// Build the legacy file by hand: BuildPortMap now uses stable keys, so write
	// the raw-URI map directly to mimic an older binary's output.
	if err := writeFileWithLock(filepath.Join(dir, nodePortMapFile), mustJSON(t, legacy), 0o644); err != nil {
		t.Fatalf("write legacy sidecar: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	got := map[string]uint16{}
	for _, n := range cfg.Nodes {
		got[n.URI] = n.Port
	}
	if got[uriA] != 24050 {
		t.Errorf("node A should keep legacy port 24050 across upgrade, got %d", got[uriA])
	}
	if got[uriB] != 24051 {
		t.Errorf("node B should keep legacy port 24051 across upgrade, got %d", got[uriB])
	}
}
