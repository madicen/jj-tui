package model

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/jj"
	"github.com/madicen/jj-tui/internal/models"
	"github.com/madicen/jj-tui/internal/tui/actions"
)

// Clipboard actions

// getErrorMessage returns the current error message (from m.err or status message)
func (m *Model) getErrorMessage() string {
	if m.err != nil {
		return m.err.Error()
	}
	statusLower := strings.ToLower(m.statusMessage)
	if strings.Contains(statusLower, "error") || strings.Contains(statusLower, "failed") {
		return m.statusMessage
	}
	return ""
}

// copyErrorMessageToClipboard copies a specific error message to clipboard
func (m *Model) copyErrorMessageToClipboard(errMsg string) tea.Cmd {
	return actions.CopyToClipboard(errMsg)
}

// Commit actions

func (m *Model) createNewCommit() tea.Cmd {
	parentCommitID := ""
	if m.isSelectedCommitValid() {
		// Always use the selected commit as parent - creating a child of any commit
		// (including immutable ones like main) is valid since we're not modifying the parent
		commit := m.repository.Graph.Commits[m.selectedCommit]
		parentCommitID = commit.ChangeID
	}
	return actions.NewCommit(m.jjService, parentCommitID)
}

func (m *Model) checkoutCommit() tea.Cmd {
	if !m.isSelectedCommitValid() {
		return nil
	}
	return actions.Checkout(m.jjService, m.repository.Graph.Commits[m.selectedCommit].ChangeID)
}

func (m *Model) squashCommit() tea.Cmd {
	if !m.isSelectedCommitValid() {
		return nil
	}
	commit := m.repository.Graph.Commits[m.selectedCommit]
	m.statusMessage = fmt.Sprintf("Squashing %s...", commit.ShortID)
	return actions.Squash(m.jjService, commit.ChangeID)
}

func (m *Model) abandonCommit() tea.Cmd {
	if !m.isSelectedCommitValid() {
		return nil
	}
	commit := m.repository.Graph.Commits[m.selectedCommit]
	m.statusMessage = fmt.Sprintf("Abandoning %s...", commit.ShortID)
	return actions.Abandon(m.jjService, commit.ChangeID)
}

func (m *Model) moveFileToParent(commitID, filePath string) tea.Cmd {
	return actions.SplitFileToParent(m.jjService, commitID, filePath)
}

func (m *Model) moveFileToChild(commitID, filePath string) tea.Cmd {
	return actions.MoveFileToChild(m.jjService, commitID, filePath)
}

func (m *Model) revertFile(commitID, filePath string) tea.Cmd {
	return actions.RevertFile(m.jjService, commitID, filePath)
}

func (m *Model) startRebaseMode() {
	if !m.isSelectedCommitValid() {
		return
	}
	commit := m.repository.Graph.Commits[m.selectedCommit]
	m.selectionMode = SelectionRebaseDestination
	m.rebaseSourceCommit = m.selectedCommit
	m.statusMessage = fmt.Sprintf("Select destination for rebasing %s (Esc to cancel)", commit.ShortID)
}

func (m *Model) cancelRebaseMode() {
	m.selectionMode = SelectionNormal
	m.rebaseSourceCommit = -1
	m.statusMessage = "Rebase cancelled"
}

func (m *Model) performRebase(destCommitIndex int) tea.Cmd {
	sourceCommit := m.repository.Graph.Commits[m.rebaseSourceCommit]
	destCommit := m.repository.Graph.Commits[destCommitIndex]

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

	return actions.Rebase(m.jjService, sourceID, destID)
}

// Description actions

func (m *Model) startEditingDescription(commit models.Commit) (tea.Model, tea.Cmd) {
	m.viewMode = ViewEditDescription
	m.editingCommitID = commit.ChangeID
	m.descriptionInput.SetWidth(m.width - 10)
	m.descriptionInput.SetHeight(m.height - 12)
	m.statusMessage = fmt.Sprintf("Loading description for %s...", commit.ShortID)
	return m, m.loadFullDescription(commit.ChangeID)
}

func (m *Model) loadFullDescription(commitID string) tea.Cmd {
	return actions.LoadDescription(m.jjService, commitID)
}

func (m *Model) saveDescription() tea.Cmd {
	return actions.SaveDescription(m.jjService, m.editingCommitID, strings.TrimSpace(m.descriptionInput.Value()))
}

// Bookmark actions

func (m *Model) startCreateBookmark() {
	if !m.isSelectedCommitValid() {
		m.statusMessage = "No commit selected"
		return
	}

	commit := m.repository.Graph.Commits[m.selectedCommit]
	m.bookmarkCommitIdx = m.selectedCommit
	m.bookmarkNameInput.SetValue("")
	m.bookmarkNameInput.Focus()
	m.bookmarkNameInput.Width = m.width - 10
	m.existingBookmarks = actions.GetExistingBookmarks(m.repository, m.selectedCommit)
	m.selectedBookmarkIdx = -1

	m.viewMode = ViewCreateBookmark
	m.statusMessage = fmt.Sprintf("Create or move bookmark on %s", commit.ShortID)
}

func (m *Model) submitBookmark() tea.Cmd {
	if m.bookmarkFromJira {
		return m.submitBookmarkFromJira()
	}

	commit := m.repository.Graph.Commits[m.bookmarkCommitIdx]
	commitID := commit.ChangeID

	if m.selectedBookmarkIdx >= 0 && m.selectedBookmarkIdx < len(m.existingBookmarks) {
		bookmarkName := m.existingBookmarks[m.selectedBookmarkIdx]
		m.statusMessage = fmt.Sprintf("Moving bookmark '%s'...", bookmarkName)
		return actions.MoveBookmark(m.jjService, bookmarkName, commitID)
	}

	bookmarkName := strings.TrimSpace(m.bookmarkNameInput.Value())

	// Sanitize bookmark name if setting is enabled
	if m.settingsSanitizeBookmarks {
		bookmarkName = jj.SanitizeBookmarkName(bookmarkName)
	}

	if err := actions.ValidateBookmarkName(bookmarkName); err != "" {
		m.statusMessage = err
		return nil
	}

	m.statusMessage = fmt.Sprintf("Creating bookmark '%s'...", bookmarkName)
	return actions.CreateBookmark(m.jjService, bookmarkName, commitID)
}

func (m *Model) submitBookmarkFromJira() tea.Cmd {
	bookmarkName := strings.TrimSpace(m.bookmarkNameInput.Value())

	// Sanitize bookmark name if setting is enabled
	if m.settingsSanitizeBookmarks {
		bookmarkName = jj.SanitizeBookmarkName(bookmarkName)
	}

	if err := actions.ValidateBookmarkName(bookmarkName); err != "" {
		m.statusMessage = err
		return nil
	}

	m.statusMessage = fmt.Sprintf("Creating branch '%s' from main...", bookmarkName)

	if m.bookmarkJiraTicketTitle != "" && m.bookmarkJiraTicketKey != "" {
		keyForTitle := m.bookmarkJiraTicketKey
		if m.bookmarkTicketDisplayKey != "" {
			keyForTitle = m.bookmarkTicketDisplayKey
		}
		m.jiraBookmarkTitles[bookmarkName] = keyForTitle + " - " + m.bookmarkJiraTicketTitle
	}

	if m.bookmarkTicketDisplayKey != "" {
		m.ticketBookmarkDisplayKeys[bookmarkName] = m.bookmarkTicketDisplayKey
	}

	// Capture ticket key for auto-transition before clearing it
	ticketKey := m.bookmarkJiraTicketKey

	m.bookmarkFromJira = false
	m.bookmarkJiraTicketKey = ""
	m.bookmarkJiraTicketTitle = ""
	m.bookmarkTicketDisplayKey = ""

	return actions.CreateBranchFromMain(m.jjService, bookmarkName, ticketKey)
}

func (m *Model) deleteBookmark() tea.Cmd {
	if !m.isSelectedCommitValid() {
		m.statusMessage = "No commit selected"
		return nil
	}

	commit := m.repository.Graph.Commits[m.selectedCommit]
	if len(commit.Branches) == 0 {
		m.statusMessage = "No bookmark on this commit to delete"
		return nil
	}

	bookmarkName := commit.Branches[0]
	m.statusMessage = fmt.Sprintf("Deleting bookmark '%s'...", bookmarkName)
	return actions.DeleteBookmark(m.jjService, bookmarkName)
}

func (m *Model) findBookmarkForCommit(commitIdx int) string {
	return actions.FindBookmarkForCommit(m.repository, commitIdx)
}

// PR actions

func (m *Model) startCreatePR() {
	if !m.isSelectedCommitValid() {
		m.statusMessage = "No commit selected"
		return
	}

	commit := m.repository.Graph.Commits[m.selectedCommit]
	var headBranch string
	var needsMoveBookmark bool

	if len(commit.Branches) > 0 {
		headBranch = commit.Branches[0]
		needsMoveBookmark = false
	} else {
		headBranch = m.findBookmarkForCommit(m.selectedCommit)
		if headBranch == "" {
			m.statusMessage = "No bookmark found. Create one first with 'b'."
			return
		}
		needsMoveBookmark = true
	}

	m.prCommitIndex = m.selectedCommit
	m.prHeadBranch = headBranch
	m.prBaseBranch = "main"
	m.prFocusedField = 0
	m.prNeedsMoveBookmark = needsMoveBookmark

	defaultTitle := m.prHeadBranch
	if jiraPRTitle, ok := m.jiraBookmarkTitles[m.prHeadBranch]; ok {
		defaultTitle = jiraPRTitle
	}
	m.prTitleInput.SetValue(defaultTitle)
	m.prTitleInput.Focus()
	m.prBodyInput.SetValue("")
	m.prBodyInput.Blur()
	m.prTitleInput.Width = m.width - 10
	m.prBodyInput.SetWidth(m.width - 10)
	// Calculate body height: total height minus header(~3), status(1), and PR form chrome(~14 lines)
	bodyHeight := m.height - 20
	bodyHeight = min(max(bodyHeight, 3), 8)
	m.prBodyInput.SetHeight(bodyHeight)

	m.viewMode = ViewCreatePR
	if needsMoveBookmark {
		m.statusMessage = fmt.Sprintf("Creating PR for %s (will move bookmark)", m.prHeadBranch)
	} else {
		m.statusMessage = "Creating PR for " + m.prHeadBranch
	}
}

func (m *Model) submitPR() tea.Cmd {
	title := strings.TrimSpace(m.prTitleInput.Value())
	if title == "" {
		m.statusMessage = "Title is required"
		return nil
	}

	m.statusMessage = fmt.Sprintf("%s %s and creating PR...", If(m.prNeedsMoveBookmark, "Moving bookmark", "Pushing"), m.prHeadBranch)

	var commitChangeID string
	if m.prNeedsMoveBookmark && m.repository != nil && m.prCommitIndex >= 0 && m.prCommitIndex < len(m.repository.Graph.Commits) {
		commitChangeID = m.repository.Graph.Commits[m.prCommitIndex].ChangeID
	}

	return actions.CreatePR(m.jjService, m.githubService, actions.PRCreateParams{
		Title:             title,
		Body:              strings.TrimSpace(m.prBodyInput.Value()),
		HeadBranch:        m.prHeadBranch,
		BaseBranch:        m.prBaseBranch,
		NeedsMoveBookmark: m.prNeedsMoveBookmark,
		CommitChangeID:    commitChangeID,
	})
}

func (m *Model) pushToPR(branch string, commitID string, moveBookmark bool) tea.Cmd {
	if moveBookmark {
		m.statusMessage = fmt.Sprintf("Moving %s and pushing...", branch)
	} else {
		m.statusMessage = fmt.Sprintf("Pushing %s...", branch)
	}
	return actions.PushToPR(m.jjService, branch, commitID, moveBookmark)
}

func (m *Model) findPRBranchForCommit(commitIndex int) string {
	return actions.FindPRBranchForCommit(m.repository, commitIndex)
}

// Settings actions

func (m *Model) saveSettings() tea.Cmd {
	params := actions.SettingsParams{
		GitHubToken:          strings.TrimSpace(m.settingsInputs[0].Value()),
		JiraURL:              strings.TrimSpace(m.settingsInputs[1].Value()),
		JiraUser:             strings.TrimSpace(m.settingsInputs[2].Value()),
		JiraToken:            strings.TrimSpace(m.settingsInputs[3].Value()),
		JiraExcludedStatuses: strings.TrimSpace(m.settingsInputs[4].Value()),
		ShowMerged:           m.settingsShowMerged,
		ShowClosed:           m.settingsShowClosed,
		OnlyMine:             m.settingsOnlyMine,
		PRLimit:              m.settingsPRLimit,
		PRRefreshInterval:    m.settingsPRRefreshInterval,
		AutoInProgress:       m.settingsAutoInProgress,
		BranchLimit:          m.settingsBranchLimit,
		SanitizeBookmarks:    m.settingsSanitizeBookmarks,
	}
	if len(m.settingsInputs) > 8 {
		params.CodecksSubdomain = strings.TrimSpace(m.settingsInputs[5].Value())
		params.CodecksToken = strings.TrimSpace(m.settingsInputs[6].Value())
		params.CodecksProject = strings.TrimSpace(m.settingsInputs[7].Value())
		params.CodecksExcludedStatuses = strings.TrimSpace(m.settingsInputs[8].Value())
	}
	return actions.SaveSettings(params)
}

func (m *Model) saveSettingsLocal() tea.Cmd {
	params := actions.SettingsParams{
		GitHubToken:          strings.TrimSpace(m.settingsInputs[0].Value()),
		JiraURL:              strings.TrimSpace(m.settingsInputs[1].Value()),
		JiraUser:             strings.TrimSpace(m.settingsInputs[2].Value()),
		JiraToken:            strings.TrimSpace(m.settingsInputs[3].Value()),
		JiraExcludedStatuses: strings.TrimSpace(m.settingsInputs[4].Value()),
		ShowMerged:           m.settingsShowMerged,
		ShowClosed:           m.settingsShowClosed,
		OnlyMine:             m.settingsOnlyMine,
		PRLimit:              m.settingsPRLimit,
		PRRefreshInterval:    m.settingsPRRefreshInterval,
		AutoInProgress:       m.settingsAutoInProgress,
		BranchLimit:          m.settingsBranchLimit,
		SanitizeBookmarks:    m.settingsSanitizeBookmarks,
	}
	if len(m.settingsInputs) > 8 {
		params.CodecksSubdomain = strings.TrimSpace(m.settingsInputs[5].Value())
		params.CodecksToken = strings.TrimSpace(m.settingsInputs[6].Value())
		params.CodecksProject = strings.TrimSpace(m.settingsInputs[7].Value())
		params.CodecksExcludedStatuses = strings.TrimSpace(m.settingsInputs[8].Value())
	}
	return actions.SaveSettingsLocal(params)
}
