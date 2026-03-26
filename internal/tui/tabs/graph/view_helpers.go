package graph

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/tui/mouse"
	"github.com/madicen/jj-tui/internal/tui/styles"
)

// FindCommitsWithEmptyDescriptions finds commits from the selected commit back to
// main that have empty descriptions (excluding immutable/root commits).
func FindCommitsWithEmptyDescriptions(repo *internal.Repository, selectedCommit int) []internal.Commit {
	if repo == nil || selectedCommit < 0 || selectedCommit >= len(repo.Graph.Commits) {
		return nil
	}
	commits := repo.Graph.Commits
	var emptyDescCommits []internal.Commit
	visited := make(map[string]bool)
	queue := []int{selectedCommit}
	idToIndex := make(map[string]int)
	for i, c := range commits {
		idToIndex[c.ID] = i
		idToIndex[c.ChangeID] = i
	}
	for len(queue) > 0 {
		idx := queue[0]
		queue = queue[1:]
		if idx < 0 || idx >= len(commits) {
			continue
		}
		commit := commits[idx]
		if visited[commit.ID] {
			continue
		}
		visited[commit.ID] = true
		if commit.Immutable {
			continue
		}
		desc := strings.TrimSpace(commit.Description)
		if desc == "" || desc == "(no description)" {
			emptyDescCommits = append(emptyDescCommits, commit)
		}
		for _, parentID := range commit.Parents {
			if parentIdx, ok := idToIndex[parentID]; ok && !visited[commits[parentIdx].ID] {
				queue = append(queue, parentIdx)
			}
		}
	}
	return emptyDescCommits
}

// isFirstParentImmutable returns true if the selected commit's first parent is immutable
// (or if the parent is not found in the list, to be safe). Used to hide Squash and Move to Parent
// when they would target an immutable parent.
func isFirstParentImmutable(commits []internal.Commit, selectedIndex int) bool {
	if selectedIndex < 0 || selectedIndex >= len(commits) {
		return true
	}
	commit := commits[selectedIndex]
	if len(commit.Parents) == 0 {
		return true // root commit, no parent
	}
	parentID := commit.Parents[0]
	idToIndex := make(map[string]int)
	for i, c := range commits {
		idToIndex[c.ID] = i
		idToIndex[c.ChangeID] = i
	}
	idx, ok := idToIndex[parentID]
	if !ok {
		return true
	}
	return commits[idx].Immutable
}

// isDefaultBranch returns true for branch names that are typically the repo default (main, master).
// Creating a PR from these pushes directly to the branch instead of opening a PR.
func isDefaultBranch(branch string) bool {
	switch strings.ToLower(branch) {
	case "main", "master":
		return true
	}
	return false
}

// GraphResult contains the split rendering for commit graph view
type GraphResult struct {
	GraphContent        string
	ActionsBar          string
	FilesContent        string
	FullContent         string
	FileIndexToLineIndex []int
}

// Graph renders the commit graph view with split panes
func (m GraphModel) Graph(data GraphData) GraphResult {
	if data.Repository == nil || len(data.Repository.Graph.Commits) == 0 {
		return GraphResult{
			FullContent: "No commits found. Press Ctrl+r to refresh.",
		}
	}

	var graphLines []string
	var actionLines []string
	var fileLines []string

	if data.InRebaseMode {
		rebaseHeader := RebaseHeaderStyle.
			Render("🔀 REBASE MODE - Select destination commit (Esc to cancel)")
		graphLines = append(graphLines, rebaseHeader)
		graphLines = append(graphLines, "")
	}

	for i, commit := range data.Repository.Graph.Commits {
		style := CommitStyle
		if data.InRebaseMode {
			switch {
			case data.RebaseSourceCommit > -1:
				style = RebaseSourceStyle
			case data.SelectedCommit > -1:
				style = RebaseDestStyle
			}
		} else if i == data.SelectedCommit {
			style = CommitSelectedStyle
		}

		var graphPrefix string
		graphStyle := GraphStyle
		if commit.IsWorking {
			graphStyle = graphStyle.Foreground(styles.ColorSecondary)
		}
		if commit.GraphPrefix != "" {
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

		selectionPrefix := "  "
		if data.InRebaseMode {
			switch {
			case data.RebaseSourceCommit == i:
				selectionPrefix = "⚡ "
			case data.SelectedCommit == i:
				selectionPrefix = "→ "
			}
		} else if i == data.SelectedCommit {
			selectionPrefix = "► "
		}

		statusIndicator := ""
		if commit.Conflicts {
			statusIndicator = " ⚠"
		}
		if commit.Divergent {
			statusIndicator += lipgloss.NewStyle().Foreground(lipgloss.Color("#FF79C6")).Render(" ⑂ divergent")
		}

		branchStr := ""
		if len(commit.Branches) > 0 {
			var branchParts []string
			conflictedSet := make(map[string]bool)
			for _, cb := range commit.ConflictedBranches {
				conflictedSet[cb] = true
			}
			for _, b := range commit.Branches {
				if conflictedSet[b] {
					branchParts = append(branchParts, lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Render(b+" ⚠"))
				} else {
					branchParts = append(branchParts, b)
				}
			}
			branchStr = " " + lipgloss.NewStyle().Foreground(styles.ColorSecondary).Render("["+strings.Join(branchParts, ", ")+"]")
		}

		beforeStatus := fmt.Sprintf("%s%s%s %s%s",
			selectionPrefix,
			graphPrefix,
			CommitIDStyle.Render(commit.ShortID),
			commit.Summary,
			branchStr,
		)
		afterStatus := statusIndicator
		var commitRow string
		onSelectedRow := !data.InRebaseMode && i == data.SelectedCommit
		showForgot := onSelectedRow && commit.HasDeltaVsBookmarkOrigin && len(commit.ConflictedBranches) == 0
		// split (z): only when graph enrichment found a viable evolog split for this change.
		showEvolog := onSelectedRow && commit.EvologSplitViable
		showResolveBookmark := onSelectedRow && len(commit.ConflictedBranches) > 0
		if showForgot || showEvolog || showResolveBookmark {
			muted := lipgloss.NewStyle().Foreground(styles.ColorMuted)
			resolveStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555"))
			var parts []string
			if showForgot {
				parts = append(parts, m.zoneManager.Mark(mouse.ZoneActionMoveOntoOriginAt(i), muted.Render("Forgot New Commit? (f)")))
			}
			if showResolveBookmark {
				parts = append(parts, m.zoneManager.Mark(mouse.ZoneActionResolveBookmarkConflictAt(i), resolveStyle.Render("Resolve diverged bookmark (C)")))
			}
			if showEvolog {
				parts = append(parts, m.zoneManager.Mark(mouse.ZoneActionEvologSplitAt(i), muted.Render("split (z)")))
			}
			var btnJoin string
			for pi, p := range parts {
				if pi > 0 {
					btnJoin += muted.Render(" | ")
				}
				btnJoin += p
			}
			commitRow = lipgloss.JoinHorizontal(lipgloss.Bottom,
				m.zoneManager.Mark(mouse.ZoneCommit(i), style.Render(beforeStatus)),
				muted.Render("  "),
				btnJoin,
				style.Render(afterStatus),
			)
		} else {
			commitRow = m.zoneManager.Mark(mouse.ZoneCommit(i), style.Render(beforeStatus+afterStatus))
		}
		graphLines = append(graphLines, commitRow)

		for _, graphLine := range commit.GraphLines {
			paddedLine := "  " + GraphStyle.Render(graphLine)
			graphLines = append(graphLines, paddedLine)
		}
	}

	if data.InRebaseMode {
		graphLines = append(graphLines, "")
		graphLines = append(graphLines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("Press Enter or click to select destination, Esc to cancel"))
		graphContent := strings.Join(graphLines, "\n")
		return GraphResult{
			GraphContent: graphContent,
			FullContent:  graphContent,
		}
	}

	if !data.GraphFocused && len(data.ChangedFiles) > 0 && data.SelectedFile >= 0 {
		actionLines = append(actionLines, "File Actions:")
		var fileActionButtons []string
		isMutable := false
		if data.SelectedCommit >= 0 && data.SelectedCommit < len(data.Repository.Graph.Commits) {
			commit := data.Repository.Graph.Commits[data.SelectedCommit]
			isMutable = !commit.Immutable
		}
		if isMutable {
			if !isFirstParentImmutable(data.Repository.Graph.Commits, data.SelectedCommit) {
				fileActionButtons = append(fileActionButtons,
					m.zoneManager.Mark(mouse.ZoneActionMoveFileUp, styles.ButtonStyle.Render("Move to Parent ([)")),
				)
			}
			fileActionButtons = append(fileActionButtons,
				m.zoneManager.Mark(mouse.ZoneActionMoveFileDown, styles.ButtonStyle.Render("Move to Child (])")),
				m.zoneManager.Mark(mouse.ZoneActionRevertFile, styles.ButtonStyle.Render("Revert Changes (v)")),
			)
			actionLines = append(actionLines, lipgloss.JoinHorizontal(lipgloss.Left, fileActionButtons...))
		} else {
			actionLines = append(actionLines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("◆ Cannot modify files in immutable commits"))
		}
	} else {
		actionLines = append(actionLines, "Actions:")
		actionButtons := []string{
			m.zoneManager.Mark(mouse.ZoneActionNewCommit, styles.ButtonStyle.Render("New (n)")),
		}
		if data.SelectedCommit >= 0 && data.SelectedCommit < len(data.Repository.Graph.Commits) {
			commit := data.Repository.Graph.Commits[data.SelectedCommit]
			if commit.Immutable {
				if len(commit.Branches) > 0 {
					actionButtons = append(actionButtons,
						m.zoneManager.Mark(mouse.ZoneActionDelBookmark, styles.ButtonStyle.Render("Del Bookmark (x)")),
					)
				}
				actionLines = append(actionLines, lipgloss.JoinHorizontal(lipgloss.Left, actionButtons...))
				actionLines = append(actionLines, "")
				actionLines = append(actionLines, lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("◆ Selected commit is immutable (pushed to remote)"))
			} else {
				actionButtons = append(actionButtons,
					m.zoneManager.Mark(mouse.ZoneActionCheckout, styles.ButtonStyle.Render("Edit (e)")),
					m.zoneManager.Mark(mouse.ZoneActionDescribe, styles.ButtonStyle.Render("Describe (d)")),
				)
				if !isFirstParentImmutable(data.Repository.Graph.Commits, data.SelectedCommit) {
					actionButtons = append(actionButtons,
						m.zoneManager.Mark(mouse.ZoneActionSquash, styles.ButtonStyle.Render("Squash (s)")),
					)
				}
				actionButtons = append(actionButtons,
					m.zoneManager.Mark(mouse.ZoneActionRebase, styles.ButtonStyle.Render("Rebase (r)")),
					m.zoneManager.Mark(mouse.ZoneActionAbandon, styles.ButtonStyle.Render("Abandon (a)")),
					m.zoneManager.Mark(mouse.ZoneActionBookmark, styles.ButtonStyle.Render("Bookmark (m)")),
				)
				if len(commit.Branches) > 0 {
					actionButtons = append(actionButtons,
						m.zoneManager.Mark(mouse.ZoneActionDelBookmark, styles.ButtonStyle.Render("Del Bookmark (x)")),
					)
				}
				if commit.Divergent {
					divergentBtnStyle := styles.ButtonStyle.Background(lipgloss.Color("#FF79C6"))
					actionButtons = append(actionButtons,
						m.zoneManager.Mark(mouse.ZoneActionResolveDivergent, divergentBtnStyle.Render("Resolve Divergent (d)")),
					)
				}
				prBranch := ""
				if data.CommitPRBranch != nil {
					prBranch = data.CommitPRBranch[data.SelectedCommit]
				}
				if prBranch != "" {
					buttonLabel := "Update PR (u)"
					if len(commit.Branches) == 0 {
						buttonLabel = fmt.Sprintf("Update PR [%s] (u)", prBranch)
					}
					actionButtons = append(actionButtons,
						m.zoneManager.Mark(mouse.ZoneActionPush, styles.ButtonStyle.Render(buttonLabel)),
					)
				} else {
					createPRBranch := ""
					if data.CommitBookmark != nil {
						createPRBranch = data.CommitBookmark[data.SelectedCommit]
					}
					if createPRBranch != "" && !isDefaultBranch(createPRBranch) {
						buttonLabel := "Create PR (c)"
						if len(commit.Branches) == 0 {
							buttonLabel = fmt.Sprintf("Create PR [%s] (c)", createPRBranch)
						}
						actionButtons = append(actionButtons,
							m.zoneManager.Mark(mouse.ZoneActionCreatePR, styles.ButtonStyle.Render(buttonLabel)),
						)
					}
				}
				actionLines = append(actionLines, lipgloss.JoinHorizontal(lipgloss.Left, actionButtons...))
			}
		} else {
			actionLines = append(actionLines, lipgloss.JoinHorizontal(lipgloss.Left, actionButtons...))
		}
	}

	var fileIndexToLineIndex []int
	var treeLines []string
	if len(data.ChangedFiles) > 0 {
		focusIndicator := "  "
		if !data.GraphFocused {
			focusIndicator = "► "
		}
		fileLines = append(fileLines, lipgloss.NewStyle().Bold(true).Render(focusIndicator+"Changed Files (Tab to switch):"))
		treeLines, fileIndexToLineIndex = m.renderFileTreeWithLineIndex(data)
		for i := range fileIndexToLineIndex {
			if fileIndexToLineIndex[i] >= 0 {
				fileIndexToLineIndex[i]++
			}
		}
		fileLines = append(fileLines, treeLines...)
	}

	var allLines []string
	allLines = append(allLines, graphLines...)
	allLines = append(allLines, "")
	allLines = append(allLines, actionLines...)
	if len(fileLines) > 0 {
		allLines = append(allLines, "")
		allLines = append(allLines, fileLines...)
	}

	graphContent := strings.Join(graphLines, "\n")
	focusIndicator := "  "
	if data.GraphFocused {
		focusIndicator = "► "
	}
	graphContent = lipgloss.NewStyle().Bold(true).Render(focusIndicator+"Graph (Tab to switch):") + "\n" + graphContent

	return GraphResult{
		GraphContent:         graphContent,
		ActionsBar:           strings.Join(actionLines, "\n"),
		FilesContent:         strings.Join(fileLines, "\n"),
		FullContent:          strings.Join(allLines, "\n"),
		FileIndexToLineIndex: fileIndexToLineIndex,
	}
}

type fileTreeNode struct {
	name      string
	status    string
	children  map[string]*fileTreeNode
	isFile    bool
	fileIndex int
}

func (m *GraphModel) renderFileTreeWithLineIndex(data GraphData) (lines []string, fileIndexToLineIndex []int) {
	fileIndexToLineIndex = make([]int, len(data.ChangedFiles))
	for i := range fileIndexToLineIndex {
		fileIndexToLineIndex[i] = -1
	}
	root := &fileTreeNode{children: make(map[string]*fileTreeNode), fileIndex: -1}
	for i, file := range data.ChangedFiles {
		parts := strings.Split(file.Path, "/")
		current := root
		for j, part := range parts {
			if current.children == nil {
				current.children = make(map[string]*fileTreeNode)
			}
			if _, exists := current.children[part]; !exists {
				current.children[part] = &fileTreeNode{name: part, children: make(map[string]*fileTreeNode), fileIndex: -1}
			}
			current = current.children[part]
			if j == len(parts)-1 {
				current.isFile = true
				current.status = file.Status
				current.fileIndex = i
			}
		}
	}
	var lineIdx int
	m.renderTreeNodeWithLineIndex(root, "", &lines, true, data, &lineIdx, fileIndexToLineIndex)
	return lines, fileIndexToLineIndex
}

func (m *GraphModel) renderTreeNodeWithLineIndex(node *fileTreeNode, indent string, lines *[]string, isRoot bool, data GraphData, lineIdx *int, fileIndexToLineIndex []int) {
	if !isRoot {
		if node.isFile {
			if node.fileIndex >= 0 && node.fileIndex < len(fileIndexToLineIndex) {
				fileIndexToLineIndex[node.fileIndex] = *lineIdx
			}
			*lineIdx++
			isSelected := !data.GraphFocused && node.fileIndex == data.SelectedFile
			statusStyle, statusChar := styles.GetStatusStyle(node.status)
			var fileLine string
			if isSelected {
				selectedStyle := lipgloss.NewStyle().Background(lipgloss.Color("#3d4f5f")).Foreground(lipgloss.Color("#ffffff"))
				fileLine = fmt.Sprintf("%s%s %s", indent, statusStyle.Render(statusChar), selectedStyle.Render(node.name))
			} else {
				fileLine = fmt.Sprintf("%s%s %s", indent, statusStyle.Render(statusChar), node.name)
			}
			*lines = append(*lines, m.zoneManager.Mark(mouse.ZoneChangedFile(node.fileIndex), fileLine))
		} else {
			*lineIdx++
			dirStyle := lipgloss.NewStyle().Foreground(styles.ColorPrimary)
			*lines = append(*lines, fmt.Sprintf("%s%s/", indent, dirStyle.Render(node.name)))
		}
	}
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
	newIndent := indent
	if !isRoot {
		newIndent = indent + "  "
	}
	for _, name := range dirs {
		m.renderTreeNodeWithLineIndex(node.children[name], newIndent, lines, false, data, lineIdx, fileIndexToLineIndex)
	}
	for _, name := range fileNodes {
		m.renderTreeNodeWithLineIndex(node.children[name], newIndent, lines, false, data, lineIdx, fileIndexToLineIndex)
	}
}
