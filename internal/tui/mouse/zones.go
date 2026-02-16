package mouse

import "fmt"

// Zone IDs - these are used with bubblezone's Mark() function
// to create clickable regions. The zone system automatically
// tracks positions - we never need to calculate coordinates.

const (
	// Tab zones
	ZoneTabGraph    = "zone:tab:graph"
	ZoneTabPRs      = "zone:tab:prs"
	ZoneTabJira     = "zone:tab:jira"
	ZoneTabBranches = "zone:tab:branches"
	ZoneTabSettings = "zone:tab:settings"
	ZoneTabHelp     = "zone:tab:help"

	// Status bar action zones
	ZoneActionQuit         = "zone:action:quit"
	ZoneActionRefresh      = "zone:action:refresh"
	ZoneActionNewCommit    = "zone:action:newcommit"
	ZoneActionCopyError    = "zone:action:copyerror"
	ZoneActionDismissError = "zone:action:dismisserror"
	ZoneActionRetry        = "zone:action:retry"
	ZoneActionUndo         = "zone:action:undo"
	ZoneActionRedo         = "zone:action:redo"

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
	ZoneBookmarkName      = "zone:bookmark:name"
	ZoneBookmarkSubmit    = "zone:bookmark:submit"
	ZoneBookmarkCancel    = "zone:bookmark:cancel"
	ZoneActionBookmark    = "zone:action:bookmark"
	ZoneActionDelBookmark = "zone:action:delbookmark"

	// Push action zone
	ZoneActionUpdatePR = "zone:action:push"

	// Init button zone (shown when not in a jj repo)
	ZoneActionJJInit = "zone:action:jj_init"

	// Warning modal zones
	ZoneWarningGoToCommit = "zone:warning:goto_commit"
	ZoneWarningDismiss    = "zone:warning:dismiss"

	// Graph view pane zones (for click-to-focus)
	ZoneGraphPane = "zone:graph:pane"
	ZoneFilesPane = "zone:files:pane"

	// Changed file action zones
	ZoneActionMoveFileUp   = "zone:action:movefileup"
	ZoneActionMoveFileDown = "zone:action:movefiledown"
	ZoneActionRevertFile   = "zone:action:revertfile"

	// Jira/Ticket action zones
	ZoneJiraCreateBranch  = "zone:jira:createbranch"
	ZoneTicketOpenBrowser = "zone:jira:openbrowser"
	ZoneJiraSetInProgress = "zone:jira:setinprogress"
	ZoneJiraSetDone       = "zone:jira:setdone"
	ZoneJiraChangeStatus  = "zone:jira:changestatus" // Toggle status change mode
	ZoneJiraTransition    = "zone:jira:transition:"  // Prefix for dynamic transitions

	// PR action zones
	ZonePROpenBrowser = "zone:pr:openbrowser"
	ZonePRMerge       = "zone:pr:merge"
	ZonePRClose       = "zone:pr:close"

	// Branch action zones
	ZoneBranchTrack           = "zone:branch:track"
	ZoneBranchUntrack         = "zone:branch:untrack"
	ZoneBranchRestore         = "zone:branch:restore"
	ZoneBranchDelete          = "zone:branch:delete"
	ZoneBranchPush            = "zone:branch:push"
	ZoneBranchFetch           = "zone:branch:fetch"
	ZoneBranchResolveConflict = "zone:branch:resolve_conflict"

	// Settings sub-tab zones
	ZoneSettingsTabGitHub   = "zone:settings:tab:github"
	ZoneSettingsTabJira     = "zone:settings:tab:jira"
	ZoneSettingsTabCodecks  = "zone:settings:tab:codecks"
	ZoneSettingsTabTickets  = "zone:settings:tab:tickets"
	ZoneSettingsTabBranches = "zone:settings:tab:branches"
	ZoneSettingsTabAdvanced = "zone:settings:tab:advanced"

	// Help sub-tab zones
	ZoneHelpTabShortcuts = "zone:help:tab:shortcuts"
	ZoneHelpTabCommands  = "zone:help:tab:commands"
	ZoneHelpCommandCopy  = "zone:help:command:copy:" // Prefix for copy buttons

	// Ticket provider selection zones
	ZoneSettingsTicketProviderNone         = "zone:settings:ticket_provider:none"
	ZoneSettingsTicketProviderJira         = "zone:settings:ticket_provider:jira"
	ZoneSettingsTicketProviderCodecks      = "zone:settings:ticket_provider:codecks"
	ZoneSettingsTicketProviderGitHubIssues = "zone:settings:ticket_provider:github_issues"
	ZoneSettingsGitHubIssuesExcluded       = "zone:settings:github_issues_excluded"
	ZoneSettingsGitHubIssuesExcludedClear  = "zone:settings:github_issues_excluded_clear"

	// Branch settings zones
	ZoneSettingsBranchLimitDecrease = "zone:settings:branch_limit_decrease"
	ZoneSettingsBranchLimitIncrease = "zone:settings:branch_limit_increase"

	// Advanced/Maintenance operations
	ZoneSettingsAdvancedDeleteBookmarks   = "zone:settings:advanced:delete_bookmarks"
	ZoneSettingsAdvancedAbandonOldCommits = "zone:settings:advanced:abandon_old_commits"
	ZoneSettingsAdvancedConfirmYes        = "zone:settings:advanced:confirm_yes"
	ZoneSettingsAdvancedConfirmNo         = "zone:settings:advanced:confirm_no"
	ZoneSettingsAutoInProgress            = "zone:settings:auto_in_progress"
	ZoneSettingsSanitizeBookmarks         = "zone:settings:sanitize_bookmarks"

	// GitHub login zones
	ZoneGitHubLoginCopyCode = "zone:github_login:copy_code"

	// Settings zones
	ZoneSettingsGitHubToken           = "zone:settings:github_token"
	ZoneSettingsGitHubShowMerged      = "zone:settings:github_show_merged"
	ZoneSettingsGitHubShowClosed      = "zone:settings:github_show_closed"
	ZoneSettingsGitHubOnlyMine        = "zone:settings:github_only_mine"
	ZoneSettingsGitHubPRLimitDecrease = "zone:settings:github_pr_limit_decrease"
	ZoneSettingsGitHubPRLimitIncrease = "zone:settings:github_pr_limit_increase"
	ZoneSettingsGitHubRefreshDecrease = "zone:settings:github_refresh_decrease"
	ZoneSettingsGitHubRefreshIncrease = "zone:settings:github_refresh_increase"
	ZoneSettingsGitHubRefreshToggle   = "zone:settings:github_refresh_toggle"
	ZoneSettingsGitHubTokenClear      = "zone:settings:github_token_clear"
	ZoneSettingsGitHubLogin           = "zone:settings:github_login"
	ZoneSettingsJiraURL               = "zone:settings:jira_url"
	ZoneSettingsJiraURLClear          = "zone:settings:jira_url_clear"
	ZoneSettingsJiraUser              = "zone:settings:jira_user"
	ZoneSettingsJiraUserClear         = "zone:settings:jira_user_clear"
	ZoneSettingsJiraToken             = "zone:settings:jira_token"
	ZoneSettingsJiraTokenClear        = "zone:settings:jira_token_clear"
	ZoneSettingsJiraProject           = "zone:settings:jira_project"
	ZoneSettingsJiraProjectClear      = "zone:settings:jira_project_clear"
	ZoneSettingsJiraJQL               = "zone:settings:jira_jql"
	ZoneSettingsJiraJQLClear          = "zone:settings:jira_jql_clear"
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

	// Bookmark conflict resolution zones
	ZoneConflictKeepLocal   = "zone:conflict:keep_local"
	ZoneConflictResetRemote = "zone:conflict:reset_remote"
	ZoneConflictConfirm     = "zone:conflict:confirm"
	ZoneConflictCancel      = "zone:conflict:cancel"

	// Divergent commit resolution zones
	ZoneDivergentConfirm       = "zone:divergent:confirm"
	ZoneDivergentCancel        = "zone:divergent:cancel"
	ZoneActionResolveDivergent = "zone:action:resolve_divergent"

	ZoneJiraOpenBrowser = "zone:jira:openbrowser"

	// Push action zone
	ZoneActionPush = "zone:action:push"

	// Help view zones
	ZoneHelpCommand = "zone:help:command:" // Prefix for command history entries
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

// ZoneDivergentCommit returns the zone ID for a divergent commit option at the given index
func ZoneDivergentCommit(index int) string {
	return fmt.Sprintf("zone:divergent:commit:%d", index)
}
