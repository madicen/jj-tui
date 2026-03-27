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

// chromeHorizontalRow draws exactly width cells as [left 1][middle width-2][right 1], so gutter
// colors can differ (see styles.HeaderGutterRightBackground). This replaces lipgloss Padding(0,1)
// on HeaderStyle/StatusBarStyle, which always used the same background for both edge cells.
func chromeHorizontalRow(width int, inner string, leftBG, midBG, rightBG, midFG lipgloss.TerminalColor) string {
	switch {
	case width < 1:
		return ""
	case width == 1:
		return lipgloss.NewStyle().Background(leftBG).Width(1).Render(" ")
	case width == 2:
		left := lipgloss.NewStyle().Background(leftBG).Width(1).Render(" ")
		right := lipgloss.NewStyle().Background(rightBG).Width(1).Render(" ")
		return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	}
	innerW := width - 2
	body := lipgloss.NewStyle().
		Background(midBG).
		Foreground(midFG).
		Width(innerW).
		Render(lipgloss.PlaceHorizontal(innerW, lipgloss.Left, inner,
			lipgloss.WithWhitespaceBackground(midBG)))
	left := lipgloss.NewStyle().Background(leftBG).Width(1).Render(" ")
	right := lipgloss.NewStyle().Background(rightBG).Width(1).Render(" ")
	return lipgloss.JoinHorizontal(lipgloss.Top, left, body, right)
}

// View implements tea.Model
func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	if m.initRepoModel.Path() != "" {
		return m.renderWithHeader(m.initRepoModel.View())
	}

	v := m.renderMainLayoutView()
	// Busy spinner must sit *under* interactive overlays; otherwise Loading hides the bookmark /
	// divergent pickers while keys still go to those modals (invisible resolve on j/k/Enter).
	v = m.applyLoadingOverlay(v)

	if m.appState.ViewMode == state.ViewEvologSplit {
		if evologContent := m.evologSplitModal.View(); evologContent != "" {
			v = applyBubbleOverlayCentered(v, evologContent, m.width, m.height)
		}
	}
	if m.appState.ViewMode == state.ViewDivergentCommit {
		if divergentContent := m.divergentModal.View(); divergentContent != "" {
			v = applyBubbleOverlayCentered(v, divergentContent, m.width, m.height)
		}
	}
	if m.appState.ViewMode == state.ViewBookmarkConflict {
		if conflictContent := m.conflictModal.View(); conflictContent != "" {
			v = applyBubbleOverlayCentered(v, conflictContent, m.width, m.height)
		}
	}

	if warningContent := m.warningModal.View(); warningContent != "" {
		v = applyBubbleOverlayCentered(v, warningContent, m.width, m.height)
	}
	if errorContent := m.errorModal.View(); errorContent != "" {
		v = applyBubbleOverlayCentered(v, errorContent, m.width, m.height)
	}

	return m.zoneManager.Scan(v)
}

// renderMainLayoutView builds header + tab content + status (no error/warning/divergent full-screen branches).
func (m *Model) renderMainLayoutView() string {
	header := m.renderHeader()
	statusBar := m.renderStatusBar()
	headerHeight := strings.Count(header, "\n") + 1
	statusHeight := strings.Count(statusBar, "\n") + 1
	contentHeight := max(m.height-headerHeight-statusHeight-2, 1)

	m.graphTabModel.SetDimensions(m.width, contentHeight)
	m.prsTabModel.SetDimensions(m.width, contentHeight)
	m.branchesTabModel.SetDimensions(m.width, contentHeight)
	m.ticketsTabModel.SetDimensions(m.width, contentHeight)
	m.settingsTabModel.SetDimensions(m.width, contentHeight)
	m.helpTabModel.SetDimensions(m.width, contentHeight)

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
		// Modal drawn as overlay in View(); keep the tab the user was on (branches vs graph) visible underneath.
		if m.bookmarkConflictReturnValid {
			switch m.bookmarkConflictReturnView {
			case state.ViewBranches:
				content = m.branchesTabModel.View()
			case state.ViewCommitGraph:
				content = m.graphTabModel.View()
			default:
				content = m.graphTabModel.View()
			}
		} else {
			content = m.graphTabModel.View()
		}
	case state.ViewDivergentCommit, state.ViewEvologSplit:
		// Modal is drawn as an overlay in View(); keep graph visible underneath.
		content = m.graphTabModel.View()
	case state.ViewGitHubLogin:
		content = m.githubLoginModel.View()
	default:
		content = m.graphTabModel.View()
	}

	contentLines := strings.Split(content, "\n")
	for len(contentLines) < contentHeight {
		contentLines = append(contentLines, "")
	}
	if len(contentLines) > contentHeight {
		contentLines = contentLines[:contentHeight]
	}
	contentPadded := strings.Join(contentLines, "\n")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		" ",
		contentPadded,
		" ",
		statusBar,
	)
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
	v = m.applyLoadingOverlay(v)
	return m.zoneManager.Scan(v)
}

// renderHeader renders the header with clickable tabs
func (m *Model) renderHeader() string {
	// Spaces inside TitleStyle (bar gutters are separate; see chromeHorizontalRow).
	title := styles.TitleStyle.Render(" jj-tui  ")

	// Hide tabs when we're in "not a jj repo" state - tabs aren't functional without a repo
	if m.initRepoModel.Path() != "" {
		return chromeHorizontalRow(m.width, title,
			styles.HeaderBarBackground, styles.HeaderBarBackground, styles.HeaderGutterRightBackground,
			styles.HeaderBarForeground)
	}

	// Create tabs wrapped in zones (with keyboard shortcuts)
	graphTabActive := m.appState.ViewMode == state.ViewCommitGraph || m.appState.ViewMode == state.ViewEvologSplit
	tabs := []string{
		m.zoneManager.Mark(mouse.ZoneTabGraph, m.renderTab("Graph (g)", graphTabActive)),
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

	line := title + repo + strings.Repeat(" ", padding) + tabsStr
	return chromeHorizontalRow(m.width, line,
		styles.HeaderBarBackground, styles.HeaderBarBackground, styles.HeaderGutterRightBackground,
		styles.HeaderBarForeground)
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

	// Sanitize status message: remove literal newlines
	status = strings.ReplaceAll(status, "\n", " ")
	status = strings.ReplaceAll(status, "\r", "")

	scrollIndicator := ""

	// Build shortcuts list
	var shortcuts []string

	// Add keyboard shortcuts with ^ notation and | separators
	// Start with undo/redo if in Graph view, then quit and refresh
	if (m.appState.ViewMode == state.ViewCommitGraph || m.appState.ViewMode == state.ViewEvologSplit) && m.appState.JJService != nil {
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

	line := status + scrollIndicator + strings.Repeat(" ", padding) + shortcutsStr
	return chromeHorizontalRow(m.width, line,
		styles.StatusBarBackground, styles.StatusBarBackground, styles.StatusBarBackground,
		styles.ColorMuted)
}
