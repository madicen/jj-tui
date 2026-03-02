package settings

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/tui/mouse"
)

// resolveTabFromZone returns the settings tab index (0–5) if zoneID is a tab zone.
func resolveTabFromZone(zoneID string) (tab int, ok bool) {
	switch zoneID {
	case mouse.ZoneSettingsTabGitHub:
		return 0, true
	case mouse.ZoneSettingsTabJira:
		return 1, true
	case mouse.ZoneSettingsTabCodecks:
		return 2, true
	case mouse.ZoneSettingsTabTickets:
		return 3, true
	case mouse.ZoneSettingsTabBranches:
		return 4, true
	case mouse.ZoneSettingsTabAdvanced:
		return 5, true
	}
	return 0, false
}

// handleGitHubZone handles zone clicks for the GitHub panel (tab 0).
func handleGitHubZone(m *Model, zoneID string) (Model, tea.Cmd) {
	if zoneID == mouse.ZoneSettingsGitHubLogin {
		return *m, StartGitHubLoginCmd()
	}
	gh := m.GetGitHubModel()
	switch zoneID {
	case mouse.ZoneSettingsGitHubOnlyMine:
		gh.SetOnlyMine(!gh.GetOnlyMine())
		return *m, nil
	case mouse.ZoneSettingsGitHubShowMerged:
		gh.SetShowMerged(!gh.GetShowMerged())
		return *m, nil
	case mouse.ZoneSettingsGitHubShowClosed:
		gh.SetShowClosed(!gh.GetShowClosed())
		return *m, nil
	case mouse.ZoneSettingsGitHubPRLimitDecrease:
		if gh.GetPRLimit() > 25 {
			gh.SetPRLimit(gh.GetPRLimit() - 25)
		}
		return *m, nil
	case mouse.ZoneSettingsGitHubPRLimitIncrease:
		if gh.GetPRLimit() < 500 {
			gh.SetPRLimit(gh.GetPRLimit() + 25)
		}
		return *m, nil
	case mouse.ZoneSettingsGitHubRefreshDecrease:
		iv := gh.GetRefreshInterval()
		if iv > 30 {
			gh.SetRefreshInterval(iv - 30)
		} else if iv > 0 {
			gh.SetRefreshInterval(0)
		}
		return *m, nil
	case mouse.ZoneSettingsGitHubRefreshIncrease:
		iv := gh.GetRefreshInterval()
		if iv == 0 {
			gh.SetRefreshInterval(30)
		} else if iv < 600 {
			gh.SetRefreshInterval(iv + 30)
		}
		return *m, nil
	case mouse.ZoneSettingsGitHubRefreshToggle:
		if gh.GetRefreshInterval() == 0 {
			gh.SetRefreshInterval(120)
		} else {
			gh.SetRefreshInterval(0)
		}
		return *m, nil
	case mouse.ZoneSettingsGitHubTokenClear:
		gh.SetToken("")
		m.SetFocusedField(0)
		return *m, nil
	case mouse.ZoneSettingsGitHubToken:
		m.SetFocusedField(0)
		return *m, nil
	}
	return *m, nil
}

// handleJiraZone handles zone clicks for the Jira panel (tab 1).
func handleJiraZone(m *Model, zoneID string) (Model, tea.Cmd) {
	jr := m.GetJiraModel()
	clearZones := []string{
		mouse.ZoneSettingsJiraURLClear, mouse.ZoneSettingsJiraUserClear, mouse.ZoneSettingsJiraTokenClear,
		mouse.ZoneSettingsJiraProjectClear, mouse.ZoneSettingsJiraProjectFilterClear, mouse.ZoneSettingsJiraIssueTypeClear, mouse.ZoneSettingsJiraJQLClear, mouse.ZoneSettingsJiraExcludedClear,
	}
	for i, zid := range clearZones {
		if zoneID == zid {
			switch i {
			case 0:
				jr.SetURL("")
			case 1:
				jr.SetUser("")
			case 2:
				jr.SetToken("")
			case 3:
				jr.SetProject("")
			case 4:
				jr.SetProjectFilter("")
			case 5:
				jr.SetIssueType("")
			case 6:
				jr.SetJQL("")
			case 7:
				jr.SetExcludedStatuses("")
			}
			m.SetFocusedField(i + 1)
			return *m, nil
		}
	}
	settingsZones := []string{
		mouse.ZoneSettingsJiraURL, mouse.ZoneSettingsJiraUser, mouse.ZoneSettingsJiraToken,
		mouse.ZoneSettingsJiraProject, mouse.ZoneSettingsJiraProjectFilter, mouse.ZoneSettingsJiraIssueType, mouse.ZoneSettingsJiraJQL, mouse.ZoneSettingsJiraExcluded,
	}
	for i, zid := range settingsZones {
		if zoneID == zid {
			m.SetFocusedField(i + 1)
			return *m, nil
		}
	}
	return *m, nil
}

// handleCodecksZone handles zone clicks for the Codecks panel (tab 2).
func handleCodecksZone(m *Model, zoneID string) (Model, tea.Cmd) {
	cc := m.GetCodecksModel()
	clearZones := []string{
		mouse.ZoneSettingsCodecksSubdomainClear, mouse.ZoneSettingsCodecksTokenClear,
		mouse.ZoneSettingsCodecksProjectClear, mouse.ZoneSettingsCodecksExcludedClear,
	}
	indices := []int{9, 10, 11, 12}
	for i, zid := range clearZones {
		if zoneID == zid {
			switch i {
			case 0:
				cc.SetSubdomain("")
			case 1:
				cc.SetToken("")
			case 2:
				cc.SetProject("")
			case 3:
				cc.SetExcludedStatuses("")
			}
			m.SetFocusedField(indices[i])
			return *m, nil
		}
	}
	settingsZones := []string{
		mouse.ZoneSettingsCodecksSubdomain, mouse.ZoneSettingsCodecksToken,
		mouse.ZoneSettingsCodecksProject, mouse.ZoneSettingsCodecksExcluded,
	}
	for i, zid := range settingsZones {
		if zoneID == zid {
			m.SetFocusedField(indices[i])
			return *m, nil
		}
	}
	return *m, nil
}

// handleTicketsZone handles zone clicks for the Tickets panel (tab 3).
func handleTicketsZone(m *Model, zoneID string) (Model, tea.Cmd) {
	tk := m.GetTicketsModel()
	switch zoneID {
	case mouse.ZoneSettingsTicketProviderNone:
		tk.SetTicketProvider("")
		return *m, nil
	case mouse.ZoneSettingsTicketProviderJira:
		tk.SetTicketProvider("jira")
		return *m, nil
	case mouse.ZoneSettingsTicketProviderCodecks:
		tk.SetTicketProvider("codecks")
		return *m, nil
	case mouse.ZoneSettingsTicketProviderGitHubIssues:
		tk.SetTicketProvider("github_issues")
		return *m, nil
	case mouse.ZoneSettingsAutoInProgress:
		tk.SetAutoInProgress(!tk.GetAutoInProgress())
		return *m, nil
	case mouse.ZoneSettingsGitHubIssuesExcludedClear:
		tk.SetGitHubIssuesExcludedStatuses("")
		tk.SetFocusedField(0)
		return *m, nil
	case mouse.ZoneSettingsGitHubIssuesExcluded:
		m.SetFocusedField(13)
		return *m, nil
	}
	return *m, nil
}

// handleBranchesZone handles zone clicks for the Branches panel (tab 4).
func handleBranchesZone(m *Model, zoneID string) (Model, tea.Cmd) {
	br := m.GetBranchesModel()
	switch zoneID {
	case mouse.ZoneSettingsBranchLimitDecrease:
		n := br.GetBranchLimit()
		if n > 10 {
			br.SetBranchLimit(n - 10)
		} else if n > 0 {
			br.SetBranchLimit(0)
		}
		return *m, nil
	case mouse.ZoneSettingsBranchLimitIncrease:
		n := br.GetBranchLimit()
		if n == 0 {
			br.SetBranchLimit(10)
		} else if n < 200 {
			br.SetBranchLimit(n + 10)
		}
		return *m, nil
	}
	return *m, nil
}

// handleAdvancedZone handles zone clicks for the Advanced panel (tab 5).
func handleAdvancedZone(m *Model, zoneID string) (Model, tea.Cmd) {
	adv := m.GetAdvancedModel()
	if adv.GetConfirmingCleanup() != "" {
		switch zoneID {
		case mouse.ZoneSettingsAdvancedConfirmYes:
			adv.SetConfirmingCleanup("")
			return *m, RequestConfirmCleanupCmd()
		case mouse.ZoneSettingsAdvancedConfirmNo:
			adv.SetConfirmingCleanup("")
			return *m, RequestCancelCleanupCmd()
		}
		return *m, nil
	}
	switch zoneID {
	case mouse.ZoneSettingsAdvancedDeleteBookmarks:
		adv.SetConfirmingCleanup("delete_bookmarks")
		return *m, RequestSetStatusCmd(StartDeleteBookmarksStatus)
	case mouse.ZoneSettingsAdvancedAbandonOldCommits:
		adv.SetConfirmingCleanup("abandon_old_commits")
		return *m, RequestSetStatusCmd(StartAbandonOldCommitsStatus)
	case mouse.ZoneSettingsSanitizeBookmarks:
		adv.SetSanitizeBookmarks(!adv.GetSanitizeBookmarks())
		return *m, nil
	case mouse.ZoneSettingsGraphRevset:
		return *m, m.SetFocusedField(14)
	case mouse.ZoneSettingsGraphRevsetClear:
		adv.SetGraphRevset("")
		return *m, m.SetFocusedField(14)
	}
	return *m, nil
}

// routeZoneToPanel routes a zone click to the active panel or tab switch or save/cancel.
func (m *Model) routeZoneToPanel(zoneID string) (Model, tea.Cmd) {
	if tab, ok := resolveTabFromZone(zoneID); ok {
		m.SetSettingsTab(tab)
		if tab == 5 {
			return *m, m.advancedModel.SetFocusedField(0)
		}
		return *m, nil
	}
	switch zoneID {
	case mouse.ZoneSettingsSave:
		return *m, Request{SaveSettings: true}.Cmd()
	case mouse.ZoneSettingsSaveLocal:
		return *m, Request{SaveSettingsLocal: true}.Cmd()
	case mouse.ZoneSettingsCancel:
		return *m, PerformCancelCmd()
	}
	switch m.settingsTab {
	case 0:
		return handleGitHubZone(m, zoneID)
	case 1:
		return handleJiraZone(m, zoneID)
	case 2:
		return handleCodecksZone(m, zoneID)
	case 3:
		return handleTicketsZone(m, zoneID)
	case 4:
		return handleBranchesZone(m, zoneID)
	case 5:
		return handleAdvancedZone(m, zoneID)
	}
	return *m, nil
}
