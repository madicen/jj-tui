package tui

import (
	"context"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen-utilities/jj-tui/v2/internal/jira"
	"github.com/madicen-utilities/jj-tui/v2/internal/models"
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

// saveSettings saves the settings and reinitializes services
func (m *Model) saveSettings() tea.Cmd {
	// Get values from inputs
	githubToken := strings.TrimSpace(m.settingsInputs[0].Value())
	jiraURL := strings.TrimSpace(m.settingsInputs[1].Value())
	jiraUser := strings.TrimSpace(m.settingsInputs[2].Value())
	jiraToken := strings.TrimSpace(m.settingsInputs[3].Value())

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

		var githubConnected, jiraConnected bool

		// Try to initialize GitHub service
		if githubToken != "" {
			// GitHub service needs owner/repo info, so we'll check if token is set
			githubConnected = true
		}

		// Try to initialize Jira service
		if jiraURL != "" && jiraUser != "" && jiraToken != "" {
			jiraConnected = jira.IsConfigured()
		}

		return settingsSavedMsg{
			githubConnected: githubConnected,
			jiraConnected:   jiraConnected,
		}
	}
}

