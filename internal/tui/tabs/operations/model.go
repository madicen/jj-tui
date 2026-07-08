// Package operations implements the operation-log browser (P4.1): a modal that
// lists recent jj operations and lets the user "time travel" by restoring the
// repository to any past operation. It reuses internal/tui/listnav for the
// shared scroll plumbing and follows the inline y/n confirm pattern used by the
// other destructive flows.
package operations

import (
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui/listnav"
	"github.com/madicen/jj-tui/internal/tui/state"
	"github.com/madicen/jj-tui/internal/tui/styles"
	"github.com/mattn/go-runewidth"
)

type mode int

const (
	modeNormal mode = iota
	modeConfirmingRestore
)

// Model is the operation-log browser modal: it lists operations (newest first,
// current flagged) and confirms a restore before time-traveling.
type Model struct {
	listnav.Model // shared list scroll state

	shown        bool
	operations   []jj.Operation
	selectedIdx  int
	termW, termH int

	mode          mode
	restoreOpID   string // operation pending restore confirmation
	restoreOpDesc string
}

// NewModel creates a new operations modal.
func NewModel() Model {
	return Model{
		Model: listnav.New(),
		termW: 100,
		termH: 24,
	}
}

// SetDimensions records terminal size.
func (m Model) SetDimensions(w, h int) Model {
	if w < 1 {
		w = 80
	}
	if h < 1 {
		h = 24
	}
	m.termW, m.termH = w, h
	return m
}

// IsShown reports whether the modal is displayed.
func (m *Model) IsShown() bool { return m.shown }

// Show displays the modal with the given operations.
func (m *Model) Show(ops []jj.Operation) {
	m.shown = true
	m.mode = modeNormal
	m.restoreOpID = ""
	m.restoreOpDesc = ""
	m.selectedIdx = 0
	m.YOffset = 0
	m.SetOperations(ops)
}

// SetOperations replaces the list, clamping the selection.
func (m *Model) SetOperations(ops []jj.Operation) {
	m.operations = append([]jj.Operation(nil), ops...)
	if m.selectedIdx >= len(m.operations) {
		m.selectedIdx = max(0, len(m.operations)-1)
	}
	if m.selectedIdx < 0 {
		m.selectedIdx = 0
	}
}

// Hide hides the modal.
func (m *Model) Hide() { m.shown = false }

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update handles messages for the operations modal.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.shown {
		return m, nil
	}
	if key, ok := msg.(tea.KeyMsg); ok {
		return m.handleKeyMsg(key)
	}
	return m, nil
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.mode == modeConfirmingRestore {
		return m.handleConfirmKey(msg)
	}
	return m.handleNormalKey(msg)
}

func (m Model) handleNormalKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.shown = false
		return m, state.NavigateTarget{Kind: state.NavigateCloseOperations, StatusMessage: "Closed operations"}.Cmd()
	case "j", "down":
		if m.selectedIdx < len(m.operations)-1 {
			m.selectedIdx++
			m.ScrollToSelected(m.selectedIdx, len(m.operations), m.listHeight())
		}
		return m, nil
	case "k", "up":
		if m.selectedIdx > 0 {
			m.selectedIdx--
			m.ScrollToSelected(m.selectedIdx, len(m.operations), m.listHeight())
		}
		return m, nil
	case "g", "home":
		m.selectedIdx = 0
		m.ScrollToSelected(m.selectedIdx, len(m.operations), m.listHeight())
		return m, nil
	case "G", "end":
		m.selectedIdx = max(0, len(m.operations)-1)
		m.ScrollToSelected(m.selectedIdx, len(m.operations), m.listHeight())
		return m, nil
	case "enter":
		if op, ok := m.selected(); ok {
			if op.IsCurrent {
				// Restoring to the current op is a no-op; skip the confirm.
				return m, nil
			}
			m.mode = modeConfirmingRestore
			m.restoreOpID = op.ID
			m.restoreOpDesc = op.Description
		}
		return m, nil
	}
	return m, nil
}

func (m Model) handleConfirmKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		id := m.restoreOpID
		m.mode = modeNormal
		m.restoreOpID = ""
		m.restoreOpDesc = ""
		if id == "" {
			return m, nil
		}
		m.shown = false
		return m, state.NavigateTarget{Kind: state.NavigateRestoreOperation, OperationID: id}.Cmd()
	case "n", "N", "esc":
		m.mode = modeNormal
		m.restoreOpID = ""
		m.restoreOpDesc = ""
		return m, nil
	}
	return m, nil
}

func (m Model) selected() (jj.Operation, bool) {
	if m.selectedIdx < 0 || m.selectedIdx >= len(m.operations) {
		return jj.Operation{}, false
	}
	return m.operations[m.selectedIdx], true
}

// listHeight is the number of operation rows visible in the scroll window,
// derived from the terminal height and leaving room for chrome + hint lines.
func (m Model) listHeight() int {
	h := m.termH - 10
	if h < 3 {
		h = 3
	}
	return h
}

// modalWidth clamps the modal to a comfortable range of the terminal width.
func (m Model) modalWidth() int {
	return min(max(48, m.termW-8), 96)
}

// View renders the operations modal body (the window title is supplied by chrome).
func (m Model) View() string {
	if !m.shown {
		return ""
	}
	muted := lipgloss.NewStyle().Foreground(styles.ColorMuted)
	idStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD"))
	currentStyle := lipgloss.NewStyle().Foreground(styles.ColorSecondary).Bold(true)
	selStyle := lipgloss.NewStyle().Foreground(styles.ColorPrimary).Bold(true)

	modalW := m.modalWidth()

	var lines []string
	lines = append(lines, muted.Render("j/k select · Enter restore · g/G top/bottom · Esc close"))
	lines = append(lines, "")

	if len(m.operations) == 0 {
		lines = append(lines, muted.Render("(no operations)"))
	}

	rows := m.operationRows(modalW, idStyle, currentStyle, selStyle, muted)
	start, end := m.VisibleRange(len(rows), m.listHeight())
	if start < end {
		lines = append(lines, rows[start:end]...)
	}
	if len(rows) > m.listHeight() {
		lines = append(lines, muted.Render(scrollHint(m.selectedIdx, len(rows))))
	}

	if m.mode == modeConfirmingRestore {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorPrimary).
			Render("Restore to operation "+shortID(m.restoreOpID)+"? (y/n)"))
		desc := strings.TrimSpace(m.restoreOpDesc)
		if desc != "" {
			lines = append(lines, muted.Render("  "+desc))
		}
		lines = append(lines, muted.Render("This is itself an operation — undo with Ctrl+z."))
	}

	outer := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorMuted).
		Padding(0, 1).
		Width(modalW)
	return outer.Render(strings.Join(lines, "\n"))
}

func (m Model) operationRows(modalW int, idStyle, currentStyle, selStyle, muted lipgloss.Style) []string {
	rows := make([]string, 0, len(m.operations))
	for i, op := range m.operations {
		prefix := "  "
		if i == m.selectedIdx {
			prefix = selStyle.Render("► ")
		}
		id := idStyle.Render(shortID(op.ID))
		header := prefix + id
		if op.IsCurrent {
			header += " " + currentStyle.Render("(current)")
		}
		desc := strings.TrimSpace(op.Description)
		if desc == "" {
			desc = "(no description)"
		}
		metaMax := max(8, modalW-12)
		meta := runewidth.Truncate(desc, metaMax, "…")
		when := strings.TrimSpace(op.Time)
		metaLine := "    " + meta
		if when != "" {
			metaLine += muted.Render("  " + when)
		}
		row := lipgloss.JoinVertical(lipgloss.Left, header, muted.Render(metaLine))
		rows = append(rows, row)
	}
	return rows
}

// shortID trims an operation id to a compact display form.
func shortID(id string) string {
	id = strings.TrimSpace(id)
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func scrollHint(selected, total int) string {
	return "  · " + strconv.Itoa(selected+1) + "/" + strconv.Itoa(total) + " ·"
}
