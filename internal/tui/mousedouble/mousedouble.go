// Package mousedouble provides time-based double-click detection and deduplication
// for overlapping bubblezone hits on the same physical mouse release.
package mousedouble

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// DefaultDoubleClickWindow is the max delay between two left releases for a double-click.
const DefaultDoubleClickWindow = 450 * time.Millisecond

// IsLeftRelease reports whether e is a left-button release (zone routing uses release).
func IsLeftRelease(e tea.MouseMsg) bool {
	return e.Button == tea.MouseButtonLeft && e.Action == tea.MouseActionRelease
}

// OnLeftPress increments the press generation so release dedupe and double-click
// state can distinguish the next release from prior ones. Call from the tab's
// tea.MouseMsg handler on MouseActionPress with the left button.
func OnLeftPress(pressGen *uint64) {
	*pressGen++
}

// OverlapRelease tracks one zone-handling pass per physical left release so
// AnyInBoundsAndUpdate does not run list logic twice when multiple zones overlap.
type OverlapRelease struct {
	gen    uint64
	x, y   int
	active bool
}

// ShouldSkipOverlappingRelease returns true for the 2nd+ zone message with the same
// (pressGen, x, y) as an earlier handled message in the same release batch.
// On the first call for a release, it records (pressGen, x, y) and returns false.
func (o *OverlapRelease) ShouldSkipOverlappingRelease(e tea.MouseMsg, pressGen uint64) bool {
	if !IsLeftRelease(e) {
		return false
	}
	if o.active && o.gen == pressGen && o.x == e.X && o.y == e.Y {
		return true
	}
	o.active = true
	o.gen = pressGen
	o.x = e.X
	o.y = e.Y
	return false
}

// DoubleClick tracks the previous left release per logical key for double-click detection.
type DoubleClick struct {
	lastKey  string
	lastTime time.Time
}

// ObserveLeftRelease returns true if this release is a double-click on the same key
// within window of the previous qualifying release. After a double-click, state is
// cleared so a third release does not immediately count as another double.
func (d *DoubleClick) ObserveLeftRelease(key string, e tea.MouseMsg, now time.Time, window time.Duration) bool {
	if !IsLeftRelease(e) {
		return false
	}
	if key == "" {
		return false
	}
	if d.lastKey == key && !d.lastTime.IsZero() && now.Sub(d.lastTime) <= window {
		d.lastKey = ""
		d.lastTime = time.Time{}
		return true
	}
	d.lastKey = key
	d.lastTime = now
	return false
}
