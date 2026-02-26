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
	ChangeID   string
	CommitIDs  []string
	Summaries  []string
}

// HandleDivergentCommitInfoMsg mutates app when err; otherwise returns info for main to show the modal.
func HandleDivergentCommitInfoMsg(msg graphtab.DivergentCommitInfoMsg, app *state.AppState) (tea.Cmd, *ShowDivergentInfo) {
	if msg.Err != nil {
		app.StatusMessage = fmt.Sprintf("Error loading divergent info: %v", msg.Err)
		app.ViewMode = state.ViewCommitGraph
		return nil, nil
	}
	return nil, &ShowDivergentInfo{
		ChangeID:  msg.ChangeID,
		CommitIDs: msg.CommitIDs,
		Summaries: msg.Summaries,
	}
}

// HandleDivergentCommitResolvedMsg mutates app (StatusMessage, ViewMode) and returns the Cmd to run.
func HandleDivergentCommitResolvedMsg(msg DivergentCommitResolvedMsg, app *state.AppState) tea.Cmd {
	if msg.Err != nil {
		app.StatusMessage = fmt.Sprintf("Error resolving divergent commit: %v", msg.Err)
		app.ViewMode = state.ViewCommitGraph
		return nil
	}
	app.StatusMessage = fmt.Sprintf("Divergent commit resolved (kept %s)", msg.KeptCommitID)
	app.ViewMode = state.ViewCommitGraph
	return data.LoadRepository(app.JJService)
}
