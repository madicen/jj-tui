package view

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Settings renders the settings view
func (r *Renderer) Settings(data SettingsData) string {
	var lines []string
	lines = append(lines, TitleStyle.Render("Settings"))
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Configure your API credentials. Press Tab/↓ to move between fields."))
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Press Ctrl+S or Enter on last field to save, Esc to cancel."))
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Italic(true).Render("Settings are saved to ~/.config/jj-tui/config.json"))
	lines = append(lines, "")

	// GitHub section
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("GitHub"))
	lines = append(lines, "")

	// GitHub Token field
	githubTokenLabel := "  Token:"
	githubTokenStyle := lipgloss.NewStyle()
	if data.FocusedField == 0 {
		githubTokenStyle = githubTokenStyle.Bold(true).Foreground(ColorPrimary)
	}
	lines = append(lines, githubTokenStyle.Render(githubTokenLabel))
	if len(data.Inputs) > 0 {
		lines = append(lines, r.Zone.Mark(ZoneSettingsGitHubToken, "  "+data.Inputs[0].View))
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    Personal access token with repo scope"))
	lines = append(lines, "")

	// Jira section
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("Jira"))
	lines = append(lines, "")

	// Jira URL field
	jiraURLLabel := "  URL:"
	jiraURLStyle := lipgloss.NewStyle()
	if data.FocusedField == 1 {
		jiraURLStyle = jiraURLStyle.Bold(true).Foreground(ColorPrimary)
	}
	lines = append(lines, jiraURLStyle.Render(jiraURLLabel))
	if len(data.Inputs) > 1 {
		lines = append(lines, r.Zone.Mark(ZoneSettingsJiraURL, "  "+data.Inputs[1].View))
	}
	lines = append(lines, "")

	// Jira User field
	jiraUserLabel := "  Email:"
	jiraUserStyle := lipgloss.NewStyle()
	if data.FocusedField == 2 {
		jiraUserStyle = jiraUserStyle.Bold(true).Foreground(ColorPrimary)
	}
	lines = append(lines, jiraUserStyle.Render(jiraUserLabel))
	if len(data.Inputs) > 2 {
		lines = append(lines, r.Zone.Mark(ZoneSettingsJiraUser, "  "+data.Inputs[2].View))
	}
	lines = append(lines, "")

	// Jira Token field
	jiraTokenLabel := "  API Token:"
	jiraTokenStyle := lipgloss.NewStyle()
	if data.FocusedField == 3 {
		jiraTokenStyle = jiraTokenStyle.Bold(true).Foreground(ColorPrimary)
	}
	lines = append(lines, jiraTokenStyle.Render(jiraTokenLabel))
	if len(data.Inputs) > 3 {
		lines = append(lines, r.Zone.Mark(ZoneSettingsJiraToken, "  "+data.Inputs[3].View))
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    Get from: https://id.atlassian.com/manage-profile/security/api-tokens"))
	lines = append(lines, "")

	// Codecks section
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("Codecks"))
	lines = append(lines, "")

	// Codecks Subdomain field
	codecksSubdomainLabel := "  Subdomain:"
	codecksSubdomainStyle := lipgloss.NewStyle()
	if data.FocusedField == 4 {
		codecksSubdomainStyle = codecksSubdomainStyle.Bold(true).Foreground(ColorPrimary)
	}
	lines = append(lines, codecksSubdomainStyle.Render(codecksSubdomainLabel))
	if len(data.Inputs) > 4 {
		lines = append(lines, r.Zone.Mark(ZoneSettingsCodecksSubdomain, "  "+data.Inputs[4].View))
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    Your team name (e.g., 'myteam' from myteam.codecks.io)"))
	lines = append(lines, "")

	// Codecks Token field
	codecksTokenLabel := "  Token:"
	codecksTokenStyle := lipgloss.NewStyle()
	if data.FocusedField == 5 {
		codecksTokenStyle = codecksTokenStyle.Bold(true).Foreground(ColorPrimary)
	}
	lines = append(lines, codecksTokenStyle.Render(codecksTokenLabel))
	if len(data.Inputs) > 5 {
		lines = append(lines, r.Zone.Mark(ZoneSettingsCodecksToken, "  "+data.Inputs[5].View))
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    Extract 'at' cookie from browser (see https://manual.codecks.io/api/)"))
	lines = append(lines, "")

	// Codecks Project field (optional filter)
	codecksProjectLabel := "  Project:"
	codecksProjectStyle := lipgloss.NewStyle()
	if data.FocusedField == 6 {
		codecksProjectStyle = codecksProjectStyle.Bold(true).Foreground(ColorPrimary)
	}
	lines = append(lines, codecksProjectStyle.Render(codecksProjectLabel))
	if len(data.Inputs) > 6 {
		lines = append(lines, r.Zone.Mark(ZoneSettingsCodecksProject, "  "+data.Inputs[6].View))
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    Optional: Filter cards by project name"))
	lines = append(lines, "")

	// Connection status
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Connection Status:"))
	if data.GithubService {
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).Render("  ✓ GitHub connected"))
	} else {
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("  ○ GitHub not connected"))
	}
	if data.JiraService {
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).Render("  ✓ Tickets connected"))
	} else {
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("  ○ Tickets not connected"))
	}
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Italic(true).Render("  Note: Configure either Jira OR Codecks for ticket integration"))

	lines = append(lines, "")
	lines = append(lines, "")

	// Action buttons
	saveButton := r.Zone.Mark(ZoneSettingsSave, ButtonStyle.Render("Save (Ctrl+S)"))
	cancelButton := r.Zone.Mark(ZoneSettingsCancel, ButtonStyle.Render("Cancel (Esc)"))
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left, saveButton, " ", cancelButton))

	return strings.Join(lines, "\n")
}

