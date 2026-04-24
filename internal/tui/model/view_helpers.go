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

	v := m.renderMainLayoutView()
	v = m.applyFormModalsOverlay(v)
	// Busy overlay sits under graph pickers (evolog / divergent / conflict) so keys still target them.
	// Form modals (PR, ticket, bookmark, GitHub login, init) are under loading so submit/init shows the
	// spinner on top; description edit and file diff skip the centered overlay (see shouldShowLoadingOverlay).
	v = m.applyLoadingOverlay(v)

	if m.appState.ViewMode == state.ViewEvologSplit {
		if evologContent := m.evologSplitModal.View(); evologContent != "" {
			v = applyBubbleOverlayCentered(v, evologContent, m.width, m.height)
		}
	}
	if m.appState.ViewMode == state.ViewFileDiff {
		if diffContent := m.fileDiffModal.View(); diffContent != "" {
			v = applyBubbleOverlayCentered(v, diffContent, m.width, m.height)
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
	if m.evologDescribePreviewActive {
		v = applyBubbleOverlayCentered(v, m.renderEvologDescribePreview(), m.width, m.height)
	}
	if errorContent := m.errorModal.View(); errorContent != "" {
		v = applyBubbleOverlayCentered(v, errorContent, m.width, m.height)
	}

	return m.zoneManager.Scan(v)
}

func (m *Model) renderEvologDescribePreview() string {
	maxW := min(m.width-8, 78)
	title := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary).Render("AI descriptions (@- and @)")
	muted := lipgloss.NewStyle().Foreground(styles.ColorMuted)
	p := runewidth.Truncate(m.evologDescribeParent, maxW, "…")
	c := runewidth.Truncate(m.evologDescribeChild, maxW, "…")
	var b strings.Builder
	b.WriteString(title)
	b.WriteString("\n\n")
	b.WriteString(muted.Render("@- (parent):"))
	b.WriteString("\n")
	b.WriteString(p)
	b.WriteString("\n\n")
	b.WriteString(muted.Render("@ (working copy):"))
	b.WriteString("\n")
	b.WriteString(c)
	b.WriteString("\n\n")
	b.WriteString(muted.Render("y apply · n or Esc discard"))
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(styles.ColorMuted).Padding(1, 2).Width(min(m.width-4, 84))
	return box.Render(b.String())
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
	switch m.layoutContentMode() {
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

// renderHeader renders the header with clickable tabs
func (m *Model) renderHeader() string {
	// Spaces inside TitleStyle (bar gutters are separate; see chromeHorizontalRow).
	title := styles.TitleStyle.Render(" jj-tui  ")

	// Create tabs wrapped in zones (with keyboard shortcuts)
	tm := m.tabHighlightMode()
	graphTabActive := tm == state.ViewCommitGraph || m.appState.ViewMode == state.ViewEvologSplit || m.appState.ViewMode == state.ViewFileDiff
	tabs := []string{
		m.zoneManager.Mark(mouse.ZoneTabGraph, m.renderTab("Graph (g)", graphTabActive)),
		m.zoneManager.Mark(mouse.ZoneTabPRs, m.renderTab("PRs (p)", tm == state.ViewPullRequests)),
		m.zoneManager.Mark(mouse.ZoneTabJira, m.renderTab("Tickets (t)", tm == state.ViewTickets)),
		m.zoneManager.Mark(mouse.ZoneTabBranches, m.renderTab("Branches (b)", tm == state.ViewBranches)),
		m.zoneManager.Mark(mouse.ZoneTabSettings, m.renderTab("Settings (,)", tm == state.ViewSettings)),
		m.zoneManager.Mark(mouse.ZoneTabHelp, m.renderTab("Help (h)", tm == state.ViewHelp)),
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
	if (m.tabHighlightMode() == state.ViewCommitGraph || m.appState.ViewMode == state.ViewEvologSplit || m.appState.ViewMode == state.ViewFileDiff) && m.appState.JJService != nil {
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
