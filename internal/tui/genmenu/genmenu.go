// Package genmenu provides a reusable long-press popover for AI generate
// buttons across modals (desc-edit, PR form, bookmark form, ticket form).
//
// The popover lists every configured AI profile from config.Config and lets
// the user pick a non-active profile for a one-shot generation (the active
// default is unchanged). Behavior mirrors the existing graph file context
// menu in internal/tui/tabs/graph/context_menu.go: a 500ms tea.Tick after a
// left-button press on a `Zone*Generate` zone opens the menu; motion before
// the tick fires cancels the press; release on a row resolves the choice.
package genmenu

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/styles"
)

// LongPressThreshold is how long a left-button press must be held over a
// generate-button zone before the profile picker opens. Matches the graph
// file context menu's threshold so the gesture feels consistent.
const LongPressThreshold = 500 * time.Millisecond

// MotionSlack is how many terminal cells the mouse may drift from the press
// anchor while still keeping the long-press armed (used only when the zone
// lookup can't confirm in-bounds, e.g. before the zone manager has laid out
// or when the cursor briefly grazes a cell border). The primary cancel rule
// is "mouse left the originating zone"; this slack just smooths over a few
// stray motion events that terminals emit when the user is essentially
// holding still. Two cells is generous in a TUI without enabling drag.
const MotionSlack = 2

// TickMsg fires after LongPressThreshold to open the menu, gated by PressID
// so a release / new press before the tick fires does not reopen the menu.
type TickMsg struct {
	// Owner identifies which modal scheduled this tick (the modal's zone id),
	// so multiple modals can coexist without crossing wires when both happen
	// to be on screen.
	Owner   string
	PressID int
}

// State holds the long-press + menu visibility for one generate button.
// Initialise to its zero value; nothing to construct.
type State struct {
	// pressID is bumped on every press over the generate zone. It's matched
	// against the PressID on TickMsg so we ignore stale ticks (release happened
	// before the threshold, then another press started).
	pressID int
	// pressActive is true between MouseActionPress and the matching release
	// (or motion-cancel). The tick only opens the menu when pressActive is
	// still set with the same pressID.
	pressActive bool
	// pressOwner is the zone id of the generate button that started this press.
	// Carried into TickMsg so multiple modals with their own State don't react
	// to each other's ticks.
	pressOwner string
	// mouseX / mouseY anchor the menu overlay on the press location.
	mouseX, mouseY int
	// shown is true while the popover is visible.
	shown bool
	// hoverIdx is the row index under the mouse while shown (-1 = none).
	hoverIdx int
}

// IsShown reports whether the popover is currently visible.
func (s *State) IsShown() bool { return s != nil && s.shown }

// IsArmed reports whether a press is currently pending (i.e. the user has
// pressed but neither the tick nor a release has fired yet). Callers can use
// this to suppress the existing click-to-generate behavior so a long-press
// that opens the menu does not also fire a click on release.
func (s *State) IsArmed() bool { return s != nil && s.pressActive }

// MouseAnchor returns the (x, y) terminal coordinate where the popover should
// be drawn. Use with bubble-overlay's OverlayViewAtPoint.
func (s *State) MouseAnchor() (int, int) {
	if s == nil {
		return 0, 0
	}
	return s.mouseX, s.mouseY
}

// HoverIndex returns the index of the row currently hovered, or -1 if none.
func (s *State) HoverIndex() int {
	if s == nil {
		return -1
	}
	return s.hoverIdx
}

// Reset clears press / shown state. Called when the modal closes or after the
// menu emits a NavigateGenerate*.
func (s *State) Reset() {
	if s == nil {
		return
	}
	s.pressActive = false
	s.pressOwner = ""
	s.shown = false
	s.hoverIdx = -1
}

// BeginPress records a press over the generate-button zone and returns a
// tea.Cmd that fires TickMsg after LongPressThreshold. zoneID identifies the
// owning button (carried through TickMsg.Owner). msg supplies the press
// coordinates used to anchor the popover.
func (s *State) BeginPress(zoneID string, msg tea.MouseMsg) tea.Cmd {
	if s == nil {
		return nil
	}
	s.pressID++
	s.pressActive = true
	s.pressOwner = zoneID
	s.mouseX = msg.X
	s.mouseY = msg.Y
	s.hoverIdx = -1
	pressID := s.pressID
	owner := zoneID
	return tea.Tick(LongPressThreshold, func(time.Time) tea.Msg {
		return TickMsg{Owner: owner, PressID: pressID}
	})
}

// CancelPress drops any pending tick (a release happened before the
// threshold, or a non-motion cancel is needed). Does nothing when the menu
// is already shown. Prefer OnMotion for the motion path so small drift
// inside the originating zone does not cancel the gesture.
func (s *State) CancelPress() {
	if s == nil || s.shown {
		return
	}
	s.pressActive = false
	s.pressOwner = ""
}

// OnMotion is the motion-event handler during the armed window. It keeps the
// press armed when the mouse stays over the originating zone, or — as a
// fallback when the zone manager can't confirm — when the cursor is still
// within MotionSlack cells of the press anchor. Anything outside both
// conditions cancels the press so the user can still click-drag other UI.
//
// This is what lets a couple of stray motion events (cell-boundary grazes,
// trackpad jitter) coexist with a held long-press without forcing the user
// to keep the mouse perfectly still.
func (s *State) OnMotion(zoneManager *zone.Manager, msg tea.MouseMsg) {
	if s == nil || s.shown || !s.pressActive {
		return
	}
	if zoneManager != nil && s.pressOwner != "" {
		if z := zoneManager.Get(s.pressOwner); z != nil && z.InBounds(msg) {
			return
		}
	}
	dx := msg.X - s.mouseX
	if dx < 0 {
		dx = -dx
	}
	dy := msg.Y - s.mouseY
	if dy < 0 {
		dy = -dy
	}
	if dx <= MotionSlack && dy <= MotionSlack {
		return
	}
	s.pressActive = false
	s.pressOwner = ""
}

// OpenIfMatches opens the menu if t belongs to this State's most recent press
// and the press is still active. Returns true when the menu transitioned to
// shown, so the caller can re-render. Returns false (no-op) when the tick is
// stale (different owner or pressID) or the user already released.
func (s *State) OpenIfMatches(t TickMsg) bool {
	if s == nil {
		return false
	}
	if !s.pressActive || s.shown {
		return false
	}
	if t.Owner != s.pressOwner || t.PressID != s.pressID {
		return false
	}
	s.shown = true
	s.hoverIdx = -1
	return true
}

// UpdateHover recomputes hoverIdx by hit-testing msg against the rendered
// ZoneGenMenuItem rows up to count. Called on MouseActionMotion / Press while
// IsShown(). No-op when the menu isn't visible.
func (s *State) UpdateHover(zoneManager *zone.Manager, msg tea.MouseMsg, count int) {
	if s == nil || !s.shown || zoneManager == nil {
		return
	}
	hit := -1
	for i := 0; i < count; i++ {
		z := zoneManager.Get(mouse.ZoneGenMenuItem(i))
		if z != nil && z.InBounds(msg) {
			hit = i
			break
		}
	}
	s.hoverIdx = hit
}

// HitTestRelease returns the row index released over, or -1 when the release
// landed outside any row. Always closes the menu before returning so the
// caller can re-render (the menu must always close on release per the design).
func (s *State) HitTestRelease(zoneManager *zone.Manager, msg tea.MouseMsg, count int) int {
	if s == nil || !s.shown {
		return -1
	}
	hit := -1
	if zoneManager != nil {
		for i := 0; i < count; i++ {
			z := zoneManager.Get(mouse.ZoneGenMenuItem(i))
			if z != nil && z.InBounds(msg) {
				hit = i
				break
			}
		}
	}
	s.shown = false
	s.hoverIdx = -1
	s.pressActive = false
	s.pressOwner = ""
	return hit
}

// Close dismisses the popover (e.g. on Esc). Does nothing when not shown.
func (s *State) Close() bool {
	if s == nil || !s.shown {
		return false
	}
	s.shown = false
	s.hoverIdx = -1
	s.pressActive = false
	s.pressOwner = ""
	return true
}

// Render returns the styled, zone-marked popover string. profiles is the list
// to show (typically cfg.AIProfileList()); activeName marks the row that is
// the current persistent active profile with a filled dot. zoneManager is
// used to mark each row so HitTestRelease can resolve clicks.
func Render(zoneManager *zone.Manager, profiles []config.AIProfile, activeName string, hoverIdx int) string {
	menuBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorPrimary).
		Padding(0, 1)

	rowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F8F8F2"))
	dimStyle := lipgloss.NewStyle().Foreground(styles.ColorMuted)
	hoverStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F8F8F2")).Background(styles.ColorPrimary)
	hoverDim := lipgloss.NewStyle().Foreground(lipgloss.Color("#CCCCCC")).Background(styles.ColorPrimary)
	activeMark := lipgloss.NewStyle().Foreground(styles.ColorSecondary)

	header := lipgloss.NewStyle().
		Foreground(styles.ColorSecondary).
		Bold(true).
		Render("AI profile")

	rows := make([]string, 0, len(profiles))
	for i, p := range profiles {
		mark := "  "
		if strings.EqualFold(p.Name, activeName) {
			mark = activeMark.Render("● ")
		}
		isHover := i == hoverIdx
		nameStyle := rowStyle
		summaryStyle := dimStyle
		if isHover {
			nameStyle = hoverStyle
			summaryStyle = hoverDim
		}
		name := nameStyle.Render(fmt.Sprintf("%s%s", mark, p.Name))
		summary := summaryStyle.Render("  " + p.Summary())
		row := name + summary
		if zoneManager != nil {
			row = zoneManager.Mark(mouse.ZoneGenMenuItem(i), row)
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		empty := dimStyle.Render("  (no profiles configured)")
		return menuBorder.Render(header + "\n" + empty)
	}
	hint := dimStyle.Render("hold to open · click to use once")
	return menuBorder.Render(header + "\n" + strings.Join(rows, "\n") + "\n" + hint)
}
