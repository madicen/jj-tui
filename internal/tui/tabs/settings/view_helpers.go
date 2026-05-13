package settings

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/styles"
	"github.com/madicen/jj-tui/internal/tui/tabs/settings/advanced"
	"github.com/madicen/jj-tui/internal/tui/tabs/settings/theme"
	"github.com/madicen/jj-tui/internal/version"
)

// ActiveTab is the selected settings sub-tab. Indices and labels:
// 0 GitHub, 1 Jira, 2 Codecks, 3 Tickets, 4 Branches, 5 Theme, 6 AI, 7 Advanced.
type ActiveTab int

const (
	TabGitHub ActiveTab = iota
	TabJira
	TabCodecks
	TabTickets
	TabBranches
	TabTheme
	TabAI
	TabAdvanced
)

// RenderData holds data passed from the main model for rendering the settings view
type RenderData struct {
	Inputs                 []struct{ View string }
	FocusedField           int
	GithubService          bool
	JiraService            bool
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
	GitHubTokenSource      string // saved | env | gh_cli
	// Repository remote section (Settings → GitHub)
	CurrentOrigin     string // live `origin` URL ("" if not configured); cached on Settings open
	OriginInputView   string // rendered view of the origin URL textinput
	GhAvailable       bool   // gh CLI present in PATH (controls the "Create new GitHub repo" button)
	GhRepoPrivate     bool   // visibility flag for "Create new GitHub repo" (true => --private)
	BranchLimit            int
	SanitizeBookmarks      bool
	ConfirmingCleanup      string
	ExternalEditorPreset   int // Advanced: selected external editor preset index (radio rows)
	AIEnabled              bool
	AIProviderID           string // openai_compatible | gemini | ollama
	AIAPIKeySet            bool   // key present (env overrides config)
	AITimeoutSeconds       int    // HTTP timeout for LLM requests; rendered as a [-] N [+] stepper
	EvologDescribeDefault  bool
	EvologFileSplitEnabled bool
	EvologHunkSplitEnabled bool
	EvologMultiStepwise    bool
	EvologMultiMax         int

	// ThemeModel is set by BuildRenderData for rendering the Theme tab (swatches + bounds).
	ThemeModel *theme.Model

	// Scroll: when ContentHeight > 0, only lines [YOffset : YOffset+ContentHeight] are shown
	YOffset       int
	ContentHeight int
}

// ViewOpts holds data from the main model needed to render settings (avoids main building RenderData).
type ViewOpts struct {
	GitHubAvailable   bool
	TicketServiceName string
	Config            *config.Config
	ContentHeight     int
	// GhAvailable mirrors `gh` CLI presence in PATH (cached by main on Settings open). Used by
	// renderGitHub to decide whether to show the "Create new GitHub repo" button or a hint.
	GhAvailable bool
}

// BuildRenderData builds RenderData from the settings model and opts. Used by RenderWithState.
func BuildRenderData(sm *Model, opts ViewOpts) RenderData {
	data := RenderData{
		FocusedField:           sm.GetFocusedField(),
		GithubService:          opts.GitHubAvailable,
		JiraService:            opts.TicketServiceName != "",
		ActiveTab:              ActiveTab(sm.GetActiveSettingsTabIndex()),
		ShowMergedPRs:          sm.GetSettingsShowMerged(),
		ShowClosedPRs:          sm.GetSettingsShowClosed(),
		OnlyMyPRs:              sm.GetSettingsOnlyMine(),
		PRLimit:                sm.GetSettingsPRLimit(),
		PRRefreshInterval:      sm.GetSettingsPRRefreshInterval(),
		TicketProvider:         sm.GetSettingsTicketProvider(),
		TicketProviderName:     opts.TicketServiceName,
		AutoInProgressOnBranch: sm.GetSettingsAutoInProgress(),
		BranchLimit:            sm.GetSettingsBranchLimit(),
		SanitizeBookmarks:      sm.GetSettingsSanitizeBookmarks(),
		ConfirmingCleanup:      sm.GetConfirmingCleanup(),
		ExternalEditorPreset:   sm.GetAdvancedModel().GetExternalEditorPreset(),
		AIEnabled:              sm.GetAIModel().GetAIEnabled(),
		AIProviderID:           sm.GetAIModel().GetAIProvider(),
		AIAPIKeySet:            opts.Config != nil && opts.Config.AISupportsGenerationCredentials(),
		AITimeoutSeconds:       sm.GetAIModel().GetAITimeoutSeconds(),
		EvologDescribeDefault:  sm.GetAIModel().GetEvologDescribeAfterSplitDefault(),
		EvologFileSplitEnabled: sm.GetAIModel().GetEvologFileSplitEnabled(),
		EvologHunkSplitEnabled: sm.GetAIModel().GetEvologHunkSplitEnabled(),
		EvologMultiStepwise:    sm.GetAIModel().GetEvologMultiStepwise(),
		EvologMultiMax:         sm.GetAIModel().GetEvologMultiMax(),
		YOffset:                sm.GetSettingsYOffset(),
		ContentHeight:          opts.ContentHeight,
		ThemeModel:             sm.GetThemeModel(),
		GitHubTokenSource:      sm.GetGitHubTokenSource(),
		CurrentOrigin:          sm.GetGitHubModel().GetCurrentOrigin(),
		OriginInputView:        sm.GetGitHubModel().GetOriginInputView(),
		GhAvailable:            opts.GhAvailable,
		GhRepoPrivate:          sm.GetGitHubModel().GetGhPrivate(),
	}
	data.Inputs = sm.GetSettingsInputs()
	data.HasLocalConfig = config.HasLocalConfig()
	if opts.Config != nil {
		data.ConfigSource = opts.Config.LoadedFrom()
	}
	jr := sm.GetJiraModel()
	data.JiraConfigured = strings.TrimSpace(jr.GetURL()) != "" &&
		strings.TrimSpace(jr.GetUser()) != "" &&
		strings.TrimSpace(jr.GetToken()) != ""
	cc := sm.GetCodecksModel()
	data.CodecksConfigured = strings.TrimSpace(cc.GetSubdomain()) != "" &&
		strings.TrimSpace(cc.GetToken()) != ""
	data.GitHubIssuesConfigured = opts.GitHubAvailable
	return data
}

// RenderWithState renders the settings view from the settings model and opts (main only passes opts).
func RenderWithState(zm *zone.Manager, sm *Model, opts ViewOpts) string {
	return Render(zm, BuildRenderData(sm, opts))
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
	case TabTheme:
		themeStartRow := len(lines)
		lines = append(lines, r.renderTheme(data, themeStartRow)...)
	case TabAI:
		lines = append(lines, r.renderAI(data)...)
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

	if data.ContentHeight > 0 {
		visibleHeight := data.ContentHeight
		totalLines := len(lines)
		maxOffset := max(0, totalLines-visibleHeight)
		start := min(data.YOffset, maxOffset)
		if start < 0 {
			start = 0
		}
		end := min(start+visibleHeight, totalLines)
		lines = lines[start:end]
	}
	return strings.Join(lines, "\n")
}

func (r renderCtx) renderTabs(active ActiveTab) string {
	githubStyle := settingsTabStyle
	jiraStyle := settingsTabStyle
	codecksStyle := settingsTabStyle
	ticketsStyle := settingsTabStyle
	branchesStyle := settingsTabStyle
	themeStyle := settingsTabStyle
	aiStyle := settingsTabStyle
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
	case TabTheme:
		themeStyle = settingsTabActive
	case TabAI:
		aiStyle = settingsTabActive
	case TabAdvanced:
		advancedStyle = settingsTabActive
	}
	githubTab := r.mark(mouse.ZoneSettingsTabGitHub, githubStyle.Render("GitHub"))
	jiraTab := r.mark(mouse.ZoneSettingsTabJira, jiraStyle.Render("Jira"))
	codecksTab := r.mark(mouse.ZoneSettingsTabCodecks, codecksStyle.Render("Codecks"))
	ticketsTab := r.mark(mouse.ZoneSettingsTabTickets, ticketsStyle.Render("Tickets"))
	branchesTab := r.mark(mouse.ZoneSettingsTabBranches, branchesStyle.Render("Branches"))
	themeTab := r.mark(mouse.ZoneSettingsTabTheme, themeStyle.Render("Theme"))
	aiTab := r.mark(mouse.ZoneSettingsTabAI, aiStyle.Render("AI"))
	advancedTab := r.mark(mouse.ZoneSettingsTabAdvanced, advancedStyle.Render("Advanced"))
	return lipgloss.JoinHorizontal(lipgloss.Left, githubTab, " │ ", jiraTab, " │ ", codecksTab, " │ ", ticketsTab, " │ ", branchesTab, " │ ", themeTab, " │ ", aiTab, " │ ", advancedTab)
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

	src := data.GitHubTokenSource
	if src == "" {
		src = config.GitHubTokenSourceSaved
	}
	authRadio := func(label, value, zone string) string {
		selected := src == value
		var radioText string
		if selected {
			radioText = toggleOnStyle.Render("(●) " + label)
		} else {
			radioText = lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render("( ) " + label)
		}
		return r.mark(zone, radioText)
	}
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("  API token source:"), "")
	lines = append(lines, "    "+authRadio("Saved in jj-tui (device flow or paste below)", config.GitHubTokenSourceSaved, mouse.ZoneSettingsGitHubAuthSaved))
	lines = append(lines, "    "+authRadio("Environment variable (GITHUB_TOKEN)", config.GitHubTokenSourceEnv, mouse.ZoneSettingsGitHubAuthEnv))
	lines = append(lines, "    "+authRadio("GitHub CLI (`gh auth token`)", config.GitHubTokenSourceGhCLI, mouse.ZoneSettingsGitHubAuthGhCLI))
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Only the selected source is used (no fallback). Restart jj-tui after changing GITHUB_TOKEN in your shell."), "")
	lines = append(lines, "")

	loginBtn := "Login with GitHub"
	if src == config.GitHubTokenSourceGhCLI {
		loginBtn = "Login with GitHub CLI"
	}
	if data.GithubService {
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).Render("  ✓ Connected to GitHub"))
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    "+r.mark(mouse.ZoneSettingsGitHubLogin, "[Reconnect]")))
	} else {
		lines = append(lines, "  "+r.mark(mouse.ZoneSettingsGitHubLogin, styles.ButtonStyle.Background(lipgloss.Color("#238636")).Render(loginBtn)))
		if src == config.GitHubTokenSourceGhCLI {
			lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Opens `gh auth login` (token from `gh auth token`)"))
		} else {
			lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Device flow stores a token under \"Saved in jj-tui\""))
		}
	}
	lines = append(lines, "")

	labelStyle := lipgloss.NewStyle().Foreground(styles.ColorMuted)
	if data.FocusedField == 0 {
		labelStyle = labelStyle.Bold(true).Foreground(styles.ColorPrimary)
	}
	lines = append(lines, labelStyle.Render("  Token (saved in jj-tui only):"))
	if len(data.Inputs) > 0 {
		if src == config.GitHubTokenSourceSaved {
			lines = append(lines, "  "+r.mark(mouse.ZoneSettingsGitHubToken, data.Inputs[0].View)+" "+r.mark(mouse.ZoneSettingsGitHubTokenClear, clearButtonStyle.Render("[Clear]")))
		} else {
			lines = append(lines, "  "+lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("(select \"Saved in jj-tui\" to edit the stored token)"))
		}
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Optional manual PAT: https://github.com/settings/tokens"), "")

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
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Auto-refresh PRs when viewing PR tab (0 = disabled)"), "")

	lines = append(lines, r.renderRepositoryRemote(data)...)
	return lines
}

// renderRepositoryRemote renders the "Repository remote" subsection of the GitHub settings tab.
// Action-oriented (not part of Save): Apply / Create / Remove fire commands directly via
// navigation messages handled by main, mirroring the welcome-screen flow.
func (r renderCtx) renderRepositoryRemote(data RenderData) []string {
	var lines []string
	muted := lipgloss.NewStyle().Foreground(styles.ColorMuted)
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary).Render("Repository remote"))
	lines = append(lines, "")
	lines = append(lines, muted.Render("  The git remote `origin` jj-tui pushes branches to. Required for `Update PR`,"))
	lines = append(lines, muted.Render("  branch push, and bookmark tracking against a remote `main`."), "")

	currentLine := "  Current origin:  " + muted.Render("(none configured)")
	if data.CurrentOrigin != "" {
		currentLine = "  Current origin:  " + lipgloss.NewStyle().Bold(true).Render(data.CurrentOrigin)
	}
	lines = append(lines, currentLine, "")

	urlLabelStyle := muted
	if data.FocusedField == 5 {
		urlLabelStyle = lipgloss.NewStyle().Foreground(styles.ColorPrimary).Bold(true)
	}
	lines = append(lines, "  "+urlLabelStyle.Render("Remote URL:"))
	lines = append(lines, "  "+r.mark(mouse.ZoneSettingsRemoteOriginInput, data.OriginInputView))
	lines = append(lines, muted.Render("    Tab/down to focus, paste a URL, then press Enter or click Apply."), "")

	// Default to the no-origin label and override only when an origin already exists; the
	// previous "Apply (^enter)" generic fallback was unused and tripped ineffassign.
	applyLabel := "Add origin (^enter)"
	if data.CurrentOrigin != "" {
		applyLabel = "Update origin (^enter)"
	}
	applyButton := styles.ButtonStyle.Background(lipgloss.Color("#238636")).Render(applyLabel)
	removeButton := styles.ButtonStyle.Background(lipgloss.Color("#8b1a1a")).Render("Remove origin (^x)")
	if data.CurrentOrigin == "" {
		// Render a disabled-looking style when there's nothing to remove. Still mark the zone so
		// the click is captured and we can show a clean status message ("no origin to remove").
		removeButton = lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("[Remove origin (n/a)]")
	}
	lines = append(lines,
		"  "+r.mark(mouse.ZoneSettingsRemoteApply, applyButton)+"  "+r.mark(mouse.ZoneSettingsRemoteRemove, removeButton),
	)
	lines = append(lines, "")

	// Push buttons: enabled only when origin is configured. When no origin is set the buttons
	// are rendered in muted style with a parenthetical hint pointing the user at Apply / Create
	// first; mouse zones are still attached so clicks produce a clear status message rather
	// than silently no-op.
	pushCurrentLabel := "Push current bookmark (p)"
	pushAllLabel := "Push all bookmarks (P)"
	var pushCurrentBtn, pushAllBtn string
	if data.CurrentOrigin == "" {
		pushCurrentBtn = lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("[" + pushCurrentLabel + " — set origin first]")
		pushAllBtn = lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("[" + pushAllLabel + " — set origin first]")
	} else {
		pushCurrentBtn = styles.ButtonStyle.Background(lipgloss.Color("#1f6feb")).Render(pushCurrentLabel)
		pushAllBtn = styles.ButtonStyle.Background(lipgloss.Color("#1f6feb")).Render(pushAllLabel)
	}
	lines = append(lines,
		"  "+r.mark(mouse.ZoneSettingsRemotePushCurrent, pushCurrentBtn)+"  "+r.mark(mouse.ZoneSettingsRemotePushAll, pushAllBtn),
	)
	lines = append(lines, muted.Render("    Runs `jj git push --allow-new` (current) or once per local bookmark via `--bookmark <name>` (all)."), "")

	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("  Or create a brand-new GitHub repo"))
	lines = append(lines, "")
	if data.GhAvailable {
		visLabel := "Public"
		if data.GhRepoPrivate {
			visLabel = "Private"
		}
		ghButton := styles.ButtonStyle.Background(lipgloss.Color("#1f6feb")).Render("Create new GitHub repo (g)")
		visButton := styles.ButtonStyle.Render(fmt.Sprintf("Visibility: %s  (^v)", visLabel))
		lines = append(lines,
			"  "+r.mark(mouse.ZoneSettingsRemoteCreateGh, ghButton)+"  "+r.mark(mouse.ZoneSettingsRemoteVisibilityToggle, visButton),
		)
		lines = append(lines, muted.Render(fmt.Sprintf("    Runs `gh repo create <dir> --%s --source=. --remote=origin` and then pushes all", strings.ToLower(visLabel))))
		lines = append(lines, muted.Render("    local bookmarks to the new origin. Requires gh CLI authentication."))
	} else {
		lines = append(lines, muted.Render("  `gh` CLI not found in PATH. Install GitHub CLI and run `gh auth login` to enable this option."))
	}
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
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("  Creating issues:"))
	addField("  Project for new issues:", 4, mouse.ZoneSettingsJiraProject, mouse.ZoneSettingsJiraProjectClear)
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Project key when creating issues (e.g., PROJ)"), "")
	addField("  Default issue type:", 6, mouse.ZoneSettingsJiraIssueType, mouse.ZoneSettingsJiraIssueTypeClear)
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Type when creating issues (e.g., Task, Bug, Story). Empty = Task"), "")
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("  Ticket Filters:"))
	addField("  Project filter(s):", 5, mouse.ZoneSettingsJiraProjectFilter, mouse.ZoneSettingsJiraProjectFilterClear)
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Optional: filter ticket list by project(s) (e.g., PROJ or PROJ,TEAM)"), "")
	addField("  Custom JQL:", 7, mouse.ZoneSettingsJiraJQL, mouse.ZoneSettingsJiraJQLClear)
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Additional JQL filter (e.g., sprint in openSprints())"), "")
	addField("  Exclude Statuses:", 8, mouse.ZoneSettingsJiraExcluded, mouse.ZoneSettingsJiraExcludedClear)
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
	addField("  Subdomain:", 9, mouse.ZoneSettingsCodecksSubdomain, mouse.ZoneSettingsCodecksSubdomainClear)
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Your team name (e.g., 'myteam' from myteam.codecks.io)"), "")
	addField("  Auth Token:", 10, mouse.ZoneSettingsCodecksToken, mouse.ZoneSettingsCodecksTokenClear)
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Extract 'at' cookie from browser DevTools"), "")
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("  Card Filters:"))
	addField("  Project Filter:", 11, mouse.ZoneSettingsCodecksProject, mouse.ZoneSettingsCodecksProjectClear)
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Optional: Only show cards from this project"), "")
	addField("  Exclude Statuses:", 12, mouse.ZoneSettingsCodecksExcluded, mouse.ZoneSettingsCodecksExcludedClear)
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
		if data.FocusedField == 13 {
			s = s.Bold(true).Foreground(styles.ColorPrimary)
		}
		lines = append(lines, s.Render("  Exclude Statuses:"))
		if len(data.Inputs) > 13 {
			lines = append(lines, "  "+r.mark(mouse.ZoneSettingsGitHubIssuesExcluded, data.Inputs[13].View)+" "+r.mark(mouse.ZoneSettingsGitHubIssuesExcludedClear, clearButtonStyle.Render("[Clear]")))
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

// themeLabelWidth is the fixed width for theme labels so swatches align in the same column.
const themeLabelWidth = 12

func (r renderCtx) renderTheme(data RenderData, startRow int) []string {
	var lines []string
	if data.ThemeModel == nil {
		return lines
	}
	tm := data.ThemeModel
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary).Render("Theme Colors"))
	lines = append(lines, "", lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("Click a swatch to change the color. Save (^s or ^l) to persist."), "")

	sw, sh := tm.Swatch(0).Size()
	const labelPrefix = "  "
	// Pad labels to themeLabelWidth so all swatches start at the same column
	primaryLabel := fmt.Sprintf("%-*s", themeLabelWidth, "Primary:")
	secondaryLabel := fmt.Sprintf("%-*s", themeLabelWidth, "Secondary:")
	mutedLabel := fmt.Sprintf("%-*s", themeLabelWidth, "Muted:")
	swatchCol := len(labelPrefix) + themeLabelWidth
	tm.SetBounds(0, startRow+2, swatchCol, sw, sh)
	tm.SetBounds(1, startRow+3, swatchCol, sw, sh)
	tm.SetBounds(2, startRow+4, swatchCol, sw, sh)

	lines = append(lines, labelPrefix+r.mark(mouse.ZoneSettingsThemePrimary, primaryLabel+tm.Swatch(0).SwatchView())+" "+r.mark(mouse.ZoneSettingsThemePrimaryDefault, clearButtonStyle.Render("[Default]")))
	lines = append(lines, labelPrefix+r.mark(mouse.ZoneSettingsThemeSecondary, secondaryLabel+tm.Swatch(1).SwatchView())+" "+r.mark(mouse.ZoneSettingsThemeSecondaryDefault, clearButtonStyle.Render("[Default]")))
	lines = append(lines, labelPrefix+r.mark(mouse.ZoneSettingsThemeMuted, mutedLabel+tm.Swatch(2).SwatchView())+" "+r.mark(mouse.ZoneSettingsThemeMutedDefault, clearButtonStyle.Render("[Default]")))
	return lines
}

func (r renderCtx) renderAI(data RenderData) []string {
	var lines []string
	focusStyle := func(i int) lipgloss.Style {
		s := lipgloss.NewStyle()
		if data.FocusedField == i {
			return s.Bold(true).Foreground(styles.ColorPrimary)
		}
		return s
	}
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary).Render("AI assist"), "")
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    API keys can be stored in config (0600) or provided via env ("+config.EnvAIAPIKey+" overrides). Sending a diff exposes code to the API."), "")
	toggleAI := "[ ]"
	if data.AIEnabled {
		toggleAI = "[✓]"
	}
	lines = append(lines, "  "+r.mark(mouse.ZoneSettingsAIEnabled, lipgloss.NewStyle().Foreground(styles.ColorPrimary).Bold(true).Render(toggleAI+" Enable AI (✧ ^g chip in description, PR, Create ticket, and bookmark modals)")))
	curProv := strings.TrimSpace(data.AIProviderID)
	if curProv == "" {
		curProv = "openai_compatible"
	}
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("  Provider:"), "")
	renderAIProv := func(idx int, id string, label string) string {
		selected := curProv == id
		var radioText string
		if selected {
			radioText = toggleOnStyle.Render("(●) " + label)
		} else {
			radioText = lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render("( ) " + label)
		}
		return r.mark(mouse.ZoneSettingsAIProvider(idx), radioText)
	}
	lines = append(lines, "    "+renderAIProv(0, "openai_compatible", "OpenAI-compatible (Chat Completions)"))
	lines = append(lines, "    "+renderAIProv(1, "gemini", "Google Gemini (Generative Language API)"))
	lines = append(lines, "    "+renderAIProv(2, "ollama", "Ollama (local Chat Completions)"))
	lines = append(lines, "")
	if data.AIAPIKeySet {
		lines = append(lines, "  "+lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).Render("LLM credentials: ready (API key, Ollama preset, or local Ollama URL)"))
	} else {
		lines = append(lines, "  "+lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("LLM credentials: set an API key ("+config.EnvAIAPIKey+" or field below), choose Ollama, or use base URL http://127.0.0.1:11434/v1"))
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Ollama: API key optional; first request after idle may exceed 60s—raise the generation timeout below if needed."), "")
	baseURLLabel := "  API base URL (OpenAI/Ollama; empty = OpenAI public):"
	if curProv == "gemini" {
		baseURLLabel = "  API base URL (ignored for Gemini):"
	}
	if curProv == "ollama" {
		baseURLLabel = "  API base URL (Ollama; empty = " + config.OllamaDefaultChatBaseURL + "):"
	}
	lines = append(lines, focusStyle(16).Render(baseURLLabel))
	if len(data.Inputs) > 16 {
		lines = append(lines, "  "+r.mark(mouse.ZoneSettingsAIBaseURL, data.Inputs[16].View))
	}
	lines = append(lines, focusStyle(17).Render("  Model (empty = default):"))
	if len(data.Inputs) > 17 {
		lines = append(lines, "  "+r.mark(mouse.ZoneSettingsAIModel, data.Inputs[17].View))
	}
	lines = append(lines, focusStyle(18).Render("  API key (optional for Ollama / local http://127.0.0.1:11434/v1; env overrides):"))
	if len(data.Inputs) > 18 {
		lines = append(lines, "  "+r.mark(mouse.ZoneSettingsAIAPIKey, data.Inputs[18].View))
	}
	lines = append(lines, "", "")
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("  Generation timeout:"))
	lines = append(lines, "    "+
		r.mark(mouse.ZoneSettingsAITimeoutDecrease, lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render("[-]"))+" "+
		lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%ds", data.AITimeoutSeconds))+" "+
		r.mark(mouse.ZoneSettingsAITimeoutIncrease, lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render("[+]")))
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Max time to wait for an LLM response. Local models (Ollama) may need 120s+ on first request (model load)."), "")
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary).Render("AI evolog split (graph z)"), "")
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(fmt.Sprintf("    Multi-split cap limits how many FAQ bases the model may suggest (1–%d; also capped by evolog row count). Stepwise: one split per Enter with evolog reload between steps.", config.EvologAIMultiSplitHardMax)), "")
	tEvDesc := "[ ]"
	if data.EvologDescribeDefault {
		tEvDesc = "[✓]"
	}
	lines = append(lines, "  "+r.mark(mouse.ZoneSettingsAIEvologDescribeDefault, lipgloss.NewStyle().Foreground(styles.ColorPrimary).Bold(true).Render(tEvDesc+" Open split modal with post-split AI describe on (still toggle with d)")))
	tEvFile := "[ ]"
	if data.EvologFileSplitEnabled {
		tEvFile = "[✓]"
	}
	lines = append(lines, "  "+r.mark(mouse.ZoneSettingsAIEvologFileSplit, lipgloss.NewStyle().Foreground(styles.ColorPrimary).Bold(true).Render(tEvFile+" Honor AI file lists for jj split after row split")))
	tEvHunk := "[ ]"
	if data.EvologHunkSplitEnabled {
		tEvHunk = "[✓]"
	}
	lines = append(lines, "  "+r.mark(mouse.ZoneSettingsAIEvologHunkSplit, lipgloss.NewStyle().Foreground(styles.ColorPrimary).Bold(true).Render(tEvHunk+" Honor AI hunk_prefix (@@-level) split after row split")))
	tEvStep := "[ ]"
	if data.EvologMultiStepwise {
		tEvStep = "[✓]"
	}
	lines = append(lines, "  "+r.mark(mouse.ZoneSettingsAIEvologMultiStepwise, lipgloss.NewStyle().Foreground(styles.ColorPrimary).Bold(true).Render(tEvStep+" Stepwise multi-split (reload evolog between steps)")))
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left,
		"  "+r.mark(mouse.ZoneSettingsAIEvologMultiMaxDecrease, lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render("−")),
		"  ", lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(fmt.Sprintf("AI multi-split max bases: %d", data.EvologMultiMax)),
		"  "+r.mark(mouse.ZoneSettingsAIEvologMultiMaxIncrease, lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render("+")),
	))
	return lines
}

func (r renderCtx) renderAdvanced(data RenderData) []string {
	var lines []string
	focusStyle := func(i int) lipgloss.Style {
		s := lipgloss.NewStyle()
		if data.FocusedField == i {
			return s.Bold(true).Foreground(styles.ColorPrimary)
		}
		return s
	}
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary).Render("Open in external editor"), "")
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Graph files pane: O opens the selected file. Install the editor CLI on your PATH (e.g. Cursor “Install cursor command”)."), "")
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("  Editor:"), "")
	renderEditorRadio := func(idx int, label string) string {
		selected := data.ExternalEditorPreset == idx
		var radioText string
		if selected {
			radioText = toggleOnStyle.Render("(●) " + label)
		} else {
			radioText = lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render("( ) " + label)
		}
		return r.mark(mouse.ZoneSettingsExternalEditorPreset(idx), radioText)
	}
	for i, label := range advanced.ExternalEditorPresetLabels {
		lines = append(lines, "    "+renderEditorRadio(i, label))
	}
	lines = append(lines, "")
	lines = append(lines, focusStyle(15).Render("  Custom command (when preset is Custom, run via sh -c):"))
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Use {path} for the absolute file path, e.g. cursor -g {path}"), "")
	if len(data.Inputs) > 15 {
		lines = append(lines, "  "+r.mark(mouse.ZoneSettingsExternalEditorCustom, data.Inputs[15].View))
	}
	lines = append(lines, "", "")

	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary).Render("Graph View"), "")
	lines = append(lines, focusStyle(14).Render("  Default revset (jj):"))
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    Which commits to show in the commit graph. Empty = built-in default (fork parents + closest immutable per mutable stack; see README)."), "")
	if len(data.Inputs) > 14 {
		lines = append(lines, "  "+r.mark(mouse.ZoneSettingsGraphRevset, data.Inputs[14].View)+" "+r.mark(mouse.ZoneSettingsGraphRevsetClear, clearButtonStyle.Render("[Clear]")))
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("    e.g. trunk() | (ancestors(@) - ancestors(trunk())) for main + your branch only"), "", "")

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
