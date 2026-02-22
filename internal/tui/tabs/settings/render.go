package settings

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/styles"
	"github.com/madicen/jj-tui/internal/version"
)

// ActiveTab is the active settings sub-tab (0=GitHub, 1=Jira, 2=Codecks, 3=Tickets, 4=Branches, 5=Advanced)
type ActiveTab int

const (
	TabGitHub ActiveTab = iota
	TabJira
	TabCodecks
	TabTickets
	TabBranches
	TabAdvanced
)

// RenderData holds data passed from the main model for rendering the settings view
type RenderData struct {
	Inputs                 []struct{ View string }
	FocusedField           int
	GithubService          bool
	JiraService           bool
	HasLocalConfig         bool
	ConfigSource           string
	ActiveTab              ActiveTab
	ShowMergedPRs          bool
	ShowClosedPRs          bool
	OnlyMyPRs              bool
	PRLimit                int
	PRRefreshInterval      int
	TicketProvider         string
	TicketProviderName     string
	AutoInProgressOnBranch bool
	JiraConfigured         bool
	CodecksConfigured      bool
	GitHubIssuesConfigured bool
	BranchLimit            int
	SanitizeBookmarks      bool
	ConfirmingCleanup      string
}

type renderCtx struct {
	zm *zone.Manager
}

func (r renderCtx) mark(id, content string) string {
	if r.zm != nil {
		return r.zm.Mark(id, content)
	}
	return content
}

var (
	clearButtonStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#F85149")).Bold(true)
	settingsTabStyle = lipgloss.NewStyle().Padding(0, 2).Foreground(styles.ColorMuted)
	settingsTabActive = lipgloss.NewStyle().Padding(0, 2).Bold(true).Foreground(styles.ColorPrimary).Underline(true)
	toggleOnStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).Bold(true)
	toggleOffStyle   = lipgloss.NewStyle().Foreground(styles.ColorMuted)
)

// Render renders the full settings view using the given zone manager and data
func Render(zm *zone.Manager, data RenderData) string {
	r := renderCtx{zm: zm}
	var lines []string

	if data.ConfigSource != "" {
		configInfo := "Config: " + data.ConfigSource
		if data.HasLocalConfig {
			configInfo += " (local override active)"
		}
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Italic(true).Render(configInfo))
	} else {
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Italic(true).Render("Config: ~/.config/jj-tui/config.json"))
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Italic(true).Render("^j/^k: switch tabs  Tab: next field"))
	lines = append(lines, "")

	lines = append(lines, r.renderTabs(data.ActiveTab))
	lines = append(lines, "")

	switch data.ActiveTab {
	case TabGitHub:
		lines = append(lines, r.renderGitHub(data)...)
	case TabJira:
		lines = append(lines, r.renderJira(data)...)
	case TabCodecks:
		lines = append(lines, r.renderCodecks(data)...)
	case TabTickets:
		lines = append(lines, r.renderTickets(data)...)
	case TabBranches:
		lines = append(lines, r.renderBranches(data)...)
	case TabAdvanced:
		lines = append(lines, r.renderAdvanced(data)...)
	}

	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Connection Status:"))
	if data.GithubService {
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).Render("  ✓ GitHub connected"))
	} else {
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("  ○ GitHub not connected"))
	}
	if data.JiraService {
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).Render("  ✓ Tickets connected"))
	} else {
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("  ○ Tickets not connected"))
	}

	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Version:"))
	versionStr := version.GetVersion()
	versionLine := lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("  " + versionStr)
	if updateInfo := version.GetUpdateInfo(); updateInfo != nil {
		if updateInfo.UpdateAvailable {
			versionLine += lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB86C")).Render(fmt.Sprintf(" → %s available", updateInfo.LatestVersion))
		} else if updateInfo.LatestVersion != "" && updateInfo.LatestVersion != "dev" {
			versionLine += lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).Render(" (up to date)")
		}
	}
	lines = append(lines, versionLine)
	lines = append(lines, "", "")

	saveBtn := r.mark(mouse.ZoneSettingsSave, styles.ButtonStyle.Render("Save Global (^s)"))
	saveLocalBtn := r.mark(mouse.ZoneSettingsSaveLocal, styles.ButtonStyle.Render("Save Local (^l)"))
	cancelBtn := r.mark(mouse.ZoneSettingsCancel, styles.ButtonStyle.Render("Cancel (Esc)"))
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left, saveBtn, " ", saveLocalBtn, " ", cancelBtn))

	return strings.Join(lines, "\n")
}

func (r renderCtx) renderTabs(active ActiveTab) string {
	githubStyle := settingsTabStyle
	jiraStyle := settingsTabStyle
	codecksStyle := settingsTabStyle
	ticketsStyle := settingsTabStyle
	branchesStyle := settingsTabStyle
	advancedStyle := settingsTabStyle
	switch active {
	case TabGitHub:
		githubStyle = settingsTabActive
	case TabJira:
		jiraStyle = settingsTabActive
	case TabCodecks:
		codecksStyle = settingsTabActive
	case TabTickets:
		ticketsStyle = settingsTabActive
	case TabBranches:
		branchesStyle = settingsTabActive
	case TabAdvanced:
		advancedStyle = settingsTabActive
	}
	githubTab := r.mark(mouse.ZoneSettingsTabGitHub, githubStyle.Render("GitHub"))
	jiraTab := r.mark(mouse.ZoneSettingsTabJira, jiraStyle.Render("Jira"))
	codecksTab := r.mark(mouse.ZoneSettingsTabCodecks, codecksStyle.Render("Codecks"))
	ticketsTab := r.mark(mouse.ZoneSettingsTabTickets, ticketsStyle.Render("Tickets"))
	branchesTab := r.mark(mouse.ZoneSettingsTabBranches, branchesStyle.Render("Branches"))
	advancedTab := r.mark(mouse.ZoneSettingsTabAdvanced, advancedStyle.Render("Advanced"))
	return lipgloss.JoinHorizontal(lipgloss.Left, githubTab, " │ ", jiraTab, " │ ", codecksTab, " │ ", ticketsTab, " │ ", branchesTab, " │ ", advancedTab)
}

func (r renderCtx) renderToggle(label string, enabled bool, zoneID string) string {
	if enabled {
		return r.mark(zoneID, toggleOnStyle.Render("[✓]")+" "+label)
	}
	return r.mark(zoneID, toggleOffStyle.Render("[ ]")+" "+lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(label))
}

func (r renderCtx) renderGitHub(data RenderData) []string {
	var lines []string
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary).Render("GitHub Integration"))
	lines = append(lines, "", lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("Connect to GitHub for PR management."), "")

	if data.GithubService {
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).Render("  ✓ Connected to GitHub"))
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    "+r.mark(mouse.ZoneSettingsGitHubLogin, "[Reconnect]")))
	} else {
		lines = append(lines, "  "+r.mark(mouse.ZoneSettingsGitHubLogin, styles.ButtonStyle.Background(lipgloss.Color("#238636")).Render("Login with GitHub")))
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Opens browser to authenticate via GitHub App"))
	}
	lines = append(lines, "")

	labelStyle := lipgloss.NewStyle().Foreground(styles.ColorMuted)
	if data.FocusedField == 0 {
		labelStyle = labelStyle.Bold(true).Foreground(styles.ColorPrimary)
	}
	lines = append(lines, labelStyle.Render("  Or enter token manually:"))
	if len(data.Inputs) > 0 {
		lines = append(lines, "  "+r.mark(mouse.ZoneSettingsGitHubToken, data.Inputs[0].View)+" "+r.mark(mouse.ZoneSettingsGitHubTokenClear, clearButtonStyle.Render("[Clear]")))
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Get a token from: https://github.com/settings/tokens"), "")

	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("  PR Filters:"), "")
	lines = append(lines, "    "+r.renderToggle("Only My PRs", data.OnlyMyPRs, mouse.ZoneSettingsGitHubOnlyMine))
	lines = append(lines, "    "+r.renderToggle("Show Merged PRs", data.ShowMergedPRs, mouse.ZoneSettingsGitHubShowMerged))
	lines = append(lines, "    "+r.renderToggle("Show Closed PRs", data.ShowClosedPRs, mouse.ZoneSettingsGitHubShowClosed))
	lines = append(lines, "")

	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("  PR Limit:"))
	lines = append(lines, "    "+r.mark(mouse.ZoneSettingsGitHubPRLimitDecrease, lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render("[-]"))+" "+
		lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%d", data.PRLimit))+" "+
		r.mark(mouse.ZoneSettingsGitHubPRLimitIncrease, lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render("[+]")))
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Max PRs to load (reduces API calls)"), "")

	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("  PR Auto-Refresh:"))
	var refreshText string
	if data.PRRefreshInterval == 0 {
		refreshText = lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("Disabled")
	} else if data.PRRefreshInterval < 60 {
		refreshText = lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%ds", data.PRRefreshInterval))
	} else {
		refreshText = lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%dm", data.PRRefreshInterval/60))
	}
	lines = append(lines, "    "+r.mark(mouse.ZoneSettingsGitHubRefreshDecrease, lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render("[-]"))+" "+
		refreshText+" "+
		r.mark(mouse.ZoneSettingsGitHubRefreshIncrease, lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render("[+]"))+" "+
		r.mark(mouse.ZoneSettingsGitHubRefreshToggle, lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render("[Toggle]")))
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Auto-refresh PRs when viewing PR tab (0 = disabled)"))
	return lines
}

func (r renderCtx) renderJira(data RenderData) []string {
	var lines []string
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary).Render("Jira Integration"))
	lines = append(lines, "", lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("Connect to Jira for ticket management."), "")

	focusStyle := func(i int) lipgloss.Style {
		s := lipgloss.NewStyle()
		if data.FocusedField == i {
			return s.Bold(true).Foreground(styles.ColorPrimary)
		}
		return s
	}
	addField := func(label string, idx int, zoneID, clearZone string) {
		lines = append(lines, focusStyle(idx).Render(label))
		if len(data.Inputs) > idx {
			lines = append(lines, "  "+r.mark(zoneID, data.Inputs[idx].View)+" "+r.mark(clearZone, clearButtonStyle.Render("[Clear]")))
		}
	}
	addField("  Instance URL:", 1, mouse.ZoneSettingsJiraURL, mouse.ZoneSettingsJiraURLClear)
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    e.g., https://yourcompany.atlassian.net"), "")
	addField("  Email:", 2, mouse.ZoneSettingsJiraUser, mouse.ZoneSettingsJiraUserClear)
	addField("  API Token:", 3, mouse.ZoneSettingsJiraToken, mouse.ZoneSettingsJiraTokenClear)
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Get from: https://id.atlassian.com/manage-profile/security/api-tokens"), "")
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("  Ticket Filters:"))
	addField("  Project(s):", 4, mouse.ZoneSettingsJiraProject, mouse.ZoneSettingsJiraProjectClear)
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Project key(s) to filter (e.g., PROJ or PROJ,TEAM)"), "")
	addField("  Custom JQL:", 5, mouse.ZoneSettingsJiraJQL, mouse.ZoneSettingsJiraJQLClear)
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Additional JQL filter (e.g., sprint in openSprints())"), "")
	addField("  Exclude Statuses:", 6, mouse.ZoneSettingsJiraExcluded, mouse.ZoneSettingsJiraExcludedClear)
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Comma-separated list (e.g., Done, Won't Do, Cancelled)"))
	return lines
}

func (r renderCtx) renderCodecks(data RenderData) []string {
	var lines []string
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary).Render("Codecks Integration"))
	lines = append(lines, "", lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("Connect to Codecks for card management."), "")

	focusStyle := func(i int) lipgloss.Style {
		s := lipgloss.NewStyle()
		if data.FocusedField == i {
			return s.Bold(true).Foreground(styles.ColorPrimary)
		}
		return s
	}
	addField := func(label string, idx int, zoneID, clearZone string) {
		lines = append(lines, focusStyle(idx).Render(label))
		if len(data.Inputs) > idx {
			lines = append(lines, "  "+r.mark(zoneID, data.Inputs[idx].View)+" "+r.mark(clearZone, clearButtonStyle.Render("[Clear]")))
		}
	}
	addField("  Subdomain:", 7, mouse.ZoneSettingsCodecksSubdomain, mouse.ZoneSettingsCodecksSubdomainClear)
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Your team name (e.g., 'myteam' from myteam.codecks.io)"), "")
	addField("  Auth Token:", 8, mouse.ZoneSettingsCodecksToken, mouse.ZoneSettingsCodecksTokenClear)
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Extract 'at' cookie from browser DevTools"), "")
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("  Card Filters:"))
	addField("  Project Filter:", 9, mouse.ZoneSettingsCodecksProject, mouse.ZoneSettingsCodecksProjectClear)
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Optional: Only show cards from this project"), "")
	addField("  Exclude Statuses:", 10, mouse.ZoneSettingsCodecksExcluded, mouse.ZoneSettingsCodecksExcludedClear)
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Comma-separated list (e.g., done, archived)"))
	return lines
}

func (r renderCtx) renderTickets(data RenderData) []string {
	var lines []string
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary).Render("Ticket Provider"))
	lines = append(lines, "", lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("Choose which ticket service to use for the Tickets tab."), "")
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("  Active Provider:"), "")

	renderRadio := func(label, provider, zone string, available bool) string {
		selected := data.TicketProvider == provider
		var radioText string
		if selected {
			radioText = toggleOnStyle.Render("(●) " + label)
		} else if available {
			radioText = lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render("( ) " + label)
		} else {
			radioText = lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("( ) " + label + " (not configured)")
		}
		if available || selected {
			return r.mark(zone, radioText)
		}
		return radioText
	}
	lines = append(lines, "    "+renderRadio("None (Disabled)", "", mouse.ZoneSettingsTicketProviderNone, true))
	lines = append(lines, "    "+renderRadio("Jira", "jira", mouse.ZoneSettingsTicketProviderJira, data.JiraConfigured))
	lines = append(lines, "    "+renderRadio("Codecks", "codecks", mouse.ZoneSettingsTicketProviderCodecks, data.CodecksConfigured))
	lines = append(lines, "    "+renderRadio("GitHub Issues", "github_issues", mouse.ZoneSettingsTicketProviderGitHubIssues, data.GitHubIssuesConfigured))
	lines = append(lines, "")

	if data.JiraService && data.TicketProviderName != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).Render("  ✓ Connected to "+data.TicketProviderName))
	} else if data.TicketProvider != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB86C")).Render("  ○ "+data.TicketProvider+" selected but not connected (check credentials)"))
	} else {
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("  ○ No ticket provider selected"))
	}
	lines = append(lines, "", "")

	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary).Render("Ticket Workflow"), "")
	toggleStr := "[ ]"
	if data.AutoInProgressOnBranch {
		toggleStr = "[✓]"
	}
	lines = append(lines, "  "+r.mark(mouse.ZoneSettingsAutoInProgress, lipgloss.NewStyle().Foreground(styles.ColorPrimary).Bold(true).Render(toggleStr+" Auto-set 'In Progress' on branch creation")))
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Automatically transition ticket when creating a branch from it"), "")

	if data.TicketProvider == "github_issues" {
		lines = append(lines, "", lipgloss.NewStyle().Bold(true).Render("  GitHub Issues Filters:"))
		s := lipgloss.NewStyle()
		if data.FocusedField == 11 {
			s = s.Bold(true).Foreground(styles.ColorPrimary)
		}
		lines = append(lines, s.Render("  Exclude Statuses:"))
		if len(data.Inputs) > 11 {
			lines = append(lines, "  "+r.mark(mouse.ZoneSettingsGitHubIssuesExcluded, data.Inputs[11].View)+" "+r.mark(mouse.ZoneSettingsGitHubIssuesExcludedClear, clearButtonStyle.Render("[Clear]")))
		}
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Comma-separated list (e.g., closed)"))
	}
	return lines
}

func (r renderCtx) renderBranches(data RenderData) []string {
	var lines []string
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary).Render("Branch Settings"))
	lines = append(lines, "", lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("Configure how branches are loaded and displayed."), "")
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("  Branch Limit:"))
	lines = append(lines, "    "+r.mark(mouse.ZoneSettingsBranchLimitDecrease, lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render("[-]"))+" "+
		lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%d", data.BranchLimit))+" "+
		r.mark(mouse.ZoneSettingsBranchLimitIncrease, lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render("[+]")))
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Total branches to show (0 = all)"))
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Local always included, remote filtered by recency"))
	return lines
}

func (r renderCtx) renderAdvanced(data RenderData) []string {
	var lines []string
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary).Render("Bookmark Settings"), "")
	toggleStr := "[ ]"
	if data.SanitizeBookmarks {
		toggleStr = "[✓]"
	}
	lines = append(lines, "  "+r.mark(mouse.ZoneSettingsSanitizeBookmarks, lipgloss.NewStyle().Foreground(styles.ColorPrimary).Bold(true).Render(toggleStr+" Auto-fix bookmark names")))
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Replace spaces and invalid characters with hyphens"), "", "")

	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary).Render("Advanced Maintenance"), "")
	lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#F85149")).Bold(true).Render("WARNING: Destructive operations. Use caution!"), "")

	if data.ConfirmingCleanup != "" {
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary).Render("Are you sure? This cannot be undone."), "")
		lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left,
			r.mark(mouse.ZoneSettingsAdvancedConfirmYes, styles.ButtonStyle.Background(lipgloss.Color("#F85149")).Render("Yes, Confirm")),
			" ", r.mark(mouse.ZoneSettingsAdvancedConfirmNo, styles.ButtonStyle.Render("Cancel"))))
		lines = append(lines, "", lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("Press Y to confirm, N/Esc to cancel"))
		return lines
	}

	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(styles.ColorMuted).Render("Available Operations:"), "")
	lines = append(lines, "  "+r.mark(mouse.ZoneSettingsAdvancedDeleteBookmarks, styles.ButtonStyle.Render("Delete All Bookmarks")))
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Delete all bookmarks in this repository"), "")
	lines = append(lines, "  "+r.mark(mouse.ZoneSettingsAdvancedAbandonOldCommits, styles.ButtonStyle.Render("Abandon Old Commits")))
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Abandon commits before origin/main"))
	return lines
}
