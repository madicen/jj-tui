package evologsplit

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui/util"
)

// EvologDiffLoadRequestedMsg is sent so main can run LoadEvologSplitDiffCmd with the current JJ service.
type EvologDiffLoadRequestedMsg struct{}

// EvologDiffLoadRequestedCmd schedules EvologDiffLoadRequestedMsg.
func EvologDiffLoadRequestedCmd() tea.Cmd {
	return func() tea.Msg {
		return EvologDiffLoadRequestedMsg{}
	}
}

// EvologSplitDiffLoadedMsg carries jj diff --summary results for the selected evolog base vs tip.
type EvologSplitDiffLoadedMsg struct {
	Seq   int
	Files []jj.ChangedFile
	Err   error
}

// LoadEvologSplitDiffCmd loads changed files between base commit and tip change (async).
func LoadEvologSplitDiffCmd(svc *jj.Service, seq int, fromCommitID, tipChangeID string) tea.Cmd {
	if svc == nil || seq <= 0 {
		return nil
	}
	from := strings.TrimSpace(fromCommitID)
	to := strings.TrimSpace(tipChangeID)
	if from == "" || to == "" {
		return nil
	}
	return func() tea.Msg {
		files, err := svc.DiffChangedFilesFromTo(context.Background(), from, to)
		return EvologSplitDiffLoadedMsg{Seq: seq, Files: files, Err: err}
	}
}

// EvologLoadedMsg is sent when evolog listing finishes (success or failure).
type EvologLoadedMsg struct {
	Entries       []jj.EvologEntry
	Err           error
	ChangeID      string // revision passed to evolog -r
	BookmarkName  string
	TipChangeID   string
	TipCommitHint string // graph commit ID (short); jj resolves full ids internally
}

// LoadEvologCmd runs jj evolog for the change and sends EvologLoadedMsg.
func LoadEvologCmd(svc *jj.Service, bookmarkName string, tip internal.Commit) tea.Cmd {
	if svc == nil {
		return nil
	}
	chID := tip.ChangeID
	hint := tip.ID
	bm := bookmarkName
	return func() tea.Msg {
		entries, err := svc.ListEvolog(context.Background(), chID)
		return EvologLoadedMsg{
			Entries:       entries,
			Err:           err,
			ChangeID:      chID,
			BookmarkName:  bm,
			TipChangeID:   chID,
			TipCommitHint: hint,
		}
	}
}

// EvologSplitCompletedMsg is sent after a successful evolog split + reload.
type EvologSplitCompletedMsg struct {
	Repository *internal.Repository
}

// MoveBookmarkDeltaOntoEvologBaseCmd runs the FAQ-style split using the chosen evolog base.
// bookmarkName may be empty to split the selected change only (no jj bookmark set).
func MoveBookmarkDeltaOntoEvologBaseCmd(svc *jj.Service, bookmarkName, localChangeID, localCommitID, baseCommitID string) tea.Cmd {
	if svc == nil {
		return nil
	}
	return func() tea.Msg {
		if err := svc.MoveBookmarkDeltaOntoEvologBase(context.Background(), bookmarkName, localChangeID, localCommitID, baseCommitID); err != nil {
			return util.ErrorMsg{Err: fmt.Errorf("evolog split: %w", err)}
		}
		repo, err := svc.GetRepository(context.Background(), "")
		if err != nil {
			return util.ErrorMsg{Err: err}
		}
		return EvologSplitCompletedMsg{Repository: repo}
	}
}
