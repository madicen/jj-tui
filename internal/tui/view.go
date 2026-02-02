package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/madicen-utilities/jj-tui/v2/internal/models"
	"github.com/madicen-utilities/jj-tui/v2/internal/tui/view"
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

	// For PR and Jira views, use split rendering with fixed header
	if m.viewMode == ViewPullRequests || m.viewMode == ViewJira {
		fixedHeader, scrollableList := m.renderSplitContent()

		if scrollableList != "" {
			// Render the fixed header with styling
			styledFixedHeader := ContentStyle.Width(m.width).Render(fixedHeader)

			// Calculate how many lines the fixed header takes
			fixedHeaderLines := strings.Count(styledFixedHeader, "\n") + 1

			// Temporarily reduce viewport height for the split view
			originalHeight := m.viewport.Height
			headerHeight := strings.Count(header, "\n") + 1
			statusHeight := strings.Count(statusBar, "\n") + 1
			availableHeight := m.height - headerHeight - statusHeight - fixedHeaderLines
			if availableHeight < 3 {
				availableHeight = 3 // Minimum height
			}
			m.viewport.Height = availableHeight

			// Put only the scrollable list in the viewport
			m.viewport.SetContent(scrollableList)
			viewportContent := m.viewport.View()

			// Restore original viewport height
			m.viewport.Height = originalHeight

			v = lipgloss.JoinVertical(
				lipgloss.Left,
				header,
				styledFixedHeader,
				viewportContent,
				statusBar,
			)
		} else {
			// No split content (e.g., error message or empty state)
			m.viewport.SetContent(fixedHeader)
			viewportContent := m.viewport.View()

			v = lipgloss.JoinVertical(
				lipgloss.Left,
				header,
				viewportContent,
				statusBar,
			)
		}
	} else {
		// Normal views: put all content in viewport
		content := m.renderContent()
		m.viewport.SetContent(content)
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
		m.zone.Mark(ZoneTabJira, m.renderTab("Jira (i)", m.viewMode == ViewJira)),
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
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555"))
	return style.Render(fmt.Sprintf("Error: %v\n\nPress 'r' to retry, 'Esc' to dismiss, or 'q' to quit.", m.err))
}

// renderCommitGraph renders the commit graph view using the view package
func (m *Model) renderCommitGraph() string {
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

	return m.renderer().Graph(view.GraphData{
		Repository:         m.repository,
		SelectedCommit:     m.selectedCommit,
		InRebaseMode:       m.selectionMode == SelectionRebaseDestination,
		RebaseSourceCommit: m.rebaseSourceCommit,
		OpenPRBranches:     openPRBranches,
		CommitPRBranch:     commitPRBranch,
		CommitBookmark:     commitBookmark,
		ChangedFiles:       changedFiles,
	})
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
	// Convert jira.Ticket to view.JiraTicket
	tickets := make([]view.JiraTicket, len(m.jiraTickets))
	for i, t := range m.jiraTickets {
		tickets[i] = view.JiraTicket{
			Key:         t.Key,
			Summary:     t.Summary,
			Status:      t.Status,
			Type:        t.Type,
			Priority:    t.Priority,
			Description: t.Description,
		}
	}

	return m.renderer().Jira(view.JiraData{
		Tickets:        tickets,
		SelectedTicket: m.selectedTicket,
		JiraService:    m.jiraService != nil,
	})
}

// renderSettings renders the settings view using the view package
func (m *Model) renderSettings() string {
	inputs := make([]view.InputView, len(m.settingsInputs))
	for i, input := range m.settingsInputs {
		inputs[i] = view.InputView{View: input.View()}
	}

	return m.renderer().Settings(view.SettingsData{
		Inputs:        inputs,
		FocusedField:  m.settingsFocusedField,
		GithubService: m.githubService != nil,
		JiraService:   m.jiraService != nil,
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

	// Show scroll position if content is scrollable
	scrollIndicator := ""
	if m.viewportReady && m.viewport.TotalLineCount() > m.viewport.Height {
		scrollPercent := m.viewport.ScrollPercent() * 100
		scrollIndicator = fmt.Sprintf(" [%.0f%%]", scrollPercent)
	}

	// Build shortcuts with zone markers (only global actions)
	shortcuts := []string{
		m.zone.Mark(ZoneActionQuit, "q:quit"),
		" ",
		m.zone.Mark(ZoneActionRefresh, "r:refresh"),
	}

	shortcutsStr := lipgloss.JoinHorizontal(lipgloss.Left, shortcuts...)

	// Layout: status on left, shortcuts on right
	padding := m.width - lipgloss.Width(status) - lipgloss.Width(scrollIndicator) - lipgloss.Width(shortcutsStr) - 2
	if padding < 0 {
		padding = 0
	}

	return StatusBarStyle.Width(m.width).Render(
		status + scrollIndicator + strings.Repeat(" ", padding) + shortcutsStr,
	)
}

// Getters for testing

// GetViewMode returns the current view mode
func (m *Model) GetViewMode() ViewMode {
	return m.viewMode
}

// GetSelectedCommit returns the selected commit index
func (m *Model) GetSelectedCommit() int {
	return m.selectedCommit
}

// GetStatusMessage returns the status message
func (m *Model) GetStatusMessage() string {
	return m.statusMessage
}

// GetRepository returns the repository
func (m *Model) GetRepository() *models.Repository {
	return m.repository
}

// Close releases resources
func (m *Model) Close() {
	if m.zone != nil {
		m.zone.Close()
	}
}
