package model

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	overlay "github.com/madicen/bubble-overlay"
	"github.com/madicen/jj-tui/internal/tui/state"
	"github.com/madicen/jj-tui/internal/tui/styles"
)

// isAIGenBusyOverlayView is true for modals that use Ctrl+G / sparkles AI generation.
func isAIGenBusyOverlayView(viewMode state.ViewMode) bool {
	switch viewMode {
	case state.ViewEditDescription, state.ViewCreatePR, state.ViewCreateTicket, state.ViewCreateBookmark:
		return true
	default:
		return false
	}
}

// Rows to shift the busy overlay upward when showing graph "Update PR" push status
// (slightly above vertical center so it sits clearer above the graph/actions area).
const loadingOverlayRaiseUpdatePRRows = 4

func isUpdatePRPushLoadingStatus(msg string) bool {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return false
	}
	if strings.Contains(msg, "and pushing...") {
		return true
	}
	return strings.HasPrefix(msg, "Pushing ") && strings.HasSuffix(msg, "...") && !strings.Contains(msg, "creating PR")
}

func shouldShowLoadingOverlay(viewMode state.ViewMode, msg string) bool {
	trimmed := strings.TrimSpace(msg)
	// Blocking pickers: never composite a centered busy box on top of these (see View() layer order).
	if viewMode == state.ViewBookmarkConflict || viewMode == state.ViewDivergentCommit {
		return false
	}
	// Full-screen file diff draws above this overlay anyway; skipping avoids a misleading box if
	// StatusMessage was overwritten while another operation left Loading true.
	if viewMode == state.ViewFileDiff {
		return false
	}
	// Background PR polling can set Loading while user is not on PR tab.
	if strings.HasPrefix(trimmed, "Loading pull requests") && viewMode != state.ViewPullRequests {
		return false
	}
	// Ticket preload should not block other tabs.
	if strings.HasPrefix(trimmed, "Loading tickets") && viewMode != state.ViewTickets {
		return false
	}
	return true
}

// showCenteredBusyOverlay is true when the main layout should composite the spinner box
// (either global Loading with shouldShowLoadingOverlay rules, or in-flight AI generation on a form modal).
func (m *Model) showCenteredBusyOverlay() bool {
	if m.width <= 0 || m.height <= 0 {
		return false
	}
	if m.appState.Loading && shouldShowLoadingOverlay(m.appState.ViewMode, m.appState.StatusMessage) {
		return true
	}
	return m.aiGenOverlayActive && isAIGenBusyOverlayView(m.appState.ViewMode)
}

func newBusySpinner() spinner.Model {
	return spinner.New(
		spinner.WithSpinner(spinner.MiniDot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(styles.ColorSecondary)),
	)
}

func (m *Model) startBusySpinnerCmd() tea.Cmd {
	return func() tea.Msg {
		return m.busySpinner.Tick()
	}
}

// wrapGraphTabCmd batches the graph tab's follow-up with a spinner tick when Loading was set
// (e.g. ApplyResult for Update PR). UpdateWithApp never goes through processGraphRequest, so main
// must attach the spinner here. Tick is scheduled before cmd so one frame can render before a
// synchronous push runs.
func (m *Model) wrapGraphTabCmd(cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	if m.appState.Loading {
		return tea.Batch(m.startBusySpinnerCmd(), cmd)
	}
	return cmd
}

// snapshotSpinnerMessage locks the spinner's caption to whatever StatusMessage was at the
// moment a busy state (Loading or aiGenOverlayActive) flipped on, and clears that lock
// when both flip off. Called via a deferred observer at the top of Model.Update so the
// snapshot reflects the final state after the message has been fully processed (including
// any cascaded submodel updates).
//
// aiGenOverlayActive is treated the same as Loading here because the AI generate flow
// (Ctrl+G on description / PR / ticket / bookmark modals) shows the same centered busy
// overlay but does not set Loading=true; without locking, background StatusMessage writes
// (e.g. "Loaded 14 PRs" from a PR poll completing) would overwrite the spinner caption.
//
// The "busy=true with empty SpinnerMessage" branch is a safety net: it catches initial
// app state where Loading might start true before the first Update arrives, and also any
// future caller that flips Loading via a path other than Model.Update.
func (m *Model) snapshotSpinnerMessage() {
	if !m.appState.Loading && !m.aiGenOverlayActive {
		m.appState.SpinnerMessage = ""
		return
	}
	if m.appState.SpinnerMessage == "" {
		m.appState.SpinnerMessage = m.appState.StatusMessage
	}
}

// applyLoadingOverlay composites a centered busy box over the full main layout when Loading
// or when an AI generate request is in flight (aiGenOverlayActive on form modals).
func (m *Model) applyLoadingOverlay(fullView string) string {
	if !m.showCenteredBusyOverlay() {
		return fullView
	}
	// Prefer the locked spinner caption (snapshot taken at the false→true edge of Loading
	// or aiGenOverlayActive); fall back to StatusMessage only as a safety net if the
	// snapshot hasn't been taken yet (e.g. very first frame).
	msg := m.appState.SpinnerMessage
	if msg == "" {
		msg = m.appState.StatusMessage
	}
	if msg == "" {
		msg = "Loading…"
	}
	msg = strings.ReplaceAll(msg, "\n", " ")
	line := lipgloss.JoinHorizontal(lipgloss.Center, m.busySpinner.View(), " ", msg)
	maxOuter := max(m.width-4, 1)
	box := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(styles.ColorMuted).
		Padding(0, 0).
		MaxWidth(maxOuter).
		Background(styles.HeaderBarBackground).
		Render(line)
	deltaTop, deltaLeft := 0, 0
	if isUpdatePRPushLoadingStatus(msg) {
		deltaTop = -loadingOverlayRaiseUpdatePRRows
	}
	return overlay.OverlayViewInCenterWithOffset(fullView, box, m.width, m.height, deltaTop, deltaLeft)
}
