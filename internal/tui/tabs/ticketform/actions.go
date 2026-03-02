package ticketform

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/tickets"
	"github.com/madicen/jj-tui/internal/tui/util"
)

// OpenCreateTicketResult is the result of opening the Create Ticket dialog
type OpenCreateTicketResult struct {
	StatusMessage string
	Ok            bool
}

// OpenCreateTicket prepares and shows the Create Ticket dialog when the provider supports it.
func OpenCreateTicket(modal *Model, ticketService tickets.Service, width, height int) OpenCreateTicketResult {
	if ticketService == nil || !ticketService.CanCreateTicket() {
		return OpenCreateTicketResult{StatusMessage: "Create ticket not available for this provider", Ok: false}
	}
	providerName := ticketService.GetProviderName()
	modal.Show(providerName)
	modal.GetTitleInput().Width = width
	modal.GetBodyInput().SetWidth(width)
	const fixedFormLines = 10
	bodyHeight := height - fixedFormLines
	if bodyHeight < 3 {
		bodyHeight = 3
	}
	modal.GetBodyInput().SetHeight(bodyHeight)
	return OpenCreateTicketResult{
		StatusMessage: fmt.Sprintf("Creating %s ticket", providerName),
		Ok:            true,
	}
}

// SubmitTicketInput contains everything needed to submit the create-ticket request
type SubmitTicketInput struct {
	Summary        string
	Description    string
	TicketService  tickets.Service
	DemoMode       bool
}

// SubmitTicketCmd validates and runs CreateTicket; returns (cmd, validationError).
func SubmitTicketCmd(input SubmitTicketInput) (tea.Cmd, string) {
	summary := strings.TrimSpace(input.Summary)
	if summary == "" {
		return nil, "Title is required"
	}
	if input.TicketService == nil {
		return nil, "No ticket provider configured"
	}
	if !input.TicketService.CanCreateTicket() {
		return nil, "This provider does not support creating tickets"
	}
	if input.DemoMode {
		return func() tea.Msg {
			ticket, err := input.TicketService.CreateTicket(context.Background(), &tickets.CreateTicketInput{
				Summary:     summary,
				Description: strings.TrimSpace(input.Description),
			})
			if err != nil {
				return util.ErrorMsg{Err: err}
			}
			return TicketCreatedMsg{Ticket: ticket}
		}, ""
	}
	svc := input.TicketService
	return func() tea.Msg {
		ticket, err := svc.CreateTicket(context.Background(), &tickets.CreateTicketInput{
			Summary:     summary,
			Description: strings.TrimSpace(input.Description),
		})
		if err != nil {
			return util.ErrorMsg{Err: err}
		}
		return TicketCreatedMsg{Ticket: ticket}
	}, ""
}

// SubmitTicketResult is the result of SubmitTicket
type SubmitTicketResult struct {
	Cmd           tea.Cmd
	StatusMessage string
}

// SubmitTicket builds input from modal and runs the create command
func SubmitTicket(modal *Model, ticketService tickets.Service, demoMode bool) SubmitTicketResult {
	input := modal.CreateTicketInput()
	cmd, errStr := SubmitTicketCmd(SubmitTicketInput{
		Summary:       input.Summary,
		Description:   input.Description,
		TicketService: ticketService,
		DemoMode:      demoMode,
	})
	if errStr != "" {
		return SubmitTicketResult{StatusMessage: errStr}
	}
	providerName := "ticket"
	if ticketService != nil {
		providerName = ticketService.GetProviderName()
	}
	return SubmitTicketResult{
		Cmd:           cmd,
		StatusMessage: fmt.Sprintf("Creating %s ticket...", providerName),
	}
}

// HandleTicketCreatedMsg updates app state and returns optional cmd (e.g. reload tickets, open URL)
func HandleTicketCreatedMsg(ticket *tickets.Ticket, ticketService tickets.Service, demoMode bool) tea.Cmd {
	if ticket == nil {
		return nil
	}
	if !demoMode && ticketService != nil {
		url := ticketService.GetTicketURL(*ticket)
		if url != "" {
			return util.OpenURL(url)
		}
	}
	return nil
}
