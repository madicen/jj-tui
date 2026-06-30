package prs

import (
	"context"
	"fmt"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/integrations/github"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/mock"
	"github.com/madicen/jj-tui/internal/tui/util"
)

// LoadPRsCmd returns a command that fetches PRs and sends PrsLoadedMsg, ReauthNeededMsg, or LoadErrorMsg.
// existingPRsCount: when demoMode and > 0, send nil Prs to keep existing. githubInfo is used in error text.
func LoadPRsCmd(ghSvc *github.Service, githubInfo string, demoMode bool, existingPRsCount int) tea.Cmd {
	if demoMode {
		if existingPRsCount > 0 {
			return func() tea.Msg { return PrsLoadedMsg{Prs: nil} }
		}
		return func() tea.Msg { return PrsLoadedMsg{Prs: mock.DemoPullRequests()} }
	}
	if ghSvc == nil {
		return func() tea.Msg { return PrsLoadedMsg{Prs: []internal.GitHubPR{}} }
	}
	svc := ghSvc
	info := githubInfo
	return func() tea.Msg {
		cfg, _ := config.Load()
		filterOpts := github.PRFilterOptions{
			Limit:      100,
			ShowMerged: true,
			ShowClosed: true,
			OnlyMine:   false,
		}
		if cfg != nil {
			filterOpts.ShowMerged = cfg.ShowMergedPRs()
			filterOpts.ShowClosed = cfg.ShowClosedPRs()
			filterOpts.OnlyMine = cfg.OnlyMyPRs()
			filterOpts.Limit = cfg.PRLimit()
		}
		prs, err := svc.GetPullRequestsWithOptions(context.Background(), filterOpts)
		if err != nil {
			if github.IsAuthError(err) {
				cfg, _ := config.Load()
				if cfg != nil && (cfg.UsedDeviceFlow() || cfg.UsedGhCLIAuth()) {
					return ReauthNeededMsg{Reason: "Your GitHub authorization has expired. Please reauthorize to continue."}
				}
			}
			errMsg := fmt.Sprintf("failed to load PRs: %v", err)
			if info != "" {
				errMsg += fmt.Sprintf(" [%s]", info)
			}
			return LoadErrorMsg{Err: fmt.Errorf("%s", errMsg)}
		}
		return PrsLoadedMsg{Prs: prs}
	}
}

// ResolveOpenPRsForBookmarksCmd looks up the open PR for each bookmark via a targeted GitHub query
// and sends OpenPRsResolvedMsg with the ones found. This guarantees the graph can detect an existing
// PR for a local bookmark even when the bulk PR list omitted it (busy repos can have thousands of PRs,
// so a newest-first capped fetch can push an older still-open PR out of the result).
func ResolveOpenPRsForBookmarksCmd(ghSvc *github.Service, bookmarks []string, demoMode bool) tea.Cmd {
	if ghSvc == nil || demoMode || len(bookmarks) == 0 {
		return nil
	}
	svc := ghSvc
	names := append([]string(nil), bookmarks...)
	return func() tea.Msg {
		ctx := context.Background()
		// Each lookup is an independent GitHub round-trip. Running them serially (up to 25) was the
		// dominant cost of loading the PR tab, especially when GitHub throttled the burst. Fan them
		// out with a bounded worker pool so the total latency is roughly one round-trip, not 25.
		const maxConcurrent = 8
		results := make([]*internal.GitHubPR, len(names))
		sem := make(chan struct{}, maxConcurrent)
		var wg sync.WaitGroup
		for i, name := range names {
			wg.Add(1)
			sem <- struct{}{}
			go func(i int, name string) {
				defer wg.Done()
				defer func() { <-sem }()
				pr, err := svc.GetOpenPRForBranch(ctx, name)
				if err == nil && pr != nil {
					results[i] = pr
				}
			}(i, name)
		}
		wg.Wait()

		// Preserve the input order so the merged result is deterministic.
		var found []internal.GitHubPR
		for _, pr := range results {
			if pr != nil {
				found = append(found, *pr)
			}
		}
		return OpenPRsResolvedMsg{Prs: found}
	}
}

// MergePRCmd returns a command that merges the PR and sends PrMergedMsg.
func MergePRCmd(ghSvc *github.Service, prNumber int, demoMode bool) tea.Cmd {
	if demoMode {
		return func() tea.Msg { return PrMergedMsg{PRNumber: prNumber, Err: nil} }
	}
	if ghSvc == nil {
		return nil
	}
	svc := ghSvc
	return func() tea.Msg {
		err := svc.MergePullRequest(context.Background(), prNumber)
		return PrMergedMsg{PRNumber: prNumber, Err: err}
	}
}

// ClosePRCmd returns a command that closes the PR and sends PrClosedMsg.
func ClosePRCmd(ghSvc *github.Service, prNumber int, demoMode bool) tea.Cmd {
	if demoMode {
		return func() tea.Msg { return PrClosedMsg{PRNumber: prNumber, Err: nil} }
	}
	if ghSvc == nil {
		return nil
	}
	svc := ghSvc
	return func() tea.Msg {
		err := svc.ClosePullRequest(context.Background(), prNumber)
		return PrClosedMsg{PRNumber: prNumber, Err: err}
	}
}

// PrTickCmd returns a command that sends PrTickMsg after the configured PR refresh interval, or nil if disabled.
func PrTickCmd() tea.Cmd {
	cfg, _ := config.Load()
	interval := 120
	if cfg != nil {
		interval = cfg.PRRefreshInterval()
	}
	if interval <= 0 {
		return nil
	}
	return tea.Tick(time.Duration(interval)*time.Second, func(t time.Time) tea.Msg {
		return PrTickMsg(t)
	})
}

// PushToPRCmd pushes updates to a PR branch (optionally moving the bookmark first).
func PushToPRCmd(svc *jj.Service, branch, commitID string, moveBookmark bool, demoMode bool) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if moveBookmark {
			if err := svc.MoveBookmark(ctx, branch, commitID); err != nil {
				return util.ErrorMsg{Err: fmt.Errorf("failed to move bookmark %s: %w", branch, err)}
			}
		}
		if demoMode {
			time.Sleep(1 * time.Second)
		}
		pushOutput, err := svc.PushToGit(ctx, branch)
		if err != nil {
			return util.ErrorMsg{Err: fmt.Errorf("failed to push: %w\nOutput: %s%s", err, pushOutput, util.MissingOriginHint(err))}
		}
		return BranchPushedMsg{Branch: branch, PushOutput: pushOutput}
	}
}

// ExecuteRequest validates the request and returns (statusMsg, cmd). Main sets statusMsg and returns the cmd.
func ExecuteRequest(r Request, ctx *RequestContext) (statusMsg string, cmd tea.Cmd) {
	if ctx == nil {
		return "", nil
	}
	if !ctx.GitHubOK {
		return "GitHub service not initialized", nil
	}
	if !ctx.SelectedPRValid() {
		return "", nil
	}
	pr := ctx.SelectedPRData()
	if pr == nil {
		return "", nil
	}

	if r.OpenInBrowser {
		if pr.URL == "" {
			return "", nil
		}
		if ctx.DemoMode {
			return fmt.Sprintf("PR #%d: %s (demo mode - browser disabled)", pr.Number, pr.URL), nil
		}
		return fmt.Sprintf("Opening PR #%d...", pr.Number), util.OpenURL(pr.URL)
	}
	if r.MergePR {
		if pr.State != "open" {
			return "Can only merge open PRs", nil
		}
		return fmt.Sprintf("Merging PR #%d...", pr.Number), MergePRCmd(ctx.GitHubService, pr.Number, ctx.DemoMode)
	}
	if r.ClosePR {
		if pr.State != "open" {
			return "Can only close open PRs", nil
		}
		return fmt.Sprintf("Closing PR #%d...", pr.Number), ClosePRCmd(ctx.GitHubService, pr.Number, ctx.DemoMode)
	}
	return "", nil
}
