package branches

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/styles"
)

func mark(z *zone.Manager, id, content string) string {
	if z == nil {
		return content
	}
	return z.Mark(id, content)
}

func findBranchIndex(branches []internal.Branch, target internal.Branch) int {
	for i, b := range branches {
		if b.Name == target.Name && b.Remote == target.Remote {
			return i
		}
	}
	return -1
}

func (m Model) renderBranches() string {
	if len(m.branchList) == 0 {
		content := []string{
			styles.TitleStyle.Render("Branches"),
			"",
			"No branches found.",
			"",
			"Press 'F' to fetch from all remotes.",
		}
		return strings.Join(content, "\n")
	}

	var headerLines []string

	if m.selectedBranch >= 0 && m.selectedBranch < len(m.branchList) {
		branch := m.branchList[m.selectedBranch]

		var detailLines []string
		var typeLabel string
		if branch.IsLocal {
			typeLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).Render("[local]")
		} else if branch.IsTracked {
			if branch.LocalDeleted {
				typeLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB86C")).Render("[tracked, local deleted]")
			} else {
				typeLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD")).Render("[tracked]")
			}
		} else {
			typeLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4")).Render("[remote]")
		}
		detailLines = append(detailLines, fmt.Sprintf("%s %s",
			lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary).Render(branch.Name),
			typeLabel,
		))
		if branch.IsLocal {
			detailLines = append(detailLines, "Location: local")
		} else if branch.Remote != "" {
			detailLines = append(detailLines, fmt.Sprintf("Remote: %s", branch.Remote))
		}
		if branch.ShortID != "" {
			detailLines = append(detailLines, fmt.Sprintf("Commit: %s", branch.ShortID))
		}

		detailsBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.ColorPrimary).
			Padding(0, 1).
			Render(strings.Join(detailLines, "\n"))
		headerLines = append(headerLines, detailsBox)

		separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
		separatorWidth := m.width - 4
		if separatorWidth < 20 {
			separatorWidth = 80
		}
		separator := separatorStyle.Render(strings.Repeat("─", separatorWidth))
		headerLines = append(headerLines, separator)
		headerLines = append(headerLines, "Actions:")

		var actionButtons []string
		if branch.IsLocal {
			actionButtons = append(actionButtons,
				mark(m.zoneManager, mouse.ZoneBranchPush, styles.ButtonStyle.Render("Push (P)")),
				mark(m.zoneManager, mouse.ZoneBranchDelete, styles.ButtonStyle.Render("Delete (x)")),
			)
			if branch.HasConflict {
				conflictBtnStyle := styles.ButtonStyle.Background(lipgloss.Color("#FF5555"))
				actionButtons = append(actionButtons,
					mark(m.zoneManager, mouse.ZoneBranchResolveConflict, conflictBtnStyle.Render("Resolve Conflict (c)")),
				)
			}
		} else if branch.IsTracked {
			actionButtons = append(actionButtons,
				mark(m.zoneManager, mouse.ZoneBranchUntrack, styles.ButtonStyle.Render("Untrack (U)")),
			)
			if branch.LocalDeleted {
				actionButtons = append(actionButtons,
					mark(m.zoneManager, mouse.ZoneBranchRestore, styles.ButtonStyle.Render("Restore Local (L)")),
				)
			}
		} else {
			actionButtons = append(actionButtons,
				mark(m.zoneManager, mouse.ZoneBranchTrack, styles.ButtonStyle.Render("Track (T)")),
			)
		}
		actionButtons = append(actionButtons,
			mark(m.zoneManager, mouse.ZoneBranchFetch, styles.ButtonStyle.Render("Fetch All (F)")),
		)
		headerLines = append(headerLines, strings.Join(actionButtons, " "))
		headerLines = append(headerLines, separator)
	}

	listContent := m.renderBranchGraph()
	listLines := strings.Split(listContent, "\n")
	fixedHeader := strings.Join(headerLines, "\n")
	headerLineCount := strings.Count(fixedHeader, "\n") + 1
	listHeight := m.height - headerLineCount
	if listHeight <= 0 {
		listHeight = 0
	}
	totalListLines := len(listLines)
	maxListOffset := 0
	if totalListLines > listHeight {
		maxListOffset = totalListLines - listHeight
	}
	if m.listYOffset > maxListOffset {
		m.listYOffset = maxListOffset
	}
	if m.listYOffset < 0 {
		m.listYOffset = 0
	}
	start := m.listYOffset
	end := start + listHeight
	if end > totalListLines {
		end = totalListLines
	}
	var visibleList string
	if start < end {
		visibleList = strings.Join(listLines[start:end], "\n")
	} else {
		visibleList = ""
	}
	out := fixedHeader + "\n" + visibleList
	outLines := strings.Split(out, "\n")
	for len(outLines) < m.height {
		outLines = append(outLines, "")
	}
	if len(outLines) > m.height {
		outLines = outLines[:m.height]
	}
	return strings.Join(outLines, "\n")
}

func (m Model) renderBranchGraph() string {
	if len(m.branchList) == 0 {
		return ""
	}
	trunkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD"))
	localStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B"))
	trackedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD"))
	remoteStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4"))
	aheadStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B"))
	behindStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB86C"))
	selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF79C6"))
	muted := lipgloss.NewStyle().Foreground(styles.ColorMuted)

	var localBranches, remoteBranches []internal.Branch
	for _, b := range m.branchList {
		if b.IsLocal {
			localBranches = append(localBranches, b)
		} else {
			remoteBranches = append(remoteBranches, b)
		}
	}

	divergedFromRemoteLocal := make(map[string]bool)
	for _, b := range localBranches {
		if b.HasConflict {
			divergedFromRemoteLocal[b.Name] = true
		}
	}

	buildStatus := func(branch internal.Branch) string {
		var statusParts []string
		if branch.Ahead > 0 {
			statusParts = append(statusParts, aheadStyle.Render(fmt.Sprintf("+%d", branch.Ahead)))
		}
		if branch.Behind > 0 {
			statusParts = append(statusParts, behindStyle.Render(fmt.Sprintf("-%d behind", branch.Behind)))
		}
		if len(statusParts) > 0 {
			line := " (" + strings.Join(statusParts, ", ") + ")" + muted.Render(" vs trunk")
			if !branch.IsLocal && branch.IsTracked && divergedFromRemoteLocal[branch.Name] {
				line += muted.Render(" · remote tip ≠ local")
			}
			return line
		}
		// No revs ahead/behind trunk(); still may differ from bookmark@remote (local amend after push).
		if branch.IsLocal && branch.HasConflict {
			return muted.Render(" (vs trunk: up to date · diverged vs remote)")
		}
		if !branch.IsLocal && divergedFromRemoteLocal[branch.Name] {
			return muted.Render(" (vs trunk: up to date · remote tip ≠ local)")
		}
		return muted.Render(" (vs trunk: up to date)")
	}

	var lines []string
	lines = append(lines, trunkStyle.Render("trunk ─────────────────────────────● (tip)"))
	totalBranches := len(localBranches) + len(remoteBranches)
	branchCount := 0

	for i, branch := range localBranches {
		branchCount++
		idx := findBranchIndex(m.branchList, branch)
		isSelected := idx == m.selectedBranch
		isLast := branchCount == totalBranches && len(remoteBranches) == 0
		if i > 0 {
			lines = append(lines, trunkStyle.Render("    │"))
		}
		status := buildStatus(branch)
		lines = append(lines, m.renderGraphBranch(branch, idx, isSelected, isLast, localStyle, selectedStyle, trunkStyle, status))
	}

	if len(localBranches) > 0 && len(remoteBranches) > 0 {
		lines = append(lines, trunkStyle.Render("    │"))
		lines = append(lines, trunkStyle.Render("    │  ")+remoteStyle.Render("── Remote ──"))
	} else if len(localBranches) == 0 && len(remoteBranches) > 0 {
		lines = append(lines, trunkStyle.Render("    │  ")+remoteStyle.Render("── Remote ──"))
	}

	for i, branch := range remoteBranches {
		idx := findBranchIndex(m.branchList, branch)
		isSelected := idx == m.selectedBranch
		isLast := i == len(remoteBranches)-1
		if i > 0 {
			lines = append(lines, trunkStyle.Render("    │"))
		}
		remoteInfo := ""
		if branch.Remote != "" {
			remoteInfo = lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(fmt.Sprintf(" @%s", branch.Remote))
		}
		status := remoteInfo + buildStatus(branch)
		nodeStyle := remoteStyle
		if branch.IsTracked {
			nodeStyle = trackedStyle
		}
		lines = append(lines, m.renderGraphBranch(branch, idx, isSelected, isLast, nodeStyle, selectedStyle, trunkStyle, status))
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderGraphBranch(branch internal.Branch, idx int, isSelected, isLast bool, nodeStyle, selectedStyle, trunkStyle lipgloss.Style, status string) string {
	connector := "├"
	if isLast {
		connector = "└"
	}
	var nodeChar string
	if branch.IsLocal {
		nodeChar = "●"
	} else if branch.IsTracked {
		nodeChar = "◐"
	} else {
		nodeChar = "○"
	}
	branchName := branch.Name
	nameStyle := nodeStyle
	if isSelected {
		branchName = "► " + branchName
		nameStyle = selectedStyle
		nodeChar = "◆"
	}
	conflictIndicator := ""
	if branch.HasConflict {
		conflictIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Render(" ⚠ diverged")
	}
	branchLine := fmt.Sprintf("    %s─%s %s%s%s",
		trunkStyle.Render(connector),
		nodeStyle.Render(nodeChar),
		nameStyle.Render(branchName),
		status,
		conflictIndicator,
	)
	return mark(m.zoneManager, mouse.ZoneBranch(idx), branchLine)
}
