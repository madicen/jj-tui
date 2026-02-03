package view

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Graph renders the commit graph view
func (r *Renderer) Graph(data GraphData) string {
	if data.Repository == nil || len(data.Repository.Graph.Commits) == 0 {
		return "No commits found. Press Ctrl+r to refresh."
	}

	var lines []string

	// Style for graph lines (muted color)
	graphStyle := lipgloss.NewStyle().Foreground(ColorMuted)

	// Special styles for rebase mode
	rebaseSourceStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#5555AA")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true)
	rebaseDestStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#55AA55")).
		Foreground(lipgloss.Color("#FFFFFF"))

	// Show rebase mode header if active
	if data.InRebaseMode {
		rebaseHeader := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFAA00")).
			Bold(true).
			Render("🔀 REBASE MODE - Select destination commit (Esc to cancel)")
		lines = append(lines, rebaseHeader)
		lines = append(lines, "")
	}

	for i, commit := range data.Repository.Graph.Commits {
		// Build commit line
		style := CommitStyle

		// In rebase mode, use special styling
		if data.InRebaseMode {
			if i == data.RebaseSourceCommit {
				// Source commit being rebased - highlighted differently
				style = rebaseSourceStyle
			} else if i == data.SelectedCommit {
				// Potential destination - green highlight
				style = rebaseDestStyle
			}
		} else if i == data.SelectedCommit {
			style = CommitSelectedStyle
		}

		// Use jj's graph prefix if available, otherwise fall back to simple display
		var graphPrefix string
		if commit.GraphPrefix != "" {
			// Use jj's native graph prefix
			graphPrefix = graphStyle.Render(commit.GraphPrefix)
		} else {
			// Fall back to simple prefix
			if commit.IsWorking {
				graphPrefix = graphStyle.Render("@  ")
			} else if commit.Immutable {
				graphPrefix = graphStyle.Render("◆  ")
			} else {
				graphPrefix = graphStyle.Render("○  ")
			}
		}

		// Selection indicator (prepended before graph)
		selectionPrefix := "  "
		if data.InRebaseMode {
			if i == data.RebaseSourceCommit {
				selectionPrefix = "⚡ " // Source being rebased
			} else if i == data.SelectedCommit {
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
		lines = append(lines, r.Zone.Mark(ZoneCommit(i), style.Render(commitLine)))

		// Render graph connector lines after this commit (if any)
		for _, graphLine := range commit.GraphLines {
			// These are the lines between commits (like │, ├─╯, etc.)
			// Add spacing to align with commit lines
			paddedLine := "  " + graphStyle.Render(graphLine)
			lines = append(lines, paddedLine)
		}
	}

	// Don't show action buttons in rebase mode - user is selecting destination
	if data.InRebaseMode {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("Press Enter or click to select destination, Esc to cancel"))
		return strings.Join(lines, "\n")
	}

	// Add action buttons
	lines = append(lines, "")
	lines = append(lines, "Actions:")

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
			lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left, actionButtons...))
			lines = append(lines, "")
			lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("◆ Selected commit is immutable (pushed to remote)"))
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
			lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left, actionButtons...))
		}
	} else {
		lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left, actionButtons...))
	}

	// Show changed files for selected commit in a tree structure
	if len(data.ChangedFiles) > 0 {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Changed Files:"))

		// Build and render the file tree
		fileLines := renderFileTree(data.ChangedFiles)
		lines = append(lines, fileLines...)
	}

	return strings.Join(lines, "\n")
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
			statusStyle, statusChar := getStatusStyle(node.status)
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

// getStatusStyle returns the style and character for a file status
func getStatusStyle(status string) (lipgloss.Style, string) {
	switch status {
	case "M":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB86C")), "M" // Orange for modified
	case "A":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")), "A" // Green for added
	case "D":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")), "D" // Red for deleted
	case "R":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD")), "R" // Cyan for renamed
	default:
		return lipgloss.NewStyle().Foreground(ColorMuted), status
	}
}

