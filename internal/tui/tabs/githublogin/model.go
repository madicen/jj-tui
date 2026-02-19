package githublogin

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/styles"
	"github.com/madicen/jj-tui/internal/tui/util"
)

// Model represents the GitHub Device Flow login screen (shown when connecting GitHub from settings).
type Model struct {
	deviceCode      string
	userCode        string
	verificationURL string
	polling         bool
	pollInterval    int
	zoneManager     *zone.Manager
}

// NewModel creates a new GitHub login model. zoneManager may be nil.
func NewModel(zoneManager *zone.Manager) Model {
	return Model{
		zoneManager: zoneManager,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles keys and zone clicks; returns PerformCancelCmd or CopyToClipboard cmd for main to run.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, PerformCancelCmd()
		case "c":
			if m.userCode != "" {
				return m, util.CopyToClipboard(m.userCode)
			}
			return m, nil
		}
		return m, nil
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

// View renders the GitHub Device Flow login screen.
func (m Model) View() string {
	var lines []string

	lines = append(lines, styles.TitleStyle.Render("GitHub Login"))
	lines = append(lines, "")
	lines = append(lines, "")

	if m.userCode != "" {
		lines = append(lines, lipgloss.NewStyle().Bold(true).Render("1. Visit this URL in your browser:"))
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#58A6FF")).Render("   "+m.verificationURL))
		lines = append(lines, "")
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Bold(true).Render("2. Enter this code:"))
		lines = append(lines, "")

		codeStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F0F6FC")).
			Background(lipgloss.Color("#238636")).
			Padding(1, 3).
			MarginLeft(3)
		lines = append(lines, codeStyle.Render(m.userCode))
		lines = append(lines, "")

		copyButton := styles.ButtonStyle.Render("Copy Code (c)")
		if m.zoneManager != nil {
			copyButton = m.zoneManager.Mark(mouse.ZoneGitHubLoginCopyCode, copyButton)
		}
		lines = append(lines, "   "+copyButton)
		lines = append(lines, "")

		if m.polling {
			lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E")).Italic(true).Render("   Waiting for authorization..."))
		}
	} else {
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E")).Render("   Starting GitHub login..."))
	}

	lines = append(lines, "")
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E")).Render("Press Esc to cancel"))

	return strings.Join(lines, "\n")
}

// ZoneIDs returns the zone IDs this view uses when rendering.
func (m Model) ZoneIDs() []string {
	return []string{mouse.ZoneGitHubLoginCopyCode}
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
	if zoneID == mouse.ZoneGitHubLoginCopyCode && m.userCode != "" {
		return m, util.CopyToClipboard(m.userCode)
	}
	return m, nil
}

// SetZoneManager sets the zone manager (main's manager).
func (m *Model) SetZoneManager(zm *zone.Manager) {
	m.zoneManager = zm
}

// ——— Device flow state (set by main when GitHubDeviceFlowStartedMsg arrives) ———

// SetDeviceFlow sets the device flow data and polling interval.
func (m *Model) SetDeviceFlow(deviceCode, userCode, verificationURL string, pollInterval int) {
	m.deviceCode = deviceCode
	m.userCode = userCode
	m.verificationURL = verificationURL
	m.polling = true
	m.pollInterval = pollInterval
}

// ClearFlow clears all device flow state (on cancel, success, or error).
func (m *Model) ClearFlow() {
	m.deviceCode = ""
	m.userCode = ""
	m.verificationURL = ""
	m.polling = false
	m.pollInterval = 0
}

// GetDeviceCode returns the device code (for polling).
func (m *Model) GetDeviceCode() string {
	return m.deviceCode
}

// GetPollInterval returns the current poll interval in seconds.
func (m *Model) GetPollInterval() int {
	return m.pollInterval
}

// SetPollInterval sets the poll interval (main updates on GitHubLoginPollMsg).
func (m *Model) SetPollInterval(seconds int) {
	m.pollInterval = seconds
}

// GetPolling returns whether we are waiting for the user to authorize.
func (m *Model) GetPolling() bool {
	return m.polling
}
