package model

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/state"
	"github.com/madicen/jj-tui/internal/tui/styles"
	"github.com/madicen/jj-tui/internal/version"
	"github.com/mattn/go-runewidth"
)

// View implements tea.Model
func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	// Init-repo screen (standalone view with header) or error modal (centered)
	if m.initRepoModel.Path() != "" {
		return m.renderWithHeader(m.initRepoModel.View())
	}
	if errorContent := m.errorModal.View(); errorContent != "" {
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
	contentHeight := max(m.height-headerHeight-statusHeight-2, 1) // -2 for blank lines after header and before status

	// Set dimensions for all tabs before View() so they know the actual content area height for scrolling
	m.graphTabModel.SetDimensions(m.width, contentHeight)
	m.prsTabModel.SetDimensions(m.width, contentHeight)
	m.branchesTabModel.SetDimensions(m.width, contentHeight)
	m.ticketsTabModel.SetDimensions(m.width, contentHeight)
	m.settingsTabModel.SetDimensions(m.width, contentHeight)
	m.helpTabModel.SetDimensions(m.width, contentHeight)

	// Delegate to tab models for their views
	var content string
	switch m.appState.ViewMode {
	case state.ViewCommitGraph:
		content = m.graphTabModel.View()
	case state.ViewPullRequests:
		content = m.prsTabModel.View()
	case state.ViewBranches:
		content = m.branchesTabModel.View()
	case state.ViewTickets:
		content = m.ticketsTabModel.View()
	case state.ViewSettings:
		content = m.settingsTabModel.View()
	case state.ViewHelp:
		content = m.helpTabModel.View()
	case state.ViewEditDescription:
		content = m.desceditModal.View()
	case state.ViewCreatePR:
		content = m.prFormModal.View()
	case state.ViewCreateTicket:
		content = m.ticketFormModal.View()
	case state.ViewCreateBookmark:
		content = m.bookmarkModal.View()
	case state.ViewBookmarkConflict:
		content = m.conflictModal.View()
	case state.ViewDivergentCommit:
		content = m.divergentModal.View()
	case state.ViewGitHubLogin:
		content = m.githubLoginModel.View()
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

	// One blank line after header and one before status bar (use single space so the line is visible)
	v := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		" ",
		contentPadded,
		" ",
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
	fullContentHeight := max(m.height-headerHeight-statusHeight-2, 1) // -2 for blank lines after header and before status
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
		" ",
		contentPadded,
		" ",
		statusBar,
	)
	return m.zoneManager.Scan(v)
}

// renderHeader renders the header with clickable tabs
func (m *Model) renderHeader() string {
	title := styles.TitleStyle.Render("jj-tui")

	// Hide tabs when we're in "not a jj repo" state - tabs aren't functional without a repo
	if m.initRepoModel.Path() != "" {
		return styles.HeaderStyle.Width(m.width).Render(title)
	}

	// Create tabs wrapped in zones (with keyboard shortcuts)
	tabs := []string{
		m.zoneManager.Mark(mouse.ZoneTabGraph, m.renderTab("Graph (g)", m.appState.ViewMode == state.ViewCommitGraph)),
		m.zoneManager.Mark(mouse.ZoneTabPRs, m.renderTab("PRs (p)", m.appState.ViewMode == state.ViewPullRequests)),
		m.zoneManager.Mark(mouse.ZoneTabJira, m.renderTab("Tickets (t)", m.appState.ViewMode == state.ViewTickets)),
		m.zoneManager.Mark(mouse.ZoneTabBranches, m.renderTab("Branches (b)", m.appState.ViewMode == state.ViewBranches)),
		m.zoneManager.Mark(mouse.ZoneTabSettings, m.renderTab("Settings (,)", m.appState.ViewMode == state.ViewSettings)),
		m.zoneManager.Mark(mouse.ZoneTabHelp, m.renderTab("Help (h)", m.appState.ViewMode == state.ViewHelp)),
	}

	tabsStr := lipgloss.JoinHorizontal(lipgloss.Right, tabs...)

	repo := ""
	if m.appState.Repository != nil {
		// Max width for the repo string is what's left over.
		// -2 for the same fudge factor as original padding calculation
		// -1 for the leading space on the repo string
		maxWidth := m.width - lipgloss.Width(title) - lipgloss.Width(tabsStr) - 3
		if maxWidth > 5 { // Only show if there's a reasonable amount of space
			repoPath := runewidth.Truncate(m.appState.Repository.Path, maxWidth, "...")
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

// renderStatusBar renders the status bar with global shortcuts (always single line).
func (m *Model) renderStatusBar() string {
	status := m.appState.StatusMessage
	if m.appState.Loading {
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
	if m.appState.ViewMode == state.ViewCommitGraph && m.appState.JJService != nil {
		if m.redoOperationID != "" {
			shortcuts = append(shortcuts,
				m.zoneManager.Mark(mouse.ZoneActionRedo, "^y redo"),
				" │ ",
			)
		}
		shortcuts = append(shortcuts,
			m.zoneManager.Mark(mouse.ZoneActionUndo, "^z undo"),
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
	maxStatusWidth := max(m.width-shortcutsWidth-2, 20)

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
	padding := max(m.width-lipgloss.Width(status)-lipgloss.Width(scrollIndicator)-lipgloss.Width(shortcutsStr)-2, 0)

	return styles.StatusBarStyle.Width(m.width).Render(
		status + scrollIndicator + strings.Repeat(" ", padding) + shortcutsStr,
	)
}
