package prs

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/integrations/github"
)

// BranchPushedMsg indicates a branch was pushed.
type BranchPushedMsg struct {
	Branch     string
	PushOutput string
}

// PrsLoadedMsg is sent when PRs have been loaded (or load failed with LoadErrorMsg).
type PrsLoadedMsg struct {
	Prs []internal.GitHubPR
}

// PrMergedMsg is sent when a PR merge completes.
type PrMergedMsg struct {
	PRNumber int
	Err      error
}

// PrClosedMsg is sent when a PR close completes.
type PrClosedMsg struct {
	PRNumber int
	Err      error
}

// LoadErrorMsg is sent when loading PRs fails (main shows error modal).
type LoadErrorMsg struct {
	Err error
}

// ReauthNeededMsg is sent when GitHub auth expired (main starts login flow).
type ReauthNeededMsg struct {
	Reason string
}

// PrTickMsg is sent on the PR refresh interval to trigger reload.
type PrTickMsg time.Time

// Request is sent to the main model to run PR actions (main has githubService, openURL, etc.).
type Request struct {
	OpenInBrowser bool
	MergePR       bool
	ClosePR       bool
}

// Cmd returns a tea.Cmd that sends this request.
func (r Request) Cmd() tea.Cmd {
	return func() tea.Msg { return r }
}

// Effect types: the tab sends these to the main model so it can update app state and status.

// ApplyPrsLoadedEffect tells main to set repository PRs and status (sent after PrsLoadedMsg).
// If Prs is nil, main keeps existing PRs and may set status to "PRs: N" when no error.
type ApplyPrsLoadedEffect struct {
	Prs           []internal.GitHubPR
	StatusMessage string
}

// ApplyPrsLoadErrorEffect tells main to show the load error in the error modal.
type ApplyPrsLoadErrorEffect struct {
	Err error
}

// ApplyPrMergeClosedEffect tells main to set status/error after merge or close, and optionally reload PRs.
type ApplyPrMergeClosedEffect struct {
	Err           error
	StatusMessage string
}

// ApplyReauthNeededEffect tells main to clear GitHub and start login flow.
type ApplyReauthNeededEffect struct {
	Reason string
}

// ApplyPrTickEffect carries the cmd to run for PR tick (reload + next tick or just next tick).
type ApplyPrTickEffect struct {
	RunCmd tea.Cmd
}

func (e ApplyPrsLoadedEffect) Cmd() tea.Cmd {
	return func() tea.Msg { return e }
}

func (e ApplyPrsLoadErrorEffect) Cmd() tea.Cmd {
	return func() tea.Msg { return e }
}

func (e ApplyPrMergeClosedEffect) Cmd() tea.Cmd {
	return func() tea.Msg { return e }
}

func (e ApplyReauthNeededEffect) Cmd() tea.Cmd {
	return func() tea.Msg { return e }
}

func (e ApplyPrTickEffect) Cmd() tea.Cmd {
	return func() tea.Msg { return e }
}

// PrTickInput is the context main sends when forwarding a PR tick so the tab can decide and build cmds.
type PrTickInput struct {
	IsPRView      bool
	Loading       bool
	HasError      bool
	GitHubService *github.Service
	GithubInfo    string
	DemoMode      bool
	ExistingCount int
}

// OpenPRURLEffect tells main to open the PR URL in the browser.
type OpenPRURLEffect struct {
	URL string
}

// OpenPRURLEffectCmd returns a cmd that sends OpenPRURLEffect to main.
func OpenPRURLEffectCmd(url string) tea.Cmd {
	return func() tea.Msg { return OpenPRURLEffect{URL: url} }
}
