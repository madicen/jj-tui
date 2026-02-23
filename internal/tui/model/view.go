package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/styles"
	helptab "github.com/madicen/jj-tui/internal/tui/tabs/help"
	settingstab "github.com/madicen/jj-tui/internal/tui/tabs/settings"
	"github.com/madicen/jj-tui/internal/version"
	"github.com/mattn/go-runewidth"
)

// View implements tea.Model
func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	// Sync error state to error modal so it can render (e.g. when set by errorMsg or tests)
	if m.err != nil {
		m.errorModal.SetError(m.err, m.notJJRepo, m.currentPath)
	}
	// Handle error modal model
	if errorContent := m.errorModal.View(); errorContent != "" {
		if m.errorModal.IsJJRepoError() {
			// "Not a jj repo" uses main model's welcome screen (has zone marks for init button)
			return m.renderWithHeader(m.renderError())
		}
		// Regular error: show as centered modal
		return m.centerModal(errorContent)
	}

	// Handle warning modal model
	if warningContent := m.warningModal.View(); warningContent != "" {
		return m.centerModal(warningContent)
	}

	// Handle divergent commit modal
	if divergentContent := m.divergentModal.View(); divergentContent != "" {
		return m.centerModal(divergentContent)
	}

	// Handle conflict modal
	if conflictContent := m.conflictModal.View(); conflictContent != "" {
		return m.centerModal(conflictContent)
	}

	// Normal UI: header and footer are owned by the main model; inner content from tab/modal View()
	header := m.renderHeader()
	statusBar := m.renderStatusBar()
	headerHeight := strings.Count(header, "\n") + 1
	statusHeight := strings.Count(statusBar, "\n") + 1
	contentHeight := max(m.height-headerHeight-statusHeight, 1)

	// Set dimensions for all tabs before View() so they know the actual content area height for scrolling
	m.graphTabModel.SetDimensions(m.width, contentHeight)
	m.prsTabModel.SetDimensions(m.width, contentHeight)
	m.branchesTabModel.SetDimensions(m.width, contentHeight)
	m.ticketsTabModel.SetDimensions(m.width, contentHeight)
	m.settingsTabModel.SetDimensions(m.width, contentHeight)
	m.helpTabModel.SetDimensions(m.width, contentHeight)

	// Delegate to tab models for their views
	var content string
	switch m.viewMode {
	case ViewCommitGraph:
		content = m.graphTabModel.View()
	case ViewPullRequests:
		content = m.prsTabModel.View()
	case ViewBranches:
		content = m.branchesTabModel.View()
	case ViewTickets:
		content = m.ticketsTabModel.View()
	case ViewSettings:
		content = m.renderSettings()
	case ViewHelp:
		m.syncHelpCommandHistory()
		content = m.helpTabModel.View()
	case ViewEditDescription:
		content = m.renderEditDescription()
	case ViewCreatePR:
		content = m.prFormModal.View()
	case ViewCreateBookmark:
		m.syncBookmarkModalState()
		content = m.bookmarkModal.View()
	case ViewBookmarkConflict:
		content = m.conflictModal.View()
	case ViewDivergentCommit:
		content = m.divergentModal.View()
	case ViewGitHubLogin:
		content = m.renderGitHubLogin()
	default:
		content = m.graphTabModel.View()
	}

	// Pin footer to bottom: pad content to fixed height (avoid lipgloss on content to preserve zone markup)
	contentLines := strings.Split(content, "\n")
	for len(contentLines) < contentHeight {
		contentLines = append(contentLines, "")
	}
	if len(contentLines) > contentHeight {
		contentLines = contentLines[:contentHeight]
	}
	contentPadded := strings.Join(contentLines, "\n")

	v := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		contentPadded,
		statusBar,
	)

	return m.zoneManager.Scan(v)
}

// centerModal centers a modal on the screen
func (m *Model) centerModal(content string) string {
	centered := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center).
		AlignVertical(lipgloss.Center).
		Render(content)
	return m.zoneManager.Scan(centered)
}

// renderWithHeader renders content with the standard header (preserves zone markup for mouse)
func (m *Model) renderWithHeader(content string) string {
	header := m.renderHeader()
	statusBar := m.renderStatusBar()

	headerHeight := strings.Count(header, "\n") + 1
	statusHeight := strings.Count(statusBar, "\n") + 1
	fullContentHeight := max(m.height-headerHeight-statusHeight, 1)
	contentLines := strings.Split(content, "\n")
	end := min(fullContentHeight, len(contentLines))
	var visible string
	if end > 0 {
		visible = strings.Join(contentLines[0:end], "\n")
	} else if len(contentLines) > 0 {
		visible = contentLines[0]
	} else {
		visible = ""
	}

	// Pin footer to bottom: pad content to fixed height (preserve zone markup)
	visibleLines := strings.Split(visible, "\n")
	for len(visibleLines) < fullContentHeight {
		visibleLines = append(visibleLines, "")
	}
	if len(visibleLines) > fullContentHeight {
		visibleLines = visibleLines[:fullContentHeight]
	}
	contentPadded := strings.Join(visibleLines, "\n")

	v := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		contentPadded,
		statusBar,
	)
	return m.zoneManager.Scan(v)
}

// renderHeader renders the header with clickable tabs
func (m *Model) renderHeader() string {
	title := styles.TitleStyle.Render("jj-tui")

	// Hide tabs when we're in "not a jj repo" state - tabs aren't functional without a repo
	if m.notJJRepo {
		return styles.HeaderStyle.Width(m.width).Render(title)
	}

	// Create tabs wrapped in zones (with keyboard shortcuts)
	tabs := []string{
		m.zoneManager.Mark(mouse.ZoneTabGraph, m.renderTab("Graph (g)", m.viewMode == ViewCommitGraph)),
		m.zoneManager.Mark(mouse.ZoneTabPRs, m.renderTab("PRs (p)", m.viewMode == ViewPullRequests)),
		m.zoneManager.Mark(mouse.ZoneTabJira, m.renderTab("Tickets (t)", m.viewMode == ViewTickets)),
		m.zoneManager.Mark(mouse.ZoneTabBranches, m.renderTab("Branches (b)", m.viewMode == ViewBranches)),
		m.zoneManager.Mark(mouse.ZoneTabSettings, m.renderTab("Settings (,)", m.viewMode == ViewSettings)),
		m.zoneManager.Mark(mouse.ZoneTabHelp, m.renderTab("Help (h)", m.viewMode == ViewHelp)),
	}

	tabsStr := lipgloss.JoinHorizontal(lipgloss.Right, tabs...)

	repo := ""
	if m.repository != nil {
		// Max width for the repo string is what's left over.
		// -2 for the same fudge factor as original padding calculation
		// -1 for the leading space on the repo string
		maxWidth := m.width - lipgloss.Width(title) - lipgloss.Width(tabsStr) - 3
		if maxWidth > 5 { // Only show if there's a reasonable amount of space
			repoPath := runewidth.Truncate(m.repository.Path, maxWidth, "...")
			repo = " " + lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(repoPath)
		}
	}

	// Layout: title on left, tabs on right
	padding := max(m.width-lipgloss.Width(title)-lipgloss.Width(repo)-lipgloss.Width(tabsStr)-2, 0)

	return styles.HeaderStyle.Width(m.width).Render(
		title + repo + strings.Repeat(" ", padding) + tabsStr,
	)
}

// renderTab renders a single tab
func (m *Model) renderTab(label string, active bool) string {
	if active {
		return styles.TabActiveStyle.Render(label)
	}
	return styles.TabStyle.Render(label)
}


// renderError renders an error message
// renderError renders an error message with text wrapping
func (m *Model) renderError() string {
	// Special handling for "not a jj repo" - show a welcome/setup screen instead of an error
	if m.notJJRepo {
		mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E"))
		pathStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#58A6FF"))

		var lines []string
		lines = append(lines, styles.TitleStyle.Render("Welcome to jj-tui"))
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("Directory: %s", pathStyle.Render(m.currentPath)))
		lines = append(lines, "")
		lines = append(lines, "This directory is not yet a Jujutsu repository.")
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Would you like to initialize it?"))
		lines = append(lines, "")

		// Init button
		initButton := m.zoneManager.Mark(mouse.ZoneActionJJInit, styles.ButtonStyle.Background(lipgloss.Color("#238636")).Render("Initialize Repository (i)"))
		lines = append(lines, initButton)
		lines = append(lines, "")
		lines = append(lines, mutedStyle.Render("This will run: jj git init"))
		lines = append(lines, mutedStyle.Render("and try to track main@origin if available"))
		lines = append(lines, "")
		lines = append(lines, mutedStyle.Render("Press Ctrl+q to quit"))

		return strings.Join(lines, "\n")
	}

	// Render error as a modal dialog box
	modalWidth := min(max(m.width-8, 50), 80)

	// Styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF5555")).
		MarginBottom(1)

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Width(modalWidth - 4)

	mutedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8B949E"))

	buttonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#30363d")).
		Padding(0, 1).
		Bold(true)

	// Build modal content
	var content strings.Builder
	content.WriteString(titleStyle.Render("⚠ Error"))
	content.WriteString("\n\n")
	content.WriteString(errorStyle.Render(m.err.Error()))
	content.WriteString("\n\n")
	content.WriteString(mutedStyle.Render("─────────────────────────────────────"))
	content.WriteString("\n\n")

	// Clickable button row
	dismissBtn := m.zoneManager.Mark(mouse.ZoneActionDismissError, buttonStyle.Render("Dismiss (Esc)"))

	// Show "Copied!" indicator if error was just copied
	var copyBtn string
	if m.errorCopied {
		copiedStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#2ea44f")).
			Bold(true)
		copyBtn = copiedStyle.Render("✓ Copied!")
	} else {
		copyBtn = m.zoneManager.Mark(mouse.ZoneActionCopyError, buttonStyle.Render("Copy (c)"))
	}

	retryBtn := m.zoneManager.Mark(mouse.ZoneActionRetry, buttonStyle.Render("Retry (^r)"))
	quitBtn := m.zoneManager.Mark(mouse.ZoneActionQuit, buttonStyle.Background(lipgloss.Color("#c9302c")).Render("Quit (^q)"))

	content.WriteString(dismissBtn + "  " + copyBtn + "  " + retryBtn + "  " + quitBtn)

	// Create the modal box with border
	modalBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF5555")).
		Padding(1, 2).
		Width(modalWidth).
		Render(content.String())

	return modalBox
}

// renderSettings renders the settings view via the settings tab package
func (m *Model) renderSettings() string {
	inputs := m.settingsTabModel.GetSettingsInputs()
	data := settingstab.RenderData{
		FocusedField:           m.settingsTabModel.GetFocusedField(),
		GithubService:          m.isGitHubAvailable(),
		JiraService:            m.ticketService != nil,
		ActiveTab:              settingstab.ActiveTab(m.settingsTabModel.GetSettingsTab()),
		ShowMergedPRs:          m.settingsTabModel.GetSettingsShowMerged(),
		ShowClosedPRs:          m.settingsTabModel.GetSettingsShowClosed(),
		OnlyMyPRs:              m.settingsTabModel.GetSettingsOnlyMine(),
		PRLimit:                m.settingsTabModel.GetSettingsPRLimit(),
		PRRefreshInterval:      m.settingsTabModel.GetSettingsPRRefreshInterval(),
		TicketProvider:         m.settingsTabModel.GetSettingsTicketProvider(),
		AutoInProgressOnBranch: m.settingsTabModel.GetSettingsAutoInProgress(),
		BranchLimit:            m.settingsTabModel.GetSettingsBranchLimit(),
		SanitizeBookmarks:      m.settingsTabModel.GetSettingsSanitizeBookmarks(),
		ConfirmingCleanup:      m.settingsTabModel.GetConfirmingCleanup(),
	}
	data.Inputs = make([]struct{ View string }, len(inputs))
	for i, input := range inputs {
		data.Inputs[i].View = input.View()
	}
	data.HasLocalConfig = config.HasLocalConfig()
	if cfg, _ := config.Load(); cfg != nil {
		data.ConfigSource = cfg.LoadedFrom()
	}
	if m.ticketService != nil {
		data.TicketProviderName = m.ticketService.GetProviderName()
	}
	data.JiraConfigured = len(inputs) > 3 &&
		strings.TrimSpace(inputs[1].Value()) != "" &&
		strings.TrimSpace(inputs[2].Value()) != "" &&
		strings.TrimSpace(inputs[3].Value()) != ""
	data.CodecksConfigured = len(inputs) > 8 &&
		strings.TrimSpace(inputs[7].Value()) != "" &&
		strings.TrimSpace(inputs[8].Value()) != ""
	data.GitHubIssuesConfigured = m.isGitHubAvailable()
	data.YOffset = m.settingsTabModel.GetSettingsYOffset()
	data.ContentHeight = m.estimatedContentHeight()

	return settingstab.Render(m.zoneManager, data)
}

// isAutoRefreshCommand returns true if the command is part of auto-refresh
// These are filtered from the command history to reduce noise
func isAutoRefreshCommand(cmd string) bool {
	// These commands are run automatically during refresh/tick cycles
	autoRefreshPatterns := []string{
		"jj log -r mutable()", // Main graph refresh (GetRepository)
		"jj log -r empty()",   // Orphan cleanup check
	}

	for _, pattern := range autoRefreshPatterns {
		if strings.HasPrefix(cmd, pattern) {
			return true
		}
	}
	return false
}

// syncHelpCommandHistory pushes the current filtered command history into the help tab so the History sub-tab can display it.
func (m *Model) syncHelpCommandHistory() {
	var entries []helptab.CommandHistoryEntry
	for _, entry := range m.getFilteredCommandHistory() {
		entries = append(entries, helptab.CommandHistoryEntry{
			Command:   entry.Command,
			Timestamp: entry.Timestamp.Format("15:04:05"),
			Duration:  formatDuration(entry.Duration),
			Success:   entry.Success,
			Error:     entry.Error,
		})
	}
	m.helpTabModel.SetCommandHistoryEntries(entries)
}

// getFilteredCommandHistory returns command history with auto-refresh commands filtered out
func (m *Model) getFilteredCommandHistory() []jj.CommandHistoryEntry {
	if m.jjService == nil {
		return nil
	}

	var filtered []jj.CommandHistoryEntry
	for _, entry := range m.jjService.GetCommandHistory() {
		if !isAutoRefreshCommand(entry.Command) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

// formatDuration formats a duration for display
func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return "<1ms"
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

// syncBookmarkModalState updates the bookmark modal with current repository; modal owns its own state.
func (m *Model) syncBookmarkModalState() {
	m.bookmarkModal.UpdateRepository(m.repository)
}

// renderEditDescription renders the description editing view
func (m *Model) renderEditDescription() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#8BE9FD"))
	subtitleStyle := lipgloss.NewStyle().Foreground(styles.ColorMuted)

	editingID := m.graphTabModel.GetEditingCommitID()
	var commitInfo string
	if m.repository != nil {
		for _, commit := range m.repository.Graph.Commits {
			if commit.ChangeID == editingID {
				changeIDShort := commit.ChangeID
				if len(changeIDShort) > 8 {
					changeIDShort = changeIDShort[:8]
				}
				commitInfo = fmt.Sprintf("%s (%s)", commit.ShortID, changeIDShort)
				break
			}
		}
	}
	if commitInfo == "" {
		commitInfo = editingID
	}

	header := titleStyle.Render("Edit Commit Description")
	commitLine := subtitleStyle.Render(fmt.Sprintf("Commit: %s", commitInfo))
	actionButtons := lipgloss.JoinHorizontal(
		lipgloss.Left,
		m.zoneManager.Mark(mouse.ZoneDescSave, styles.ButtonStyle.Render("Save (Ctrl+S)")),
		m.zoneManager.Mark(mouse.ZoneDescCancel, styles.ButtonStyle.Render("Cancel (Esc)")),
	)
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		commitLine,
		"",
		m.graphTabModel.GetDescriptionInput().View(),
		"",
		actionButtons,
	)
}

// renderStatusBar renders the status bar with global shortcuts
// renderStatusBar renders the status bar with global shortcuts (always single line)
func (m *Model) renderStatusBar() string {
	status := m.statusMessage
	if m.loading {
		status = "⏳ " + status
	}

	// Sanitize status message: remove literal newlines
	status = strings.ReplaceAll(status, "\n", " ")
	status = strings.ReplaceAll(status, "\r", "")

	scrollIndicator := ""

	// Build shortcuts list
	var shortcuts []string

	// Add keyboard shortcuts with ^ notation and | separators
	// Start with undo/redo if in Graph view, then quit and refresh
	if m.viewMode == ViewCommitGraph && m.jjService != nil {
		shortcuts = append(shortcuts,
			m.zoneManager.Mark(mouse.ZoneActionUndo, "^z undo"),
			" │ ",
			m.zoneManager.Mark(mouse.ZoneActionRedo, "^y redo"),
			" │ ",
		)
	}

	// Always add quit and refresh (in same position for all tabs)
	shortcuts = append(shortcuts,
		m.zoneManager.Mark(mouse.ZoneActionRefresh, "^r refresh"),
		" │ ",
		m.zoneManager.Mark(mouse.ZoneActionQuit, "^q quit"),
	)

	// Add update notification if available
	if updateInfo := version.GetUpdateInfo(); updateInfo != nil && updateInfo.UpdateAvailable {
		updateNotice := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFB86C")).
			Bold(true).
			Render(fmt.Sprintf(" │ Update: %s", updateInfo.LatestVersion))
		shortcuts = append(shortcuts, updateNotice)
	}

	shortcutsStr := lipgloss.JoinHorizontal(lipgloss.Left, shortcuts...)

	// Calculate available width for status message
	shortcutsWidth := lipgloss.Width(shortcutsStr) + lipgloss.Width(scrollIndicator) + 4
	maxStatusWidth := m.width - shortcutsWidth - 2
	if maxStatusWidth < 20 {
		maxStatusWidth = 20
	}

	// Always truncate status to fit on single line
	statusWidth := lipgloss.Width(status)
	if statusWidth > maxStatusWidth {
		truncated := ""
		for _, r := range status {
			if lipgloss.Width(truncated+"…") >= maxStatusWidth {
				break
			}
			truncated += string(r)
		}
		status = truncated + "…"
	}

	// Layout: status on left, shortcuts on right
	padding := m.width - lipgloss.Width(status) - lipgloss.Width(scrollIndicator) - lipgloss.Width(shortcutsStr) - 2
	if padding < 0 {
		padding = 0
	}

	return styles.StatusBarStyle.Width(m.width).Render(
		status + scrollIndicator + strings.Repeat(" ", padding) + shortcutsStr,
	)
}

// renderGitHubLogin renders the GitHub Device Flow login screen
func (m *Model) renderGitHubLogin() string {
	var lines []string

	lines = append(lines, styles.TitleStyle.Render("GitHub Login"))
	lines = append(lines, "")
	lines = append(lines, "")

	if m.settingsTabModel.GetGitHubUserCode() != "" {
		lines = append(lines, lipgloss.NewStyle().Bold(true).Render("1. Visit this URL in your browser:"))
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#58A6FF")).Render("   "+m.settingsTabModel.GetGitHubVerificationURL()))
		lines = append(lines, "")
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Bold(true).Render("2. Enter this code:"))
		lines = append(lines, "")

		// Display the user code prominently
		codeStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F0F6FC")).
			Background(lipgloss.Color("#238636")).
			Padding(1, 3).
			MarginLeft(3)
		lines = append(lines, codeStyle.Render(m.settingsTabModel.GetGitHubUserCode()))
		lines = append(lines, "")

		// Add copy button
		copyButton := styles.ButtonStyle.Render("Copy Code (c)")
		lines = append(lines, "   "+m.zoneManager.Mark(mouse.ZoneGitHubLoginCopyCode, copyButton))
		lines = append(lines, "")

		if m.settingsTabModel.GetGitHubLoginPolling() {
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
