package githublogin

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/state"
	"github.com/madicen/jj-tui/internal/tui/styles"
	"github.com/madicen/jj-tui/internal/tui/util"
)

// LoginMode selects device flow vs GitHub CLI login UI.
type LoginMode int

const (
	LoginModeDevice LoginMode = iota
	LoginModeGhCLI
)

// Model represents the GitHub login screen (device flow or `gh auth login`).
type Model struct {
	loginMode       LoginMode
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
			return m, state.NavigateTarget{Kind: state.NavigateGitHubLoginCancel, StatusMessage: "GitHub login cancelled"}.Cmd()
		case "enter":
			if m.loginMode == LoginModeGhCLI {
				return m, GhAuthLoginCmd()
			}
			if m.userCode != "" {
				return m, tea.Batch(
					util.CopyToClipboard(m.userCode),
					util.OpenURL(m.verificationURL),
				)
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

// View renders the GitHub login screen (device flow or GitHub CLI).
func (m Model) View() string {
	if m.loginMode == LoginModeGhCLI {
		return m.viewGhCLI()
	}
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

		copyButton := styles.ButtonStyle.Render("Copy Code & Open Browser (Enter)")
		cancelButton := styles.ButtonStyle.Render("Cancel (Esc)")
		if m.zoneManager != nil {
			copyButton = m.zoneManager.Mark(mouse.ZoneGitHubLoginCopyAndOpen, copyButton)
			cancelButton = m.zoneManager.Mark(mouse.ZoneGitHubLoginCancel, cancelButton)
		}
		lines = append(lines, "   "+lipgloss.JoinHorizontal(lipgloss.Left, copyButton, "  ", cancelButton))
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

func (m Model) viewGhCLI() string {
	var lines []string
	lines = append(lines, styles.TitleStyle.Render("GitHub CLI login"))
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E")).Render("This will temporarily suspend jj-tui and run:"))
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("   gh auth login"))
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E")).Render("Complete the prompts in your terminal; jj-tui will then verify `gh auth token`."))
	lines = append(lines, "")
	runBtn := styles.ButtonStyle.Render("Run gh auth login (Enter)")
	cancelBtn := styles.ButtonStyle.Render("Cancel (Esc)")
	if m.zoneManager != nil {
		runBtn = m.zoneManager.Mark(mouse.ZoneGitHubLoginRunGhAuth, runBtn)
		cancelBtn = m.zoneManager.Mark(mouse.ZoneGitHubLoginCancel, cancelBtn)
	}
	lines = append(lines, "   "+lipgloss.JoinHorizontal(lipgloss.Left, runBtn, "  ", cancelBtn))
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E")).Render("Press Esc to cancel without running gh."))
	return strings.Join(lines, "\n")
}

// ZoneIDs returns the zone IDs this view uses when rendering.
func (m Model) ZoneIDs() []string {
	if m.loginMode == LoginModeGhCLI {
		return []string{mouse.ZoneGitHubLoginRunGhAuth, mouse.ZoneGitHubLoginCancel}
	}
	return []string{mouse.ZoneGitHubLoginCopyAndOpen, mouse.ZoneGitHubLoginCancel}
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
	if m.loginMode == LoginModeGhCLI {
		switch zoneID {
		case mouse.ZoneGitHubLoginRunGhAuth:
			return m, GhAuthLoginCmd()
		case mouse.ZoneGitHubLoginCancel:
			return m, state.NavigateTarget{Kind: state.NavigateGitHubLoginCancel, StatusMessage: "GitHub login cancelled"}.Cmd()
		}
		return m, nil
	}
	if zoneID == mouse.ZoneGitHubLoginCopyAndOpen && m.userCode != "" {
		return m, tea.Batch(
			util.CopyToClipboard(m.userCode),
			util.OpenURL(m.verificationURL),
		)
	}
	if zoneID == mouse.ZoneGitHubLoginCancel {
		return m, state.NavigateTarget{Kind: state.NavigateGitHubLoginCancel, StatusMessage: "GitHub login cancelled"}.Cmd()
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
	m.loginMode = LoginModeDevice
	m.deviceCode = deviceCode
	m.userCode = userCode
	m.verificationURL = verificationURL
	m.polling = true
	m.pollInterval = pollInterval
}

// SetGhCLILoginMode switches the modal to the GitHub CLI login flow (before running gh).
func (m *Model) SetGhCLILoginMode() {
	m.loginMode = LoginModeGhCLI
	m.deviceCode = ""
	m.userCode = ""
	m.verificationURL = ""
	m.polling = false
	m.pollInterval = 0
}

// ClearFlow clears all login state (on cancel, success, or error).
func (m *Model) ClearFlow() {
	m.loginMode = LoginModeDevice
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
