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
	"github.com/madicen/jj-tui/internal/tui/tabs/graph"
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

	// Delegate to tab models for their views
	var content string
	switch m.viewMode {
	case ViewCommitGraph:
		// Keep graph tab in sync with main selection and rebase state (e.g. when set by tests or handlers)
		m.graphTabModel.SelectCommit(m.selectedCommit)
		m.graphTabModel.SetSelectionMode(graph.SelectionMode(m.selectionMode))
		m.graphTabModel.SetRebaseSourceCommit(m.rebaseSourceCommit)
		content = m.graphTabModel.View()
		if content == "" {
			content = m.renderGraphContent()
		}
	case ViewPullRequests:
		content = m.prsTabModel.View()
		if content == "" {
			content = m.renderPRsContent()
		}
	case ViewBranches:
		content = m.branchesTabModel.View()
		if content == "" {
			content = m.renderBranchesContent()
		}
	case ViewTickets:
		content = m.ticketsTabModel.View()
		if content == "" {
			content = m.renderTicketsContent()
		}
	case ViewSettings:
		content = m.renderSettings()
	case ViewHelp:
		content = m.helpTabModel.View()
		if content == "" {
			content = m.renderHelpContent()
		}
	case ViewEditDescription:
		content = m.renderEditDescription()
	case ViewCreatePR:
		content = m.prFormModal.View()
	case ViewCreateBookmark:
		m.syncBookmarkModalState()
		content = m.bookmarkModal.View()
		if content == "" {
			content = m.renderCreateBookmark()
		}
	case ViewBookmarkConflict:
		content = m.conflictModal.View()
	case ViewDivergentCommit:
		content = m.divergentModal.View()
	case ViewGitHubLogin:
		content = m.renderGitHubLogin()
	default:
		content = m.renderGraphContent()
	}

	// Keep header and footer always visible: put inner content in a viewport with height = total - header - footer
	headerHeight := strings.Count(header, "\n") + 1
	statusHeight := strings.Count(statusBar, "\n") + 1
	contentHeight := max(m.height-headerHeight-statusHeight, 1)

	if m.viewportReady {
		m.viewport.Width = m.width
		m.viewport.Height = contentHeight
		m.viewport.SetContent(content)
		// Clamp YOffset if content shortened
		if total := m.viewport.TotalLineCount(); total > 0 && contentHeight > 0 {
			maxOffset := max(total-contentHeight, 0)
			if m.viewport.YOffset > maxOffset {
				m.viewport.YOffset = maxOffset
			}
		}
		content = m.viewport.View()
	}

	v := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		content,
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

// renderWithHeader renders content with the standard header
func (m *Model) renderWithHeader(content string) string {
	header := m.renderHeader()
	statusBar := m.renderStatusBar()

	headerHeight := strings.Count(header, "\n") + 1
	statusHeight := strings.Count(statusBar, "\n") + 1
	fullContentHeight := max(m.height-headerHeight-statusHeight, 1)
	m.viewport.Height = fullContentHeight
	m.viewport.SetContent(content)

	v := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		m.viewport.View(),
		statusBar,
	)
	return m.zoneManager.Scan(v)
}

// Rendering helper methods (simplified stubs that delegate to models)

func (m *Model) renderGraphContent() string {
	return "Graph view - rendering delegated to graphTabModel"
}

func (m *Model) renderPRsContent() string {
	return "Pull Requests view - rendering delegated to prsTabModel"
}

func (m *Model) renderBranchesContent() string {
	return "Branches view - rendering delegated to branchesTabModel"
}

func (m *Model) renderTicketsContent() string {
	return "Tickets view - rendering delegated to ticketsTabModel"
}

func (m *Model) renderSettingsContent() string {
	return "Settings view - rendering delegated to settingsTabModel"
}

func (m *Model) renderHelpContent() string {
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
	return m.helpTabModel.View()
}

// renderHeader renders the header with clickable tabs
func (m *Model) renderHeader() string {
	title := TitleStyle.Render("jj-tui")

	// Hide tabs when we're in "not a jj repo" state - tabs aren't functional without a repo
	if m.notJJRepo {
		return HeaderStyle.Width(m.width).Render(title)
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
			repo = " " + lipgloss.NewStyle().Foreground(colorMuted).Render(repoPath)
		}
	}

	// Layout: title on left, tabs on right
	padding := max(m.width-lipgloss.Width(title)-lipgloss.Width(repo)-lipgloss.Width(tabsStr)-2, 0)

	return HeaderStyle.Width(m.width).Render(
		title + repo + strings.Repeat(" ", padding) + tabsStr,
	)
}

// renderTab renders a single tab
func (m *Model) renderTab(label string, active bool) string {
	if active {
		return TabActiveStyle.Render(label)
	}
	return TabStyle.Render(label)
}

// renderContent renders the main content based on view mode (viewport path; uses tab/modal View())
func (m *Model) renderContent() string {
	var content string

	if m.err != nil {
		content = m.renderError()
	} else if m.loading {
		content = "Loading..."
	} else {
		switch m.viewMode {
		case ViewCommitGraph:
			m.graphTabModel.SelectCommit(m.selectedCommit)
			m.graphTabModel.SetSelectionMode(graph.SelectionMode(m.selectionMode))
			m.graphTabModel.SetRebaseSourceCommit(m.rebaseSourceCommit)
			content = m.graphTabModel.View()
			if content == "" {
				content = "Loading..."
			}
		case ViewPullRequests:
			content = m.prsTabModel.View()
		case ViewTickets:
			content = m.ticketsTabModel.View()
		case ViewBranches:
			content = m.branchesTabModel.View()
		case ViewSettings:
			content = m.renderSettings()
		case ViewHelp:
			content = m.renderHelp()
		case ViewCreatePR:
			content = m.prFormModal.View()
		case ViewEditDescription:
			content = m.renderEditDescription()
		case ViewCreateBookmark:
			m.syncBookmarkModalState()
			content = m.bookmarkModal.View()
		case ViewGitHubLogin:
			content = m.renderGitHubLogin()
		case ViewBookmarkConflict:
			content = m.conflictModal.View()
		case ViewDivergentCommit:
			content = m.divergentModal.View()
		default:
			content = "Loading..."
		}
	}

	// Don't apply height constraint - viewport handles scrolling
	return ContentStyle.Width(m.width).Render(content)
}

// renderSplitContent returns fixed header and scrollable list for PR/Jira views
func (m *Model) renderSplitContent() (string, string) {
	if m.err != nil {
		return m.renderError(), ""
	}
	if m.loading {
		return "Loading...", ""
	}

	switch m.viewMode {
	case ViewPullRequests:
		return m.prsTabModel.View(), ""
	case ViewTickets:
		return m.ticketsTabModel.View(), ""
	case ViewBranches:
		return m.branchesTabModel.View(), ""
	default:
		return m.renderContent(), ""
	}
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

// renderWarningModal renders the warning modal (e.g., for empty commit descriptions)
func (m *Model) renderWarningModal() string {
	modalWidth := min(max(m.width-8, 50), 80)

	// Styles - amber/yellow theme for warnings
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#E3B341")).
		MarginBottom(1)

	messageStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#C9D1D9")).
		Width(modalWidth - 4)

	mutedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8B949E"))

	commitStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#58A6FF"))

	selectedCommitStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#30363d")).
		Bold(true)

	buttonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#30363d")).
		Padding(0, 1).
		Bold(true)

	// Build modal content
	var content strings.Builder
	content.WriteString(titleStyle.Render("⚠ " + m.warningTitle))
	content.WriteString("\n\n")
	content.WriteString(messageStyle.Render(m.warningMessage))
	content.WriteString("\n\n")

	// List commits with empty descriptions
	if len(m.warningCommits) > 0 {
		content.WriteString(mutedStyle.Render("Commits without descriptions:"))
		content.WriteString("\n")
		for i, commit := range m.warningCommits {
			changeID := commit.ChangeID
			if len(changeID) > 8 {
				changeID = changeID[:8]
			}
			summary := commit.Summary
			if summary == "" {
				summary = "(no description)"
			}
			if len(summary) > 40 {
				summary = summary[:37] + "..."
			}

			line := fmt.Sprintf("  %s  %s", changeID, summary)
			if i == m.warningSelectedIdx {
				content.WriteString(selectedCommitStyle.Render(line))
			} else {
				content.WriteString(commitStyle.Render(line))
			}
			content.WriteString("\n")
		}
	}

	content.WriteString("\n")
	content.WriteString(mutedStyle.Render("─────────────────────────────────────"))
	content.WriteString("\n\n")

	// Clickable button row
	goToBtn := m.zoneManager.Mark(mouse.ZoneWarningGoToCommit, buttonStyle.Background(lipgloss.Color("#238636")).Render("Go to Commit (Enter)"))
	dismissBtn := m.zoneManager.Mark(mouse.ZoneWarningDismiss, buttonStyle.Render("Cancel (Esc)"))

	content.WriteString(goToBtn + "  " + dismissBtn)
	content.WriteString("\n\n")
	content.WriteString(mutedStyle.Render("Use ↑/↓ to select a commit, Enter to edit its description"))

	// Create the modal box with amber border
	modalBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#E3B341")).
		Padding(1, 2).
		Width(modalWidth).
		Render(content.String())

	return modalBox
}

// renderSettings renders the settings view via the settings tab package
func (m *Model) renderSettings() string {
	data := settingstab.RenderData{
		FocusedField:           m.settingsFocusedField,
		GithubService:          m.isGitHubAvailable(),
		JiraService:            m.ticketService != nil,
		ActiveTab:              settingstab.ActiveTab(m.settingsTab),
		ShowMergedPRs:          m.settingsShowMerged,
		ShowClosedPRs:          m.settingsShowClosed,
		OnlyMyPRs:              m.settingsOnlyMine,
		PRLimit:                m.settingsPRLimit,
		PRRefreshInterval:      m.settingsPRRefreshInterval,
		TicketProvider:         m.settingsTicketProvider,
		AutoInProgressOnBranch: m.settingsAutoInProgress,
		BranchLimit:            m.settingsBranchLimit,
		SanitizeBookmarks:      m.settingsSanitizeBookmarks,
		ConfirmingCleanup:      m.confirmingCleanup,
	}
	data.Inputs = make([]struct{ View string }, len(m.settingsInputs))
	for i, input := range m.settingsInputs {
		data.Inputs[i].View = input.View()
	}
	data.HasLocalConfig = config.HasLocalConfig()
	if cfg, _ := config.Load(); cfg != nil {
		data.ConfigSource = cfg.LoadedFrom()
	}
	if m.ticketService != nil {
		data.TicketProviderName = m.ticketService.GetProviderName()
	}
	data.JiraConfigured = strings.TrimSpace(m.settingsInputs[1].Value()) != "" &&
		strings.TrimSpace(m.settingsInputs[2].Value()) != "" &&
		strings.TrimSpace(m.settingsInputs[3].Value()) != ""
	data.CodecksConfigured = len(m.settingsInputs) > 8 &&
		strings.TrimSpace(m.settingsInputs[7].Value()) != "" &&
		strings.TrimSpace(m.settingsInputs[8].Value()) != ""
	data.GitHubIssuesConfigured = m.isGitHubAvailable()

	return settingstab.Render(m.zoneManager, data)
}

// renderHelp renders the help view via the help tab
func (m *Model) renderHelp() string {
	return m.renderHelpContent()
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

// syncBookmarkModalState copies main model bookmark state into the bookmark modal before rendering
func (m *Model) syncBookmarkModalState() {
	m.bookmarkModal.UpdateRepository(m.repository)
	m.bookmarkModal.SetBookmarkName(m.bookmarkNameInput.Value())
	m.bookmarkModal.SetExistingBookmarks(m.existingBookmarks)
	m.bookmarkModal.SetCommitIdx(m.bookmarkCommitIdx)
	m.bookmarkModal.SetSelectedBookmarkIdx(m.selectedBookmarkIdx)
	m.bookmarkModal.SetNameExists(m.bookmarkNameExists)
	if m.bookmarkFromJira {
		m.bookmarkModal.SetFromJira(m.bookmarkJiraTicketKey, m.bookmarkJiraTicketTitle, m.bookmarkTicketDisplayKey)
	} else {
		m.bookmarkModal.ClearJiraContext()
	}
}

// renderCreateBookmark is fallback when bookmark modal View() returns "" (modal owns rendering now)
func (m *Model) renderCreateBookmark() string {
	return ""
}

// renderBookmarkConflict and renderDivergentCommit removed; conflict/divergent modals own rendering

// renderEditDescription renders the description editing view
func (m *Model) renderEditDescription() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#8BE9FD"))
	subtitleStyle := lipgloss.NewStyle().Foreground(styles.ColorMuted)

	var commitInfo string
	if m.repository != nil {
		for _, commit := range m.repository.Graph.Commits {
			if commit.ChangeID == m.editingCommitID {
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
		commitInfo = m.editingCommitID
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
		m.descriptionInput.View(),
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

	// Show scroll position if content is scrollable
	scrollIndicator := ""
	if m.viewportReady && m.viewport.TotalLineCount() > m.viewport.Height {
		scrollPercent := m.viewport.ScrollPercent() * 100
		scrollIndicator = fmt.Sprintf(" [%.0f%%]", scrollPercent)
	}

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

	return StatusBarStyle.Width(m.width).Render(
		status + scrollIndicator + strings.Repeat(" ", padding) + shortcutsStr,
	)
}

// renderGitHubLogin renders the GitHub Device Flow login screen
func (m *Model) renderGitHubLogin() string {
	var lines []string

	lines = append(lines, styles.TitleStyle.Render("GitHub Login"))
	lines = append(lines, "")
	lines = append(lines, "")

	if m.githubUserCode != "" {
		lines = append(lines, lipgloss.NewStyle().Bold(true).Render("1. Visit this URL in your browser:"))
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#58A6FF")).Render("   "+m.githubVerificationURL))
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
		lines = append(lines, codeStyle.Render(m.githubUserCode))
		lines = append(lines, "")

		// Add copy button
		copyButton := styles.ButtonStyle.Render("Copy Code (c)")
		lines = append(lines, "   "+m.zoneManager.Mark(mouse.ZoneGitHubLoginCopyCode, copyButton))
		lines = append(lines, "")

		if m.githubLoginPolling {
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
