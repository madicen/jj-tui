package mousedouble

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestOverlapRelease(t *testing.T) {
	var o OverlapRelease
	var pressGen uint64 = 1
	e := tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease, X: 3, Y: 4}

	if o.ShouldSkipOverlappingRelease(e, pressGen) {
		t.Fatal("first release should not skip")
	}
	if !o.ShouldSkipOverlappingRelease(e, pressGen) {
		t.Fatal("second overlapping release should skip")
	}

	OnLeftPress(&pressGen) // 2
	e2 := tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease, X: 3, Y: 4}
	if o.ShouldSkipOverlappingRelease(e2, pressGen) {
		t.Fatal("first release after new press should not skip")
	}
	if !o.ShouldSkipOverlappingRelease(e2, pressGen) {
		t.Fatal("second overlapping after new press should skip")
	}
}

func TestDoubleClick(t *testing.T) {
	var d DoubleClick
	t0 := time.Unix(1000, 0)
	e := tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease}

	if d.ObserveLeftRelease("row:1", e, t0, DefaultDoubleClickWindow) {
		t.Fatal("first click should not be double")
	}
	if !d.ObserveLeftRelease("row:1", e, t0.Add(100*time.Millisecond), DefaultDoubleClickWindow) {
		t.Fatal("second release same key within window should be double")
	}
	if d.ObserveLeftRelease("row:1", e, t0.Add(200*time.Millisecond), DefaultDoubleClickWindow) {
		t.Fatal("right after double, cleared state: next release is first of a new pair")
	}
	if d.ObserveLeftRelease("row:2", e, t0.Add(300*time.Millisecond), DefaultDoubleClickWindow) {
		t.Fatal("different key is never double on first observe")
	}
	if d.ObserveLeftRelease("row:2", e, t0.Add(DefaultDoubleClickWindow+time.Second), DefaultDoubleClickWindow) {
		t.Fatal("second observe same key after long gap should not be double")
	}
	if !d.ObserveLeftRelease("row:2", e, t0.Add(DefaultDoubleClickWindow+time.Second+50*time.Millisecond), DefaultDoubleClickWindow) {
		t.Fatal("third observe: second click within window after gap should double")
	}
}

func TestObserveLeftReleaseEmptyKey(t *testing.T) {
	var d DoubleClick
	e := tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease}
	if d.ObserveLeftRelease("", e, time.Now(), DefaultDoubleClickWindow) {
		t.Fatal("empty key should not count as double")
	}
}
