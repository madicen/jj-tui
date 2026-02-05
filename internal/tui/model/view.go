package model

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/tui/view"
)

// View implements tea.Model
func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	// Build the view with zone markers
	header := m.renderHeader()
	statusBar := m.renderStatusBar()

	var v string

	// Handle errors first - especially "not a jj repo" which needs special UI
	if m.err != nil {
		headerHeight := strings.Count(header, "\n") + 1
		statusHeight := strings.Count(statusBar, "\n") + 1
		fullContentHeight := m.height - headerHeight - statusHeight
		if fullContentHeight < 1 {
			fullContentHeight = 1
		}
		m.viewport.Height = fullContentHeight

		// Render error content (includes init button for non-jj repos)
		errorContent := m.renderError()
		m.viewport.SetContent(errorContent)
		viewportContent := m.viewport.View()

		v = lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			viewportContent,
			statusBar,
		)
		return m.zone.Scan(v)
	}

	// For PR and Jira views, use split rendering with fixed header
	if m.viewMode == ViewPullRequests || m.viewMode == ViewJira {
		fixedHeader, scrollableList := m.renderSplitContent()

		// Calculate full content height first (may have been reduced by graph view)
		headerHeight := strings.Count(header, "\n") + 1
		statusHeight := strings.Count(statusBar, "\n") + 1
		fullContentHeight := m.height - headerHeight - statusHeight
		if fullContentHeight < 1 {
			fullContentHeight = 1
		}

		if scrollableList != "" {
			// Render the fixed header with styling
			styledFixedHeader := ContentStyle.Width(m.width).Render(fixedHeader)

			// Calculate how many lines the fixed header takes
			fixedHeaderLines := strings.Count(styledFixedHeader, "\n") + 1

			// Calculate viewport height for the split view
			availableHeight := fullContentHeight - fixedHeaderLines
			if availableHeight < 3 {
				availableHeight = 3 // Minimum height
			}
			m.viewport.Height = availableHeight

			// Save scroll position before SetContent (which resets YOffset)
			savedYOffset := m.viewport.YOffset

			// Put only the scrollable list in the viewport
			m.viewport.SetContent(scrollableList)

			// Restore scroll position and clamp to valid range
			m.viewport.YOffset = savedYOffset
			maxOffset := m.viewport.TotalLineCount() - availableHeight
			if maxOffset < 0 {
				maxOffset = 0
			}
			if m.viewport.YOffset > maxOffset {
				m.viewport.YOffset = maxOffset
			}
			if m.viewport.YOffset < 0 {
				m.viewport.YOffset = 0
			}

			viewportContent := m.viewport.View()

			v = lipgloss.JoinVertical(
				lipgloss.Left,
				header,
				styledFixedHeader,
				viewportContent,
				statusBar,
			)
		} else {
			// No split content (e.g., error message or empty state)
			// Reset viewport to full height for non-split display
			m.viewport.Height = fullContentHeight
			m.viewport.SetContent(fixedHeader)
			viewportContent := m.viewport.View()

			v = lipgloss.JoinVertical(
				lipgloss.Left,
				header,
				viewportContent,
				statusBar,
			)
		}
	} else if m.viewMode == ViewCommitGraph {
		// Graph view with split panes: graph (scrollable) | actions (fixed) | files (scrollable)
		graphResult := m.getGraphResult()

		headerHeight := strings.Count(header, "\n") + 1
		statusHeight := strings.Count(statusBar, "\n") + 1
		separatorLines := 2 // Two separator lines between sections
		paddingLines := 1   // Padding after header

		// Use a minimum actions height during loading to keep layout stable
		actionsContent := graphResult.ActionsBar
		if actionsContent == "" {
			actionsContent = "Actions:"
		}
		actionsHeight := strings.Count(actionsContent, "\n") + 1

		// Calculate available height for the two scrollable panes
		availableHeight := m.height - headerHeight - statusHeight - actionsHeight - separatorLines - paddingLines
		if availableHeight < 6 {
			availableHeight = 6
		}

		// Split height: 60% for graph, 40% for files
		graphHeight := (availableHeight * 60) / 100
		filesHeight := availableHeight - graphHeight
		if graphHeight < 3 {
			graphHeight = 3
		}
		if filesHeight < 3 {
			filesHeight = 3
		}

		// Set up graph viewport
		m.viewport.Height = graphHeight

		// Save scroll position before SetContent (which resets YOffset)
		savedGraphOffset := m.viewport.YOffset

		// Always set content if we have valid graph content (even during loading, to avoid stale content from other views)
		if graphResult.GraphContent != "" {
			m.viewport.SetContent(graphResult.GraphContent)
		}

		// Restore scroll position and clamp to valid range
		m.viewport.YOffset = savedGraphOffset
		maxGraphOffset := m.viewport.TotalLineCount() - graphHeight
		if maxGraphOffset < 0 {
			maxGraphOffset = 0
		}
		if m.viewport.YOffset > maxGraphOffset {
			m.viewport.YOffset = maxGraphOffset
		}
		if m.viewport.YOffset < 0 {
			m.viewport.YOffset = 0
		}

		// Set up files viewport - show placeholder if no files yet
		m.filesViewport.Height = filesHeight
		filesContent := graphResult.FilesContent
		if filesContent == "" {
			filesContent = lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render("  Loading changed files...")
		}

		// Save scroll position before SetContent
		savedFilesOffset := m.filesViewport.YOffset
		m.filesViewport.SetContent(filesContent)

		// Restore scroll position and clamp to valid range
		m.filesViewport.YOffset = savedFilesOffset
		maxFilesOffset := m.filesViewport.TotalLineCount() - filesHeight
		if maxFilesOffset < 0 {
			maxFilesOffset = 0
		}
		if m.filesViewport.YOffset > maxFilesOffset {
			m.filesViewport.YOffset = maxFilesOffset
		}
		if m.filesViewport.YOffset < 0 {
			m.filesViewport.YOffset = 0
		}

		// Simple separator line
		separator := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#444444")).
			Render(strings.Repeat("─", m.width-2))

		// Wrap viewports in zones for click-to-focus
		graphPane := m.zone.Mark(ZoneGraphPane, m.viewport.View())
		filesPane := m.zone.Mark(ZoneFilesPane, m.filesViewport.View())

		v = lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			"", // Padding line after header
			graphPane,
			separator,
			actionsContent,
			separator,
			filesPane,
			statusBar,
		)
	} else {
		// Normal views: put all content in viewport
		// Reset viewport height to full available space (may have been reduced by graph view)
		headerHeight := strings.Count(header, "\n") + 1
		statusHeight := strings.Count(statusBar, "\n") + 1
		fullContentHeight := m.height - headerHeight - statusHeight
		if fullContentHeight < 1 {
			fullContentHeight = 1
		}
		m.viewport.Height = fullContentHeight

		// Save scroll position before SetContent (which resets YOffset)
		savedYOffset := m.viewport.YOffset

		content := m.renderContent()
		m.viewport.SetContent(content)

		// Restore scroll position and clamp to valid range
		m.viewport.YOffset = savedYOffset
		maxOffset := m.viewport.TotalLineCount() - fullContentHeight
		if maxOffset < 0 {
			maxOffset = 0
		}
		if m.viewport.YOffset > maxOffset {
			m.viewport.YOffset = maxOffset
		}
		if m.viewport.YOffset < 0 {
			m.viewport.YOffset = 0
		}

		viewportContent := m.viewport.View()

		v = lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			viewportContent,
			statusBar,
		)
	}

	// CRITICAL: Scan the view to register zone positions
	return m.zone.Scan(v)
}

// renderer returns a view renderer with the zone manager
func (m *Model) renderer() *view.Renderer {
	return view.New(m.zone)
}

// renderHeader renders the header with clickable tabs
func (m *Model) renderHeader() string {
	title := TitleStyle.Render("jj-tui")
	if m.repository != nil {
		title += " " + lipgloss.NewStyle().Foreground(colorMuted).Render(m.repository.Path)
	}

	// Create tabs wrapped in zones (with keyboard shortcuts)
	tabs := []string{
		m.zone.Mark(ZoneTabGraph, m.renderTab("Graph (g)", m.viewMode == ViewCommitGraph)),
		m.zone.Mark(ZoneTabPRs, m.renderTab("PRs (p)", m.viewMode == ViewPullRequests)),
		m.zone.Mark(ZoneTabJira, m.renderTab("Tickets (t)", m.viewMode == ViewJira)),
		m.zone.Mark(ZoneTabSettings, m.renderTab("Settings (,)", m.viewMode == ViewSettings)),
		m.zone.Mark(ZoneTabHelp, m.renderTab("Help (h)", m.viewMode == ViewHelp)),
	}

	tabsStr := lipgloss.JoinHorizontal(lipgloss.Left, tabs...)

	// Layout: title on left, tabs on right
	padding := m.width - lipgloss.Width(title) - lipgloss.Width(tabsStr) - 2
	if padding < 0 {
		padding = 0
	}

	return HeaderStyle.Width(m.width).Render(
		title + strings.Repeat(" ", padding) + tabsStr,
	)
}

// renderTab renders a single tab
func (m *Model) renderTab(label string, active bool) string {
	if active {
		return TabActiveStyle.Render(label)
	}
	return TabStyle.Render(label)
}

// renderContent renders the main content based on view mode
func (m *Model) renderContent() string {
	var content string

	if m.err != nil {
		content = m.renderError()
	} else if m.loading {
		content = "Loading..."
	} else {
		switch m.viewMode {
		case ViewCommitGraph:
			content = m.renderCommitGraph()
		case ViewPullRequests:
			content = m.renderPullRequests()
		case ViewJira:
			content = m.renderJira()
		case ViewSettings:
			content = m.renderSettings()
		case ViewHelp:
			content = m.renderHelp()
		case ViewCreatePR:
			content = m.renderCreatePR()
		case ViewEditDescription:
			content = m.renderEditDescription()
		case ViewCreateBookmark:
			content = m.renderCreateBookmark()
		case ViewGitHubLogin:
			content = m.renderGitHubLogin()
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
		return m.renderPullRequestsSplit()
	case ViewJira:
		return m.renderJiraSplit()
	default:
		return m.renderContent(), ""
	}
}

// renderError renders an error message
func (m *Model) renderError() string {
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555"))

	// Special handling for "not a jj repo" error - show init button
	if m.notJJRepo {
		var lines []string
		lines = append(lines, view.TitleStyle.Render("Not a Jujutsu Repository"))
		lines = append(lines, "")
		lines = append(lines, errorStyle.Render(fmt.Sprintf("Directory: %s", m.currentPath)))
		lines = append(lines, "")
		lines = append(lines, "This directory is not initialized as a Jujutsu repository.")
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Would you like to initialize it?"))
		lines = append(lines, "")

		// Init button
		initButton := m.zone.Mark(ZoneActionJJInit, view.ButtonStyle.Background(lipgloss.Color("#238636")).Render("Initialize Repository (i)"))
		lines = append(lines, initButton)
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E")).Render("This will run: jj git init"))
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E")).Render("and try to track main@origin if available"))
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E")).Render("Press Ctrl+q to quit"))

		return strings.Join(lines, "\n")
	}

	return errorStyle.Render(fmt.Sprintf("Error: %v\n\nPress Ctrl+r to retry, Esc to dismiss, or Ctrl+q to quit.", m.err))
}

// getGraphResult returns the GraphResult for the commit graph view
func (m *Model) getGraphResult() view.GraphResult {
	return m.renderer().Graph(m.buildGraphData())
}

// buildGraphData builds the GraphData for the commit graph
func (m *Model) buildGraphData() view.GraphData {
	// Build a map of branches that have open PRs
	openPRBranches := make(map[string]bool)
	if m.repository != nil {
		for _, pr := range m.repository.PRs {
			if pr.State == "open" {
				openPRBranches[pr.HeadBranch] = true
			}
		}
	}

	// Build a map of commit index -> PR branch name for commits that can push to a PR
	// This includes commits with the bookmark AND their descendants
	commitPRBranch := make(map[int]string)
	if m.repository != nil && len(m.repository.Graph.Commits) > 0 {
		// First, find commits that directly have PR bookmarks
		commitIDToIndex := make(map[string]int)
		for i, commit := range m.repository.Graph.Commits {
			commitIDToIndex[commit.ID] = i
			commitIDToIndex[commit.ChangeID] = i
			// Check if this commit has a PR bookmark
			for _, branch := range commit.Branches {
				if openPRBranches[branch] {
					commitPRBranch[i] = branch
					break
				}
			}
		}

		// Now propagate PR branch info to descendants (commits whose parents have PR branches)
		// We iterate multiple times to handle chains of descendants
		changed := true
		for changed {
			changed = false
			for i, commit := range m.repository.Graph.Commits {
				if commitPRBranch[i] != "" {
					continue // Already has a PR branch
				}
				// Check if any parent has a PR branch
				for _, parentID := range commit.Parents {
					if parentIdx, ok := commitIDToIndex[parentID]; ok {
						if branch := commitPRBranch[parentIdx]; branch != "" {
							commitPRBranch[i] = branch
							changed = true
							break
						}
					}
				}
			}
		}
	}

	// Build a map of commit index -> bookmark name for commits that can create a PR
	// This includes commits with bookmarks (that don't have open PRs) AND their descendants
	commitBookmark := make(map[int]string)
	if m.repository != nil && len(m.repository.Graph.Commits) > 0 {
		commitIDToIndex := make(map[string]int)
		for i, commit := range m.repository.Graph.Commits {
			commitIDToIndex[commit.ID] = i
			commitIDToIndex[commit.ChangeID] = i
			// Check if this commit has a bookmark without an open PR
			for _, branch := range commit.Branches {
				if !openPRBranches[branch] {
					commitBookmark[i] = branch
					break
				}
			}
		}

		// Propagate bookmark info to descendants
		changed := true
		for changed {
			changed = false
			for i, commit := range m.repository.Graph.Commits {
				if commitBookmark[i] != "" || commitPRBranch[i] != "" {
					continue // Already has a bookmark or PR branch
				}
				// Check if any parent has a bookmark (without PR)
				for _, parentID := range commit.Parents {
					if parentIdx, ok := commitIDToIndex[parentID]; ok {
						if branch := commitBookmark[parentIdx]; branch != "" {
							commitBookmark[i] = branch
							changed = true
							break
						}
					}
				}
			}
		}
	}

	// Convert changed files to view format
	var changedFiles []view.ChangedFile
	for _, f := range m.changedFiles {
		changedFiles = append(changedFiles, view.ChangedFile{
			Path:   f.Path,
			Status: f.Status,
		})
	}

	return view.GraphData{
		Repository:         m.repository,
		SelectedCommit:     m.selectedCommit,
		InRebaseMode:       m.selectionMode == SelectionRebaseDestination,
		RebaseSourceCommit: m.rebaseSourceCommit,
		OpenPRBranches:     openPRBranches,
		CommitPRBranch:     commitPRBranch,
		CommitBookmark:     commitBookmark,
		ChangedFiles:       changedFiles,
		GraphFocused:       m.graphFocused,
	}
}

// renderCommitGraph renders the commit graph view using the view package
func (m *Model) renderCommitGraph() string {
	return m.renderer().Graph(m.buildGraphData()).FullContent
}

// renderPullRequests renders the PR list view using the view package
func (m *Model) renderPullRequests() string {
	result := m.renderer().PullRequests(view.PRData{
		Repository:    m.repository,
		SelectedPR:    m.selectedPR,
		GithubService: m.githubService != nil,
	})
	return result.FullContent
}

// renderPullRequestsSplit returns split header and list for the PR view
// Returns (fixedHeader, scrollableList) - if scrollableList is empty, use FullContent in fixedHeader
func (m *Model) renderPullRequestsSplit() (string, string) {
	result := m.renderer().PullRequests(view.PRData{
		Repository:    m.repository,
		SelectedPR:    m.selectedPR,
		GithubService: m.githubService != nil,
	})
	// If there's no scrollable list (error states), return full content as the "header"
	if result.ScrollableList == "" {
		return result.FullContent, ""
	}
	return result.FixedHeader, result.ScrollableList
}

// renderJira renders the Jira tickets view using the view package
func (m *Model) renderJira() string {
	result := m.getJiraResult()
	return result.FullContent
}

// renderJiraSplit returns split header and list for the Jira view
// Returns (fixedHeader, scrollableList) - if scrollableList is empty, use FullContent in fixedHeader
func (m *Model) renderJiraSplit() (string, string) {
	result := m.getJiraResult()
	// If there's no scrollable list (error states), return full content as the "header"
	if result.ScrollableList == "" {
		return result.FullContent, ""
	}
	return result.FixedHeader, result.ScrollableList
}

// getJiraResult returns the JiraResult for rendering
func (m *Model) getJiraResult() view.JiraResult {
	// Convert tickets.Ticket to view.JiraTicket
	ticketViews := make([]view.JiraTicket, len(m.ticketList))
	for i, t := range m.ticketList {
		ticketViews[i] = view.JiraTicket{
			Key:         t.Key,
			DisplayKey:  t.DisplayKey,
			Summary:     t.Summary,
			Status:      t.Status,
			Type:        t.Type,
			Priority:    t.Priority,
			Description: t.Description,
		}
	}

	var providerName string
	if m.ticketService != nil {
		providerName = m.ticketService.GetProviderName()
	}

	return m.renderer().Jira(view.JiraData{
		Tickets:        ticketViews,
		SelectedTicket: m.selectedTicket,
		JiraService:    m.ticketService != nil,
		ProviderName:   providerName,
	})
}

// renderSettings renders the settings view using the view package
func (m *Model) renderSettings() string {
	inputs := make([]view.InputView, len(m.settingsInputs))
	for i, input := range m.settingsInputs {
		inputs[i] = view.InputView{View: input.View()}
	}

	// Check for local config
	hasLocalConfig := config.HasLocalConfig()
	cfg, _ := config.Load()
	var configSource string
	if cfg != nil {
		configSource = cfg.LoadedFrom()
	}

	return m.renderer().Settings(view.SettingsData{
		Inputs:            inputs,
		FocusedField:      m.settingsFocusedField,
		GithubService:     m.githubService != nil,
		JiraService:       m.ticketService != nil,
		HasLocalConfig:    hasLocalConfig,
		ConfigSource:      configSource,
		ActiveTab:         view.SettingsTab(m.settingsTab),
		ShowMergedPRs:     m.settingsShowMerged,
		ShowClosedPRs:     m.settingsShowClosed,
		ConfirmingCleanup: m.confirmingCleanup,
	})
}

// renderHelp renders the help view using the view package
func (m *Model) renderHelp() string {
	return m.renderer().Help()
}

// renderCreatePR renders the create PR view using the view package
func (m *Model) renderCreatePR() string {
	return m.renderer().CreatePR(view.CreatePRData{
		Repository:     m.repository,
		SelectedCommit: m.prCommitIndex,
		GithubService:  m.githubService != nil,
		TitleInput:     m.prTitleInput.View(),
		BodyInput:      m.prBodyInput.View(),
		HeadBranch:     m.prHeadBranch,
		BaseBranch:     m.prBaseBranch,
		FocusedField:   m.prFocusedField,
	})
}

// renderCreateBookmark renders the bookmark creation view using the view package
func (m *Model) renderCreateBookmark() string {
	return m.renderer().Bookmark(view.BookmarkData{
		Repository:        m.repository,
		CommitIndex:       m.bookmarkCommitIdx,
		NameInput:         m.bookmarkNameInput.View(),
		ExistingBookmarks: m.existingBookmarks,
		SelectedBookmark:  m.selectedBookmarkIdx,
		FromJira:          m.bookmarkFromJira,
		JiraTicketKey:     m.bookmarkJiraTicketKey,
	})
}

// renderEditDescription renders the description editing view using the view package
func (m *Model) renderEditDescription() string {
	return m.renderer().Description(view.DescriptionData{
		Repository:      m.repository,
		EditingCommitID: m.editingCommitID,
		InputView:       m.descriptionInput.View(),
	})
}

// renderStatusBar renders the status bar with global shortcuts
func (m *Model) renderStatusBar() string {
	status := m.statusMessage
	if m.loading {
		status = "⏳ " + status
	}

	// Sanitize status message: remove newlines and truncate if needed
	status = strings.ReplaceAll(status, "\n", " ")
	status = strings.ReplaceAll(status, "\r", "")

	// Show scroll position if content is scrollable
	scrollIndicator := ""
	if m.viewportReady && m.viewport.TotalLineCount() > m.viewport.Height {
		scrollPercent := m.viewport.ScrollPercent() * 100
		scrollIndicator = fmt.Sprintf(" [%.0f%%]", scrollPercent)
	}

	// Build shortcuts with zone markers (only global actions)
	shortcuts := []string{
		m.zone.Mark(ZoneActionQuit, "ctrl+q:quit"),
		" ",
		m.zone.Mark(ZoneActionRefresh, "ctrl+r:refresh"),
	}

	// Add undo/redo shortcuts in Graph view
	if m.viewMode == ViewCommitGraph && m.jjService != nil {
		shortcuts = append(shortcuts, " ", "ctrl+z:undo", " ", "ctrl+y:redo")
	}

	// Add error action buttons if there's an error (check both m.err and status message)
	hasError := m.err != nil || strings.Contains(strings.ToLower(m.statusMessage), "error")
	if hasError {
		copyBtn := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6B6B")).
			Bold(true).
			Render("[Copy]")
		dismissBtn := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Bold(true).
			Render("[X]")
		shortcuts = append(shortcuts, " ", m.zone.Mark(ZoneActionCopyError, copyBtn), " ", m.zone.Mark(ZoneActionDismissError, dismissBtn))
	}

	shortcutsStr := lipgloss.JoinHorizontal(lipgloss.Left, shortcuts...)

	// Calculate available space for status message
	// Reserve space for: scroll indicator + shortcuts + padding
	reservedWidth := lipgloss.Width(scrollIndicator) + lipgloss.Width(shortcutsStr) + 4
	maxStatusWidth := m.width - reservedWidth
	if maxStatusWidth < 20 {
		maxStatusWidth = 20
	}

	// Truncate status if too long
	if lipgloss.Width(status) > maxStatusWidth {
		// Truncate and add ellipsis
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

	lines = append(lines, view.TitleStyle.Render("GitHub Login"))
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
