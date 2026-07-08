package workspaces

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui/state"
	"github.com/madicen/jj-tui/internal/tui/styles"
	"github.com/mattn/go-runewidth"
)

type mode int

const (
	modeNormal mode = iota
	modeAdding
	modeConfirmingForget
)

// Model is the workspaces view-only MVP modal: it lists workspaces, marks the
// current one, and supports add (path input) / forget (confirm) actions.
type Model struct {
	shown        bool
	workspaces   []jj.Workspace
	selectedIdx  int
	termW, termH int

	mode       mode
	addInput   textinput.Model
	forgetName string // workspace pending forget confirmation
}

// NewModel creates a new workspaces modal.
func NewModel() Model {
	ti := textinput.New()
	ti.Placeholder = "../my-workspace"
	ti.Prompt = "› "
	ti.CharLimit = 512
	return Model{
		termW:    100,
		termH:    24,
		addInput: ti,
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

// Show displays the modal with the given workspaces.
func (m *Model) Show(ws []jj.Workspace) {
	m.shown = true
	m.mode = modeNormal
	m.forgetName = ""
	m.addInput.Blur()
	m.addInput.SetValue("")
	m.SetWorkspaces(ws)
}

// SetWorkspaces replaces the list, clamping the selection.
func (m *Model) SetWorkspaces(ws []jj.Workspace) {
	m.workspaces = append([]jj.Workspace(nil), ws...)
	if m.selectedIdx >= len(m.workspaces) {
		m.selectedIdx = max(0, len(m.workspaces)-1)
	}
	if m.selectedIdx < 0 {
		m.selectedIdx = 0
	}
}

// Hide hides the modal.
func (m *Model) Hide() { m.shown = false }

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update handles messages for the workspaces modal.
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
	switch m.mode {
	case modeAdding:
		return m.handleAddingKey(msg)
	case modeConfirmingForget:
		return m.handleForgetConfirmKey(msg)
	default:
		return m.handleNormalKey(msg)
	}
}

func (m Model) handleNormalKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.shown = false
		return m, state.NavigateTarget{Kind: state.NavigateCloseWorkspaces, StatusMessage: "Closed workspaces"}.Cmd()
	case "j", "down":
		if m.selectedIdx < len(m.workspaces)-1 {
			m.selectedIdx++
		}
		return m, nil
	case "k", "up":
		if m.selectedIdx > 0 {
			m.selectedIdx--
		}
		return m, nil
	case "a":
		m.mode = modeAdding
		m.addInput.SetValue("")
		m.addInput.Focus()
		return m, textinput.Blink
	case "f":
		if ws, ok := m.selected(); ok {
			if ws.Current {
				return m, nil
			}
			m.mode = modeConfirmingForget
			m.forgetName = ws.Name
		}
		return m, nil
	}
	return m, nil
}

func (m Model) handleAddingKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeNormal
		m.addInput.Blur()
		m.addInput.SetValue("")
		return m, nil
	case "enter":
		path := strings.TrimSpace(m.addInput.Value())
		m.mode = modeNormal
		m.addInput.Blur()
		m.addInput.SetValue("")
		if path == "" {
			return m, nil
		}
		return m, state.NavigateTarget{Kind: state.NavigateAddWorkspace, WorkspacePath: path}.Cmd()
	}
	var cmd tea.Cmd
	m.addInput, cmd = m.addInput.Update(msg)
	return m, cmd
}

func (m Model) handleForgetConfirmKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		name := m.forgetName
		m.mode = modeNormal
		m.forgetName = ""
		if name == "" {
			return m, nil
		}
		return m, state.NavigateTarget{Kind: state.NavigateForgetWorkspace, WorkspaceName: name}.Cmd()
	case "n", "N", "esc":
		m.mode = modeNormal
		m.forgetName = ""
		return m, nil
	}
	return m, nil
}

func (m Model) selected() (jj.Workspace, bool) {
	if m.selectedIdx < 0 || m.selectedIdx >= len(m.workspaces) {
		return jj.Workspace{}, false
	}
	return m.workspaces[m.selectedIdx], true
}

// View renders the workspaces modal body (the window title is supplied by chrome).
func (m Model) View() string {
	if !m.shown {
		return ""
	}
	muted := lipgloss.NewStyle().Foreground(styles.ColorMuted)
	cidStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD"))
	currentStyle := lipgloss.NewStyle().Foreground(styles.ColorSecondary).Bold(true)

	modalW := min(max(48, m.termW-8), 84)

	var lines []string
	lines = append(lines, muted.Render("j/k select · a add · f forget · Esc close"))
	lines = append(lines, "")

	if len(m.workspaces) == 0 {
		lines = append(lines, muted.Render("(no workspaces)"))
	}
	for i, ws := range m.workspaces {
		prefix := "  "
		if i == m.selectedIdx {
			prefix = "► "
		}
		name := ws.Name
		if ws.Current {
			name = currentStyle.Render(name + " (current)")
		} else {
			name = lipgloss.NewStyle().Bold(true).Render(name)
		}
		desc := strings.TrimSpace(ws.Description)
		if desc == "" {
			desc = "(no description)"
		}
		idPart := cidStyle.Render(ws.ChangeID)
		metaMax := max(8, modalW-10)
		meta := muted.Render(runewidth.Truncate(desc, metaMax, "…"))
		row := lipgloss.JoinVertical(lipgloss.Left,
			prefix+name+"  "+idPart,
			"    "+meta,
		)
		lines = append(lines, row)
	}

	lines = append(lines, "")
	switch m.mode {
	case modeAdding:
		lines = append(lines, muted.Render("New workspace path (Enter to add · Esc cancel):"))
		lines = append(lines, m.addInput.View())
	case modeConfirmingForget:
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorPrimary).
			Render("Forget workspace \""+m.forgetName+"\"? (y/n)"))
		lines = append(lines, muted.Render("The workspace directory on disk is left untouched."))
	}

	outer := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorMuted).
		Padding(0, 1).
		Width(modalW)
	return outer.Render(strings.Join(lines, "\n"))
}
