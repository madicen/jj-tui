package graph

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/view"
)

// GraphResult contains the split rendering for commit graph view
type GraphResult struct {
	GraphContent string // Commit graph (scrollable)
	ActionsBar   string // Actions buttons (fixed in middle)
	FilesContent string // Changed files (scrollable)
	FullContent  string // Full content for non-split views
}

// Graph renders the commit graph view with split panes
func (m graphModel) Graph(data GraphData) GraphResult {
	if data.Repository == nil || len(data.Repository.Graph.Commits) == 0 {
		return GraphResult{
			FullContent: "No commits found. Press Ctrl+r to refresh.",
		}
	}

	var graphLines []string
	var actionLines []string
	var fileLines []string

	// Show rebase mode header if active
	if data.InRebaseMode {
		rebaseHeader := RebaseHeaderStyle.
			Render("🔀 REBASE MODE - Select destination commit (Esc to cancel)")
		graphLines = append(graphLines, rebaseHeader)
		graphLines = append(graphLines, "")
	}

	for i, commit := range data.Repository.Graph.Commits {
		// Build commit line
		style := CommitStyle

		// In rebase mode, use special styling
		if data.InRebaseMode {
			switch {
			case data.RebaseSourceCommit > -1:
				// Source commit being rebased - highlighted differently
				style = RebaseSourceStyle
			case data.SelectedCommit > -1:
				// Potential destination - green highlight
				style = RebaseDestStyle
			}
		} else if i == data.SelectedCommit {
			style = CommitSelectedStyle
		}

		var graphPrefix string
		var graphStyle lipgloss.Style = GraphStyle
		if commit.IsWorking {
			// Highlight working copy with a different color
			graphStyle = graphStyle.Foreground(view.ColorSecondary)
		}
		if commit.GraphPrefix != "" {
			// Use jj's native graph prefix
			graphPrefix = graphStyle.Render(commit.GraphPrefix)
		} else {
			switch {
			case commit.IsWorking:
				graphPrefix = graphStyle.Render("@  ")
			case commit.Immutable:
				graphPrefix = graphStyle.Render("◆  ")
			default:
				graphPrefix = graphStyle.Render("○  ")
			}
		}

		// Selection indicator (prepended before graph)
		selectionPrefix := "  "
		if data.InRebaseMode {
			switch {
			case data.RebaseSourceCommit == i:
				selectionPrefix = "⚡ " // Source being rebased
			case data.SelectedCommit == i:
				selectionPrefix = "→ " // Target destination
			}
		} else if i == data.SelectedCommit {
			selectionPrefix = "► "
		}

		// Show conflict/divergent indicators
		statusIndicator := ""
		if commit.Conflicts {
			statusIndicator = " ⚠"
		}
		if commit.Divergent {
			// Divergent commit - multiple versions of this change ID exist
			statusIndicator += lipgloss.NewStyle().Foreground(lipgloss.Color("#FF79C6")).Render(" ⑂ divergent")
		}

		// Show branches/bookmarks with diverged indicator
		branchStr := ""
		if len(commit.Branches) > 0 {
			// Build branch display with conflict markers for diverged bookmarks
			var branchParts []string
			conflictedSet := make(map[string]bool)
			for _, cb := range commit.ConflictedBranches {
				conflictedSet[cb] = true
			}
			for _, b := range commit.Branches {
				if conflictedSet[b] {
					// Diverged bookmark - show warning indicator
					branchParts = append(branchParts, lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Render(b+" ⚠"))
				} else {
					branchParts = append(branchParts, b)
				}
			}
			branchStr = " " + lipgloss.NewStyle().Foreground(view.ColorSecondary).Render("["+strings.Join(branchParts, ", ")+"]")
		}

		// Format the commit line: selection + graph + commit_id + summary + branches + status
		commitLine := fmt.Sprintf("%s%s%s %s%s%s",
			selectionPrefix,
			graphPrefix,
			CommitIDStyle.Render(commit.ShortID),
			commit.Summary,
			branchStr,
			statusIndicator,
		)

		// Wrap in zone for click detection
		graphLines = append(graphLines, m.zoneManager.Mark(mouse.ZoneCommit(i), style.Render(commitLine)))

		// Render graph connector lines after this commit (if any)
		for _, graphLine := range commit.GraphLines {
			// These are the lines between commits (like │, ├─╯, etc.)
			// Add spacing to align with commit lines
			paddedLine := "  " + GraphStyle.Render(graphLine)
			graphLines = append(graphLines, paddedLine)
		}
	}

	// Don't show action buttons in rebase mode - user is selecting destination
	if data.InRebaseMode {
		graphLines = append(graphLines, "")
		graphLines = append(graphLines, lipgloss.NewStyle().Foreground(view.ColorMuted).Render("Press Enter or click to select destination, Esc to cancel"))
		graphContent := strings.Join(graphLines, "\n")
		return GraphResult{
			GraphContent: graphContent,
			FullContent:  graphContent,
		}
	}

	// Add action buttons - context-aware based on focus
	// When files pane is focused: show file actions
	// When graph pane is focused: show commit actions
	if !data.GraphFocused && len(data.ChangedFiles) > 0 && data.SelectedFile >= 0 {
		// File actions mode
		actionLines = append(actionLines, "File Actions:")

		var fileActionButtons []string

		// Check if commit is mutable (file actions only work on mutable commits)
		isMutable := false
		if data.SelectedCommit >= 0 && data.SelectedCommit < len(data.Repository.Graph.Commits) {
			commit := data.Repository.Graph.Commits[data.SelectedCommit]
			isMutable = !commit.Immutable
		}

		if isMutable {
			fileActionButtons = append(fileActionButtons,
				m.zoneManager.Mark(mouse.ZoneActionMoveFileUp, view.ButtonStyle.Render("Move to Parent ([)")),
				m.zoneManager.Mark(mouse.ZoneActionMoveFileDown, view.ButtonStyle.Render("Move to Child (])")),
				m.zoneManager.Mark(mouse.ZoneActionRevertFile, view.ButtonStyle.Render("Revert Changes (v)")),
			)
			actionLines = append(actionLines, lipgloss.JoinHorizontal(lipgloss.Left, fileActionButtons...))
		} else {
			actionLines = append(actionLines, lipgloss.NewStyle().Foreground(view.ColorMuted).Render("◆ Cannot modify files in immutable commits"))
		}
	} else {
		// Commit actions mode
		actionLines = append(actionLines, "Actions:")

		// Always show "New" action
		actionButtons := []string{
			m.zoneManager.Mark(mouse.ZoneActionNewCommit, view.ButtonStyle.Render("New (n)")),
		}

		// Add commit-specific actions if a commit is selected
		if data.SelectedCommit >= 0 && data.SelectedCommit < len(data.Repository.Graph.Commits) {
			commit := data.Repository.Graph.Commits[data.SelectedCommit]

			if commit.Immutable {
				// For immutable commits, only show delete bookmark if it has one
				if len(commit.Branches) > 0 {
					actionButtons = append(actionButtons,
						m.zoneManager.Mark(mouse.ZoneActionDelBookmark, view.ButtonStyle.Render("Del Bookmark (x)")),
					)
				}
				actionLines = append(actionLines, lipgloss.JoinHorizontal(lipgloss.Left, actionButtons...))
				actionLines = append(actionLines, "")
				actionLines = append(actionLines, lipgloss.NewStyle().Foreground(view.ColorMuted).Render("◆ Selected commit is immutable (pushed to remote)"))
			} else {
				actionButtons = append(actionButtons,
					m.zoneManager.Mark(mouse.ZoneActionCheckout, view.ButtonStyle.Render("Edit (e)")),
					m.zoneManager.Mark(mouse.ZoneActionDescribe, view.ButtonStyle.Render("Describe (d)")),
					m.zoneManager.Mark(mouse.ZoneActionSquash, view.ButtonStyle.Render("Squash (s)")),
					m.zoneManager.Mark(mouse.ZoneActionRebase, view.ButtonStyle.Render("Rebase (r)")),
					m.zoneManager.Mark(mouse.ZoneActionAbandon, view.ButtonStyle.Render("Abandon (a)")),
					m.zoneManager.Mark(mouse.ZoneActionBookmark, view.ButtonStyle.Render("Bookmark (m)")),
				)

				// Show delete bookmark button if commit has bookmarks
				if len(commit.Branches) > 0 {
					actionButtons = append(actionButtons,
						m.zoneManager.Mark(mouse.ZoneActionDelBookmark, view.ButtonStyle.Render("Del Bookmark (x)")),
					)
				}

				// Show resolve divergent button if commit is divergent
				if commit.Divergent {
					divergentBtnStyle := view.ButtonStyle.Background(lipgloss.Color("#FF79C6"))
					actionButtons = append(actionButtons,
						m.zoneManager.Mark(mouse.ZoneActionResolveDivergent, divergentBtnStyle.Render("Resolve Divergent (a)")),
					)
				}

				// Check if this commit can push to a PR (either has the bookmark or is a descendant)
				prBranch := ""
				if data.CommitPRBranch != nil {
					prBranch = data.CommitPRBranch[data.SelectedCommit]
				}

				if prBranch != "" {
					// This commit (or an ancestor) has an open PR - show Update PR button
					buttonLabel := "Update PR (u)"
					if len(commit.Branches) == 0 {
						// This is a descendant without the bookmark - indicate we'll add commits to the PR
						buttonLabel = fmt.Sprintf("Update PR [%s] (u)", prBranch)
					}
					actionButtons = append(actionButtons,
						m.zoneManager.Mark(mouse.ZoneActionPush, view.ButtonStyle.Render(buttonLabel)),
					)
				} else {
					// Check if this commit can create a PR (has a bookmark or is a descendant of one)
					createPRBranch := ""
					if data.CommitBookmark != nil {
						createPRBranch = data.CommitBookmark[data.SelectedCommit]
					}
					if createPRBranch != "" {
						// Can create a NEW PR - show button
						buttonLabel := "Create PR (c)"
						if len(commit.Branches) == 0 {
							// This is a descendant - indicate we'll move the bookmark to include all commits
							buttonLabel = fmt.Sprintf("Create PR [%s] (c)", createPRBranch)
						}
						actionButtons = append(actionButtons,
							m.zoneManager.Mark(mouse.ZoneActionCreatePR, view.ButtonStyle.Render(buttonLabel)),
						)
					}
				}
				actionLines = append(actionLines, lipgloss.JoinHorizontal(lipgloss.Left, actionButtons...))
			}
		} else {
			actionLines = append(actionLines, lipgloss.JoinHorizontal(lipgloss.Left, actionButtons...))
		}
	}

	// Build changed files section with focus indicator and tree view
	if len(data.ChangedFiles) > 0 {
		focusIndicator := "  "
		if !data.GraphFocused {
			focusIndicator = "► "
		}
		fileLines = append(fileLines, lipgloss.NewStyle().Bold(true).Render(focusIndicator+"Changed Files (Tab to switch):"))

		// Render files as a tree structure with selection highlighting
		// Move buttons are rendered inline with the selected file
		treeLines := m.renderFileTree(data)
		fileLines = append(fileLines, treeLines...)
	}

	// Build full content for backward compatibility
	var allLines []string
	allLines = append(allLines, graphLines...)
	allLines = append(allLines, "")
	allLines = append(allLines, actionLines...)
	if len(fileLines) > 0 {
		allLines = append(allLines, "")
		allLines = append(allLines, fileLines...)
	}

	// Always add focus indicator to graph header for consistent layout
	graphContent := strings.Join(graphLines, "\n")
	focusIndicator := "  "
	if data.GraphFocused {
		focusIndicator = "► "
	}
	graphContent = lipgloss.NewStyle().Bold(true).Render(focusIndicator+"Graph (Tab to switch):") + "\n" + graphContent

	return GraphResult{
		GraphContent: graphContent,
		ActionsBar:   strings.Join(actionLines, "\n"),
		FilesContent: strings.Join(fileLines, "\n"),
		FullContent:  strings.Join(allLines, "\n"),
	}
}

// fileTreeNode represents a node in the file tree
type fileTreeNode struct {
	name      string
	status    string                   // Empty for directories
	children  map[string]*fileTreeNode // Child nodes (directories and files)
	isFile    bool
	fileIndex int // Original index in the files list (for selection tracking)
}

// renderFileTree builds a tree structure from the changed files and renders it
// Accepts GraphData to support selection highlighting
func (m *graphModel) renderFileTree(data GraphData) []string {
	// Build the tree
	root := &fileTreeNode{children: make(map[string]*fileTreeNode), fileIndex: -1}

	for i, file := range data.ChangedFiles {
		parts := strings.Split(file.Path, "/")
		current := root

		for j, part := range parts {
			if current.children == nil {
				current.children = make(map[string]*fileTreeNode)
			}

			if _, exists := current.children[part]; !exists {
				current.children[part] = &fileTreeNode{
					name:      part,
					children:  make(map[string]*fileTreeNode),
					fileIndex: -1,
				}
			}
			current = current.children[part]

			// If this is the last part, it's a file
			if j == len(parts)-1 {
				current.isFile = true
				current.status = file.Status
				current.fileIndex = i // Store original index for selection
			}
		}
	}

	// Render the tree
	var lines []string
	m.renderTreeNode(root, "", &lines, true, data)
	return lines
}

// renderTreeNode recursively renders a tree node with selection support
func (m *graphModel) renderTreeNode(node *fileTreeNode, indent string, lines *[]string, isRoot bool, data GraphData) {
	if !isRoot {
		if node.isFile {
			// Check if this file is selected
			isSelected := !data.GraphFocused && node.fileIndex == data.SelectedFile
			statusStyle, statusChar := GetStatusStyle(node.status)

			var fileLine string
			if isSelected {
				// Render with selection highlight
				selectedStyle := lipgloss.NewStyle().
					Background(lipgloss.Color("#3d4f5f")).
					Foreground(lipgloss.Color("#ffffff"))
				fileLine = fmt.Sprintf("%s%s %s", indent, statusStyle.Render(statusChar), selectedStyle.Render(node.name))
			} else {
				fileLine = fmt.Sprintf("%s%s %s", indent, statusStyle.Render(statusChar), node.name)
			}
			*lines = append(*lines, m.zoneManager.Mark(mouse.ZoneChangedFile(node.fileIndex), fileLine))
		} else {
			// Render directory
			dirStyle := lipgloss.NewStyle().Foreground(view.ColorPrimary)
			*lines = append(*lines, fmt.Sprintf("%s%s/", indent, dirStyle.Render(node.name)))
		}
	}

	// Sort children: directories first, then files, both alphabetically
	var dirs, fileNodes []string
	for name, child := range node.children {
		if child.isFile {
			fileNodes = append(fileNodes, name)
		} else {
			dirs = append(dirs, name)
		}
	}
	sort.Strings(dirs)
	sort.Strings(fileNodes)

	// Calculate new indent
	newIndent := indent
	if !isRoot {
		newIndent = indent + "  "
	}

	// Render directories first, then files
	for _, name := range dirs {
		m.renderTreeNode(node.children[name], newIndent, lines, false, data)
	}
	for _, name := range fileNodes {
		m.renderTreeNode(node.children[name], newIndent, lines, false, data)
	}
}
