package monitor

import (
	"testing"
	"time"
)

// TestProbeAllNodes_NoProbeTargetMarksAvailable is the regression guard for the
// probe de-duplication refactor. The per-pool startup probe used to mark all
// nodes available when no probe target was configured. After removing it and
// routing all probing through the monitor, probeAllNodes must take over that
// role: with no probe target (entry.probe == nil), it marks every node
// initialCheckDone+available — otherwise nodes stay unchecked forever and the
// export / "healthy online" count both read zero.
func TestProbeAllNodes_NoProbeTargetMarksAvailable(t *testing.T) {
	// Empty ProbeTarget => probeReady=false, and Register without SetProbe leaves
	// entry.probe == nil, mirroring the "no probe target" runtime state.
	mgr, err := NewManager(Config{ProbeTarget: ""})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	if mgr.probeReady {
		t.Fatal("probeReady should be false with an empty probe target")
	}

	mgr.Register(NodeInfo{Tag: "n1", URI: "vless://a@a.com:443"})
	mgr.Register(NodeInfo{Tag: "n2", URI: "vless://b@b.com:443"})

	// StartPeriodicHealthCheck's !probeReady branch runs probeAllNodes once.
	mgr.StartPeriodicHealthCheck(5*time.Minute, 8*time.Second)

	snaps := mgr.SnapshotFiltered(true) // strict: InitialCheckDone && Available
	if len(snaps) != 2 {
		t.Fatalf("expected both nodes marked available without a probe target, got %d", len(snaps))
	}
	for _, s := range snaps {
		if !s.InitialCheckDone || !s.Available {
			t.Errorf("node %q: InitialCheckDone=%v Available=%v, want both true",
				s.Tag, s.InitialCheckDone, s.Available)
		}
	}
}

// TestClampProbeConcurrency_Bounds verifies the widened [8, 1024] range so large
// inventories can be configured for a fast initial sweep while small/invalid
// values still floor at a safe worker count.
func TestClampProbeConcurrency_Bounds(t *testing.T) {
	cases := []struct {
		in, want int
	}{
		{0, 32},    // unset -> default
		{-5, 32},   // invalid -> default
		{3, 8},     // below floor -> 8
		{8, 8},     // floor
		{512, 512}, // mid-range honored (the point of widening)
		{1024, 1024},
		{5000, 1024}, // above ceiling -> 1024
	}
	for _, c := range cases {
		if got := clampProbeConcurrency(c.in); got != c.want {
			t.Errorf("clampProbeConcurrency(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}
