package initrepo

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/styles"
)

// Model is the "not a jj repo" welcome screen. Path non-empty means the screen is active.
type Model struct {
	path        string
	zoneManager *zone.Manager
}

// NewModel creates a new init-repo screen model.
func NewModel() Model {
	return Model{}
}

// Init is required by tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles keys and zone clicks; sends request messages to main.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if m.path == "" {
		return m, nil
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

func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m, RequestDismissCmd()
	case "i":
		return m, RequestInitCmd()
	case "ctrl+q", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) resolveClickedZone(msg zone.MsgZoneInBounds) string {
	if msg.Zone == nil {
		return ""
	}
	z := m.zoneManager.Get(mouse.ZoneActionJJInit)
	if z != nil && z.InBounds(msg.Event) {
		return mouse.ZoneActionJJInit
	}
	return ""
}

func (m Model) handleZoneClick(zoneID string) (Model, tea.Cmd) {
	if zoneID == mouse.ZoneActionJJInit {
		return m, RequestInitCmd()
	}
	return m, nil
}

// View renders the welcome screen (path must be non-empty). Main applies header layout.
func (m Model) View() string {
	if m.path == "" {
		return ""
	}
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E"))
	pathStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#58A6FF"))

	var lines []string
	lines = append(lines, styles.TitleStyle.Render("Welcome to jj-tui"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Directory: %s", pathStyle.Render(m.path)))
	lines = append(lines, "")
	lines = append(lines, "This directory is not yet a Jujutsu repository.")
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Would you like to initialize it?"))
	lines = append(lines, "")

	initButton := styles.ButtonStyle.Background(lipgloss.Color("#238636")).Render("Initialize Repository (i)")
	if m.zoneManager != nil {
		initButton = m.zoneManager.Mark(mouse.ZoneActionJJInit, initButton)
	}
	lines = append(lines, initButton)
	lines = append(lines, "")
	lines = append(lines, mutedStyle.Render("This will run: jj git init"))
	lines = append(lines, mutedStyle.Render("and try to track main@origin if available"))
	lines = append(lines, "")
	lines = append(lines, mutedStyle.Render("Press Ctrl+q to quit"))

	return strings.Join(lines, "\n")
}

// SetPath sets the directory path and activates the screen; empty path hides it.
func (m *Model) SetPath(path string) {
	m.path = path
}

// Path returns the current path (empty if screen is not active).
func (m *Model) Path() string {
	return m.path
}

// SetZoneManager sets the zone manager for clickable elements.
func (m *Model) SetZoneManager(zm *zone.Manager) {
	m.zoneManager = zm
}
