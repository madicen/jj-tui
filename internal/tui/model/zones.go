package model

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
	ZoneActionNewCommit    = "zone:action:newcommit"
	ZoneActionCopyError    = "zone:action:copyerror"
	ZoneActionDismissError = "zone:action:dismisserror"

	// Commit action zones
	ZoneActionCheckout = "zone:action:checkout"
	ZoneActionEdit     = "zone:action:edit"
	ZoneActionDescribe = "zone:action:describe"
	ZoneActionSquash   = "zone:action:squash"
	ZoneActionRebase   = "zone:action:rebase"
	ZoneActionAbandon  = "zone:action:abandon"

	// Description editor zones
	ZoneDescSave   = "zone:desc:save"
	ZoneDescCancel = "zone:desc:cancel"

	// PR creation zones
	ZonePRTitle        = "zone:pr:title"
	ZonePRBody         = "zone:pr:body"
	ZonePRSubmit       = "zone:pr:submit"
	ZonePRCancel       = "zone:pr:cancel"
	ZoneActionCreatePR = "zone:action:createpr"

	// Bookmark creation zones
	ZoneBookmarkName       = "zone:bookmark:name"
	ZoneBookmarkSubmit     = "zone:bookmark:submit"
	ZoneBookmarkCancel     = "zone:bookmark:cancel"
	ZoneActionBookmark     = "zone:action:bookmark"
	ZoneActionDelBookmark  = "zone:action:delbookmark"

	// Push action zone
	ZoneActionPush = "zone:action:push"

	// Init button zone (shown when not in a jj repo)
	ZoneActionJJInit = "zone:action:jj_init"

	// Graph view pane zones (for click-to-focus)
	ZoneGraphPane = "zone:graph:pane"
	ZoneFilesPane = "zone:files:pane"

	// Jira action zones
	ZoneJiraCreateBranch = "zone:jira:createbranch"
	ZoneJiraOpenBrowser  = "zone:jira:openbrowser"

	// Settings sub-tab zones
	ZoneSettingsTabGitHub  = "zone:settings:tab:github"
	ZoneSettingsTabJira    = "zone:settings:tab:jira"
	ZoneSettingsTabCodecks = "zone:settings:tab:codecks"

	// Settings zones
	ZoneSettingsGitHubToken      = "zone:settings:github_token"
	ZoneSettingsGitHubShowMerged = "zone:settings:github_show_merged"
	ZoneSettingsGitHubShowClosed = "zone:settings:github_show_closed"
	ZoneSettingsGitHubTokenClear      = "zone:settings:github_token_clear"
	ZoneSettingsGitHubLogin           = "zone:settings:github_login"
	ZoneSettingsJiraURL               = "zone:settings:jira_url"
	ZoneSettingsJiraURLClear          = "zone:settings:jira_url_clear"
	ZoneSettingsJiraUser              = "zone:settings:jira_user"
	ZoneSettingsJiraUserClear         = "zone:settings:jira_user_clear"
	ZoneSettingsJiraToken             = "zone:settings:jira_token"
	ZoneSettingsJiraTokenClear        = "zone:settings:jira_token_clear"
	ZoneSettingsJiraExcluded          = "zone:settings:jira_excluded"
	ZoneSettingsJiraExcludedClear     = "zone:settings:jira_excluded_clear"
	ZoneSettingsCodecksSubdomain      = "zone:settings:codecks_subdomain"
	ZoneSettingsCodecksSubdomainClear = "zone:settings:codecks_subdomain_clear"
	ZoneSettingsCodecksToken          = "zone:settings:codecks_token"
	ZoneSettingsCodecksTokenClear     = "zone:settings:codecks_token_clear"
	ZoneSettingsCodecksProject        = "zone:settings:codecks_project"
	ZoneSettingsCodecksProjectClear   = "zone:settings:codecks_project_clear"
	ZoneSettingsCodecksExcluded       = "zone:settings:codecks_excluded"
	ZoneSettingsCodecksExcludedClear  = "zone:settings:codecks_excluded_clear"
	ZoneSettingsSave                  = "zone:settings:save"
	ZoneSettingsSaveLocal             = "zone:settings:save_local"
	ZoneSettingsCancel                = "zone:settings:cancel"
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

// ZoneExistingBookmark returns the zone ID for an existing bookmark at the given index
func ZoneExistingBookmark(index int) string {
	return fmt.Sprintf("zone:bookmark:existing:%d", index)
}

