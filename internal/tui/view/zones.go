package view

import "fmt"

// Zone IDs for clickable elements
const (
	ZoneActionNewCommit = "zone:action:newcommit"
	ZoneActionCheckout  = "zone:action:checkout"
	ZoneActionDescribe  = "zone:action:describe"
	ZoneActionSquash    = "zone:action:squash"
	ZoneActionRebase    = "zone:action:rebase"
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

	// PR creation zones
	ZonePRTitle        = "zone:pr:title"
	ZonePRBody         = "zone:pr:body"
	ZonePRSubmit       = "zone:pr:submit"
	ZonePRCancel       = "zone:pr:cancel"
	ZoneActionCreatePR = "zone:action:createpr"

	// Bookmark creation zones
	ZoneBookmarkName   = "zone:bookmark:name"
	ZoneBookmarkSubmit = "zone:bookmark:submit"
	ZoneBookmarkCancel = "zone:bookmark:cancel"
	ZoneActionBookmark = "zone:action:bookmark"

	// Push action zone
	ZoneActionPush = "zone:action:push"
)

// ZoneExistingBookmark returns the zone ID for an existing bookmark at the given index
func ZoneExistingBookmark(index int) string {
	return fmt.Sprintf("zone:bookmark:existing:%d", index)
}

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

