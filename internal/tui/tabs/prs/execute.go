package prs

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// Callbacks are provided by the main model to run PR actions (merge, close, open URL).
// The tab validates and then calls the appropriate callback.
type Callbacks struct {
	MergePR       func(prNumber int) tea.Cmd
	ClosePR       func(prNumber int) tea.Cmd
	OpenInBrowser func(url string) tea.Cmd
}

// ExecuteRequest validates the request and runs the appropriate callback.
// Returns (cmd, "") when valid, or (nil, statusMsg) on validation error.
func ExecuteRequest(r Request, ctx *RequestContext, cb *Callbacks) (tea.Cmd, string) {
	if ctx == nil || cb == nil {
		return nil, ""
	}
	if !ctx.GitHubOK {
		return nil, "GitHub service not initialized"
	}
	if !ctx.SelectedPRValid() {
		return nil, ""
	}
	pr := ctx.SelectedPRData()
	if pr == nil {
		return nil, ""
	}

	if r.OpenInBrowser {
		if pr.URL == "" {
			return nil, ""
		}
		if ctx.DemoMode {
			return nil, fmt.Sprintf("PR #%d: %s (demo mode - browser disabled)", pr.Number, pr.URL)
		}
		if cb.OpenInBrowser != nil {
			return cb.OpenInBrowser(pr.URL), ""
		}
		return nil, ""
	}
	if r.MergePR {
		if pr.State != "open" {
			return nil, "Can only merge open PRs"
		}
		if cb.MergePR != nil {
			return cb.MergePR(pr.Number), ""
		}
		return nil, ""
	}
	if r.ClosePR {
		if pr.State != "open" {
			return nil, "Can only close open PRs"
		}
		if cb.ClosePR != nil {
			return cb.ClosePR(pr.Number), ""
		}
		return nil, ""
	}
	return nil, ""
}
