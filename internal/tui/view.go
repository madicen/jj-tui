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
	content := m.renderContent()
	statusBar := m.renderStatusBar()

	// Put content in viewport for scrolling
	m.viewport.SetContent(content)
	viewportContent := m.viewport.View()

	v := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		viewportContent,
		statusBar,
	)

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

	// Create tabs wrapped in zones
	tabs := []string{
		m.zone.Mark(ZoneTabGraph, m.renderTab("Graph", m.viewMode == ViewCommitGraph)),
		m.zone.Mark(ZoneTabPRs, m.renderTab("PRs", m.viewMode == ViewPullRequests)),
		m.zone.Mark(ZoneTabJira, m.renderTab("Jira", m.viewMode == ViewJira)),
		m.zone.Mark(ZoneTabSettings, m.renderTab("Settings", m.viewMode == ViewSettings)),
		m.zone.Mark(ZoneTabHelp, m.renderTab("Help", m.viewMode == ViewHelp)),
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

	return m.renderer().Graph(view.GraphData{
		Repository:         m.repository,
		SelectedCommit:     m.selectedCommit,
		InRebaseMode:       m.selectionMode == SelectionRebaseDestination,
		RebaseSourceCommit: m.rebaseSourceCommit,
		OpenPRBranches:     openPRBranches,
	})
}

// renderPullRequests renders the PR list view using the view package
func (m *Model) renderPullRequests() string {
	return m.renderer().PullRequests(view.PRData{
		Repository:    m.repository,
		SelectedPR:    m.selectedPR,
		GithubService: m.githubService != nil,
	})
}

// renderJira renders the Jira tickets view using the view package
func (m *Model) renderJira() string {
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
		Repository:  m.repository,
		CommitIndex: m.bookmarkCommitIdx,
		NameInput:   m.bookmarkNameInput.View(),
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
