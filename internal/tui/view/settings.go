package view

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/version"
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
	case SettingsTabTickets:
		lines = append(lines, r.renderTicketsSettings(data)...)
	case SettingsTabBranches:
		lines = append(lines, r.renderBranchesSettings(data)...)
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

	// Version info
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Version:"))
	versionStr := version.GetVersion()
	versionLine := lipgloss.NewStyle().Foreground(ColorMuted).Render("  " + versionStr)
	if updateInfo := version.GetUpdateInfo(); updateInfo != nil {
		if updateInfo.UpdateAvailable {
			versionLine += lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB86C")).Render(
				fmt.Sprintf(" → %s available", updateInfo.LatestVersion))
		} else if updateInfo.LatestVersion != "" && updateInfo.LatestVersion != "dev" {
			versionLine += lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).Render(" (up to date)")
		}
	}
	lines = append(lines, versionLine)

	lines = append(lines, "")
	lines = append(lines, "")

	// Action buttons
	saveButton := r.Mark(mouse.ZoneSettingsSave, ButtonStyle.Render("Save Global (^s)"))
	saveLocalButton := r.Mark(mouse.ZoneSettingsSaveLocal, ButtonStyle.Render("Save Local (^l)"))
	cancelButton := r.Mark(mouse.ZoneSettingsCancel, ButtonStyle.Render("Cancel (Esc)"))
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left, saveButton, " ", saveLocalButton, " ", cancelButton))

	return strings.Join(lines, "\n")
}

// renderSettingsTabs renders the tab bar
func (r *Renderer) renderSettingsTabs(activeTab SettingsTab) string {
	githubStyle := settingsTabStyle
	jiraStyle := settingsTabStyle
	codecksStyle := settingsTabStyle
	ticketsStyle := settingsTabStyle
	branchesStyle := settingsTabStyle
	advancedStyle := settingsTabStyle

	switch activeTab {
	case SettingsTabGitHub:
		githubStyle = settingsTabActiveStyle
	case SettingsTabJira:
		jiraStyle = settingsTabActiveStyle
	case SettingsTabCodecks:
		codecksStyle = settingsTabActiveStyle
	case SettingsTabTickets:
		ticketsStyle = settingsTabActiveStyle
	case SettingsTabBranches:
		branchesStyle = settingsTabActiveStyle
	case SettingsTabAdvanced:
		advancedStyle = settingsTabActiveStyle
	}

	githubTab := r.Mark(mouse.ZoneSettingsTabGitHub, githubStyle.Render("GitHub"))
	jiraTab := r.Mark(mouse.ZoneSettingsTabJira, jiraStyle.Render("Jira"))
	codecksTab := r.Mark(mouse.ZoneSettingsTabCodecks, codecksStyle.Render("Codecks"))
	ticketsTab := r.Mark(mouse.ZoneSettingsTabTickets, ticketsStyle.Render("Tickets"))
	branchesTab := r.Mark(mouse.ZoneSettingsTabBranches, branchesStyle.Render("Branches"))
	advancedTab := r.Mark(mouse.ZoneSettingsTabAdvanced, advancedStyle.Render("Advanced"))

	return lipgloss.JoinHorizontal(lipgloss.Left, githubTab, " │ ", jiraTab, " │ ", codecksTab, " │ ", ticketsTab, " │ ", branchesTab, " │ ", advancedTab)
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
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    "+r.Mark(mouse.ZoneSettingsGitHubLogin, "[Reconnect]")))
	} else {
		loginButton := r.Mark(mouse.ZoneSettingsGitHubLogin, ButtonStyle.Background(lipgloss.Color("#238636")).Render("Login with GitHub"))
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
		clearBtn := r.Mark(mouse.ZoneSettingsGitHubTokenClear, clearButtonStyle.Render("[Clear]"))
		lines = append(lines, "  "+r.Mark(mouse.ZoneSettingsGitHubToken, data.Inputs[0].View)+" "+clearBtn)
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    Get a token from: https://github.com/settings/tokens"))
	lines = append(lines, "")

	// PR Filter toggles
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("  PR Filters:"))
	lines = append(lines, "")

	// Only My PRs toggle
	onlyMineToggle := r.renderToggle("Only My PRs", data.OnlyMyPRs, mouse.ZoneSettingsGitHubOnlyMine)
	lines = append(lines, "    "+onlyMineToggle)

	// Show Merged PRs toggle
	mergedToggle := r.renderToggle("Show Merged PRs", data.ShowMergedPRs, mouse.ZoneSettingsGitHubShowMerged)
	lines = append(lines, "    "+mergedToggle)

	// Show Closed PRs toggle
	closedToggle := r.renderToggle("Show Closed PRs", data.ShowClosedPRs, mouse.ZoneSettingsGitHubShowClosed)
	lines = append(lines, "    "+closedToggle)

	lines = append(lines, "")

	// PR Limit control
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("  PR Limit:"))
	limitText := lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%d", data.PRLimit))
	decreaseBtn := r.Mark(mouse.ZoneSettingsGitHubPRLimitDecrease, lipgloss.NewStyle().Foreground(ColorPrimary).Render("[-]"))
	increaseBtn := r.Mark(mouse.ZoneSettingsGitHubPRLimitIncrease, lipgloss.NewStyle().Foreground(ColorPrimary).Render("[+]"))
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
	refreshDecBtn := r.Mark(mouse.ZoneSettingsGitHubRefreshDecrease, lipgloss.NewStyle().Foreground(ColorPrimary).Render("[-]"))
	refreshIncBtn := r.Mark(mouse.ZoneSettingsGitHubRefreshIncrease, lipgloss.NewStyle().Foreground(ColorPrimary).Render("[+]"))
	toggleBtn := r.Mark(mouse.ZoneSettingsGitHubRefreshToggle, lipgloss.NewStyle().Foreground(ColorPrimary).Render("[Toggle]"))
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
	return r.Mark(zoneID, toggleText)
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
		clearBtn := r.Mark(mouse.ZoneSettingsJiraURLClear, clearButtonStyle.Render("[Clear]"))
		lines = append(lines, "  "+r.Mark(mouse.ZoneSettingsJiraURL, data.Inputs[1].View)+" "+clearBtn)
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
		clearBtn := r.Mark(mouse.ZoneSettingsJiraUserClear, clearButtonStyle.Render("[Clear]"))
		lines = append(lines, "  "+r.Mark(mouse.ZoneSettingsJiraUser, data.Inputs[2].View)+" "+clearBtn)
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
		clearBtn := r.Mark(mouse.ZoneSettingsJiraTokenClear, clearButtonStyle.Render("[Clear]"))
		lines = append(lines, "  "+r.Mark(mouse.ZoneSettingsJiraToken, data.Inputs[3].View)+" "+clearBtn)
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    Get from: https://id.atlassian.com/manage-profile/security/api-tokens"))
	lines = append(lines, "")

	// Jira Filters section
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("  Ticket Filters:"))

	// Jira Project filter field (index 4)
	projectLabel := "  Project(s):"
	projectStyle := lipgloss.NewStyle()
	if data.FocusedField == 4 {
		projectStyle = projectStyle.Bold(true).Foreground(ColorPrimary)
	}
	lines = append(lines, projectStyle.Render(projectLabel))
	if len(data.Inputs) > 4 {
		clearBtn := r.Mark(mouse.ZoneSettingsJiraProjectClear, clearButtonStyle.Render("[Clear]"))
		lines = append(lines, "  "+r.Mark(mouse.ZoneSettingsJiraProject, data.Inputs[4].View)+" "+clearBtn)
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    Project key(s) to filter (e.g., PROJ or PROJ,TEAM)"))
	lines = append(lines, "")

	// Jira JQL filter field (index 5)
	jqlLabel := "  Custom JQL:"
	jqlStyle := lipgloss.NewStyle()
	if data.FocusedField == 5 {
		jqlStyle = jqlStyle.Bold(true).Foreground(ColorPrimary)
	}
	lines = append(lines, jqlStyle.Render(jqlLabel))
	if len(data.Inputs) > 5 {
		clearBtn := r.Mark(mouse.ZoneSettingsJiraJQLClear, clearButtonStyle.Render("[Clear]"))
		lines = append(lines, "  "+r.Mark(mouse.ZoneSettingsJiraJQL, data.Inputs[5].View)+" "+clearBtn)
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    Additional JQL filter (e.g., sprint in openSprints())"))
	lines = append(lines, "")

	// Jira Excluded Statuses field (index 6)
	excludedLabel := "  Exclude Statuses:"
	excludedStyle := lipgloss.NewStyle()
	if data.FocusedField == 6 {
		excludedStyle = excludedStyle.Bold(true).Foreground(ColorPrimary)
	}
	lines = append(lines, excludedStyle.Render(excludedLabel))
	if len(data.Inputs) > 6 {
		clearBtn := r.Mark(mouse.ZoneSettingsJiraExcludedClear, clearButtonStyle.Render("[Clear]"))
		lines = append(lines, "  "+r.Mark(mouse.ZoneSettingsJiraExcluded, data.Inputs[6].View)+" "+clearBtn)
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    Comma-separated list (e.g., Done, Won't Do, Cancelled)"))

	return lines
}

// renderCodecksSettings renders the Codecks settings content
// Input indices: 7=Subdomain, 8=Token, 9=Project, 10=Excluded Statuses
func (r *Renderer) renderCodecksSettings(data SettingsData) []string {
	var lines []string

	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("Codecks Integration"))
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Connect to Codecks for card management."))
	lines = append(lines, "")

	// Codecks Subdomain field (index 7)
	codecksSubdomainLabel := "  Subdomain:"
	codecksSubdomainStyle := lipgloss.NewStyle()
	if data.FocusedField == 7 {
		codecksSubdomainStyle = codecksSubdomainStyle.Bold(true).Foreground(ColorPrimary)
	}
	lines = append(lines, codecksSubdomainStyle.Render(codecksSubdomainLabel))
	if len(data.Inputs) > 7 {
		clearBtn := r.Mark(mouse.ZoneSettingsCodecksSubdomainClear, clearButtonStyle.Render("[Clear]"))
		lines = append(lines, "  "+r.Mark(mouse.ZoneSettingsCodecksSubdomain, data.Inputs[7].View)+" "+clearBtn)
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    Your team name (e.g., 'myteam' from myteam.codecks.io)"))
	lines = append(lines, "")

	// Codecks Token field (index 8)
	codecksTokenLabel := "  Auth Token:"
	codecksTokenStyle := lipgloss.NewStyle()
	if data.FocusedField == 8 {
		codecksTokenStyle = codecksTokenStyle.Bold(true).Foreground(ColorPrimary)
	}
	lines = append(lines, codecksTokenStyle.Render(codecksTokenLabel))
	if len(data.Inputs) > 8 {
		clearBtn := r.Mark(mouse.ZoneSettingsCodecksTokenClear, clearButtonStyle.Render("[Clear]"))
		lines = append(lines, "  "+r.Mark(mouse.ZoneSettingsCodecksToken, data.Inputs[8].View)+" "+clearBtn)
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    Extract 'at' cookie from browser DevTools"))
	lines = append(lines, "")

	// Codecks Project field (index 9)
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("  Card Filters:"))
	codecksProjectLabel := "  Project Filter:"
	codecksProjectStyle := lipgloss.NewStyle()
	if data.FocusedField == 9 {
		codecksProjectStyle = codecksProjectStyle.Bold(true).Foreground(ColorPrimary)
	}
	lines = append(lines, codecksProjectStyle.Render(codecksProjectLabel))
	if len(data.Inputs) > 9 {
		clearBtn := r.Mark(mouse.ZoneSettingsCodecksProjectClear, clearButtonStyle.Render("[Clear]"))
		lines = append(lines, "  "+r.Mark(mouse.ZoneSettingsCodecksProject, data.Inputs[9].View)+" "+clearBtn)
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    Optional: Only show cards from this project"))
	lines = append(lines, "")

	// Codecks Excluded Statuses field (index 10)
	excludedLabel := "  Exclude Statuses:"
	excludedStyle := lipgloss.NewStyle()
	if data.FocusedField == 10 {
		excludedStyle = excludedStyle.Bold(true).Foreground(ColorPrimary)
	}
	lines = append(lines, excludedStyle.Render(excludedLabel))
	if len(data.Inputs) > 10 {
		clearBtn := r.Mark(mouse.ZoneSettingsCodecksExcludedClear, clearButtonStyle.Render("[Clear]"))
		lines = append(lines, "  "+r.Mark(mouse.ZoneSettingsCodecksExcluded, data.Inputs[10].View)+" "+clearBtn)
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    Comma-separated list (e.g., done, archived)"))

	return lines
}

// renderTicketsSettings renders the Tickets settings tab with provider selection
func (r *Renderer) renderTicketsSettings(data SettingsData) []string {
	var lines []string

	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("Ticket Provider"))
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Choose which ticket service to use for the Tickets tab."))
	lines = append(lines, "")

	// Provider selection radio buttons
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("  Active Provider:"))
	lines = append(lines, "")

	// Helper to render a radio option
	renderRadio := func(label, provider, zone string, available bool) string {
		selected := data.TicketProvider == provider
		var radioText string
		if selected {
			radioText = toggleOnStyle.Render("(●) " + label)
		} else if available {
			radioText = lipgloss.NewStyle().Foreground(ColorPrimary).Render("( ) " + label)
		} else {
			radioText = lipgloss.NewStyle().Foreground(ColorMuted).Render("( ) " + label + " (not configured)")
		}
		if available || selected {
			return r.Mark(zone, radioText)
		}
		return radioText
	}

	// None/Disabled option
	lines = append(lines, "    "+renderRadio("None (Disabled)", "", mouse.ZoneSettingsTicketProviderNone, true))

	// Jira option
	lines = append(lines, "    "+renderRadio("Jira", "jira", mouse.ZoneSettingsTicketProviderJira, data.JiraConfigured))

	// Codecks option
	lines = append(lines, "    "+renderRadio("Codecks", "codecks", mouse.ZoneSettingsTicketProviderCodecks, data.CodecksConfigured))

	// GitHub Issues option
	lines = append(lines, "    "+renderRadio("GitHub Issues", "github_issues", mouse.ZoneSettingsTicketProviderGitHubIssues, data.GitHubIssuesConfigured))

	lines = append(lines, "")

	// Show current connection status
	if data.JiraService && data.TicketProviderName != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).Render(
			"  ✓ Connected to "+data.TicketProviderName))
	} else if data.TicketProvider != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB86C")).Render(
			"  ○ "+data.TicketProvider+" selected but not connected (check credentials)"))
	} else {
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render(
			"  ○ No ticket provider selected"))
	}

	lines = append(lines, "")
	lines = append(lines, "")

	// Ticket Workflow section (moved from Advanced)
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("Ticket Workflow"))
	lines = append(lines, "")

	// Auto-transition toggle
	autoInProgressToggle := "[ ]"
	if data.AutoInProgressOnBranch {
		autoInProgressToggle = "[✓]"
	}
	toggleStyle := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	autoBtn := r.Mark(mouse.ZoneSettingsAutoInProgress, toggleStyle.Render(autoInProgressToggle+" Auto-set 'In Progress' on branch creation"))
	lines = append(lines, "  "+autoBtn)
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    Automatically transition ticket when creating a branch from it"))
	lines = append(lines, "")

	// GitHub Issues excluded statuses (only show if GitHub Issues is selected)
	if data.TicketProvider == "github_issues" {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Bold(true).Render("  GitHub Issues Filters:"))
		excludedLabel := "  Exclude Statuses:"
		excludedStyle := lipgloss.NewStyle()
		if data.FocusedField == 11 { // Index for GitHub Issues excluded statuses input
			excludedStyle = excludedStyle.Bold(true).Foreground(ColorPrimary)
		}
		lines = append(lines, excludedStyle.Render(excludedLabel))
		if len(data.Inputs) > 11 {
			clearBtn := r.Mark(mouse.ZoneSettingsGitHubIssuesExcludedClear, clearButtonStyle.Render("[Clear]"))
			lines = append(lines, "  "+r.Mark(mouse.ZoneSettingsGitHubIssuesExcluded, data.Inputs[11].View)+" "+clearBtn)
		}
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    Comma-separated list (e.g., closed)"))
	}

	return lines
}

// renderBranchesSettings renders the Branches settings content
func (r *Renderer) renderBranchesSettings(data SettingsData) []string {
	var lines []string

	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("Branch Settings"))
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Configure how branches are loaded and displayed."))
	lines = append(lines, "")

	// Branch Limit control
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("  Branch Limit:"))
	limitText := lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%d", data.BranchLimit))
	decreaseBtn := r.Mark(mouse.ZoneSettingsBranchLimitDecrease, lipgloss.NewStyle().Foreground(ColorPrimary).Render("[-]"))
	increaseBtn := r.Mark(mouse.ZoneSettingsBranchLimitIncrease, lipgloss.NewStyle().Foreground(ColorPrimary).Render("[+]"))
	lines = append(lines, "    "+decreaseBtn+" "+limitText+" "+increaseBtn)
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    Total branches to show (0 = all)"))
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    Local always included, remote filtered by recency"))

	return lines
}

// renderAdvancedSettings renders the Advanced/Maintenance settings tab
func (r *Renderer) renderAdvancedSettings(data SettingsData) []string {
	var lines []string

	// Bookmark Settings section
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("Bookmark Settings"))
	lines = append(lines, "")

	// Sanitize bookmark names toggle
	sanitizeToggle := "[ ]"
	if data.SanitizeBookmarks {
		sanitizeToggle = "[✓]"
	}
	toggleStyle := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	sanitizeBtn := r.Mark(mouse.ZoneSettingsSanitizeBookmarks, toggleStyle.Render(sanitizeToggle+" Auto-fix bookmark names"))
	lines = append(lines, "  "+sanitizeBtn)
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    Replace spaces and invalid characters with hyphens"))
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

		confirmYes := r.Mark(mouse.ZoneSettingsAdvancedConfirmYes, ButtonStyle.Background(lipgloss.Color("#F85149")).Render("Yes, Confirm"))
		confirmNo := r.Mark(mouse.ZoneSettingsAdvancedConfirmNo, ButtonStyle.Render("Cancel"))
		lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left, confirmYes, " ", confirmNo))
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Press Y to confirm, N/Esc to cancel"))

		return lines
	}

	// Normal operation listing
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorMuted).Render("Available Operations:"))
	lines = append(lines, "")

	deleteBtn := r.Mark(mouse.ZoneSettingsAdvancedDeleteBookmarks, ButtonStyle.Render("Delete All Bookmarks"))
	lines = append(lines, "  "+deleteBtn)
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    Delete all bookmarks in this repository"))
	lines = append(lines, "")

	abandonBtn := r.Mark(mouse.ZoneSettingsAdvancedAbandonOldCommits, ButtonStyle.Render("Abandon Old Commits"))
	lines = append(lines, "  "+abandonBtn)
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("    Abandon commits before origin/main"))

	return lines
}
