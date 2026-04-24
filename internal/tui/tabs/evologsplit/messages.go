package evologsplit

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
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

// EvologSplitSuggestRequestedMsg tells main to run the LLM evolog-split suggestion with current modal entries.
type EvologSplitSuggestRequestedMsg struct {
	ReqID int
}

// OverlaySpinTickMsg advances the AI-suggest overlay spinner. Scheduled via OverlaySpinCmd so animation
// keeps running even when bubbles spinner.TickMsg is dropped or starved on the main message queue.
type OverlaySpinTickMsg struct {
	Time time.Time
}

// OverlaySpinCmd schedules the next overlay spinner tick (same cadence as spinner.MiniDot).
func OverlaySpinCmd() tea.Cmd {
	return tea.Tick(spinner.MiniDot.FPS, func(t time.Time) tea.Msg {
		return OverlaySpinTickMsg{Time: t}
	})
}

// EvologSplitDiffLoadedMsg carries jj diff --summary for selected row vs the newer neighbor above it,
// plus the full git unified diff for the same revision pair (colored in the modal).
type EvologSplitDiffLoadedMsg struct {
	Seq     int
	Files   []jj.ChangedFile
	GitDiff string
	Err     error
}

// LoadEvologSplitDiffCmd loads changed files for one evolog step (async). prevStepFrom/prevStepTo may be
// empty; when both are set, files whose git patch matches the prior step are omitted.
func LoadEvologSplitDiffCmd(svc *jj.Service, seq int, fromCommitID, toCommitID, prevStepFrom, prevStepTo string) tea.Cmd {
	if svc == nil || seq <= 0 {
		return nil
	}
	from := strings.TrimSpace(fromCommitID)
	to := strings.TrimSpace(toCommitID)
	if from == "" || to == "" {
		return nil
	}
	pf := strings.TrimSpace(prevStepFrom)
	pt := strings.TrimSpace(prevStepTo)
	return func() tea.Msg {
		var files []jj.ChangedFile
		var git string
		var err error
		if pf != "" && pt != "" {
			files, git, err = svc.DiffChangedFilesEvologStep(context.Background(), from, to, pf, pt)
		} else {
			files, git, err = svc.DiffChangedFilesFromTo(context.Background(), from, to)
		}
		return EvologSplitDiffLoadedMsg{Seq: seq, Files: files, GitDiff: git, Err: err}
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

// PerformEvologSplitCmd runs FAQ-style evolog split(s) and optional `jj split` by fileset or hunk prefix.
// When len(multiBaseCommitIDs) > 1, runs EvologMultiSplit (deepest-first list); otherwise a single MoveBookmarkDeltaOntoEvologBase using baseFromSelection or multiBaseCommitIDs[0].
func PerformEvologSplitCmd(svc *jj.Service, bookmarkName, localChangeID, localCommitHint, baseFromSelection string, multiBaseCommitIDs []string, splitFilesetsFirst []string, hunkPrefixFirst map[string]int) tea.Cmd {
	if svc == nil {
		return nil
	}
	return func() tea.Msg {
		ctx := context.Background()
		bases := append([]string(nil), multiBaseCommitIDs...)
		var err error
		if len(bases) > 1 {
			err = svc.EvologMultiSplit(ctx, bookmarkName, localChangeID, localCommitHint, bases, splitFilesetsFirst, hunkPrefixFirst)
		} else {
			base := strings.TrimSpace(baseFromSelection)
			if len(bases) == 1 {
				base = strings.TrimSpace(bases[0])
			}
			err = svc.MoveBookmarkDeltaOntoEvologBase(ctx, bookmarkName, localChangeID, localCommitHint, base, splitFilesetsFirst, hunkPrefixFirst)
		}
		if err != nil {
			return util.ErrorMsg{Err: fmt.Errorf("evolog split: %w", err)}
		}
		repo, rerr := svc.GetRepository(ctx, "")
		if rerr != nil {
			return util.ErrorMsg{Err: rerr}
		}
		return EvologSplitCompletedMsg{Repository: repo}
	}
}
