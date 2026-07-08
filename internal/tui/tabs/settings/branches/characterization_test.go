package branches

import "testing"

// This file characterizes the Branches settings sub-tab's current behavior
// BEFORE the P2.3 Tab-interface refactor. The parent settings model reads the
// branch limit and show-all-remotes toggle when saving config; the tests lock
// the accessor surface and clamping.
//
// PLAN(P2.3): keep this green across the interface migration.

// TestNewModelDefaults locks the documented defaults (limit 100, filter on).
func TestNewModelDefaults(t *testing.T) {
	m := NewModel()
	if m.GetBranchLimit() != 100 {
		t.Fatalf("default GetBranchLimit() = %d, want 100", m.GetBranchLimit())
	}
	if m.GetShowAllRemotes() {
		t.Fatal("default GetShowAllRemotes() should be false (filter on)")
	}
}

// TestSetBranchLimitClamps locks the [0,500] clamp so a stray value can't wedge
// the branch list query.
func TestSetBranchLimitClamps(t *testing.T) {
	cases := []struct{ in, want int }{
		{-5, 0},
		{0, 0},
		{250, 250},
		{500, 500},
		{9999, 500},
	}
	for _, tc := range cases {
		m := NewModel()
		m.SetBranchLimit(tc.in)
		if got := m.GetBranchLimit(); got != tc.want {
			t.Fatalf("SetBranchLimit(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

// TestToggleShowAllRemotesFlipsTwice locks that two toggles return to start.
func TestToggleShowAllRemotesFlipsTwice(t *testing.T) {
	m := NewModel()
	start := m.GetShowAllRemotes()
	m.ToggleShowAllRemotes()
	if m.GetShowAllRemotes() == start {
		t.Fatal("ToggleShowAllRemotes did not flip")
	}
	m.ToggleShowAllRemotes()
	if m.GetShowAllRemotes() != start {
		t.Fatal("two toggles should return to start")
	}
}

// TestSetShowAllRemotes locks the direct setter used by zone clicks.
func TestSetShowAllRemotes(t *testing.T) {
	m := NewModel()
	m.SetShowAllRemotes(true)
	if !m.GetShowAllRemotes() {
		t.Fatal("SetShowAllRemotes(true) did not take effect")
	}
	m.SetShowAllRemotes(false)
	if m.GetShowAllRemotes() {
		t.Fatal("SetShowAllRemotes(false) did not take effect")
	}
}
