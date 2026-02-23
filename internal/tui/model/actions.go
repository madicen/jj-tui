package model

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/integrations/jj"
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
		commit := m.repository.Graph.Commits[m.GetSelectedCommit()]
		parentCommitID = commit.ChangeID
	}
	return actions.NewCommit(m.jjService, parentCommitID)
}

func (m *Model) checkoutCommit() tea.Cmd {
	if !m.isSelectedCommitValid() {
		return nil
	}
	return actions.Checkout(m.jjService, m.repository.Graph.Commits[m.GetSelectedCommit()].ChangeID)
}

func (m *Model) squashCommit() tea.Cmd {
	if !m.isSelectedCommitValid() {
		return nil
	}
	commit := m.repository.Graph.Commits[m.GetSelectedCommit()]
	m.statusMessage = fmt.Sprintf("Squashing %s...", commit.ShortID)
	return actions.Squash(m.jjService, commit.ChangeID)
}

func (m *Model) abandonCommit() tea.Cmd {
	if !m.isSelectedCommitValid() {
		return nil
	}
	commit := m.repository.Graph.Commits[m.GetSelectedCommit()]
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
	idx := m.GetSelectedCommit()
	commit := m.repository.Graph.Commits[idx]
	m.selectionMode = SelectionRebaseDestination
	m.rebaseSourceCommit = idx
	m.statusMessage = fmt.Sprintf("Select destination for rebasing %s (Esc to cancel)", commit.ShortID)
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

func (m *Model) startEditingDescription(commit internal.Commit) (tea.Model, tea.Cmd) {
	m.viewMode = ViewEditDescription
	m.graphTabModel.SetEditingCommitID(commit.ChangeID)
	descInput := m.graphTabModel.GetDescriptionInput()
	descInput.SetWidth(m.width - 10)
	descInput.SetHeight(m.height - 12)
	m.graphTabModel.SetDescriptionInput(*descInput)
	m.statusMessage = fmt.Sprintf("Loading description for %s...", commit.ShortID)
	return m, m.loadFullDescription(commit.ChangeID)
}

func (m *Model) loadFullDescription(commitID string) tea.Cmd {
	return actions.LoadDescription(m.jjService, commitID)
}

func (m *Model) saveDescription() tea.Cmd {
	return actions.SaveDescription(m.jjService, m.graphTabModel.GetEditingCommitID(), strings.TrimSpace(m.graphTabModel.GetDescriptionInput().Value()))
}

// Bookmark actions

func (m *Model) startCreateBookmark() {
	if !m.isSelectedCommitValid() {
		m.statusMessage = "No commit selected"
		return
	}

	idx := m.GetSelectedCommit()
	commit := m.repository.Graph.Commits[idx]
	m.bookmarkModal.Show(idx, actions.GetExistingBookmarks(m.repository, idx))
	ni := m.bookmarkModal.GetNameInput()
	ni.SetValue("")
	ni.Focus()
	ni.Width = m.width - 10

	m.viewMode = ViewCreateBookmark
	m.statusMessage = fmt.Sprintf("Create or move bookmark on %s", commit.ShortID)
}

func (m *Model) submitBookmark() tea.Cmd {
	if m.bookmarkModal.IsFromJira() {
		return m.submitBookmarkFromJira()
	}

	commitIdx := m.bookmarkModal.GetCommitIdx()
	if m.repository == nil || commitIdx < 0 || commitIdx >= len(m.repository.Graph.Commits) {
		return nil
	}
	commit := m.repository.Graph.Commits[commitIdx]
	commitID := commit.ChangeID

	existingBookmarks := m.bookmarkModal.GetNameInput() // modal doesn't expose slice; we need GetExistingBookmarks from modal
	_ = existingBookmarks
	selIdx := m.bookmarkModal.GetNameInput() // modal needs GetSelectedBookmarkIdx
	_ = selIdx
	// Use modal's state via new getters if we add them
	if m.bookmarkModal.GetSelectedBookmarkIdx() >= 0 {
		// Moving existing bookmark - need existing bookmarks list from modal
		// Bookmark modal has existingBookmarks set in Show(); we need GetExistingBookmarks() on modal
		return nil
	}

	bookmarkName := strings.TrimSpace(m.bookmarkModal.GetBookmarkName())

	// Sanitize bookmark name if setting is enabled
	if m.settingsTabModel.GetSettingsSanitizeBookmarks() {
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
	bookmarkName := strings.TrimSpace(m.bookmarkModal.GetBookmarkName())

	if m.settingsTabModel.GetSettingsSanitizeBookmarks() {
		bookmarkName = jj.SanitizeBookmarkName(bookmarkName)
	}

	if err := actions.ValidateBookmarkName(bookmarkName); err != "" {
		m.statusMessage = err
		return nil
	}

	m.statusMessage = fmt.Sprintf("Creating branch '%s' from main...", bookmarkName)

	title := m.bookmarkModal.GetJiraTicketTitle()
	key := m.bookmarkModal.GetJiraKey()
	displayKey := m.bookmarkModal.GetTicketDisplayKey()
	if title != "" && key != "" {
		keyForTitle := key
		if displayKey != "" {
			keyForTitle = displayKey
		}
		titles := m.bookmarkModal.GetJiraBookmarkTitles()
		if titles == nil {
			titles = make(map[string]string)
		}
		titles[bookmarkName] = keyForTitle + " - " + title
		m.bookmarkModal.SetJiraBookmarkTitles(titles)
	}
	if displayKey != "" {
		keys := m.bookmarkModal.GetTicketBookmarkDisplayKeys()
		if keys == nil {
			keys = make(map[string]string)
		}
		keys[bookmarkName] = displayKey
		m.bookmarkModal.SetTicketBookmarkDisplayKeys(keys)
	}

	ticketKey := m.bookmarkModal.GetJiraKey()
	m.bookmarkModal.ClearJiraContext()

	return actions.CreateBranchFromMain(m.jjService, bookmarkName, ticketKey)
}

func (m *Model) deleteBookmark() tea.Cmd {
	if !m.isSelectedCommitValid() {
		m.statusMessage = "No commit selected"
		return nil
	}

	commit := m.repository.Graph.Commits[m.GetSelectedCommit()]
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

	idx := m.GetSelectedCommit()
	commit := m.repository.Graph.Commits[idx]
	var headBranch string
	var needsMoveBookmark bool

	if len(commit.Branches) > 0 {
		headBranch = commit.Branches[0]
		needsMoveBookmark = false
	} else {
		headBranch = m.findBookmarkForCommit(idx)
		if headBranch == "" {
			m.statusMessage = "No bookmark found. Create one first with 'b'."
			return
		}
		needsMoveBookmark = true
	}

	m.prFormModal.Show(idx, "main", headBranch)
	m.prFormModal.SetNeedsMoveBookmark(needsMoveBookmark)

	defaultTitle := headBranch
	if titles := m.bookmarkModal.GetJiraBookmarkTitles(); titles != nil {
		if jiraPRTitle, ok := titles[headBranch]; ok {
			defaultTitle = jiraPRTitle
		}
	}
	m.prFormModal.SetTitle(defaultTitle)
	m.prFormModal.GetTitleInput().Focus()
	m.prFormModal.SetBody("")
	m.prFormModal.GetBodyInput().Blur()
	m.prFormModal.GetTitleInput().Width = m.width - 10
	m.prFormModal.GetBodyInput().SetWidth(m.width - 10)
	bodyHeight := m.height - 20
	bodyHeight = min(max(bodyHeight, 3), 8)
	m.prFormModal.GetBodyInput().SetHeight(bodyHeight)

	m.viewMode = ViewCreatePR
	if needsMoveBookmark {
		m.statusMessage = fmt.Sprintf("Creating PR for %s (will move bookmark)", headBranch)
	} else {
		m.statusMessage = "Creating PR for " + headBranch
	}
}

func (m *Model) submitPR() tea.Cmd {
	title := strings.TrimSpace(m.prFormModal.GetTitle())
	if title == "" {
		m.statusMessage = "Title is required"
		return nil
	}

	// In demo mode, return a fake PR created message
	if m.demoMode {
		m.statusMessage = "Creating PR (demo)..."
		body := strings.TrimSpace(m.prFormModal.GetBody())
		headBranch := m.prFormModal.GetHeadBranch()
		baseBranch := m.prFormModal.GetBaseBranch()

		var commitIDs []string
		if m.repository != nil {
			idx := m.prFormModal.GetCommitIndex()
			if idx >= 0 && idx < len(m.repository.Graph.Commits) {
				commit := m.repository.Graph.Commits[idx]
				commitIDs = []string{commit.ID}
			}
		}

		return func() tea.Msg {
			return actions.PRCreatedMsg{PR: &internal.GitHubPR{
				Number:       999,
				Title:        title,
				Body:         body,
				State:        "open",
				HeadBranch:   headBranch,
				BaseBranch:   baseBranch,
				URL:          "https://github.com/example/repo/pull/999",
				CommitIDs:    commitIDs,
				CheckStatus:  internal.CheckStatusPending,
				ReviewStatus: internal.ReviewStatusNone,
			}}
		}
	}

	m.statusMessage = fmt.Sprintf("%s %s and creating PR...", If(m.prFormModal.NeedsMoveBookmark(), "Moving bookmark", "Pushing"), m.prFormModal.GetHeadBranch())

	var commitChangeID string
	if m.prFormModal.NeedsMoveBookmark() && m.repository != nil {
		idx := m.prFormModal.GetCommitIndex()
		if idx >= 0 && idx < len(m.repository.Graph.Commits) {
			commitChangeID = m.repository.Graph.Commits[idx].ChangeID
		}
	}

	return actions.CreatePR(m.jjService, m.githubService, actions.PRCreateParams{
		Title:             title,
		Body:              strings.TrimSpace(m.prFormModal.GetBody()),
		HeadBranch:        m.prFormModal.GetHeadBranch(),
		BaseBranch:        m.prFormModal.GetBaseBranch(),
		NeedsMoveBookmark: m.prFormModal.NeedsMoveBookmark(),
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
	inputs := m.settingsTabModel.GetSettingsInputs()
	params := actions.SettingsParams{
		GitHubToken:          strings.TrimSpace(inputs[0].Value()),
		JiraURL:              strings.TrimSpace(inputs[1].Value()),
		JiraUser:             strings.TrimSpace(inputs[2].Value()),
		JiraToken:            strings.TrimSpace(inputs[3].Value()),
		JiraProject:          strings.TrimSpace(inputs[4].Value()),
		JiraJQL:              strings.TrimSpace(inputs[5].Value()),
		JiraExcludedStatuses: strings.TrimSpace(inputs[6].Value()),
		TicketProvider:       m.settingsTabModel.GetSettingsTicketProvider(),
		ShowMerged:           m.settingsTabModel.GetSettingsShowMerged(),
		ShowClosed:           m.settingsTabModel.GetSettingsShowClosed(),
		OnlyMine:             m.settingsTabModel.GetSettingsOnlyMine(),
		PRLimit:              m.settingsTabModel.GetSettingsPRLimit(),
		PRRefreshInterval:    m.settingsTabModel.GetSettingsPRRefreshInterval(),
		AutoInProgress:       m.settingsTabModel.GetSettingsAutoInProgress(),
		BranchLimit:          m.settingsTabModel.GetSettingsBranchLimit(),
		SanitizeBookmarks:    m.settingsTabModel.GetSettingsSanitizeBookmarks(),
	}
	if len(inputs) > 10 {
		params.CodecksSubdomain = strings.TrimSpace(inputs[7].Value())
		params.CodecksToken = strings.TrimSpace(inputs[8].Value())
		params.CodecksProject = strings.TrimSpace(inputs[9].Value())
		params.CodecksExcludedStatuses = strings.TrimSpace(inputs[10].Value())
	}
	if len(inputs) > 11 {
		params.GitHubIssuesExcludedStatuses = strings.TrimSpace(inputs[11].Value())
	}
	// Pass GitHub repo info for GitHub Issues provider
	if m.githubService != nil {
		params.GitHubOwner = m.githubService.GetOwner()
		params.GitHubRepo = m.githubService.GetRepo()
	}
	return actions.SaveSettings(params)
}

func (m *Model) saveSettingsLocal() tea.Cmd {
	inputs := m.settingsTabModel.GetSettingsInputs()
	params := actions.SettingsParams{
		GitHubToken:          strings.TrimSpace(inputs[0].Value()),
		JiraURL:              strings.TrimSpace(inputs[1].Value()),
		JiraUser:             strings.TrimSpace(inputs[2].Value()),
		JiraToken:            strings.TrimSpace(inputs[3].Value()),
		JiraProject:          strings.TrimSpace(inputs[4].Value()),
		JiraJQL:              strings.TrimSpace(inputs[5].Value()),
		JiraExcludedStatuses: strings.TrimSpace(inputs[6].Value()),
		TicketProvider:       m.settingsTabModel.GetSettingsTicketProvider(),
		ShowMerged:           m.settingsTabModel.GetSettingsShowMerged(),
		ShowClosed:           m.settingsTabModel.GetSettingsShowClosed(),
		OnlyMine:             m.settingsTabModel.GetSettingsOnlyMine(),
		PRLimit:              m.settingsTabModel.GetSettingsPRLimit(),
		PRRefreshInterval:    m.settingsTabModel.GetSettingsPRRefreshInterval(),
		AutoInProgress:       m.settingsTabModel.GetSettingsAutoInProgress(),
		BranchLimit:          m.settingsTabModel.GetSettingsBranchLimit(),
		SanitizeBookmarks:    m.settingsTabModel.GetSettingsSanitizeBookmarks(),
	}
	if len(inputs) > 10 {
		params.CodecksSubdomain = strings.TrimSpace(inputs[7].Value())
		params.CodecksToken = strings.TrimSpace(inputs[8].Value())
		params.CodecksProject = strings.TrimSpace(inputs[9].Value())
		params.CodecksExcludedStatuses = strings.TrimSpace(inputs[10].Value())
	}
	if len(inputs) > 11 {
		params.GitHubIssuesExcludedStatuses = strings.TrimSpace(inputs[11].Value())
	}
	// Pass GitHub repo info for GitHub Issues provider
	if m.githubService != nil {
		params.GitHubOwner = m.githubService.GetOwner()
		params.GitHubRepo = m.githubService.GetRepo()
	}
	return actions.SaveSettingsLocal(params)
}
