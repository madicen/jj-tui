package view

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Graph renders the commit graph view with split panes
func (r *Renderer) Graph(data GraphData) GraphResult {
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
			graphStyle = graphStyle.Foreground(ColorSecondary)
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

		// Show conflict indicator
		conflictIndicator := ""
		if commit.Conflicts {
			conflictIndicator = " ⚠"
		}

		// Show branches/bookmarks
		branchStr := ""
		if len(commit.Branches) > 0 {
			branchStr = " " + lipgloss.NewStyle().Foreground(ColorSecondary).Render("["+strings.Join(commit.Branches, ", ")+"]")
		}

		// Format the commit line: selection + graph + commit_id + summary + branches + conflict
		commitLine := fmt.Sprintf("%s%s%s %s%s%s",
			selectionPrefix,
			graphPrefix,
			CommitIDStyle.Render(commit.ShortID),
			commit.Summary,
			branchStr,
			conflictIndicator,
		)

		// Wrap in zone for click detection
		graphLines = append(graphLines, r.Zone.Mark(ZoneCommit(i), style.Render(commitLine)))

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
		graphLines = append(graphLines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Press Enter or click to select destination, Esc to cancel"))
		graphContent := strings.Join(graphLines, "\n")
		return GraphResult{
			GraphContent: graphContent,
			FullContent:  graphContent,
		}
	}

	// Add action buttons
	actionLines = append(actionLines, "Actions:")

	// Always show "New" action
	actionButtons := []string{
		r.Zone.Mark(ZoneActionNewCommit, ButtonStyle.Render("New (n)")),
	}

	// Add commit-specific actions if a commit is selected
	if data.SelectedCommit >= 0 && data.SelectedCommit < len(data.Repository.Graph.Commits) {
		commit := data.Repository.Graph.Commits[data.SelectedCommit]

		if commit.Immutable {
			// For immutable commits, only show delete bookmark if it has one
			if len(commit.Branches) > 0 {
				actionButtons = append(actionButtons,
					r.Zone.Mark(ZoneActionDelBookmark, ButtonStyle.Render("Del Bookmark (x)")),
				)
			}
			actionLines = append(actionLines, lipgloss.JoinHorizontal(lipgloss.Left, actionButtons...))
			actionLines = append(actionLines, "")
			actionLines = append(actionLines, lipgloss.NewStyle().Foreground(ColorMuted).Render("◆ Selected commit is immutable (pushed to remote)"))
		} else {
			actionButtons = append(actionButtons,
				r.Zone.Mark(ZoneActionCheckout, ButtonStyle.Render("Edit (e)")),
				r.Zone.Mark(ZoneActionDescribe, ButtonStyle.Render("Describe (d)")),
				r.Zone.Mark(ZoneActionSquash, ButtonStyle.Render("Squash (s)")),
				r.Zone.Mark(ZoneActionRebase, ButtonStyle.Render("Rebase (r)")),
				r.Zone.Mark(ZoneActionAbandon, ButtonStyle.Render("Abandon (a)")),
				r.Zone.Mark(ZoneActionBookmark, ButtonStyle.Render("Bookmark (b)")),
			)

			// Show delete bookmark button if commit has bookmarks
			if len(commit.Branches) > 0 {
				actionButtons = append(actionButtons,
					r.Zone.Mark(ZoneActionDelBookmark, ButtonStyle.Render("Del Bookmark (x)")),
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
					r.Zone.Mark(ZoneActionPush, ButtonStyle.Render(buttonLabel)),
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
						r.Zone.Mark(ZoneActionCreatePR, ButtonStyle.Render(buttonLabel)),
					)
				}
			}
			actionLines = append(actionLines, lipgloss.JoinHorizontal(lipgloss.Left, actionButtons...))
		}
	} else {
		actionLines = append(actionLines, lipgloss.JoinHorizontal(lipgloss.Left, actionButtons...))
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
		treeLines := r.renderFileTree(data)
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
func (r *Renderer) renderFileTree(data GraphData) []string {
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
	r.renderTreeNode(root, "", &lines, true, data)
	return lines
}

// renderTreeNode recursively renders a tree node with selection support
func (r *Renderer) renderTreeNode(node *fileTreeNode, indent string, lines *[]string, isRoot bool, data GraphData) {
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

				// Add inline move buttons for mutable commits
				if data.SelectedCommit >= 0 && data.SelectedCommit < len(data.Repository.Graph.Commits) {
					selectedCommit := data.Repository.Graph.Commits[data.SelectedCommit]
					if !selectedCommit.Immutable {
						moveUp := r.Zone.Mark(ZoneActionMoveFileUp, ButtonStyle.Render("Move Up ([)"))
						moveDown := r.Zone.Mark(ZoneActionMoveFileDown, ButtonStyle.Render("Move Down (])"))
						fileLine = fileLine + "  " + moveUp + " " + moveDown
					}
				}
			} else {
				fileLine = fmt.Sprintf("%s%s %s", indent, statusStyle.Render(statusChar), node.name)
			}
			*lines = append(*lines, r.Zone.Mark(ZoneChangedFile(node.fileIndex), fileLine))
		} else {
			// Render directory
			dirStyle := lipgloss.NewStyle().Foreground(ColorPrimary)
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
		r.renderTreeNode(node.children[name], newIndent, lines, false, data)
	}
	for _, name := range fileNodes {
		r.renderTreeNode(node.children[name], newIndent, lines, false, data)
	}
}
