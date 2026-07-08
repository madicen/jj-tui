package model

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// undo_hint.go implements the status-line undo hint (P5.4). After a mutating jj
// command completes and the repo reloads, applyRepositoryLoaded fires
// fetchUndoHintCmd, which cheaply reads the latest operation description
// (`jj op log --limit 1`). The result is shown as "Ctrl+z undoes: <op>" in the
// status bar for undoHintDuration, then expires via a tea.Tick. A monotonically
// increasing sequence number makes a newer hint cancel any pending expiry so a
// stale tick can't clear a fresh hint early.

// undoHintDuration is how long the "Ctrl+z undoes: …" hint stays in the footer.
const undoHintDuration = 6 * time.Second

// undoHintReadyMsg carries the latest operation description fetched after a
// mutating command.
type undoHintReadyMsg struct {
	Desc string
}

// undoHintExpiredMsg clears the hint if Seq still matches the current hint.
type undoHintExpiredMsg struct {
	Seq int
}

// fetchUndoHintCmd reads the most recent operation description and turns it into
// an undoHintReadyMsg. It is a no-op (nil) without a jj service.
func (m *Model) fetchUndoHintCmd() tea.Cmd {
	svc := m.appState.JJService
	if svc == nil {
		return nil
	}
	return func() tea.Msg {
		desc, err := svc.LatestOperationDescription(context.Background())
		if err != nil || desc == "" {
			return nil
		}
		return undoHintReadyMsg{Desc: desc}
	}
}

// handleUndoHintReady sets the hint text and schedules its expiry.
func (m *Model) handleUndoHintReady(msg undoHintReadyMsg) (tea.Model, tea.Cmd) {
	if msg.Desc == "" {
		return m, nil
	}
	m.undoHint = "Ctrl+z undoes: " + msg.Desc
	m.undoHintSeq++
	seq := m.undoHintSeq
	return m, tea.Tick(undoHintDuration, func(time.Time) tea.Msg {
		return undoHintExpiredMsg{Seq: seq}
	})
}

// handleUndoHintExpired clears the hint when its expiry tick is the current one.
func (m *Model) handleUndoHintExpired(msg undoHintExpiredMsg) (tea.Model, tea.Cmd) {
	if msg.Seq == m.undoHintSeq {
		m.undoHint = ""
	}
	return m, nil
}
