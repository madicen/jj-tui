// Package longpress holds shared mouse-gesture helpers for every long-press
// menu in the TUI (graph file/commit context menus, branches/PRs/tickets
// context menus, and the AI generate-button popover provided by genmenu).
//
// Historically each long-press handler cancelled its armed press on the
// first MouseActionMotion. That made the gesture unforgiving: trackpad
// jitter, cell-boundary grazes, or even one stray motion event between
// MouseActionPress and the threshold tick would tear down the press, so a
// user holding the mouse "still" could never reach the 500ms threshold.
//
// StillArmed centralises a two-rule tolerance:
//
//   - If the cursor is still inside the originating zone, stay armed
//     (primary check — bubblezone gives us authoritative hit-tests).
//   - Otherwise, if the cursor has drifted by at most MotionSlack cells from
//     the press anchor, stay armed (fallback for terminals that briefly
//     report sub-cell motion, or pre-layout frames where the zone manager
//     doesn't yet know about the originating region).
//
// Anything outside both conditions returns false so callers can cancel the
// press and let drag/scroll gestures elsewhere in the UI proceed.
package longpress

import (
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
)

// MotionSlack is the maximum cell distance (Chebyshev) the cursor may drift
// from the press anchor while still keeping a long-press armed when the
// originating zone can't confirm in-bounds. Two cells is generous in a TUI
// without enabling a real drag — most "the user is holding still" motion
// events fall within one cell.
const MotionSlack = 2

// StillArmed reports whether a long-press anchored at (anchorX, anchorY)
// over originZone should remain armed given the motion event msg. originZone
// may be empty when no zone id is associated with the press; in that case
// only the slack box around the anchor is used. zm may be nil — same
// behavior as an empty originZone.
func StillArmed(zm *zone.Manager, originZone string, anchorX, anchorY int, msg tea.MouseMsg) bool {
	if zm != nil && originZone != "" {
		if z := zm.Get(originZone); z != nil && z.InBounds(msg) {
			return true
		}
	}
	dx := msg.X - anchorX
	if dx < 0 {
		dx = -dx
	}
	dy := msg.Y - anchorY
	if dy < 0 {
		dy = -dy
	}
	return dx <= MotionSlack && dy <= MotionSlack
}
