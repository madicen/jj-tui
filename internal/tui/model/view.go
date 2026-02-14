package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui/view"
	"github.com/madicen/jj-tui/internal/version"
	"github.com/mattn/go-runewidth"
)

// View implements tea.Model
func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	var v string

	// Handle errors first - regular errors show as centered modal, "not a jj repo" shows welcome screen
	if m.err != nil {
		if m.notJJRepo {
			// "Not a jj repo" shows a welcome screen with normal UI
			header := m.renderHeader()
			statusBar := m.renderStatusBar()
			headerHeight := strings.Count(header, "\n") + 1
			statusHeight := strings.Count(statusBar, "\n") + 1
			fullContentHeight := max(m.height-headerHeight-statusHeight, 1)
			m.viewport.Height = fullContentHeight

			errorContent := m.renderError()
			m.viewport.SetContent(errorContent)
			viewportContent := m.viewport.View()

			v = lipgloss.JoinVertical(
				lipgloss.Left,
				header,
				viewportContent,
				statusBar,
			)
			return m.zoneManager.Scan(v)
		}

		// Regular errors: show centered modal without header/tabs/status bar
		errorModal := m.renderError()

		// Center the modal both horizontally and vertically
		centeredModal := lipgloss.NewStyle().
			Width(m.width).
			Height(m.height).
			Align(lipgloss.Center).
			AlignVertical(lipgloss.Center).
			Render(errorModal)

		return m.zoneManager.Scan(centeredModal)
	}

	// Handle warning modal (e.g., empty commit descriptions)
	if m.showWarningModal {
		warningModal := m.renderWarningModal()

		// Center the modal both horizontally and vertically
		centeredModal := lipgloss.NewStyle().
			Width(m.width).
			Height(m.height).
			Align(lipgloss.Center).
			AlignVertical(lipgloss.Center).
			Render(warningModal)

		return m.zoneManager.Scan(centeredModal)
	}

	// Build the view with zone markers
	header := m.renderHeader()
	statusBar := m.renderStatusBar()

	// For PR, Jira, and Branches views, use split rendering with fixed header
	switch m.viewMode {
	case ViewPullRequests, ViewTickets, ViewBranches:
		fixedHeader, scrollableList := m.renderSplitContent()

		// Calculate full content height first (may have been reduced by graph view)
		headerHeight := strings.Count(header, "\n") + 1
		statusHeight := strings.Count(statusBar, "\n") + 1
		fullContentHeight := max(m.height-headerHeight-statusHeight, 1)

		if scrollableList != "" {
			// Render the fixed header with styling
			styledFixedHeader := ContentStyle.Width(m.width).Render(fixedHeader)

			// Calculate how many lines the fixed header takes
			fixedHeaderLines := strings.Count(styledFixedHeader, "\n") + 1

			// Calculate viewport height for the split view
			availableHeight := max(fullContentHeight-fixedHeaderLines,
				// Minimum height
				3)
			m.viewport.Height = availableHeight

			// Save scroll position before SetContent (which resets YOffset)
			savedYOffset := m.viewport.YOffset

			// Put only the scrollable list in the viewport
			m.viewport.SetContent(scrollableList)

			// Restore scroll position and clamp to valid range
			m.viewport.YOffset = savedYOffset
			maxOffset := max(m.viewport.TotalLineCount()-availableHeight, 0)
			m.viewport.YOffset = max(min(m.viewport.YOffset, maxOffset), 0)

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
	case ViewCommitGraph:
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
		availableHeight := max(m.height-headerHeight-statusHeight-actionsHeight-separatorLines-paddingLines, 6)

		// Split height: 60% for graph, 40% for files
		graphHeight := (availableHeight * 60) / 100
		filesHeight := availableHeight - graphHeight
		graphHeight = max(graphHeight, 3)
		filesHeight = max(filesHeight, 3)

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
		maxGraphOffset := max(m.viewport.TotalLineCount()-graphHeight, 0)
		m.viewport.YOffset = max(min(m.viewport.YOffset, maxGraphOffset), 0)

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
		maxFilesOffset := max(m.filesViewport.TotalLineCount()-filesHeight, 0)
		m.filesViewport.YOffset = max(min(m.filesViewport.YOffset, maxFilesOffset), 0)

		// Simple separator line
		separator := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#444444")).
			Render(strings.Repeat("─", m.width-2))

		// Wrap viewports in zones for click-to-focus
		graphPane := m.zoneManager.Mark(ZoneGraphPane, m.viewport.View())
		filesPane := m.zoneManager.Mark(ZoneFilesPane, m.filesViewport.View())

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
	default:
		// Normal views: put all content in viewport
		// Reset viewport height to full available space (may have been reduced by graph view)
		headerHeight := strings.Count(header, "\n") + 1
		statusHeight := strings.Count(statusBar, "\n") + 1
		fullContentHeight := max(m.height-headerHeight-statusHeight, 1)
		m.viewport.Height = fullContentHeight

		// Save scroll position before SetContent (which resets YOffset)
		savedYOffset := m.viewport.YOffset

		content := m.renderContent()
		m.viewport.SetContent(content)

		// Restore scroll position and clamp to valid range
		m.viewport.YOffset = savedYOffset
		maxOffset := max(m.viewport.TotalLineCount()-fullContentHeight, 0)
		m.viewport.YOffset = max(min(m.viewport.YOffset, maxOffset), 0)

		viewportContent := m.viewport.View()

		v = lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			viewportContent,
			statusBar,
		)
	}

	// CRITICAL: Scan the view to register zone positions
	return m.zoneManager.Scan(v)
}

// renderer returns a view renderer with the zone manager
func (m *Model) renderer() *view.Renderer {
	return view.New(m.zoneManager)
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
		m.zoneManager.Mark(ZoneTabGraph, m.renderTab("Graph (g)", m.viewMode == ViewCommitGraph)),
		m.zoneManager.Mark(ZoneTabPRs, m.renderTab("PRs (p)", m.viewMode == ViewPullRequests)),
		m.zoneManager.Mark(ZoneTabJira, m.renderTab("Tickets (t)", m.viewMode == ViewTickets)),
		m.zoneManager.Mark(ZoneTabBranches, m.renderTab("Branches (b)", m.viewMode == ViewBranches)),
		m.zoneManager.Mark(ZoneTabSettings, m.renderTab("Settings (,)", m.viewMode == ViewSettings)),
		m.zoneManager.Mark(ZoneTabHelp, m.renderTab("Help (h)", m.viewMode == ViewHelp)),
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
		case ViewTickets:
			content = m.renderJira()
		case ViewBranches:
			content = m.renderBranches()
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
		case ViewBookmarkConflict:
			content = m.renderBookmarkConflict()
		case ViewDivergentCommit:
			content = m.renderDivergentCommit()
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
	case ViewTickets:
		return m.renderJiraSplit()
	case ViewBranches:
		return m.renderBranchesSplit()
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
		lines = append(lines, view.TitleStyle.Render("Welcome to jj-tui"))
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("Directory: %s", pathStyle.Render(m.currentPath)))
		lines = append(lines, "")
		lines = append(lines, "This directory is not yet a Jujutsu repository.")
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Would you like to initialize it?"))
		lines = append(lines, "")

		// Init button
		initButton := m.zoneManager.Mark(ZoneActionJJInit, view.ButtonStyle.Background(lipgloss.Color("#238636")).Render("Initialize Repository (i)"))
		lines = append(lines, initButton)
		lines = append(lines, "")
		lines = append(lines, mutedStyle.Render("This will run: jj git init"))
		lines = append(lines, mutedStyle.Render("and try to track main@origin if available"))
		lines = append(lines, "")
		lines = append(lines, mutedStyle.Render("Press Ctrl+q to quit"))

		return strings.Join(lines, "\n")
	}

	// Render error as a modal dialog box
	modalWidth := m.width - 8
	if modalWidth < 50 {
		modalWidth = 50
	}
	if modalWidth > 80 {
		modalWidth = 80
	}

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
	dismissBtn := m.zoneManager.Mark(ZoneActionDismissError, buttonStyle.Render("Dismiss (Esc)"))

	// Show "Copied!" indicator if error was just copied
	var copyBtn string
	if m.errorCopied {
		copiedStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#2ea44f")).
			Bold(true)
		copyBtn = copiedStyle.Render("✓ Copied!")
	} else {
		copyBtn = m.zoneManager.Mark(ZoneActionCopyError, buttonStyle.Render("Copy (c)"))
	}

	retryBtn := m.zoneManager.Mark(ZoneActionRetry, buttonStyle.Render("Retry (^r)"))
	quitBtn := m.zoneManager.Mark(ZoneActionQuit, buttonStyle.Background(lipgloss.Color("#c9302c")).Render("Quit (^q)"))

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
	modalWidth := m.width - 8
	if modalWidth < 50 {
		modalWidth = 50
	}
	if modalWidth > 80 {
		modalWidth = 80
	}

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
	goToBtn := m.zoneManager.Mark(ZoneWarningGoToCommit, buttonStyle.Background(lipgloss.Color("#238636")).Render("Go to Commit (Enter)"))
	dismissBtn := m.zoneManager.Mark(ZoneWarningDismiss, buttonStyle.Render("Cancel (Esc)"))

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
		SelectedFile:       m.selectedFile,
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
		GithubService: m.isGitHubAvailable(),
		Width:         m.width,
	})
	return result.FullContent
}

// renderPullRequestsSplit returns split header and list for the PR view
// Returns (fixedHeader, scrollableList) - if scrollableList is empty, use FullContent in fixedHeader
func (m *Model) renderPullRequestsSplit() (string, string) {
	result := m.renderer().PullRequests(view.PRData{
		Repository:    m.repository,
		SelectedPR:    m.selectedPR,
		GithubService: m.isGitHubAvailable(),
		Width:         m.width,
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

	// Convert transitions to view format
	var transitionViews []view.TicketTransition
	for _, t := range m.availableTransitions {
		transitionViews = append(transitionViews, view.TicketTransition{
			ID:   t.ID,
			Name: t.Name,
		})
	}

	return m.renderer().Jira(view.JiraData{
		Tickets:              ticketViews,
		SelectedTicket:       m.selectedTicket,
		JiraService:          m.ticketService != nil,
		ProviderName:         providerName,
		AvailableTransitions: transitionViews,
		TransitionInProgress: m.transitionInProgress,
		StatusChangeMode:     m.statusChangeMode,
		Width:                m.width,
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

	// Get ticket provider display name
	var ticketProviderName string
	if m.ticketService != nil {
		ticketProviderName = m.ticketService.GetProviderName()
	}

	// Check what's configured for provider availability
	jiraConfigured := strings.TrimSpace(m.settingsInputs[1].Value()) != "" &&
		strings.TrimSpace(m.settingsInputs[2].Value()) != "" &&
		strings.TrimSpace(m.settingsInputs[3].Value()) != ""
	codecksConfigured := len(m.settingsInputs) > 8 &&
		strings.TrimSpace(m.settingsInputs[7].Value()) != "" &&
		strings.TrimSpace(m.settingsInputs[8].Value()) != ""
	githubIssuesConfigured := m.isGitHubAvailable()

	return m.renderer().Settings(view.SettingsData{
		Inputs:                 inputs,
		FocusedField:           m.settingsFocusedField,
		GithubService:          m.isGitHubAvailable(),
		JiraService:            m.ticketService != nil,
		HasLocalConfig:         hasLocalConfig,
		ConfigSource:           configSource,
		ActiveTab:              view.SettingsTab(m.settingsTab),
		ShowMergedPRs:          m.settingsShowMerged,
		ShowClosedPRs:          m.settingsShowClosed,
		OnlyMyPRs:              m.settingsOnlyMine,
		PRLimit:                m.settingsPRLimit,
		PRRefreshInterval:      m.settingsPRRefreshInterval,
		TicketProvider:         m.settingsTicketProvider,
		TicketProviderName:     ticketProviderName,
		AutoInProgressOnBranch: m.settingsAutoInProgress,
		JiraConfigured:         jiraConfigured,
		CodecksConfigured:      codecksConfigured,
		GitHubIssuesConfigured: githubIssuesConfigured,
		BranchLimit:            m.settingsBranchLimit,
		SanitizeBookmarks:      m.settingsSanitizeBookmarks,
		ConfirmingCleanup:      m.confirmingCleanup,
	})
}

// renderHelp renders the help view using the view package
func (m *Model) renderHelp() string {
	data := view.HelpData{
		ActiveTab:       view.HelpTab(m.helpTab),
		SelectedCommand: m.helpSelectedCommand,
	}

	// Get command history from jj service (filtered to exclude auto-refresh commands)
	if m.jjService != nil {
		history := m.jjService.GetCommandHistory()
		for _, entry := range history {
			// Filter out auto-refresh commands that spam the list
			if isAutoRefreshCommand(entry.Command) {
				continue
			}
			data.CommandHistory = append(data.CommandHistory, view.CommandHistoryEntry{
				Command:   entry.Command,
				Timestamp: entry.Timestamp.Format("15:04:05"),
				Duration:  formatDuration(entry.Duration),
				Success:   entry.Success,
				Error:     entry.Error,
			})
		}
	}

	return m.renderer().Help(data)
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

// renderBranches renders the branches view using the view package
func (m *Model) renderBranches() string {
	result := m.getBranchesResult()
	return result.FullContent
}

// renderBranchesSplit returns split header and list for the branches view
func (m *Model) renderBranchesSplit() (string, string) {
	result := m.getBranchesResult()
	if result.ScrollableList == "" {
		return result.FullContent, ""
	}
	return result.FixedHeader, result.ScrollableList
}

// getBranchesResult returns the BranchResult for rendering
func (m *Model) getBranchesResult() view.BranchResult {
	return m.renderer().Branches(view.BranchData{
		Branches:       m.branchList,
		SelectedBranch: m.selectedBranch,
		Width:          m.width,
	})
}

// renderCreatePR renders the create PR view using the view package
func (m *Model) renderCreatePR() string {
	return m.renderer().CreatePR(view.CreatePRData{
		Repository:     m.repository,
		SelectedCommit: m.prCommitIndex,
		GithubService:  m.isGitHubAvailable(),
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
		NameExists:        m.bookmarkNameExists,
	})
}

// renderBookmarkConflict renders the bookmark conflict resolution view
func (m *Model) renderBookmarkConflict() string {
	return m.renderer().BookmarkConflict(view.BookmarkConflictData{
		BookmarkName:   m.conflictBookmarkName,
		LocalCommitID:  m.conflictLocalCommitID,
		RemoteCommitID: m.conflictRemoteCommitID,
		LocalSummary:   m.conflictLocalSummary,
		RemoteSummary:  m.conflictRemoteSummary,
		SelectedOption: m.conflictSelectedOption,
	})
}

// renderDivergentCommit renders the divergent commit resolution view
func (m *Model) renderDivergentCommit() string {
	return m.renderer().DivergentCommit(view.DivergentCommitData{
		ChangeID:    m.divergentChangeID,
		CommitIDs:   m.divergentCommitIDs,
		Summaries:   m.divergentCommitSummaries,
		SelectedIdx: m.divergentSelectedIdx,
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
			m.zoneManager.Mark(ZoneActionUndo, "^z undo"),
			" │ ",
			m.zoneManager.Mark(ZoneActionRedo, "^y redo"),
			" │ ",
		)
	}

	// Always add quit and refresh (in same position for all tabs)
	shortcuts = append(shortcuts,
		m.zoneManager.Mark(ZoneActionQuit, "^q quit"),
		" │ ",
		m.zoneManager.Mark(ZoneActionRefresh, "^r refresh"),
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

		// Add copy button
		copyButton := view.ButtonStyle.Render("Copy Code (c)")
		lines = append(lines, "   "+m.renderer().Mark(ZoneGitHubLoginCopyCode, copyButton))
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
