package tickets

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// Callbacks are provided by the main model to run ticket actions.
type Callbacks struct {
	OpenInBrowser     func(url string) tea.Cmd
	TransitionTicket  func(transitionID string) tea.Cmd
}

// ExecuteResult holds the result of ExecuteRequest.
// When NeedToggleMode or NeedStartBookmark is true, the model should run the corresponding handler (no async cmd).
type ExecuteResult struct {
	Cmd              tea.Cmd
	StatusMsg         string
	NeedToggleMode   bool
	NeedStartBookmark bool
	TransitionStatus  string // status message when running a transition (e.g. "Setting X to Y...")
}

// ExecuteRequest validates the request and runs the appropriate callback or returns sync actions.
func ExecuteRequest(r Request, ctx *RequestContext, cb *Callbacks) ExecuteResult {
	if ctx == nil {
		return ExecuteResult{}
	}

	if r.ToggleStatusChangeMode {
		if ctx.TicketService == nil || ctx.TransitionInProgress {
			return ExecuteResult{}
		}
		return ExecuteResult{NeedToggleMode: true}
	}
	if r.StartBookmarkFromTicket {
		if !ctx.SelectedTicketValid() || ctx.TicketService == nil {
			return ExecuteResult{}
		}
		return ExecuteResult{NeedStartBookmark: true}
	}
	if r.OpenInBrowser {
		if ctx.TicketService == nil || !ctx.SelectedTicketValid() {
			return ExecuteResult{}
		}
		ticket := ctx.SelectedTicketData()
		if ticket == nil {
			return ExecuteResult{}
		}
		url := ctx.TicketService.GetTicketURL(*ticket)
		if url == "" {
			return ExecuteResult{}
		}
		if cb != nil && cb.OpenInBrowser != nil {
			return ExecuteResult{Cmd: cb.OpenInBrowser(url)}
		}
		return ExecuteResult{}
	}
	if r.TransitionID != "" {
		if ctx.TicketService == nil || ctx.TransitionInProgress {
			return ExecuteResult{}
		}
		if !ctx.SelectedTicketValid() {
			return ExecuteResult{}
		}
		transitionName := ctx.TransitionName(r.TransitionID)
		ticket := ctx.SelectedTicketData()
		if ticket == nil {
			return ExecuteResult{}
		}
		statusMsg := fmt.Sprintf("Setting %s to %s...", ticket.DisplayKey, transitionName)
		if cb != nil && cb.TransitionTicket != nil {
			return ExecuteResult{Cmd: cb.TransitionTicket(r.TransitionID), TransitionStatus: statusMsg}
		}
		return ExecuteResult{}
	}
	return ExecuteResult{}
}
