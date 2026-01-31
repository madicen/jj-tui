package view

import "fmt"

// Zone IDs for clickable elements
const (
	ZoneActionNewCommit = "zone:action:newcommit"
	ZoneActionCheckout  = "zone:action:checkout"
	ZoneActionDescribe  = "zone:action:describe"
	ZoneActionSquash    = "zone:action:squash"
	ZoneActionAbandon   = "zone:action:abandon"

	ZoneJiraCreateBranch = "zone:jira:createbranch"

	ZoneSettingsGitHubToken = "zone:settings:github_token"
	ZoneSettingsJiraURL     = "zone:settings:jira_url"
	ZoneSettingsJiraUser    = "zone:settings:jira_user"
	ZoneSettingsJiraToken   = "zone:settings:jira_token"
	ZoneSettingsSave        = "zone:settings:save"
	ZoneSettingsCancel      = "zone:settings:cancel"

	ZoneDescSave   = "zone:desc:save"
	ZoneDescCancel = "zone:desc:cancel"
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

