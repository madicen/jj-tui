package view

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// clearButtonStyle for the clear buttons
var clearButtonStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#F85149")).
	Bold(true)

// settingsTabStyle for inactive tabs
var settingsTabStyle = lipgloss.NewStyle().
	Padding(0, 2).
	Foreground(ColorMuted)

// settingsTabActiveStyle for the active tab
var settingsTabActiveStyle = lipgloss.NewStyle().
	Padding(0, 2).
	Bold(true).
	Foreground(ColorPrimary).
	Underline(true)

// toggleOnStyle for enabled toggles
var toggleOnStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#50FA7B")).
	Bold(true)

// toggleOffStyle for disabled toggles
var toggleOffStyle = lipgloss.NewStyle().
	Foreground(ColorMuted)

// Settings renders the settings view with sub-tabs
func (r *Renderer) Settings(data SettingsData) string {
	var lines []string
	lines = append(lines, TitleStyle.Render("Settings"))
	lines = append(lines, "")

	// Show config source
	if data.ConfigSource != "" {
		configInfo := "Config: " + data.ConfigSource
		if data.HasLocalConfig {
			configInfo += " (local override active)"
		}
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Italic(true).Render(configInfo))
	} else {
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Italic(true).Render("Config: ~/.config/jj-tui/config.json"))
	}

	// Navigation hint
	navHint := lipgloss.NewStyle().Foreground(ColorMuted).Italic(true).Render("^j/^k: switch tabs  Tab: next field")
	lines = append(lines, navHint)
	lines = append(lines, "")

	// Render sub-tabs
	tabs := r.renderSettingsTabs(data.ActiveTab)
	lines = append(lines, tabs)
	lines = append(lines, "")

	// Render content based on active tab
	switch data.ActiveTab {
	case SettingsTabGitHub:
		lines = append(lines, r.renderGitHubSettings(data)...)
	case SettingsTabJira:
		lines = append(lines, r.renderJiraSettings(data)...)
	case SettingsTabCodecks:
		lines = append(lines, r.renderCodecksSettings(data)...)
	case SettingsTabAdvanced:
		lines = append(lines, r.renderAdvancedSettings(data)...)
	}

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
	lines = append(lines, "")

	// Action buttons
	saveButton := r.Zone.Mark(ZoneSettingsSave, ButtonStyle.Render("Save Global (^s)"))
	saveLocalButton := r.Zone.Mark(ZoneSettingsSaveLocal, ButtonStyle.Render("Save Local (^l)"))
	cancelButton := r.Zone.Mark(ZoneSettingsCancel, ButtonStyle.Render("Cancel (Esc)"))
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left, saveButton, " ", saveLocalButton, " ", cancelButton))

	return strings.Join(lines, "\n")
}

// renderSettingsTabs renders the tab bar
func (r *Renderer) renderSettingsTabs(activeTab SettingsTab) string {
	githubStyle := settingsTabStyle
	jiraStyle := settingsTabStyle
	codecksStyle := settingsTabStyle
	advancedStyle := settingsTabStyle

	switch activeTab {
	case SettingsTabGitHub:
		githubStyle = settingsTabActiveStyle
	case SettingsTabJira:
		jiraStyle = settingsTabActiveStyle
	case SettingsTabCodecks:
		codecksStyle = settingsTabActiveStyle
	case SettingsTabAdvanced:
		advancedStyle = settingsTabActiveStyle
	}

	githubTab := r.Zone.Mark(ZoneSettingsTabGitHub, githubStyle.Render("GitHub"))
	jiraTab := r.Zone.Mark(ZoneSettingsTabJira, jiraStyle.Render("Jira"))
	codecksTab := r.Zone.Mark(ZoneSettingsTabCodecks, codecksStyle.Render("Codecks"))
	advancedTab := r.Zone.Mark(ZoneSettingsTabAdvanced, advancedStyle.Render("Advanced"))

	return lipgloss.JoinHorizontal(lipgloss.Left, githubTab, " │ ", jiraTab, " │ ", codecksTab, " │ ", advancedTab)
}

// renderGitHubSettings renders the GitHub settings content
func (r *Renderer) renderGitHubSettings(data SettingsData) []string {
	var lines []string

	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("GitHub Integration"))
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Connect to GitHub for PR management."))
	lines = append(lines, "")

	// GitHub login button or connected status
	if data.GithubService {
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).Render("  ✓ Connected to GitHub"))
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    "+r.Zone.Mark(ZoneSettingsGitHubLogin, "[Reconnect]")))
	} else {
		loginButton := r.Zone.Mark(ZoneSettingsGitHubLogin, ButtonStyle.Background(lipgloss.Color("#238636")).Render("Login with GitHub"))
		lines = append(lines, "  "+loginButton)
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    Opens browser to authenticate via GitHub App"))
	}
	lines = append(lines, "")

	// Manual token input as alternative
	githubTokenLabel := "  Or enter token manually:"
	githubTokenStyle := lipgloss.NewStyle().Foreground(ColorMuted)
	if data.FocusedField == 0 {
		githubTokenStyle = githubTokenStyle.Bold(true).Foreground(ColorPrimary)
	}
	lines = append(lines, githubTokenStyle.Render(githubTokenLabel))
	if len(data.Inputs) > 0 {
		clearBtn := r.Zone.Mark(ZoneSettingsGitHubTokenClear, clearButtonStyle.Render("[Clear]"))
		lines = append(lines, "  "+r.Zone.Mark(ZoneSettingsGitHubToken, data.Inputs[0].View)+" "+clearBtn)
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    Get a token from: https://github.com/settings/tokens"))
	lines = append(lines, "")

	// PR Filter toggles
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("  PR Filters:"))
	lines = append(lines, "")

	// Only My PRs toggle
	onlyMineToggle := r.renderToggle("Only My PRs", data.OnlyMyPRs, ZoneSettingsGitHubOnlyMine)
	lines = append(lines, "    "+onlyMineToggle)

	// Show Merged PRs toggle
	mergedToggle := r.renderToggle("Show Merged PRs", data.ShowMergedPRs, ZoneSettingsGitHubShowMerged)
	lines = append(lines, "    "+mergedToggle)

	// Show Closed PRs toggle
	closedToggle := r.renderToggle("Show Closed PRs", data.ShowClosedPRs, ZoneSettingsGitHubShowClosed)
	lines = append(lines, "    "+closedToggle)

	lines = append(lines, "")

	// PR Limit control
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("  PR Limit:"))
	limitText := lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%d", data.PRLimit))
	decreaseBtn := r.Zone.Mark(ZoneSettingsGitHubPRLimitDecrease, lipgloss.NewStyle().Foreground(ColorPrimary).Render("[-]"))
	increaseBtn := r.Zone.Mark(ZoneSettingsGitHubPRLimitIncrease, lipgloss.NewStyle().Foreground(ColorPrimary).Render("[+]"))
	lines = append(lines, "    "+decreaseBtn+" "+limitText+" "+increaseBtn)
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    Max PRs to load (reduces API calls)"))
	lines = append(lines, "")

	// PR Auto-Refresh Interval control
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("  PR Auto-Refresh:"))
	var refreshText string
	if data.PRRefreshInterval == 0 {
		refreshText = lipgloss.NewStyle().Foreground(ColorMuted).Render("Disabled")
	} else if data.PRRefreshInterval < 60 {
		refreshText = lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%ds", data.PRRefreshInterval))
	} else {
		mins := data.PRRefreshInterval / 60
		refreshText = lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%dm", mins))
	}
	refreshDecBtn := r.Zone.Mark(ZoneSettingsGitHubRefreshDecrease, lipgloss.NewStyle().Foreground(ColorPrimary).Render("[-]"))
	refreshIncBtn := r.Zone.Mark(ZoneSettingsGitHubRefreshIncrease, lipgloss.NewStyle().Foreground(ColorPrimary).Render("[+]"))
	toggleBtn := r.Zone.Mark(ZoneSettingsGitHubRefreshToggle, lipgloss.NewStyle().Foreground(ColorPrimary).Render("[Toggle]"))
	lines = append(lines, "    "+refreshDecBtn+" "+refreshText+" "+refreshIncBtn+" "+toggleBtn)
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    Auto-refresh PRs when viewing PR tab (0 = disabled)"))

	return lines
}

// renderToggle renders a clickable toggle switch
func (r *Renderer) renderToggle(label string, enabled bool, zoneID string) string {
	var toggleText string
	if enabled {
		toggleText = toggleOnStyle.Render("[✓]") + " " + label
	} else {
		toggleText = toggleOffStyle.Render("[ ]") + " " + lipgloss.NewStyle().Foreground(ColorMuted).Render(label)
	}
	return r.Zone.Mark(zoneID, toggleText)
}

// renderJiraSettings renders the Jira settings content
// Input indices: 1=URL, 2=User, 3=Token, 4=Excluded Statuses
func (r *Renderer) renderJiraSettings(data SettingsData) []string {
	var lines []string

	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("Jira Integration"))
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Connect to Jira for ticket management."))
	lines = append(lines, "")

	// Jira URL field (index 1)
	jiraURLLabel := "  Instance URL:"
	jiraURLStyle := lipgloss.NewStyle()
	if data.FocusedField == 1 {
		jiraURLStyle = jiraURLStyle.Bold(true).Foreground(ColorPrimary)
	}
	lines = append(lines, jiraURLStyle.Render(jiraURLLabel))
	if len(data.Inputs) > 1 {
		clearBtn := r.Zone.Mark(ZoneSettingsJiraURLClear, clearButtonStyle.Render("[Clear]"))
		lines = append(lines, "  "+r.Zone.Mark(ZoneSettingsJiraURL, data.Inputs[1].View)+" "+clearBtn)
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    e.g., https://yourcompany.atlassian.net"))
	lines = append(lines, "")

	// Jira User field (index 2)
	jiraUserLabel := "  Email:"
	jiraUserStyle := lipgloss.NewStyle()
	if data.FocusedField == 2 {
		jiraUserStyle = jiraUserStyle.Bold(true).Foreground(ColorPrimary)
	}
	lines = append(lines, jiraUserStyle.Render(jiraUserLabel))
	if len(data.Inputs) > 2 {
		clearBtn := r.Zone.Mark(ZoneSettingsJiraUserClear, clearButtonStyle.Render("[Clear]"))
		lines = append(lines, "  "+r.Zone.Mark(ZoneSettingsJiraUser, data.Inputs[2].View)+" "+clearBtn)
	}
	lines = append(lines, "")

	// Jira Token field (index 3)
	jiraTokenLabel := "  API Token:"
	jiraTokenStyle := lipgloss.NewStyle()
	if data.FocusedField == 3 {
		jiraTokenStyle = jiraTokenStyle.Bold(true).Foreground(ColorPrimary)
	}
	lines = append(lines, jiraTokenStyle.Render(jiraTokenLabel))
	if len(data.Inputs) > 3 {
		clearBtn := r.Zone.Mark(ZoneSettingsJiraTokenClear, clearButtonStyle.Render("[Clear]"))
		lines = append(lines, "  "+r.Zone.Mark(ZoneSettingsJiraToken, data.Inputs[3].View)+" "+clearBtn)
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    Get from: https://id.atlassian.com/manage-profile/security/api-tokens"))
	lines = append(lines, "")

	// Jira Excluded Statuses field (index 4)
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("  Ticket Filters:"))
	excludedLabel := "  Exclude Statuses:"
	excludedStyle := lipgloss.NewStyle()
	if data.FocusedField == 4 {
		excludedStyle = excludedStyle.Bold(true).Foreground(ColorPrimary)
	}
	lines = append(lines, excludedStyle.Render(excludedLabel))
	if len(data.Inputs) > 4 {
		clearBtn := r.Zone.Mark(ZoneSettingsJiraExcludedClear, clearButtonStyle.Render("[Clear]"))
		lines = append(lines, "  "+r.Zone.Mark(ZoneSettingsJiraExcluded, data.Inputs[4].View)+" "+clearBtn)
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    Comma-separated list (e.g., Done, Won't Do, Cancelled)"))

	return lines
}

// renderCodecksSettings renders the Codecks settings content
// Input indices: 5=Subdomain, 6=Token, 7=Project, 8=Excluded Statuses
func (r *Renderer) renderCodecksSettings(data SettingsData) []string {
	var lines []string

	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("Codecks Integration"))
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Connect to Codecks for card management."))
	lines = append(lines, "")

	// Codecks Subdomain field (index 5)
	codecksSubdomainLabel := "  Subdomain:"
	codecksSubdomainStyle := lipgloss.NewStyle()
	if data.FocusedField == 5 {
		codecksSubdomainStyle = codecksSubdomainStyle.Bold(true).Foreground(ColorPrimary)
	}
	lines = append(lines, codecksSubdomainStyle.Render(codecksSubdomainLabel))
	if len(data.Inputs) > 5 {
		clearBtn := r.Zone.Mark(ZoneSettingsCodecksSubdomainClear, clearButtonStyle.Render("[Clear]"))
		lines = append(lines, "  "+r.Zone.Mark(ZoneSettingsCodecksSubdomain, data.Inputs[5].View)+" "+clearBtn)
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    Your team name (e.g., 'myteam' from myteam.codecks.io)"))
	lines = append(lines, "")

	// Codecks Token field (index 6)
	codecksTokenLabel := "  Auth Token:"
	codecksTokenStyle := lipgloss.NewStyle()
	if data.FocusedField == 6 {
		codecksTokenStyle = codecksTokenStyle.Bold(true).Foreground(ColorPrimary)
	}
	lines = append(lines, codecksTokenStyle.Render(codecksTokenLabel))
	if len(data.Inputs) > 6 {
		clearBtn := r.Zone.Mark(ZoneSettingsCodecksTokenClear, clearButtonStyle.Render("[Clear]"))
		lines = append(lines, "  "+r.Zone.Mark(ZoneSettingsCodecksToken, data.Inputs[6].View)+" "+clearBtn)
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    Extract 'at' cookie from browser DevTools"))
	lines = append(lines, "")

	// Codecks Project field (index 7)
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("  Card Filters:"))
	codecksProjectLabel := "  Project Filter:"
	codecksProjectStyle := lipgloss.NewStyle()
	if data.FocusedField == 7 {
		codecksProjectStyle = codecksProjectStyle.Bold(true).Foreground(ColorPrimary)
	}
	lines = append(lines, codecksProjectStyle.Render(codecksProjectLabel))
	if len(data.Inputs) > 7 {
		clearBtn := r.Zone.Mark(ZoneSettingsCodecksProjectClear, clearButtonStyle.Render("[Clear]"))
		lines = append(lines, "  "+r.Zone.Mark(ZoneSettingsCodecksProject, data.Inputs[7].View)+" "+clearBtn)
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    Optional: Only show cards from this project"))
	lines = append(lines, "")

	// Codecks Excluded Statuses field (index 8)
	excludedLabel := "  Exclude Statuses:"
	excludedStyle := lipgloss.NewStyle()
	if data.FocusedField == 8 {
		excludedStyle = excludedStyle.Bold(true).Foreground(ColorPrimary)
	}
	lines = append(lines, excludedStyle.Render(excludedLabel))
	if len(data.Inputs) > 8 {
		clearBtn := r.Zone.Mark(ZoneSettingsCodecksExcludedClear, clearButtonStyle.Render("[Clear]"))
		lines = append(lines, "  "+r.Zone.Mark(ZoneSettingsCodecksExcluded, data.Inputs[8].View)+" "+clearBtn)
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    Comma-separated list (e.g., done, archived)"))

	return lines
}

// renderAdvancedSettings renders the Advanced/Maintenance settings tab
func (r *Renderer) renderAdvancedSettings(data SettingsData) []string {
	var lines []string

	// Ticket Workflow section
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("Ticket Workflow"))
	lines = append(lines, "")

	// Auto-transition toggle
	autoInProgressToggle := "[ ]"
	if data.AutoInProgressOnBranch {
		autoInProgressToggle = "[✓]"
	}
	toggleStyle := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	autoBtn := r.Zone.Mark(ZoneSettingsAutoInProgress, toggleStyle.Render(autoInProgressToggle+" Auto-set 'In Progress' on branch creation"))
	lines = append(lines, "  "+autoBtn)
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    Automatically transition ticket to 'In Progress' when creating a branch from it"))
	lines = append(lines, "")
	lines = append(lines, "")

	// Advanced Maintenance section
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("Advanced Maintenance"))
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#F85149")).Bold(true).Render("WARNING: Destructive operations. Use caution!"))
	lines = append(lines, "")

	// Check if confirming
	if data.ConfirmingCleanup != "" {
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("Are you sure? This cannot be undone."))
		lines = append(lines, "")

		confirmYes := r.Zone.Mark(ZoneSettingsAdvancedConfirmYes, ButtonStyle.Background(lipgloss.Color("#F85149")).Render("Yes, Confirm"))
		confirmNo := r.Zone.Mark(ZoneSettingsAdvancedConfirmNo, ButtonStyle.Render("Cancel"))
		lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left, confirmYes, " ", confirmNo))
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Press Y to confirm, N/Esc to cancel"))

		return lines
	}

	// Normal operation listing
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorMuted).Render("Available Operations:"))
	lines = append(lines, "")

	deleteBtn := r.Zone.Mark(ZoneSettingsAdvancedDeleteBookmarks, ButtonStyle.Render("Delete All Bookmarks"))
	lines = append(lines, "  "+deleteBtn)
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    Delete all bookmarks in this repository"))
	lines = append(lines, "")

	abandonBtn := r.Zone.Mark(ZoneSettingsAdvancedAbandonOldCommits, ButtonStyle.Render("Abandon Old Commits"))
	lines = append(lines, "  "+abandonBtn)
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    Abandon commits before origin/main"))
	lines = append(lines, "")

	trackBtn := r.Zone.Mark(ZoneSettingsAdvancedTrackOriginMain, ButtonStyle.Render("Track origin/main"))
	lines = append(lines, "  "+trackBtn)
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    Fetch and track origin/main"))

	return lines
}
