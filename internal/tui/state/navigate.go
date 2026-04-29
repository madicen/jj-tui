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
	NavigateDismissErrorAndRefresh
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
