package warning

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/state"
	"github.com/madicen/jj-tui/internal/tui/util"
)

// Model represents the warning modal (e.g., for empty commit descriptions)
type Model struct {
	shown       bool
	title       string
	message     string
	commits     []internal.Commit
	selectedIdx int
	zoneManager *zone.Manager // set by main (zones may be in main's view)
}

// NewModel creates a new Warning model
func NewModel() Model {
	return Model{
		shown:       false,
		selectedIdx: 0,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages for the Warning modal
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.shown {
		return m, nil
	}
	// Handle request message (main forwards EditCommitRequestedMsg to us)
	if req, ok := msg.(EditCommitRequestedMsg); ok {
		m.shown = false
		m.commits = nil
		return m, state.NavigateTarget{Kind: state.NavigateEditDescription, Commit: req.Commit}.Cmd()
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case zone.MsgZoneInBounds:
		if m.zoneManager != nil {
			if zoneID := m.resolveClickedZone(msg); zoneID != "" {
				return m.handleZoneClick(zoneID)
			}
		}
		return m, nil
	}
	return m, nil
}

// View renders the Warning modal
func (m Model) View() string {
	if !m.shown {
		return ""
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("3")).
		Padding(1, 2).
		Width(70)

	content := m.title + "\n\n" + m.message
	if len(m.commits) > 0 {
		content += "\n\nCommits with issues:\n"
		for i, c := range m.commits {
			marker := " "
			if i == m.selectedIdx {
				marker = ">"
			}
			content += marker + " " + c.Summary + "\n"
		}
		content += "\n(Use j/k to select, enter to edit, esc to cancel)"
	}

	return style.Render(content)
}

// handleKeyMsg handles keyboard input; returns PerformCancelCmd or tea.Quit for main to handle.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.shown = false
		m.commits = nil
		return m, state.NavigateTarget{Kind: state.NavigateWarningCancel, StatusMessage: "Cancelled"}.Cmd()
	case "enter":
		if len(m.commits) > 0 && m.selectedIdx < len(m.commits) {
			commit := m.commits[m.selectedIdx]
			m.shown = false
			m.commits = nil
			return m, state.NavigateTarget{Kind: state.NavigateEditDescription, Commit: commit}.Cmd()
		}
		return m, nil
	case "j", "down":
		if len(m.commits) > 0 && m.selectedIdx < len(m.commits)-1 {
			m.selectedIdx++
		}
		return m, nil
	case "k", "up":
		if m.selectedIdx > 0 {
			m.selectedIdx--
		}
		return m, nil
	case "ctrl+q", "ctrl+c":
		util.FlushMouse()
		return m, tea.Quit
	}
	return m, nil
}

// ZoneIDs returns the zone IDs used when main renders this modal's buttons. Used to resolve clicks.
func (m Model) ZoneIDs() []string {
	return []string{mouse.ZoneWarningGoToCommit, mouse.ZoneWarningDismiss}
}

func (m Model) resolveClickedZone(msg zone.MsgZoneInBounds) string {
	if msg.Zone == nil {
		return ""
	}
	for _, id := range m.ZoneIDs() {
		z := m.zoneManager.Get(id)
		if z != nil && z.InBounds(msg.Event) {
			return id
		}
	}
	return ""
}

func (m Model) handleZoneClick(zoneID string) (Model, tea.Cmd) {
	switch zoneID {
	case mouse.ZoneWarningGoToCommit:
		if len(m.commits) > 0 && m.selectedIdx < len(m.commits) {
			commit := m.commits[m.selectedIdx]
			m.shown = false
			m.commits = nil
			return m, state.NavigateTarget{Kind: state.NavigateEditDescription, Commit: commit}.Cmd()
		}
		m.shown = false
		m.commits = nil
		return m, nil
	case mouse.ZoneWarningDismiss:
		m.shown = false
		m.commits = nil
		return m, state.NavigateTarget{Kind: state.NavigateWarningCancel, StatusMessage: "Cancelled"}.Cmd()
	}
	return m, nil
}

// SetZoneManager sets the zone manager used to resolve clicks (main's manager).
func (m *Model) SetZoneManager(zm *zone.Manager) {
	m.zoneManager = zm
}

// Accessors

// IsShown returns whether the modal is displayed
func (m *Model) IsShown() bool {
	return m.shown
}

// Show displays the warning modal
func (m *Model) Show(title, message string, commits []internal.Commit) {
	m.shown = true
	m.title = title
	m.message = message
	m.commits = commits
	m.selectedIdx = 0
}

// Hide hides the modal
func (m *Model) Hide() {
	m.shown = false
	m.commits = nil
}

// GetSelectedCommit returns the selected commit
func (m *Model) GetSelectedCommit() *internal.Commit {
	if len(m.commits) > 0 && m.selectedIdx < len(m.commits) {
		return &m.commits[m.selectedIdx]
	}
	return nil
}

// UpdateRepository updates the repository
func (m *Model) UpdateRepository(repo *internal.Repository) {
	// Warning modal doesn't use repository directly
}
