package state

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
)

// NavigateKind is the discriminant for a navigation request from a submodel.
type NavigateKind int

const (
	NavigateEditDescription NavigateKind = iota
	NavigateCreateBookmark
	NavigateCreateBookmarkFromTicket
	NavigateWarning
	NavigateCreatePR
	NavigateBackToGraph
	NavigateBackToBranches
	NavigateBackToSettings
	NavigateDismissError
	NavigateDismissInit
	NavigateGitHubLoginCancel
	// Modal "perform" actions: main runs the cmd or applies the effect.
	NavigateSaveDescription
	NavigateSubmitBookmark
	NavigateSubmitPR
	NavigateResolveConflict
	NavigateResolveDivergent
	NavigateWarningCancel
	NavigateRunInit
	// NavigateRetryError is fired by the Retry button on the error modal. Main replays the most
	// recent retry-eligible action (currently: in-flight AI generation kind saved on Model). If
	// nothing is replayable, main falls back to a repository refresh.
	NavigateRetryError
	NavigateBackFromPRForm // back to graph and hide PR form modal
	NavigateCreateTicket   // open Create Ticket modal (from Tickets tab)
	NavigateBackFromTicketForm
	NavigateSubmitTicket          // run create ticket and close modal
	NavigateOpenEvologSplit       // experimental: show evolog picker for stack split
	NavigatePerformEvologSplit    // run jj new + restore (+ bookmark set if EvologBookmarkName set) after user picked base
	NavigateCloseBookmarkConflict // close diverged-bookmark dialog; main restores the tab stored when opening
	NavigateOpenFileDiff          // show full jj diff for one changed file (graph modal) or preloaded raw git text (evolog)
	NavigateCloseFileDiff         // dismiss file diff overlay only (return to evolog split or graph)
	// Optional LLM: modals request generation; main runs tea.Cmd and applies TextGeneratedMsg.
	NavigateGenerateCommitDescription
	NavigateGeneratePRForm
	NavigateGenerateBookmarkName
	NavigateGenerateTicketForm
	// Repository remote management from Settings → GitHub. Apply adds-or-updates origin to the
	// supplied URL, CreateGh runs `gh repo create` (and wires up origin), Remove deletes origin.
	// Main owns the dispatch because it has the jj service and refreshes the repo on success.
	NavigateRemoteApply
	NavigateRemoteCreateGh
	NavigateRemoteRemove
	// NavigatePushBookmarks runs `jj git push --allow-new` against the configured origin —
	// without `--bookmark` for PushAll=false (jj selects the bookmark on @), or with one
	// `--bookmark <name>` per local bookmark for PushAll=true. We avoid `--all-bookmarks`
	// because some currently-supported jj versions reject it. Wired to the Push current /
	// Push all buttons in the Repository remote panel. Separate from the auto-push-after-create
	// flow so users can retry pushes after configuration changes without re-creating the
	// GitHub repo.
	NavigatePushBookmarks
)

// NavigateTarget describes a navigation request. Only main can perform these
// (it owns modals and cross-tab state). Submodels set AppState.ViewMode/StatusMessage
// and return tea.Cmd for everything else.
type NavigateTarget struct {
	Kind NavigateKind

	// Payloads for specific kinds (only one set per kind).
	Commit           internal.Commit
	WarningTitle     string
	WarningMessage   string
	WarningCommits   []internal.Commit
	TicketKey        string
	TicketTitle      string
	TicketDisplayKey string
	StatusMessage    string
	ClearError       bool
	ClearInit        bool
	// Save description
	SaveCommitID    string
	SaveDescription string
	// Resolve conflict
	ConflictBookmarkName string
	ConflictResolution   string
	// Resolve divergent
	DivergentChangeID     string
	DivergentKeepCommitID string
	// Dismiss error and then run refresh
	RefreshAfterDismiss bool
	// Evolog split (experimental)
	EvologBookmarkName  string
	EvologTipChangeID   string
	EvologTipCommitHint string
	EvologBaseCommitID  string
	// EvologDescribeAfterSplit runs AI jj describe after a successful split (@- and @ when @- is mutable, otherwise @ only).
	EvologDescribeAfterSplit bool
	// EvologPrecomputedDescribeParent / Child, when set, skip the post-split describe LLM and apply these strings (from AI suggest chain preview).
	EvologPrecomputedDescribeParent string
	EvologPrecomputedDescribeChild  string
	// EvologFilesetsFirst optional paths for jj split -r @ after FAQ split(s).
	EvologFilesetsFirst []string
	// EvologHunkPeelRounds optional sequence of path→prefix hunk peels after FAQ split(s); each round is one jj split on @.
	EvologHunkPeelRounds []map[string]int
	// EvologMultiBaseCommitIDs optional deepest-first bases for sequential FAQ splits (len>1 triggers multi-split).
	EvologMultiBaseCommitIDs []string
	// EvologStepwiseRemainder, when non-empty, is bases still to run after this split (stepwise mode); main reloads evolog without closing the modal.
	EvologStepwiseRemainder []string
	// Repository remote payload (set on NavigateRemoteApply / NavigateRemoteCreateGh):
	// RemoteURL is the new origin URL for Apply; RemoteRepoName / RemoteRepoPrivate parameterise
	// `gh repo create` for CreateGh. Remove takes no payload.
	RemoteURL         string
	RemoteRepoName    string
	RemoteRepoPrivate bool
	// PushAll is the payload for NavigatePushBookmarks: true => enumerate local bookmarks and
	// pass each as `--bookmark <name>` (portable across jj versions), false => push only the
	// current bookmark (jj's default selection on @).
	PushAll bool

	// AIOverrideProfile, when non-empty, names the AI profile to use for a one-shot
	// NavigateGenerate* call instead of the active profile. Set by the long-press
	// generate-button menu when the user picks a non-active model. Empty = use the
	// active profile (today's behavior). The profile name is resolved against
	// cfg.AIProfiles in handleNavigate.
	AIOverrideProfile string

	// Init-repo screen options: forwarded to data.RunJJInit when the user accepts the welcome
	// screen. Defaults (zero values) reproduce today's behavior of plain `jj git init`.
	InitColocate     bool   // run `jj git init --colocate` instead of plain `jj git init`
	InitRemoteURL    string // when non-empty, add as `origin` after init and run `jj git fetch`
	InitGhCreateRepo bool   // run `gh repo create` after init (requires gh CLI in PATH)
	InitGhRepoName   string // name passed to `gh repo create`; empty -> filepath.Base(cwd)
	InitGhRepoPrivate bool  // visibility for `gh repo create`: true => --private, else --public
	// File diff modal (graph): path relative to repo; Commit holds change id / short id.
	FileDiffPath string
	// When non-empty, NavigateOpenFileDiff shows this git unified diff immediately (no jj call). Used by evolog split.
	FileDiffRawGit          string
	FileDiffOverlayTitle    string // e.g. "Evolog step"; empty => default "File diff"
	FileDiffOverlaySubtitle string // e.g. "abc… → def…"; empty => path @ change id
}

// NavigateMsg is the only callback from submodels to main: they request a view change or
// modal open that only main can perform. Submodels set AppState (ViewMode, StatusMessage)
// and return tea.Cmd for everything else.
type NavigateMsg struct {
	Target NavigateTarget
}

// Cmd returns a tea.Cmd that sends this navigation to the main model.
func (t NavigateTarget) Cmd() tea.Cmd {
	return func() tea.Msg { return NavigateMsg{Target: t} }
}
