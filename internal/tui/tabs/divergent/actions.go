package divergent

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui/data"
	"github.com/madicen/jj-tui/internal/tui/state"
)

// ResolveDivergentCommitCmd runs jj resolve for the divergent commit and sends DivergentCommitResolvedMsg.
func ResolveDivergentCommitCmd(jjSvc *jj.Service, changeID, keepCommitID string) tea.Cmd {
	if jjSvc == nil {
		return nil
	}
	svc := jjSvc
	return func() tea.Msg {
		err := svc.ResolveDivergentCommit(context.Background(), changeID, keepCommitID)
		return DivergentCommitResolvedMsg{
			ChangeID:     changeID,
			KeptCommitID: keepCommitID,
			Err:          err,
		}
	}
}

// ShowDivergentInfo is returned when the handler wants main to show the divergent modal.
type ShowDivergentInfo struct {
	ChangeID string
	Versions []jj.DivergentVersion
}

// viableDivergentKeepIndices returns indices of revisions that can be kept while abandoning all others.
// A revision can be kept only if every *other* divergent head is mutable (jj will not abandon immutable commits).
func viableDivergentKeepIndices(v []jj.DivergentVersion) []int {
	if len(v) < 2 {
		return nil
	}
	var out []int
	for i := range v {
		ok := true
		for j := range v {
			if i == j {
				continue
			}
			if v[j].Immutable {
				ok = false
				break
			}
		}
		if ok {
			out = append(out, i)
		}
	}
	return out
}

// DivergentCommitInfoInput carries the loaded divergent-commit info from main
// (P2.5): the graph tab's DivergentCommitInfoMsg is destructured by the root so
// this package no longer imports the graph tab for the message type.
type DivergentCommitInfoInput struct {
	ChangeID string
	Versions []jj.DivergentVersion
	Err      error
}

// HandleDivergentCommitInfoMsg mutates app when err; otherwise returns info for main to show the modal,
// or a resolve command when exactly one head can be discarded (no pointless two-button choice).
func HandleDivergentCommitInfoMsg(in DivergentCommitInfoInput, app *state.AppState) (tea.Cmd, *ShowDivergentInfo) {
	if in.Err != nil {
		app.StatusMessage = fmt.Sprintf("Error loading divergent info: %v", in.Err)
		app.ViewMode = state.ViewCommitGraph
		return nil, nil
	}
	if app.JJService != nil {
		if viable := viableDivergentKeepIndices(in.Versions); len(viable) == 1 {
			app.StatusMessage = "Resolving divergent change (only one side can be discarded)…"
			return ResolveDivergentCommitCmd(app.JJService, in.ChangeID, in.Versions[viable[0]].CommitID), nil
		}
	}
	return nil, &ShowDivergentInfo{
		ChangeID: in.ChangeID,
		Versions: in.Versions,
	}
}

// HandleDivergentCommitResolvedMsg mutates app (StatusMessage, ViewMode) and returns the Cmd to run.
func HandleDivergentCommitResolvedMsg(msg DivergentCommitResolvedMsg, app *state.AppState) tea.Cmd {
	if msg.Err != nil {
		app.StatusMessage = fmt.Sprintf("Error resolving divergent commit: %v", msg.Err)
		app.ViewMode = state.ViewCommitGraph
		return nil
	}
	kept := msg.KeptCommitID
	if len(kept) > 14 {
		kept = kept[:12] + "…"
	}
	app.StatusMessage = fmt.Sprintf("Divergent commit resolved (kept %s)", kept)
	app.ViewMode = state.ViewCommitGraph
	return data.LoadRepository(app.JJService, app.GraphFilterRevset)
}
