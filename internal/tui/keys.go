package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// handleKeyMsg handles keyboard input
func (m *Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Special handling for edit description view
	if m.viewMode == ViewEditDescription {
		return m.handleDescriptionEditKeyMsg(msg)
	}

	// Special handling for settings view
	if m.viewMode == ViewSettings {
		return m.handleSettingsKeyMsg(msg)
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "g":
		m.viewMode = ViewCommitGraph
	case "p":
		m.viewMode = ViewPullRequests
		// Load PRs when switching to PR view
		if m.githubService != nil {
			m.statusMessage = "Loading PRs..."
			return m, m.loadPRs()
		}
	case "i": // 'i' for issues (Jira) since 'j' is used for down navigation
		m.viewMode = ViewJira
		// Load tickets when switching to Jira view
		if m.jiraService != nil {
			m.statusMessage = "Loading Jira tickets..."
			return m, m.loadJiraTickets()
		}
	case ",": // ',' for settings (like many apps use comma for settings)
		m.viewMode = ViewSettings
		// Focus first input when entering settings
		m.settingsFocusedField = 0
		for i := range m.settingsInputs {
			if i == 0 {
				m.settingsInputs[i].Focus()
			} else {
				m.settingsInputs[i].Blur()
			}
		}
	case "h", "?":
		m.viewMode = ViewHelp
	case "n":
		if m.viewMode == ViewCommitGraph && m.jjService != nil {
			// Create a new commit
			m.statusMessage = "Creating new commit..."
			return m, m.createNewCommit()
		}
	case "d":
		// Edit description of selected commit
		if m.viewMode == ViewCommitGraph && m.selectedCommit >= 0 && m.jjService != nil && m.repository != nil {
			commit := m.repository.Graph.Commits[m.selectedCommit]
			if commit.Immutable {
				m.statusMessage = "Cannot edit description: commit is immutable"
				return m, nil
			}
			return m.startEditingDescription(commit)
		}
	case "r":
		m.statusMessage = "Refreshing..."
		m.loading = true
		return m, m.loadRepository()
	case "esc":
		if m.viewMode != ViewCommitGraph {
			m.viewMode = ViewCommitGraph
		}
	case "j", "down":
		if m.viewMode == ViewPullRequests {
			if m.repository != nil && m.selectedPR < len(m.repository.PRs)-1 {
				m.selectedPR++
			}
		} else if m.viewMode == ViewJira {
			if m.selectedTicket < len(m.jiraTickets)-1 {
				m.selectedTicket++
			}
		} else {
			if m.repository != nil && m.selectedCommit < len(m.repository.Graph.Commits)-1 {
				m.selectedCommit++
			}
		}
	case "k", "up":
		if m.viewMode == ViewPullRequests {
			if m.selectedPR > 0 {
				m.selectedPR--
			}
		} else if m.viewMode == ViewJira {
			if m.selectedTicket > 0 {
				m.selectedTicket--
			}
		} else {
			if m.selectedCommit > 0 {
				m.selectedCommit--
			}
		}
	case "o":
		// Open PR URL in browser (PR view only)
		if m.viewMode == ViewPullRequests && m.repository != nil && m.selectedPR >= 0 && m.selectedPR < len(m.repository.PRs) {
			pr := m.repository.PRs[m.selectedPR]
			if pr.URL != "" {
				m.statusMessage = fmt.Sprintf("Opening PR #%d...", pr.Number)
				return m, openURL(pr.URL)
			}
		}
	case "enter", "e":
		// In PR view, open the PR in browser
		if m.viewMode == ViewPullRequests && m.repository != nil && m.selectedPR >= 0 && m.selectedPR < len(m.repository.PRs) {
			pr := m.repository.PRs[m.selectedPR]
			if pr.URL != "" {
				m.statusMessage = fmt.Sprintf("Opening PR #%d...", pr.Number)
				return m, openURL(pr.URL)
			}
		}
		// In Jira view, create branch from ticket
		if m.viewMode == ViewJira && m.selectedTicket >= 0 && m.selectedTicket < len(m.jiraTickets) && m.jjService != nil {
			ticket := m.jiraTickets[m.selectedTicket]
			m.statusMessage = fmt.Sprintf("Creating branch from %s...", ticket.Key)
			return m, m.createBranchFromTicket(ticket)
		}
		// In commit view, edit selected commit (jj edit)
		if m.viewMode == ViewCommitGraph && m.selectedCommit >= 0 && m.jjService != nil && m.repository != nil {
			commit := m.repository.Graph.Commits[m.selectedCommit]
			if commit.Immutable {
				m.statusMessage = "Cannot edit: commit is immutable"
				return m, nil
			}
			return m, m.checkoutCommit()
		}
	case "s":
		// Squash selected commit
		if m.selectedCommit >= 0 && m.jjService != nil && m.repository != nil {
			commit := m.repository.Graph.Commits[m.selectedCommit]
			if commit.Immutable {
				m.statusMessage = "Cannot squash: commit is immutable"
				return m, nil
			}
			return m, m.squashCommit()
		}
	case "a":
		// Abandon selected commit
		if m.viewMode == ViewCommitGraph && m.selectedCommit >= 0 && m.jjService != nil && m.repository != nil {
			commit := m.repository.Graph.Commits[m.selectedCommit]
			if commit.Immutable {
				m.statusMessage = "Cannot abandon: commit is immutable"
				return m, nil
			}
			return m, m.abandonCommit()
		}
	}
	return m, nil
}

// handleDescriptionEditKeyMsg handles keys while editing description
func (m *Model) handleDescriptionEditKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel editing
		m.viewMode = ViewCommitGraph
		m.editingCommitID = ""
		m.statusMessage = "Description edit cancelled"
		return m, nil
	case "ctrl+s":
		// Save the description
		return m, m.saveDescription()
	}

	// Pass other keys to the textarea
	var cmd tea.Cmd
	m.descriptionInput, cmd = m.descriptionInput.Update(msg)
	return m, cmd
}

// handleSettingsKeyMsg handles keys while in settings view
func (m *Model) handleSettingsKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel and go back
		m.viewMode = ViewCommitGraph
		m.statusMessage = "Settings cancelled"
		return m, nil
	case "ctrl+s", "enter":
		// If on a field and press enter, move to next field
		// If on last field, save
		if msg.String() == "enter" && m.settingsFocusedField < len(m.settingsInputs)-1 {
			m.settingsFocusedField++
			for i := range m.settingsInputs {
				if i == m.settingsFocusedField {
					m.settingsInputs[i].Focus()
				} else {
					m.settingsInputs[i].Blur()
				}
			}
			return m, nil
		}
		// Save settings
		return m, m.saveSettings()
	case "tab", "down":
		// Move to next field
		m.settingsFocusedField = (m.settingsFocusedField + 1) % len(m.settingsInputs)
		for i := range m.settingsInputs {
			if i == m.settingsFocusedField {
				m.settingsInputs[i].Focus()
			} else {
				m.settingsInputs[i].Blur()
			}
		}
		return m, nil
	case "shift+tab", "up":
		// Move to previous field
		m.settingsFocusedField--
		if m.settingsFocusedField < 0 {
			m.settingsFocusedField = len(m.settingsInputs) - 1
		}
		for i := range m.settingsInputs {
			if i == m.settingsFocusedField {
				m.settingsInputs[i].Focus()
			} else {
				m.settingsInputs[i].Blur()
			}
		}
		return m, nil
	}

	// Pass other keys to the focused input
	var cmd tea.Cmd
	m.settingsInputs[m.settingsFocusedField], cmd = m.settingsInputs[m.settingsFocusedField].Update(msg)
	return m, cmd
}

