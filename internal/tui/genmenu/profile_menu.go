package genmenu

import (
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/jj-tui/internal/config"
)

// ProfileMenu embeds the long-press AI profile picker glue shared by the four
// generate-bearing form tabs (descedit, prform, bookmark, ticketform). Bind a
// generate-chip zoneID at construction; supply onSelect when handling mouse.
type ProfileMenu struct {
	State
	profiles      []config.AIProfile
	activeProfile string
	zoneID        string
}

// NewProfileMenu returns a menu bound to the given generate-chip zone id.
func NewProfileMenu(zoneID string) ProfileMenu {
	return ProfileMenu{zoneID: zoneID}
}

// SetAIProfiles updates the profile list and the active-profile mark.
func (p *ProfileMenu) SetAIProfiles(profiles []config.AIProfile, activeProfile string) {
	if p == nil {
		return
	}
	p.profiles = profiles
	p.activeProfile = activeProfile
}

// MenuState returns a pointer to the underlying long-press State.
func (p *ProfileMenu) MenuState() *State {
	if p == nil {
		return nil
	}
	return &p.State
}

// Overlay returns the rendered popover (empty when hidden) and its (x, y) anchor.
func (p *ProfileMenu) Overlay(zm *zone.Manager) (string, int, int) {
	if p == nil || !p.IsShown() {
		return "", 0, 0
	}
	view := Render(zm, p.profiles, p.activeProfile, p.HoverIndex())
	x, y := p.MouseAnchor()
	return view, x, y
}

// HandleMouse drives BeginPress / OnMotion / UpdateHover / HitTestRelease.
// canArm gates arming (bookmark passes false when the Generate chip is inactive).
// onSelect receives the chosen profile name on row release; may be nil.
func (p *ProfileMenu) HandleMouse(zm *zone.Manager, msg tea.MouseMsg, canArm bool, onSelect func(string) tea.Cmd) tea.Cmd {
	if p == nil || zm == nil {
		return nil
	}
	if p.IsShown() {
		switch msg.Action {
		case tea.MouseActionMotion, tea.MouseActionPress:
			p.UpdateHover(zm, msg, len(p.profiles))
			return nil
		case tea.MouseActionRelease:
			if msg.Button != tea.MouseButtonLeft {
				return nil
			}
			idx := p.HitTestRelease(zm, msg, len(p.profiles))
			if idx >= 0 && idx < len(p.profiles) && onSelect != nil {
				return onSelect(p.profiles[idx].Name)
			}
			return nil
		}
		return nil
	}
	switch msg.Action {
	case tea.MouseActionPress:
		if msg.Button != tea.MouseButtonLeft || !canArm {
			return nil
		}
		z := zm.Get(p.zoneID)
		if z != nil && z.InBounds(msg) && len(p.profiles) > 0 {
			return p.BeginPress(p.zoneID, msg)
		}
	case tea.MouseActionMotion:
		p.OnMotion(zm, msg)
	}
	return nil
}
