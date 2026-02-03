package tui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/codecks"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/jira"
	"github.com/madicen/jj-tui/internal/models"
	"github.com/madicen/jj-tui/internal/tickets"
)

// createNewCommit creates a new commit
func (m *Model) createNewCommit() tea.Cmd {
	return func() tea.Msg {
		if err := m.jjService.NewCommit(context.Background()); err != nil {
			return errorMsg{err: fmt.Errorf("failed to create commit: %w", err)}
		}
		// Reload repository after change
		repo, err := m.jjService.GetRepository(context.Background())
		if err != nil {
			return errorMsg{err: err}
		}
		return repositoryLoadedMsg{repository: repo}
	}
}

// checkoutCommit checks out (edits) the selected commit
func (m *Model) checkoutCommit() tea.Cmd {
	if !m.isSelectedCommitValid() {
		return nil
	}
	commit := m.repository.Graph.Commits[m.selectedCommit]
	return func() tea.Msg {
		if err := m.jjService.CheckoutCommit(context.Background(), commit.ChangeID); err != nil {
			return errorMsg{err: fmt.Errorf("failed to checkout: %w", err)}
		}
		// Reload repository after change
		repo, err := m.jjService.GetRepository(context.Background())
		if err != nil {
			return errorMsg{err: err}
		}
		// Return editCompletedMsg to select the working copy after reload
		return editCompletedMsg{repository: repo}
	}
}

// squashCommit squashes the selected commit into its parent
func (m *Model) squashCommit() tea.Cmd {
	if !m.isSelectedCommitValid() {
		return nil
	}
	commit := m.repository.Graph.Commits[m.selectedCommit]
	m.statusMessage = fmt.Sprintf("Squashing %s...", commit.ShortID)
	return func() tea.Msg {
		if err := m.jjService.SquashCommit(context.Background(), commit.ChangeID); err != nil {
			return errorMsg{err: fmt.Errorf("failed to squash: %w", err)}
		}
		// Reload repository after change
		repo, err := m.jjService.GetRepository(context.Background())
		if err != nil {
			return errorMsg{err: err}
		}
		return repositoryLoadedMsg{repository: repo}
	}
}

// abandonCommit abandons the selected commit
func (m *Model) abandonCommit() tea.Cmd {
	if !m.isSelectedCommitValid() {
		return nil
	}
	commit := m.repository.Graph.Commits[m.selectedCommit]
	m.statusMessage = fmt.Sprintf("Abandoning %s...", commit.ShortID)
	return func() tea.Msg {
		if err := m.jjService.AbandonCommit(context.Background(), commit.ChangeID); err != nil {
			return errorMsg{err: fmt.Errorf("failed to abandon: %w", err)}
		}
		// Reload repository after change
		repo, err := m.jjService.GetRepository(context.Background())
		if err != nil {
			return errorMsg{err: err}
		}
		return repositoryLoadedMsg{repository: repo}
	}
}

// startRebaseMode enters rebase selection mode
func (m *Model) startRebaseMode() {
	if !m.isSelectedCommitValid() {
		return
	}
	commit := m.repository.Graph.Commits[m.selectedCommit]
	m.selectionMode = SelectionRebaseDestination
	m.rebaseSourceCommit = m.selectedCommit
	m.statusMessage = fmt.Sprintf("Select destination for rebasing %s (Esc to cancel)", commit.ShortID)
}

// cancelRebaseMode exits rebase selection mode
func (m *Model) cancelRebaseMode() {
	m.selectionMode = SelectionNormal
	m.rebaseSourceCommit = -1
	m.statusMessage = "Rebase cancelled"
}

// performRebase executes the rebase operation
func (m *Model) performRebase(destCommitIndex int) tea.Cmd {
	sourceCommit := m.repository.Graph.Commits[m.rebaseSourceCommit]
	destCommit := m.repository.Graph.Commits[destCommitIndex]

	// Can't rebase onto itself
	if m.rebaseSourceCommit == destCommitIndex {
		m.selectionMode = SelectionNormal
		m.rebaseSourceCommit = -1
		m.statusMessage = "Cannot rebase commit onto itself"
		return nil
	}

	m.statusMessage = fmt.Sprintf("Rebasing %s onto %s...", sourceCommit.ShortID, destCommit.ShortID)
	m.selectionMode = SelectionNormal
	sourceID := sourceCommit.ChangeID
	destID := destCommit.ChangeID
	m.rebaseSourceCommit = -1

	return func() tea.Msg {
		if err := m.jjService.RebaseCommit(context.Background(), sourceID, destID); err != nil {
			return errorMsg{err: fmt.Errorf("failed to rebase: %w", err)}
		}
		// Reload repository after change
		repo, err := m.jjService.GetRepository(context.Background())
		if err != nil {
			return errorMsg{err: err}
		}
		return repositoryLoadedMsg{repository: repo}
	}
}

// startEditingDescription starts editing a commit's description
func (m *Model) startEditingDescription(commit models.Commit) (tea.Model, tea.Cmd) {
	m.viewMode = ViewEditDescription
	m.editingCommitID = commit.ChangeID

	// Resize textarea to fit the content area
	m.descriptionInput.SetWidth(m.width - 10)
	m.descriptionInput.SetHeight(m.height - 12)

	m.statusMessage = fmt.Sprintf("Loading description for %s...", commit.ShortID)

	// Fetch the full description asynchronously
	return m, m.loadFullDescription(commit.ChangeID)
}

// loadFullDescription fetches the complete description for a commit
func (m *Model) loadFullDescription(commitID string) tea.Cmd {
	return func() tea.Msg {
		if m.jjService == nil {
			return errorMsg{err: fmt.Errorf("jj service not available")}
		}

		// Get the full description from jj
		desc, err := m.jjService.GetCommitDescription(context.Background(), commitID)
		if err != nil {
			return errorMsg{err: fmt.Errorf("failed to load description: %w", err)}
		}

		return descriptionLoadedMsg{
			commitID:    commitID,
			description: desc,
		}
	}
}

// saveDescription saves the edited description
func (m *Model) saveDescription() tea.Cmd {
	commitID := m.editingCommitID
	description := strings.TrimSpace(m.descriptionInput.Value())

	return func() tea.Msg {
		ctx := context.Background()

		// Use jj describe to set the new description
		if err := m.jjService.DescribeCommit(ctx, commitID, description); err != nil {
			return errorMsg{err: fmt.Errorf("failed to update description: %w", err)}
		}

		return descriptionSavedMsg{commitID: commitID}
	}
}

// startCreatePR initializes the PR creation form for a commit
func (m *Model) startCreatePR() {
	if m.repository == nil || m.selectedCommit < 0 || m.selectedCommit >= len(m.repository.Graph.Commits) {
		m.statusMessage = "No commit selected"
		return
	}

	commit := m.repository.Graph.Commits[m.selectedCommit]

	// Find the bookmark - either directly on this commit or from an ancestor
	var headBranch string
	var needsMoveBookmark bool

	if len(commit.Branches) > 0 {
		// Commit has a bookmark directly
		headBranch = commit.Branches[0]
		needsMoveBookmark = false
	} else {
		// Need to find a bookmark from ancestors
		headBranch = m.findBookmarkForCommit(m.selectedCommit)
		if headBranch == "" {
			m.statusMessage = "No bookmark found. Create one first with 'm'."
			return
		}
		needsMoveBookmark = true
	}

	// Set up the PR creation form
	m.prCommitIndex = m.selectedCommit
	m.prHeadBranch = headBranch
	m.prBaseBranch = "main"
	m.prFocusedField = 0
	m.prNeedsMoveBookmark = needsMoveBookmark

	// Default title: use the stored "KEY - Title" if we have a Jira mapping, otherwise just the branch name
	defaultTitle := m.prHeadBranch
	if jiraPRTitle, ok := m.jiraBookmarkTitles[m.prHeadBranch]; ok {
		defaultTitle = jiraPRTitle
	}
	m.prTitleInput.SetValue(defaultTitle)
	m.prTitleInput.Focus()

	// Default body from commit description
	m.prBodyInput.SetValue("")
	m.prBodyInput.Blur()

	// Resize inputs
	m.prTitleInput.Width = m.width - 10
	m.prBodyInput.SetWidth(m.width - 10)
	m.prBodyInput.SetHeight(m.height - 15)

	m.viewMode = ViewCreatePR
	if needsMoveBookmark {
		m.statusMessage = fmt.Sprintf("Creating PR for %s (will move bookmark to include all commits)", m.prHeadBranch)
	} else {
		m.statusMessage = "Creating PR for " + m.prHeadBranch
	}
}

// findBookmarkForCommit finds a bookmark from ancestors of the given commit
func (m *Model) findBookmarkForCommit(commitIdx int) string {
	if m.repository == nil || commitIdx < 0 || commitIdx >= len(m.repository.Graph.Commits) {
		return ""
	}

	// Build a map of commit ID to index
	commitIDToIndex := make(map[string]int)
	for i, commit := range m.repository.Graph.Commits {
		commitIDToIndex[commit.ID] = i
		commitIDToIndex[commit.ChangeID] = i
	}

	// BFS to find an ancestor with a bookmark
	visited := make(map[int]bool)
	queue := []int{commitIdx}

	for len(queue) > 0 {
		idx := queue[0]
		queue = queue[1:]

		if visited[idx] {
			continue
		}
		visited[idx] = true

		commit := m.repository.Graph.Commits[idx]

		// Check if this commit has a bookmark
		if len(commit.Branches) > 0 {
			return commit.Branches[0]
		}

		// Add parents to queue
		for _, parentID := range commit.Parents {
			if parentIdx, ok := commitIDToIndex[parentID]; ok {
				queue = append(queue, parentIdx)
			}
		}
	}

	return ""
}

// submitPR pushes the branch and creates the PR
func (m *Model) submitPR() tea.Cmd {
	title := strings.TrimSpace(m.prTitleInput.Value())
	body := strings.TrimSpace(m.prBodyInput.Value())
	headBranch := m.prHeadBranch
	baseBranch := m.prBaseBranch
	needsMoveBookmark := m.prNeedsMoveBookmark
	commitIndex := m.prCommitIndex

	if title == "" {
		m.statusMessage = "Title is required"
		return nil
	}

	if needsMoveBookmark {
		m.statusMessage = fmt.Sprintf("Moving bookmark %s and creating PR...", headBranch)
	} else {
		m.statusMessage = fmt.Sprintf("Pushing %s and creating PR...", headBranch)
	}

	return func() tea.Msg {
		ctx := context.Background()

		// If we need to move the bookmark to include all commits
		if needsMoveBookmark && m.repository != nil && commitIndex >= 0 && commitIndex < len(m.repository.Graph.Commits) {
			commit := m.repository.Graph.Commits[commitIndex]
			if err := m.jjService.MoveBookmark(ctx, headBranch, commit.ChangeID); err != nil {
				return errorMsg{err: fmt.Errorf("failed to move bookmark %s: %w", headBranch, err)}
			}
		}

		// Push the branch to GitHub
		pushOutput, err := m.jjService.PushToGit(ctx, headBranch)
		if err != nil {
			return errorMsg{err: fmt.Errorf("failed to push branch: %w\nOutput: %s", err, pushOutput)}
		}

		// Give GitHub a moment to index the branch
		time.Sleep(3 * time.Second)

		// Create the PR with retry logic for eventual consistency
		var pr *models.GitHubPR
		var lastErr error
		for attempt := 0; attempt < 5; attempt++ {
			pr, lastErr = m.githubService.CreatePullRequest(ctx, &models.CreatePRRequest{
				Title:      title,
				Body:       body,
				HeadBranch: headBranch,
				BaseBranch: baseBranch,
			})
			if lastErr == nil {
				break
			}
			// If it's a ref-related error, wait and retry
			if strings.Contains(lastErr.Error(), "not all refs") || strings.Contains(lastErr.Error(), "422") {
				time.Sleep(3 * time.Second)
				continue
			}
			// For other errors, fail immediately
			break
		}
		if lastErr != nil {
			return errorMsg{err: fmt.Errorf("failed to create PR (branch: %s): %w\nPush output: %s", headBranch, lastErr, pushOutput)}
		}

		return prCreatedMsg{pr: pr}
	}
}

// startCreateBookmark initializes the bookmark creation form for a commit
func (m *Model) startCreateBookmark() {
	if m.repository == nil || m.selectedCommit < 0 || m.selectedCommit >= len(m.repository.Graph.Commits) {
		m.statusMessage = "No commit selected"
		return
	}

	commit := m.repository.Graph.Commits[m.selectedCommit]

	// Set up the bookmark creation form
	m.bookmarkCommitIdx = m.selectedCommit
	m.bookmarkNameInput.SetValue("")
	m.bookmarkNameInput.Focus()
	m.bookmarkNameInput.Width = m.width - 10

	// Collect existing bookmarks from all commits (excluding ones already on this commit)
	existingOnCommit := make(map[string]bool)
	for _, b := range commit.Branches {
		existingOnCommit[b] = true
	}

	bookmarkSet := make(map[string]bool)
	for _, c := range m.repository.Graph.Commits {
		for _, b := range c.Branches {
			// Don't include bookmarks that are already on this commit
			if !existingOnCommit[b] {
				bookmarkSet[b] = true
			}
		}
	}

	// Convert to sorted slice
	m.existingBookmarks = make([]string, 0, len(bookmarkSet))
	for b := range bookmarkSet {
		m.existingBookmarks = append(m.existingBookmarks, b)
	}
	// Sort alphabetically
	for i := 0; i < len(m.existingBookmarks); i++ {
		for j := i + 1; j < len(m.existingBookmarks); j++ {
			if m.existingBookmarks[i] > m.existingBookmarks[j] {
				m.existingBookmarks[i], m.existingBookmarks[j] = m.existingBookmarks[j], m.existingBookmarks[i]
			}
		}
	}

	m.selectedBookmarkIdx = -1 // Start with "new bookmark" selected

	m.viewMode = ViewCreateBookmark
	m.statusMessage = fmt.Sprintf("Create or move bookmark on %s", commit.ShortID)
}

// submitBookmark creates or moves a bookmark on the selected commit
func (m *Model) submitBookmark() tea.Cmd {
	// Special handling for Jira flow: create new branch from main
	if m.bookmarkFromJira {
		return m.submitBookmarkFromJira()
	}

	commit := m.repository.Graph.Commits[m.bookmarkCommitIdx]
	commitID := commit.ChangeID

	// Check if we're moving an existing bookmark or creating a new one
	if m.selectedBookmarkIdx >= 0 && m.selectedBookmarkIdx < len(m.existingBookmarks) {
		// Moving an existing bookmark
		bookmarkName := m.existingBookmarks[m.selectedBookmarkIdx]
		m.statusMessage = fmt.Sprintf("Moving bookmark '%s'...", bookmarkName)

		return func() tea.Msg {
			ctx := context.Background()

			if err := m.jjService.MoveBookmark(ctx, bookmarkName, commitID); err != nil {
				return errorMsg{err: fmt.Errorf("failed to move bookmark: %w", err)}
			}

			return bookmarkCreatedOnCommitMsg{
				bookmarkName: bookmarkName,
				commitID:     commitID,
				wasMoved:     true,
			}
		}
	}

	// Creating a new bookmark
	bookmarkName := strings.TrimSpace(m.bookmarkNameInput.Value())

	if bookmarkName == "" {
		m.statusMessage = "Bookmark name is required"
		return nil
	}

	// Validate bookmark name (no spaces, no special chars except - and _)
	for _, r := range bookmarkName {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '/') {
			m.statusMessage = "Invalid bookmark name. Use letters, numbers, -, _, or /"
			return nil
		}
	}

	m.statusMessage = fmt.Sprintf("Creating bookmark '%s'...", bookmarkName)

	return func() tea.Msg {
		ctx := context.Background()

		if err := m.jjService.CreateBookmarkOnCommit(ctx, bookmarkName, commitID); err != nil {
			return errorMsg{err: fmt.Errorf("failed to create bookmark: %w", err)}
		}

		return bookmarkCreatedOnCommitMsg{
			bookmarkName: bookmarkName,
			commitID:     commitID,
			wasMoved:     false,
		}
	}
}

// submitBookmarkFromJira creates a new branch from main with the Jira ticket as bookmark
// and rebases the current work onto it
func (m *Model) submitBookmarkFromJira() tea.Cmd {
	bookmarkName := strings.TrimSpace(m.bookmarkNameInput.Value())

	if bookmarkName == "" {
		m.statusMessage = "Bookmark name is required"
		return nil
	}

	// Validate bookmark name
	for _, r := range bookmarkName {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '/') {
			m.statusMessage = "Invalid bookmark name. Use letters, numbers, -, _, or /"
			return nil
		}
	}

	m.statusMessage = fmt.Sprintf("Creating branch '%s' from main...", bookmarkName)

	// Save the Jira PR title mapping (formatted as "KEY - Title")
	if m.bookmarkJiraTicketTitle != "" && m.bookmarkJiraTicketKey != "" {
		m.jiraBookmarkTitles[bookmarkName] = m.bookmarkJiraTicketKey + " - " + m.bookmarkJiraTicketTitle
	}

	// Save the ticket short ID for commit message prepopulation
	if m.bookmarkTicketDisplayKey != "" {
		m.ticketBookmarkDisplayKeys[bookmarkName] = m.bookmarkTicketDisplayKey
	}

	// Reset ticket state
	m.bookmarkFromJira = false
	m.bookmarkJiraTicketKey = ""
	m.bookmarkJiraTicketTitle = ""
	m.bookmarkTicketDisplayKey = ""

	return func() tea.Msg {
		ctx := context.Background()

		// Create the branch from main with the bookmark
		if err := m.jjService.CreateBranchFromMain(ctx, bookmarkName); err != nil {
			return errorMsg{err: fmt.Errorf("failed to create branch from main: %w", err)}
		}

		return bookmarkCreatedOnCommitMsg{
			bookmarkName: bookmarkName,
			commitID:     "main",
			wasMoved:     false,
		}
	}
}

// deleteBookmark deletes a bookmark from the selected commit
func (m *Model) deleteBookmark() tea.Cmd {
	if m.repository == nil || m.selectedCommit < 0 || m.selectedCommit >= len(m.repository.Graph.Commits) {
		m.statusMessage = "No commit selected"
		return nil
	}

	commit := m.repository.Graph.Commits[m.selectedCommit]

	if len(commit.Branches) == 0 {
		m.statusMessage = "No bookmark on this commit to delete"
		return nil
	}

	// For now, delete the first bookmark on the commit
	// TODO: If multiple bookmarks, show a selection UI
	bookmarkName := commit.Branches[0]
	m.statusMessage = fmt.Sprintf("Deleting bookmark '%s'...", bookmarkName)

	return func() tea.Msg {
		ctx := context.Background()

		if err := m.jjService.DeleteBookmark(ctx, bookmarkName); err != nil {
			return errorMsg{err: fmt.Errorf("failed to delete bookmark: %w", err)}
		}

		return bookmarkDeletedMsg{bookmarkName: bookmarkName}
	}
}

// pushToPR pushes updates to a PR, moving the bookmark if necessary
func (m *Model) pushToPR(branch string, commitID string, moveBookmark bool) tea.Cmd {
	if moveBookmark {
		m.statusMessage = fmt.Sprintf("Moving %s to include new commits and pushing...", branch)
	} else {
		m.statusMessage = fmt.Sprintf("Pushing %s...", branch)
	}

	return func() tea.Msg {
		ctx := context.Background()

		// If needed, move the bookmark to include this commit
		if moveBookmark {
			if err := m.jjService.MoveBookmark(ctx, branch, commitID); err != nil {
				return errorMsg{err: fmt.Errorf("failed to move bookmark %s: %w", branch, err)}
			}
		}

		// Push the branch to GitHub
		pushOutput, err := m.jjService.PushToGit(ctx, branch)
		if err != nil {
			return errorMsg{err: fmt.Errorf("failed to push: %w\nOutput: %s", err, pushOutput)}
		}

		return branchPushedMsg{
			branch:     branch,
			pushOutput: pushOutput,
		}
	}
}

// findPRBranchForCommit finds the PR branch that this commit can push to
// (either the commit has the branch directly, or it's a descendant of a commit with the branch)
func (m *Model) findPRBranchForCommit(commitIndex int) string {
	if m.repository == nil || commitIndex < 0 || commitIndex >= len(m.repository.Graph.Commits) {
		return ""
	}

	// Build set of open PR branches
	openPRBranches := make(map[string]bool)
	for _, pr := range m.repository.PRs {
		if pr.State == "open" {
			openPRBranches[pr.HeadBranch] = true
		}
	}

	// Build commit ID to index map
	commitIDToIndex := make(map[string]int)
	for i, commit := range m.repository.Graph.Commits {
		commitIDToIndex[commit.ID] = i
		commitIDToIndex[commit.ChangeID] = i
	}

	// Check this commit and traverse ancestors to find a PR branch
	visited := make(map[int]bool)
	queue := []int{commitIndex}

	for len(queue) > 0 {
		idx := queue[0]
		queue = queue[1:]

		if visited[idx] {
			continue
		}
		visited[idx] = true

		commit := m.repository.Graph.Commits[idx]

		// Check if this commit has a PR branch
		for _, branch := range commit.Branches {
			if openPRBranches[branch] {
				return branch
			}
		}

		// Add parents to queue
		for _, parentID := range commit.Parents {
			if parentIdx, ok := commitIDToIndex[parentID]; ok {
				if !visited[parentIdx] {
					queue = append(queue, parentIdx)
				}
			}
		}
	}

	return ""
}

// saveSettings saves the settings and reinitializes services
func (m *Model) saveSettings() tea.Cmd {
	// Get values from inputs
	// Index mapping:
	// 0: GitHub Token
	// 1: Jira URL
	// 2: Jira User
	// 3: Jira Token
	// 4: Codecks Subdomain
	// 5: Codecks Token
	githubToken := strings.TrimSpace(m.settingsInputs[0].Value())
	jiraURL := strings.TrimSpace(m.settingsInputs[1].Value())
	jiraUser := strings.TrimSpace(m.settingsInputs[2].Value())
	jiraToken := strings.TrimSpace(m.settingsInputs[3].Value())

	var codecksSubdomain, codecksToken, codecksProject string
	if len(m.settingsInputs) > 6 {
		codecksSubdomain = strings.TrimSpace(m.settingsInputs[4].Value())
		codecksToken = strings.TrimSpace(m.settingsInputs[5].Value())
		codecksProject = strings.TrimSpace(m.settingsInputs[6].Value())
	}

	return func() tea.Msg {
		// Set environment variables for the current process
		if githubToken != "" {
			os.Setenv("GITHUB_TOKEN", githubToken)
		}
		if jiraURL != "" {
			os.Setenv("JIRA_URL", jiraURL)
		}
		if jiraUser != "" {
			os.Setenv("JIRA_USER", jiraUser)
		}
		if jiraToken != "" {
			os.Setenv("JIRA_TOKEN", jiraToken)
		}
		if codecksSubdomain != "" {
			os.Setenv("CODECKS_SUBDOMAIN", codecksSubdomain)
		}
		if codecksToken != "" {
			os.Setenv("CODECKS_TOKEN", codecksToken)
		}
		if codecksProject != "" {
			os.Setenv("CODECKS_PROJECT", codecksProject)
		} else {
			os.Unsetenv("CODECKS_PROJECT") // Clear filter if empty
		}

		// Determine ticket provider based on what's configured
		var ticketProvider string
		if codecksSubdomain != "" && codecksToken != "" {
			ticketProvider = "codecks"
		} else if jiraURL != "" && jiraUser != "" && jiraToken != "" {
			ticketProvider = "jira"
		}

		// Save to config file for persistence across restarts
		cfg := &config.Config{
			GitHubToken:      githubToken,
			TicketProvider:   ticketProvider,
			JiraURL:          jiraURL,
			JiraUser:         jiraUser,
			JiraToken:        jiraToken,
			CodecksSubdomain: codecksSubdomain,
			CodecksToken:     codecksToken,
			CodecksProject:   codecksProject,
		}
		// Ignore save errors - settings will still work for current session
		_ = cfg.Save()

		var githubConnected bool
		var ticketSvc tickets.Service

		// Try to initialize GitHub service
		if githubToken != "" {
			githubConnected = true
		}

		// Try to initialize ticket service based on provider
		if ticketProvider == "codecks" && codecks.IsConfigured() {
			ticketSvc, _ = codecks.NewService()
		} else if ticketProvider == "jira" && jira.IsConfigured() {
			ticketSvc, _ = jira.NewService()
		}

		return settingsSavedMsg{
			githubConnected: githubConnected,
			ticketService:   ticketSvc,
			ticketProvider:  ticketProvider,
		}
	}
}

