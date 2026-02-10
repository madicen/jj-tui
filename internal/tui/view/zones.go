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

	ZoneJiraCreateBranch    = "zone:jira:createbranch"
	ZoneJiraOpenBrowser     = "zone:jira:openbrowser"
	ZoneJiraSetInProgress   = "zone:jira:setinprogress"
	ZoneJiraSetDone         = "zone:jira:setdone"
	ZoneJiraChangeStatus    = "zone:jira:changestatus"  // Toggle status change mode
	ZoneJiraTransition      = "zone:jira:transition:"   // Prefix for dynamic transitions
	ZonePROpenBrowser = "zone:pr:openbrowser"
	ZonePRMerge       = "zone:pr:merge"
	ZonePRClose       = "zone:pr:close"

	// Settings sub-tabs
	ZoneSettingsTabGitHub    = "zone:settings:tab:github"
	ZoneSettingsTabJira      = "zone:settings:tab:jira"
	ZoneSettingsTabCodecks   = "zone:settings:tab:codecks"
	ZoneSettingsTabBranches  = "zone:settings:tab:branches"
	ZoneSettingsTabAdvanced  = "zone:settings:tab:advanced"

	// Branch settings zones
	ZoneSettingsBranchLimitDecrease = "zone:settings:branch_limit_decrease"
	ZoneSettingsBranchLimitIncrease = "zone:settings:branch_limit_increase"

	ZoneSettingsGitHubToken           = "zone:settings:github_token"
	ZoneSettingsGitHubTokenClear      = "zone:settings:github_token_clear"
	ZoneSettingsGitHubLogin           = "zone:settings:github_login"
	ZoneSettingsGitHubShowMerged      = "zone:settings:github_show_merged"
	ZoneSettingsGitHubShowClosed      = "zone:settings:github_show_closed"
	ZoneSettingsGitHubOnlyMine        = "zone:settings:github_only_mine"
	ZoneSettingsGitHubPRLimitDecrease       = "zone:settings:github_pr_limit_decrease"
	ZoneSettingsGitHubPRLimitIncrease       = "zone:settings:github_pr_limit_increase"
	ZoneSettingsGitHubRefreshDecrease       = "zone:settings:github_refresh_decrease"
	ZoneSettingsGitHubRefreshIncrease       = "zone:settings:github_refresh_increase"
	ZoneSettingsGitHubRefreshToggle         = "zone:settings:github_refresh_toggle"
	ZoneSettingsJiraExcluded                = "zone:settings:jira_excluded"
	ZoneSettingsJiraExcludedClear     = "zone:settings:jira_excluded_clear"
	ZoneSettingsCodecksExcluded       = "zone:settings:codecks_excluded"
	ZoneSettingsCodecksExcludedClear  = "zone:settings:codecks_excluded_clear"
	ZoneSettingsJiraURL               = "zone:settings:jira_url"
	ZoneSettingsJiraURLClear          = "zone:settings:jira_url_clear"
	ZoneSettingsJiraUser              = "zone:settings:jira_user"
	ZoneSettingsJiraUserClear         = "zone:settings:jira_user_clear"
	ZoneSettingsJiraToken             = "zone:settings:jira_token"
	ZoneSettingsJiraTokenClear        = "zone:settings:jira_token_clear"
	ZoneSettingsCodecksSubdomain      = "zone:settings:codecks_subdomain"
	ZoneSettingsCodecksSubdomainClear = "zone:settings:codecks_subdomain_clear"
	ZoneSettingsCodecksToken          = "zone:settings:codecks_token"
	ZoneSettingsCodecksTokenClear     = "zone:settings:codecks_token_clear"
	ZoneSettingsCodecksProject        = "zone:settings:codecks_project"
	ZoneSettingsCodecksProjectClear   = "zone:settings:codecks_project_clear"

	// Advanced/Maintenance operations
	ZoneSettingsAdvancedDeleteBookmarks   = "zone:settings:advanced:delete_bookmarks"
	ZoneSettingsAdvancedAbandonOldCommits = "zone:settings:advanced:abandon_old_commits"
	ZoneSettingsAdvancedConfirmYes        = "zone:settings:advanced:confirm_yes"
	ZoneSettingsAdvancedConfirmNo         = "zone:settings:advanced:confirm_no"
	ZoneSettingsAutoInProgress            = "zone:settings:auto_in_progress"
	ZoneSettingsSanitizeBookmarks         = "zone:settings:sanitize_bookmarks"

	ZoneSettingsSave      = "zone:settings:save"
	ZoneSettingsSaveLocal = "zone:settings:save_local"
	ZoneSettingsCancel    = "zone:settings:cancel"

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

	// Graph view pane zones (for click-to-focus)
	ZoneGraphPane = "zone:graph:pane"
	ZoneFilesPane = "zone:files:pane"

	// Changed file action zones
	ZoneActionMoveFileUp   = "zone:action:movefileup"
	ZoneActionMoveFileDown = "zone:action:movefiledown"
	ZoneActionRevertFile   = "zone:action:revertfile"

	// Branch action zones
	ZoneBranchTrack   = "zone:branch:track"
	ZoneBranchUntrack = "zone:branch:untrack"
	ZoneBranchRestore = "zone:branch:restore"
	ZoneBranchDelete  = "zone:branch:delete"
	ZoneBranchPush    = "zone:branch:push"
	ZoneBranchFetch   = "zone:branch:fetch"
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

// ZoneChangedFile returns the zone ID for a changed file at the given index
func ZoneChangedFile(index int) string {
	return fmt.Sprintf("zone:file:%d", index)
}

// ZoneBranch returns the zone ID for a branch at the given index
func ZoneBranch(index int) string {
	return fmt.Sprintf("zone:branch:%d", index)
}
