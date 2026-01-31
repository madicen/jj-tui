package tui

import "fmt"

// Zone IDs - these are used with bubblezone's Mark() function
// to create clickable regions. The zone system automatically
// tracks positions - we never need to calculate coordinates.

const (
	// Tab zones
	ZoneTabGraph    = "zone:tab:graph"
	ZoneTabPRs      = "zone:tab:prs"
	ZoneTabJira     = "zone:tab:jira"
	ZoneTabSettings = "zone:tab:settings"
	ZoneTabHelp     = "zone:tab:help"

	// Status bar action zones
	ZoneActionQuit      = "zone:action:quit"
	ZoneActionRefresh   = "zone:action:refresh"
	ZoneActionNewCommit = "zone:action:newcommit"

	// Commit action zones
	ZoneActionCheckout = "zone:action:checkout"
	ZoneActionEdit     = "zone:action:edit"
	ZoneActionDescribe = "zone:action:describe"
	ZoneActionSquash   = "zone:action:squash"
	ZoneActionAbandon  = "zone:action:abandon"

	// Description editor zones
	ZoneDescSave   = "zone:desc:save"
	ZoneDescCancel = "zone:desc:cancel"

	// Jira action zones
	ZoneJiraCreateBranch = "zone:jira:createbranch"

	// Settings zones
	ZoneSettingsGitHubToken = "zone:settings:github_token"
	ZoneSettingsJiraURL     = "zone:settings:jira_url"
	ZoneSettingsJiraUser    = "zone:settings:jira_user"
	ZoneSettingsJiraToken   = "zone:settings:jira_token"
	ZoneSettingsSave        = "zone:settings:save"
	ZoneSettingsCancel      = "zone:settings:cancel"
)

// ZoneCommit returns the zone ID for a commit at the given index
func ZoneCommit(index int) string {
	return fmt.Sprintf("zone:commit:%d", index)
}

// ZonePR returns the zone ID for a PR at the given index
func ZonePR(index int) string {
	return fmt.Sprintf("zone:pr:%d", index)
}

// ZoneJiraTicket returns the zone ID for a Jira ticket at the given index
func ZoneJiraTicket(index int) string {
	return fmt.Sprintf("zone:jira:ticket:%d", index)
}

