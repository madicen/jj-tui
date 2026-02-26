package tickets

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/config"
	ticketdomain "github.com/madicen/jj-tui/internal/tickets"
)

// LoadTicketsCmd returns a command that fetches tickets and sends TicketsLoadedMsg or LoadErrorMsg.
// Pass nil svc to send empty list; demoMode skips status filtering.
func LoadTicketsCmd(svc ticketdomain.Service, demoMode bool) tea.Cmd {
	if svc == nil {
		return func() tea.Msg { return TicketsLoadedMsg{Tickets: []ticketdomain.Ticket{}} }
	}
	service := svc
	return func() tea.Msg {
		ticketList, err := service.GetAssignedTickets(context.Background())
		if err != nil {
			return LoadErrorMsg{Err: fmt.Errorf("failed to load tickets: %w", err)}
		}
		if !demoMode {
			cfg, _ := config.Load()
			if cfg != nil {
				excludedStatuses := make(map[string]bool)
				var excludedStr string
				switch service.GetProviderName() {
				case "Jira":
					excludedStr = cfg.JiraExcludedStatuses
				case "Codecks":
					excludedStr = cfg.CodecksExcludedStatuses
				case "GitHub Issues":
					excludedStr = cfg.GitHubIssuesExcludedStatuses
				}
				if excludedStr != "" {
					for status := range strings.SplitSeq(excludedStr, ",") {
						status = strings.TrimSpace(strings.ToLower(status))
						if status != "" {
							excludedStatuses[status] = true
						}
					}
				}
				if len(excludedStatuses) > 0 {
					var filtered []ticketdomain.Ticket
					for _, ticket := range ticketList {
						statusLower := strings.ToLower(ticket.Status)
						if !excludedStatuses[statusLower] {
							filtered = append(filtered, ticket)
						}
					}
					ticketList = filtered
				}
			}
		}
		return TicketsLoadedMsg{Tickets: ticketList}
	}
}

// LoadTransitionsCmd returns a command that loads transitions for the selected ticket and sends TransitionsLoadedMsg.
func LoadTransitionsCmd(svc ticketdomain.Service, ticketList []ticketdomain.Ticket, selectedIdx int) tea.Cmd {
	if svc == nil || selectedIdx < 0 || selectedIdx >= len(ticketList) {
		return func() tea.Msg { return TransitionsLoadedMsg{Transitions: nil} }
	}
	service := svc
	ticket := ticketList[selectedIdx]
	return func() tea.Msg {
		transitions, err := service.GetAvailableTransitions(context.Background(), ticket.Key)
		if err != nil {
			return TransitionsLoadedMsg{Transitions: nil}
		}
		return TransitionsLoadedMsg{Transitions: transitions}
	}
}

// TransitionTicketCmd returns a command that runs the transition and sends TransitionCompletedMsg.
func TransitionTicketCmd(svc ticketdomain.Service, ticketKey, transitionID string) tea.Cmd {
	if svc == nil {
		return nil
	}
	service := svc
	return func() tea.Msg {
		err := service.TransitionTicket(context.Background(), ticketKey, transitionID)
		if err != nil {
			return TransitionCompletedMsg{TicketKey: ticketKey, Err: err}
		}
		transitions, _ := service.GetAvailableTransitions(context.Background(), ticketKey)
		var newStatus string
		for _, t := range transitions {
			if t.ID == transitionID {
				newStatus = t.Name
				break
			}
		}
		if newStatus == "" {
			newStatus = transitionID
		}
		return TransitionCompletedMsg{TicketKey: ticketKey, NewStatus: newStatus}
	}
}

// TransitionTicketToInProgressCmd returns a command that finds an "in progress" transition and runs it.
func TransitionTicketToInProgressCmd(svc ticketdomain.Service, ticketKey string) tea.Cmd {
	if svc == nil {
		return nil
	}
	service := svc
	return func() tea.Msg {
		transitions, err := service.GetAvailableTransitions(context.Background(), ticketKey)
		if err != nil {
			return TransitionCompletedMsg{TicketKey: ticketKey, Err: err}
		}
		var inProgressID string
		for _, t := range transitions {
			lowerName := strings.ToLower(t.Name)
			isInProgress := strings.Contains(lowerName, "progress") ||
				(strings.Contains(lowerName, "start") && !strings.Contains(lowerName, "not start") && !strings.Contains(lowerName, "not_start"))
			if isInProgress {
				inProgressID = t.ID
				break
			}
		}
		if inProgressID == "" {
			return TransitionCompletedMsg{TicketKey: ticketKey, NewStatus: ""}
		}
		err = service.TransitionTicket(context.Background(), ticketKey, inProgressID)
		if err != nil {
			return TransitionCompletedMsg{TicketKey: ticketKey, Err: err}
		}
		return TransitionCompletedMsg{TicketKey: ticketKey, NewStatus: "In Progress"}
	}
}

// ExecuteRequest validates the request and returns (statusMsg, cmd). Main sets statusMsg and returns the cmd.
func ExecuteRequest(r Request, ctx *RequestContext) (statusMsg string, cmd tea.Cmd) {
	if ctx == nil {
		return "", nil
	}

	if r.ToggleStatusChangeMode {
		if ctx.TicketService == nil || ctx.TransitionInProgress {
			return "", nil
		}
		status := "Ready"
		if !ctx.IsStatusChangeMode {
			status = "Change status (i/D/B/N)"
		}
		return "", ToggleModeEffectCmd(status)
	}
	if r.StartBookmarkFromTicket {
		if !ctx.SelectedTicketValid() || ctx.TicketService == nil {
			return "", nil
		}
		ticket := ctx.SelectedTicketData()
		if ticket == nil {
			return "", nil
		}
		return "", OpenCreateBookmarkFromTicketEffect{
			TicketKey:  ticket.Key,
			Title:     ticket.Summary,
			DisplayKey: ticket.DisplayKey,
		}.Cmd()
	}
	if r.OpenInBrowser {
		if ctx.TicketService == nil || !ctx.SelectedTicketValid() {
			return "", nil
		}
		ticket := ctx.SelectedTicketData()
		if ticket == nil {
			return "", nil
		}
		url := ctx.TicketService.GetTicketURL(*ticket)
		if url == "" {
			return "", nil
		}
		return "", OpenURLEffectCmd(url)
	}
	if r.TransitionID != "" {
		if ctx.TicketService == nil || ctx.TransitionInProgress {
			return "", nil
		}
		if !ctx.SelectedTicketValid() {
			return "", nil
		}
		transitionName := ctx.TransitionName(r.TransitionID)
		ticket := ctx.SelectedTicketData()
		if ticket == nil {
			return "", nil
		}
		statusMsg := fmt.Sprintf("Setting %s to %s...", ticket.DisplayKey, transitionName)
		return statusMsg, TransitionTicketCmd(ctx.TicketService, ticket.Key, r.TransitionID)
	}
	return "", nil
}
