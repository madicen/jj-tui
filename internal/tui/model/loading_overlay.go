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

// applyLoadingOverlay composites a centered busy box over the full main layout when Loading.
func (m *Model) applyLoadingOverlay(fullView string) string {
	if !m.appState.Loading || m.width <= 0 || m.height <= 0 {
		return fullView
	}
	msg := m.appState.StatusMessage
	if msg == "" {
		msg = "Loading…"
	}
	if !shouldShowLoadingOverlay(m.appState.ViewMode, msg) {
		return fullView
	}
	msg = strings.ReplaceAll(msg, "\n", " ")
	line := lipgloss.JoinHorizontal(lipgloss.Center, m.busySpinner.View(), " ", msg)
	maxOuter := m.width - 4
	if maxOuter < 1 {
		maxOuter = 1
	}
	box := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(styles.ColorMuted).
		Padding(0, 0).
		MaxWidth(maxOuter).
		Background(styles.HeaderBarBackground).
		Render(line)
	boxLines := strings.Split(box, "\n")
	modalH := len(boxLines)
	modalW := 0
	for _, l := range boxLines {
		if w := lipgloss.Width(l); w > modalW {
			modalW = w
		}
	}
	top := max((m.height-modalH)/2, 0)
	if isUpdatePRPushLoadingStatus(msg) {
		top = max(top-loadingOverlayRaiseUpdatePRRows, 0)
	}
	left := max((m.width-modalW)/2, 0)
	return overlay.OverlayView(fullView, box, m.width, m.height, top, left)
}
