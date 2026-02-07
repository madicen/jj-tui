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
		switch {
		case commit.IsWorking:
			graphPrefix = lipgloss.NewStyle().Foreground(ColorSecondary).Render("@  ")
		case commit.Immutable:
			graphPrefix = GraphStyle.Render("◆  ")
		default:
			graphPrefix = GraphStyle.Render("○  ")
		}

		// Selection indicator (prepended before graph)
		selectionPrefix := "  "
		if data.InRebaseMode {
			switch {
			case data.RebaseSourceCommit > -1:
				selectionPrefix = "⚡ " // Source being rebased
			case data.SelectedCommit > -1:
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

	// Build changed files section with focus indicator
	if len(data.ChangedFiles) > 0 {
		focusIndicator := "  "
		if !data.GraphFocused {
			focusIndicator = "► "
		}
		fileLines = append(fileLines, lipgloss.NewStyle().Bold(true).Render(focusIndicator+"Changed Files (Tab to switch):"))

		// Build and render the file tree
		treeLines := renderFileTree(data.ChangedFiles)
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
	name     string
	status   string                   // Empty for directories
	children map[string]*fileTreeNode // Child nodes (directories and files)
	isFile   bool
}

// renderFileTree builds a tree structure from the changed files and renders it
func renderFileTree(files []ChangedFile) []string {
	// Build the tree
	root := &fileTreeNode{children: make(map[string]*fileTreeNode)}

	for _, file := range files {
		parts := strings.Split(file.Path, "/")
		current := root

		for i, part := range parts {
			if current.children == nil {
				current.children = make(map[string]*fileTreeNode)
			}

			if _, exists := current.children[part]; !exists {
				current.children[part] = &fileTreeNode{
					name:     part,
					children: make(map[string]*fileTreeNode),
				}
			}
			current = current.children[part]

			// If this is the last part, it's a file
			if i == len(parts)-1 {
				current.isFile = true
				current.status = file.Status
			}
		}
	}

	// Render the tree
	var lines []string
	renderTreeNode(root, "", &lines, true)
	return lines
}

// renderTreeNode recursively renders a tree node
func renderTreeNode(node *fileTreeNode, indent string, lines *[]string, isRoot bool) {
	if !isRoot {
		if node.isFile {
			// Render file with status
			statusStyle, statusChar := GetStatusStyle(node.status)
			*lines = append(*lines, fmt.Sprintf("%s%s %s", indent, statusStyle.Render(statusChar), node.name))
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
		renderTreeNode(node.children[name], newIndent, lines, false)
	}
	for _, name := range fileNodes {
		renderTreeNode(node.children[name], newIndent, lines, false)
	}
}
