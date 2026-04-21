package divergent

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui/data"
	"github.com/madicen/jj-tui/internal/tui/state"
	graphtab "github.com/madicen/jj-tui/internal/tui/tabs/graph"
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

// HandleDivergentCommitInfoMsg mutates app when err; otherwise returns info for main to show the modal,
// or a resolve command when exactly one head can be discarded (no pointless two-button choice).
func HandleDivergentCommitInfoMsg(msg graphtab.DivergentCommitInfoMsg, app *state.AppState) (tea.Cmd, *ShowDivergentInfo) {
	if msg.Err != nil {
		app.StatusMessage = fmt.Sprintf("Error loading divergent info: %v", msg.Err)
		app.ViewMode = state.ViewCommitGraph
		return nil, nil
	}
	if app.JJService != nil {
		if viable := viableDivergentKeepIndices(msg.Versions); len(viable) == 1 {
			app.StatusMessage = "Resolving divergent change (only one side can be discarded)…"
			return ResolveDivergentCommitCmd(app.JJService, msg.ChangeID, msg.Versions[viable[0]].CommitID), nil
		}
	}
	return nil, &ShowDivergentInfo{
		ChangeID: msg.ChangeID,
		Versions: msg.Versions,
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
	return data.LoadRepository(app.JJService)
}
