package monitor

import (
	"testing"
	"time"
)

// TestSnapshotFiltered_OnlyAvailableExcludesUnchecked is the regression test for
// the export-count mismatch bug:导出了 6000 个未检查节点，但 WebUI 只显示 1000
// 个已验证可用的。SnapshotFiltered(true) must use the same strict criterion as
// the "healthy online" statistic: InitialCheckDone && Available && !Blacklisted.
func TestSnapshotFiltered_OnlyAvailableExcludesUnchecked(t *testing.T) {
	mgr, err := NewManager(Config{ProbeTarget: "https://www.google.com"})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	// Register 4 nodes with different states:
	// 1. Checked & available → included in filtered snapshot
	// 2. Checked & unavailable → excluded
	// 3. Not yet checked (InitialCheckDone=false) → excluded (the fix)
	// 4. Blacklisted → excluded

	available := mgr.Register(NodeInfo{Tag: "available", URI: "vless://a@a.com:443"})
	unavailable := mgr.Register(NodeInfo{Tag: "unavailable", URI: "vless://b@b.com:443"})
	_ = mgr.Register(NodeInfo{Tag: "unchecked", URI: "vless://c@c.com:443"}) // unchecked: defaults to initialCheckDone=false
	blacklisted := mgr.Register(NodeInfo{Tag: "blacklisted", URI: "vless://d@d.com:443"})

	// Simulate health check results. EntryHandle holds a ref pointer to the
	// internal entry; since this test is in the same package, we can access
	// unexported fields directly.
	available.ref.mu.Lock()
	available.ref.initialCheckDone = true
	available.ref.available = true
	available.ref.lastOK = time.Now()
	available.ref.mu.Unlock()

	unavailable.ref.mu.Lock()
	unavailable.ref.initialCheckDone = true
	unavailable.ref.available = false
	unavailable.ref.lastFail = time.Now()
	unavailable.ref.mu.Unlock()

	// unchecked: leave initialCheckDone=false (simulates a node that hasn't been
	// probed yet, or whose startup probe failed/timed out)

	blacklisted.ref.mu.Lock()
	blacklisted.ref.initialCheckDone = true
	blacklisted.ref.available = true
	blacklisted.ref.blacklist = true
	blacklisted.ref.mu.Unlock()

	// Test the strict filter used by export
	filtered := mgr.SnapshotFiltered(true)

	if len(filtered) != 1 {
		t.Errorf("SnapshotFiltered(true) returned %d nodes, want 1 (only the verified-available node)", len(filtered))
		for _, s := range filtered {
			t.Logf("  - %s: InitialCheckDone=%v Available=%v Blacklisted=%v",
				s.Tag, s.InitialCheckDone, s.Available, s.Blacklisted)
		}
	}

	if len(filtered) > 0 && filtered[0].Tag != "available" {
		t.Errorf("SnapshotFiltered(true) included %q, want only 'available'", filtered[0].Tag)
	}

	// Verify the unchecked node was excluded (the fix for the export mismatch)
	for _, s := range filtered {
		if s.Tag == "unchecked" {
			t.Errorf("SnapshotFiltered(true) included the unchecked node; it should be excluded so export count matches the WebUI 'healthy online' display")
		}
	}
}

// TestSnapshotFiltered_UnfilteredIncludesAll verifies that SnapshotFiltered(false)
// still returns all nodes regardless of health state (used by the full node list).
func TestSnapshotFiltered_UnfilteredIncludesAll(t *testing.T) {
	mgr, err := NewManager(Config{ProbeTarget: "https://www.google.com"})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	mgr.Register(NodeInfo{Tag: "node1", URI: "vless://1@a.com:443"})
	mgr.Register(NodeInfo{Tag: "node2", URI: "vless://2@b.com:443"})

	unfiltered := mgr.SnapshotFiltered(false)
	if len(unfiltered) != 2 {
		t.Errorf("SnapshotFiltered(false) returned %d nodes, want 2", len(unfiltered))
	}
}
