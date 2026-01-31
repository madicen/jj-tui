package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
)

// handleZoneClick handles clicks detected by bubblezone
func (m *Model) handleZoneClick(zoneInfo *zone.ZoneInfo) (tea.Model, tea.Cmd) {
	if zoneInfo == nil {
		return m, nil
	}

	// Check tab zones
	if m.zone.Get(ZoneTabGraph) == zoneInfo {
		return m, func() tea.Msg { return TabSelectedMsg{Tab: ViewCommitGraph} }
	}
	if m.zone.Get(ZoneTabPRs) == zoneInfo {
		return m, func() tea.Msg { return TabSelectedMsg{Tab: ViewPullRequests} }
	}
	if m.zone.Get(ZoneTabJira) == zoneInfo {
		m.viewMode = ViewJira
		if m.jiraService != nil {
			m.statusMessage = "Loading Jira tickets..."
			return m, m.loadJiraTickets()
		}
		return m, nil
	}
	if m.zone.Get(ZoneTabSettings) == zoneInfo {
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
		return m, nil
	}
	if m.zone.Get(ZoneTabHelp) == zoneInfo {
		return m, func() tea.Msg { return TabSelectedMsg{Tab: ViewHelp} }
	}

	// Check status bar action zones
	if m.zone.Get(ZoneActionQuit) == zoneInfo {
		return m, tea.Batch(
			func() tea.Msg { return ActionMsg{Action: ActionQuit} },
			tea.Quit,
		)
	}
	if m.zone.Get(ZoneActionRefresh) == zoneInfo {
		m.statusMessage = "Refreshing..."
		m.loading = true
		return m, m.loadRepository()
	}
	if m.zone.Get(ZoneActionNewCommit) == zoneInfo {
		// Create a new commit (same as pressing 'n')
		if m.jjService != nil {
			m.statusMessage = "Creating new commit..."
			return m, m.createNewCommit()
		}
	}

	// Check commit zones
	if m.repository != nil {
		for i := range m.repository.Graph.Commits {
			if m.zone.Get(ZoneCommit(i)) == zoneInfo {
				return m, func() tea.Msg {
					return CommitSelectedMsg{
						Index:    i,
						CommitID: m.repository.Graph.Commits[i].ID,
					}
				}
			}
		}
	}

	// Check commit action zones (only for mutable commits)
	if m.zone.Get(ZoneActionCheckout) == zoneInfo {
		if m.selectedCommit >= 0 && m.jjService != nil && m.repository != nil {
			commit := m.repository.Graph.Commits[m.selectedCommit]
			if commit.Immutable {
				m.statusMessage = "Cannot edit: commit is immutable"
				return m, nil
			}
			return m, m.checkoutCommit()
		}
	}
	if m.zone.Get(ZoneActionSquash) == zoneInfo {
		if m.selectedCommit >= 0 && m.jjService != nil && m.repository != nil {
			commit := m.repository.Graph.Commits[m.selectedCommit]
			if commit.Immutable {
				m.statusMessage = "Cannot squash: commit is immutable"
				return m, nil
			}
			return m, m.squashCommit()
		}
	}
	if m.zone.Get(ZoneActionDescribe) == zoneInfo {
		if m.selectedCommit >= 0 && m.jjService != nil && m.repository != nil {
			commit := m.repository.Graph.Commits[m.selectedCommit]
			if commit.Immutable {
				m.statusMessage = "Cannot edit description: commit is immutable"
				return m, nil
			}
			return m.startEditingDescription(commit)
		}
	}
	if m.zone.Get(ZoneActionAbandon) == zoneInfo {
		if m.selectedCommit >= 0 && m.jjService != nil && m.repository != nil {
			commit := m.repository.Graph.Commits[m.selectedCommit]
			if commit.Immutable {
				m.statusMessage = "Cannot abandon: commit is immutable"
				return m, nil
			}
			return m, m.abandonCommit()
		}
	}

	// Check PR zones
	if m.repository != nil {
		for i := range m.repository.PRs {
			if m.zone.Get(ZonePR(i)) == zoneInfo {
				m.selectedPR = i
				return m, nil
			}
		}
	}

	// Description editor zones
	if m.zone.Get(ZoneDescSave) == zoneInfo {
		if m.viewMode == ViewEditDescription {
			return m, m.saveDescription()
		}
	}
	if m.zone.Get(ZoneDescCancel) == zoneInfo {
		if m.viewMode == ViewEditDescription {
			m.viewMode = ViewCommitGraph
			m.editingCommitID = ""
			m.statusMessage = "Description edit cancelled"
			return m, nil
		}
	}

	// Check Jira ticket zones
	for i := range m.jiraTickets {
		if m.zone.Get(ZoneJiraTicket(i)) == zoneInfo {
			m.selectedTicket = i
			return m, nil
		}
	}

	// Check Jira create branch button
	if m.zone.Get(ZoneJiraCreateBranch) == zoneInfo {
		if m.viewMode == ViewJira && m.selectedTicket >= 0 && m.selectedTicket < len(m.jiraTickets) && m.jjService != nil {
			ticket := m.jiraTickets[m.selectedTicket]
			m.statusMessage = fmt.Sprintf("Creating branch from %s...", ticket.Key)
			return m, m.createBranchFromTicket(ticket)
		}
	}

	// Settings input field clicks
	if m.viewMode == ViewSettings {
		settingsZones := []string{
			ZoneSettingsGitHubToken,
			ZoneSettingsJiraURL,
			ZoneSettingsJiraUser,
			ZoneSettingsJiraToken,
		}
		for i, zoneID := range settingsZones {
			if m.zone.Get(zoneID) == zoneInfo {
				m.settingsFocusedField = i
				for j := range m.settingsInputs {
					if j == i {
						m.settingsInputs[j].Focus()
					} else {
						m.settingsInputs[j].Blur()
					}
				}
				return m, nil
			}
		}

		// Save button
		if m.zone.Get(ZoneSettingsSave) == zoneInfo {
			return m, m.saveSettings()
		}

		// Cancel button
		if m.zone.Get(ZoneSettingsCancel) == zoneInfo {
			m.viewMode = ViewCommitGraph
			m.statusMessage = "Settings cancelled"
			return m, nil
		}
	}

	return m, nil
}

// handleAction handles action messages
func (m *Model) handleAction(action ActionType) (tea.Model, tea.Cmd) {
	switch action {
	case ActionQuit:
		return m, tea.Quit
	case ActionRefresh:
		m.statusMessage = "Refreshing..."
		m.loading = true
		return m, m.loadRepository()
	case ActionNewPR:
		m.viewMode = ViewCreatePR
	case ActionCheckout, ActionEdit:
		if m.selectedCommit >= 0 && m.jjService != nil && m.repository != nil {
			commit := m.repository.Graph.Commits[m.selectedCommit]
			if commit.Immutable {
				m.statusMessage = "Cannot edit: commit is immutable"
				return m, nil
			}
			return m, m.checkoutCommit()
		}
	case ActionSquash:
		if m.selectedCommit >= 0 && m.jjService != nil && m.repository != nil {
			commit := m.repository.Graph.Commits[m.selectedCommit]
			if commit.Immutable {
				m.statusMessage = "Cannot squash: commit is immutable"
				return m, nil
			}
			return m, m.squashCommit()
		}
	case ActionHelp:
		m.viewMode = ViewHelp
	}
	return m, nil
}

