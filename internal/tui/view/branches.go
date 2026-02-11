package view

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/madicen/jj-tui/internal/models"
)

// BranchData contains data needed for branches rendering
type BranchData struct {
	Branches       []models.Branch
	SelectedBranch int
	Width          int
}

// BranchResult contains the split rendering for branches
type BranchResult struct {
	FixedHeader    string // Details section that stays fixed
	ScrollableList string // List that scrolls
	FullContent    string // Full content for non-split views
}

// Branches renders the branches view with split header/list for scrolling
func (r *Renderer) Branches(data BranchData) BranchResult {
	if len(data.Branches) == 0 {
		content := []string{
			TitleStyle.Render("Branches"),
			"",
			"No branches found.",
			"",
			"Press 'F' to fetch from all remotes.",
		}
		return BranchResult{FullContent: strings.Join(content, "\n")}
	}

	// Build fixed header section
	var headerLines []string

	// Show selected branch details
	if data.SelectedBranch >= 0 && data.SelectedBranch < len(data.Branches) {
		branch := data.Branches[data.SelectedBranch]

		// Build details content
		var detailLines []string

		// Branch name with type indicator
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
			lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(branch.Name),
			typeLabel,
		))

		// Location info - show "local" for local branches, remote name for remote branches
		if branch.IsLocal {
			detailLines = append(detailLines, "Location: local")
		} else if branch.Remote != "" {
			detailLines = append(detailLines, fmt.Sprintf("Remote: %s", branch.Remote))
		}

		// Commit info
		if branch.ShortID != "" {
			detailLines = append(detailLines, fmt.Sprintf("Commit: %s", branch.ShortID))
		}

		detailsBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 1).
			Render(strings.Join(detailLines, "\n"))
		headerLines = append(headerLines, detailsBox)

		// Separator line style (same as graph tab)
		separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
		separatorWidth := data.Width - 4
		if separatorWidth < 20 {
			separatorWidth = 80 // fallback
		}
		separator := separatorStyle.Render(strings.Repeat("─", separatorWidth))

		// Actions section with separators (like Graph tab)
		headerLines = append(headerLines, separator)
		headerLines = append(headerLines, "Actions:")

		// Build action buttons based on branch type
		var actionButtons []string

		if branch.IsLocal {
			// Local branch: can push or delete
			actionButtons = append(actionButtons,
				r.Mark(ZoneBranchPush, ButtonStyle.Render("Push (P)")),
			)
			actionButtons = append(actionButtons,
				r.Mark(ZoneBranchDelete, ButtonStyle.Render("Delete (x)")),
			)
		} else if branch.IsTracked {
			// Tracked remote branch: can untrack
			actionButtons = append(actionButtons,
				r.Mark(ZoneBranchUntrack, ButtonStyle.Render("Untrack (U)")),
			)
			// If local was deleted, offer to restore it
			if branch.LocalDeleted {
				actionButtons = append(actionButtons,
					r.Mark(ZoneBranchRestore, ButtonStyle.Render("Restore Local (L)")),
				)
			}
		} else {
			// Untracked remote branch: can track
			actionButtons = append(actionButtons,
				r.Mark(ZoneBranchTrack, ButtonStyle.Render("Track (T)")),
			)
		}

		// Fetch is always available
		actionButtons = append(actionButtons,
			r.Mark(ZoneBranchFetch, ButtonStyle.Render("Fetch All (F)")),
		)

		headerLines = append(headerLines, strings.Join(actionButtons, " "))
		headerLines = append(headerLines, separator)
	}

	// Build scrollable list section - unified Branch Graph
	var listLines []string
	listLines = append(listLines, r.renderBranchGraph(data.Branches, data.SelectedBranch))

	fixedHeader := strings.Join(headerLines, "\n")
	scrollableList := strings.Join(listLines, "\n")
	fullContent := fixedHeader + "\n" + scrollableList

	return BranchResult{
		FixedHeader:    fixedHeader,
		ScrollableList: scrollableList,
		FullContent:    fullContent,
	}
}

// findBranchIndex finds the index of a branch in the full list
func findBranchIndex(branches []models.Branch, target models.Branch) int {
	for i, b := range branches {
		if b.Name == target.Name && b.Remote == target.Remote {
			return i
		}
	}
	return -1
}

// renderBranchGraph renders a visual tree showing all branches relative to trunk
func (r *Renderer) renderBranchGraph(branches []models.Branch, selectedIdx int) string {
	if len(branches) == 0 {
		return ""
	}

	var lines []string

	// Style definitions
	trunkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD"))
	localStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B"))
	trackedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD"))
	remoteStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4"))
	aheadStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B"))
	behindStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB86C"))
	selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF79C6"))

	// Trunk line
	trunkLine := trunkStyle.Render("trunk ─────────────────────────────● (tip)")
	lines = append(lines, trunkLine)

	// Separate into local and remote for grouping
	var localBranches, remoteBranches []models.Branch
	for _, b := range branches {
		if b.IsLocal {
			localBranches = append(localBranches, b)
		} else {
			remoteBranches = append(remoteBranches, b)
		}
	}

	totalBranches := len(localBranches) + len(remoteBranches)
	branchCount := 0

	// Render local branches first
	for i, branch := range localBranches {
		branchCount++
		idx := findBranchIndex(branches, branch)
		isSelected := idx == selectedIdx
		isLast := branchCount == totalBranches && len(remoteBranches) == 0

		// Add vertical connector before branch (except first)
		if i > 0 {
			lines = append(lines, trunkStyle.Render("    │"))
		}

		// Build status string for local branches
		var statusParts []string
		if branch.Ahead > 0 {
			statusParts = append(statusParts, aheadStyle.Render(fmt.Sprintf("+%d", branch.Ahead)))
		}
		if branch.Behind > 0 {
			statusParts = append(statusParts, behindStyle.Render(fmt.Sprintf("-%d behind", branch.Behind)))
		}
		status := ""
		if len(statusParts) > 0 {
			status = " (" + strings.Join(statusParts, ", ") + ")"
		} else {
			status = lipgloss.NewStyle().Foreground(ColorMuted).Render(" (up to date)")
		}

		lines = append(lines, r.renderGraphBranch(branch, idx, isSelected, isLast, localStyle, selectedStyle, trunkStyle, status))
	}

	// Add section separator if we have both local and remote
	if len(localBranches) > 0 && len(remoteBranches) > 0 {
		lines = append(lines, trunkStyle.Render("    │"))
		lines = append(lines, trunkStyle.Render("    │  ")+remoteStyle.Render("── Remote ──"))
	}

	// Render remote branches
	for i, branch := range remoteBranches {
		branchCount++
		idx := findBranchIndex(branches, branch)
		isSelected := idx == selectedIdx
		isLast := i == len(remoteBranches)-1

		// Add vertical connector before branch (except first remote after separator)
		if i > 0 {
			lines = append(lines, trunkStyle.Render("    │"))
		}

		// Remote info
		remoteInfo := ""
		if branch.Remote != "" {
			remoteInfo = lipgloss.NewStyle().Foreground(ColorMuted).Render(fmt.Sprintf(" @%s", branch.Remote))
		}

		// Build ahead/behind status for remote branches too
		var statusParts []string
		if branch.Ahead > 0 {
			statusParts = append(statusParts, aheadStyle.Render(fmt.Sprintf("+%d", branch.Ahead)))
		}
		if branch.Behind > 0 {
			statusParts = append(statusParts, behindStyle.Render(fmt.Sprintf("-%d behind", branch.Behind)))
		}

		status := remoteInfo
		if len(statusParts) > 0 {
			status += " (" + strings.Join(statusParts, ", ") + ")"
		}

		// Choose style based on tracked status
		nodeStyle := remoteStyle
		if branch.IsTracked {
			nodeStyle = trackedStyle
		}

		lines = append(lines, r.renderGraphBranch(branch, idx, isSelected, isLast, nodeStyle, selectedStyle, trunkStyle, status))
	}

	return strings.Join(lines, "\n")
}

// renderGraphBranch renders a single branch in the graph
func (r *Renderer) renderGraphBranch(branch models.Branch, idx int, isSelected, isLast bool, nodeStyle, selectedStyle, trunkStyle lipgloss.Style, status string) string {
	// Determine connector
	connector := "├"
	if isLast {
		connector = "└"
	}

	// Node character based on type
	var nodeChar string
	if branch.IsLocal {
		nodeChar = "●"
	} else if branch.IsTracked {
		nodeChar = "◐"
	} else {
		nodeChar = "○"
	}

	// Selection styling
	branchName := branch.Name
	nameStyle := nodeStyle
	if isSelected {
		branchName = "► " + branchName
		nameStyle = selectedStyle
		nodeChar = "◆"
	}

	// Build the branch line with clickable zone
	branchLine := fmt.Sprintf("    %s─%s %s%s",
		trunkStyle.Render(connector),
		nodeStyle.Render(nodeChar),
		nameStyle.Render(branchName),
		status,
	)

	return r.Mark(ZoneBranch(idx), branchLine)
}
