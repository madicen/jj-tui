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
	ZoneJiraOpenBrowser  = "zone:jira:openbrowser"

	ZoneSettingsGitHubToken      = "zone:settings:github_token"
	ZoneSettingsGitHubTokenClear = "zone:settings:github_token_clear"
	ZoneSettingsGitHubLogin      = "zone:settings:github_login"
	ZoneSettingsJiraURL          = "zone:settings:jira_url"
	ZoneSettingsJiraURLClear     = "zone:settings:jira_url_clear"
	ZoneSettingsJiraUser         = "zone:settings:jira_user"
	ZoneSettingsJiraUserClear    = "zone:settings:jira_user_clear"
	ZoneSettingsJiraToken        = "zone:settings:jira_token"
	ZoneSettingsJiraTokenClear   = "zone:settings:jira_token_clear"
	ZoneSettingsCodecksSubdomain      = "zone:settings:codecks_subdomain"
	ZoneSettingsCodecksSubdomainClear = "zone:settings:codecks_subdomain_clear"
	ZoneSettingsCodecksToken          = "zone:settings:codecks_token"
	ZoneSettingsCodecksTokenClear     = "zone:settings:codecks_token_clear"
	ZoneSettingsCodecksProject        = "zone:settings:codecks_project"
	ZoneSettingsCodecksProjectClear   = "zone:settings:codecks_project_clear"
	ZoneSettingsSave                  = "zone:settings:save"
	ZoneSettingsCancel                = "zone:settings:cancel"

	ZoneDescSave   = "zone:desc:save"
	ZoneDescCancel = "zone:desc:cancel"

	// PR creation zones
	ZonePRTitle        = "zone:pr:title"
	ZonePRBody         = "zone:pr:body"
	ZonePRSubmit       = "zone:pr:submit"
	ZonePRCancel       = "zone:pr:cancel"
	ZoneActionCreatePR = "zone:action:createpr"

	// Bookmark creation zones
	ZoneBookmarkName      = "zone:bookmark:name"
	ZoneBookmarkSubmit    = "zone:bookmark:submit"
	ZoneBookmarkCancel    = "zone:bookmark:cancel"
	ZoneActionBookmark    = "zone:action:bookmark"
	ZoneActionDelBookmark = "zone:action:delbookmark"

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

