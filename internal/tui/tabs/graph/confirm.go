package graph

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/tui/state"
)

// destructiveConfirm is a pending y/n confirmation for a destructive graph request
// (abandon / backout). It is only created when the ui.confirm_destructive toggle is on.
type destructiveConfirm struct {
	req    Request
	prompt string
}

// maybeConfirmDestructive returns true (and stores a pending confirmation) when req is a
// destructive operation and the ui.confirm_destructive toggle is on. Callers that get true
// must NOT run the request yet — the graph now shows a y/n prompt and will run it once the
// user presses y. When the toggle is off (or req is not destructive) it returns false and the
// caller runs the request immediately, preserving the pre-P5.3 behavior.
func (m *GraphModel) maybeConfirmDestructive(req Request, app *state.AppState) bool {
	// ConfirmDestructiveOps is nil-safe on a nil *Config, defaulting to true (confirm).
	if !app.Config.ConfirmDestructiveOps() {
		return false
	}
	prompt := m.destructiveRequestPrompt(req)
	if prompt == "" {
		return false
	}
	m.confirm = &destructiveConfirm{req: req, prompt: prompt}
	app.StatusMessage = prompt
	return true
}

// resolveConfirm handles a key press while a destructive confirmation is pending. It returns
// (request, true) when the user confirmed (y/Y) so the caller runs it; otherwise it returns
// (_, false) — cancelling on n/N/Esc/q and swallowing every other key so the confirmation
// can't be dismissed by accident.
func (m *GraphModel) resolveConfirm(msg tea.KeyMsg, app *state.AppState) (Request, bool) {
	switch msg.String() {
	case "y", "Y":
		req := m.confirm.req
		m.confirm = nil
		return req, true
	case "n", "N", "esc", "q":
		m.confirm = nil
		app.StatusMessage = "Cancelled"
		return Request{}, false
	default:
		// Any other key is ignored while confirming (do not fall through to normal handling).
		return Request{}, false
	}
}

// destructiveRequestPrompt returns a one-line consequence prompt for a destructive request,
// or "" if req needs no confirmation. The revision short id is included when available.
func (m *GraphModel) destructiveRequestPrompt(req Request) string {
	shortID := m.selectedCommitShortID()
	switch {
	case req.Abandon:
		if shortID != "" {
			return fmt.Sprintf("Abandon revision %s? Undo with Ctrl+z  ·  y = yes, n = cancel", shortID)
		}
		return "Abandon this revision? Undo with Ctrl+z  ·  y = yes, n = cancel"
	case req.BatchAbandon:
		ids := m.multiSelectShortIDs()
		if len(ids) > 0 {
			return fmt.Sprintf("Abandon %d revisions (%s)? One undo restores all  ·  y = yes, n = cancel",
				len(ids), strings.Join(ids, ", "))
		}
		return "Abandon selected revisions? Undo with Ctrl+z  ·  y = yes, n = cancel"
	case req.Backout:
		if shortID != "" {
			return fmt.Sprintf("Back out revision %s? Creates a revert commit; undo with Ctrl+z  ·  y = yes, n = cancel", shortID)
		}
		return "Back out this revision? Creates a revert commit; undo with Ctrl+z  ·  y = yes, n = cancel"
	}
	return ""
}

// selectedCommitShortID returns the short id of the selected commit, or "" if none.
func (m *GraphModel) selectedCommitShortID() string {
	if m.repository != nil && m.selectedCommit >= 0 && m.selectedCommit < len(m.repository.Graph.Commits) {
		return m.repository.Graph.Commits[m.selectedCommit].ShortID
	}
	return ""
}
