package keys

import "github.com/charmbracelet/bubbles/key"

// ScopeTickets is the config prefix for tickets tab shortcuts.
const ScopeTickets = "tickets"

// TicketsKeyMap holds the tickets tab's keybindings.
type TicketsKeyMap struct {
	MoveDown         key.Binding
	MoveUp           key.Binding
	ChangeStatus     key.Binding
	StatusInProgress key.Binding
	StatusDone       key.Binding
	StatusBlocked    key.Binding
	StatusNotStarted key.Binding
	Open             key.Binding
	NewTicket        key.Binding
	CreateBranch     key.Binding
}

// DefaultTicketsKeyMap returns the tickets bindings, applying any config overrides.
func DefaultTicketsKeyMap(overrides map[string]string) TicketsKeyMap {
	return TicketsKeyMap{
		MoveDown:         bind(overrides, "tickets.move_down", "j/↓", "Move down", "j", "down"),
		MoveUp:           bind(overrides, "tickets.move_up", "k/↑", "Move up", "k", "up"),
		ChangeStatus:     bind(overrides, "tickets.change_status", "c", "Change ticket status", "c"),
		StatusInProgress: bind(overrides, "tickets.status_in_progress", "i", "Set status: In Progress", "i"),
		StatusDone:       bind(overrides, "tickets.status_done", "D", "Set status: Done", "D"),
		StatusBlocked:    bind(overrides, "tickets.status_blocked", "B", "Set status: Blocked", "B"),
		StatusNotStarted: bind(overrides, "tickets.status_not_started", "N", "Set status: Not Started", "N"),
		Open:             bind(overrides, "tickets.open", "o", "Open ticket in browser", "o"),
		NewTicket:        bind(overrides, "tickets.new_ticket", "n", "Create new ticket", "n"),
		CreateBranch:     bind(overrides, "tickets.create_branch", "Enter", "Create branch from ticket", "enter", "e"),
	}
}

func (k TicketsKeyMap) entries() []entry {
	return []entry{
		{"tickets.move_down", k.MoveDown},
		{"tickets.move_up", k.MoveUp},
		{"tickets.change_status", k.ChangeStatus},
		{"tickets.status_in_progress", k.StatusInProgress},
		{"tickets.status_done", k.StatusDone},
		{"tickets.status_blocked", k.StatusBlocked},
		{"tickets.status_not_started", k.StatusNotStarted},
		{"tickets.open", k.Open},
		{"tickets.new_ticket", k.NewTicket},
		{"tickets.create_branch", k.CreateBranch},
	}
}
