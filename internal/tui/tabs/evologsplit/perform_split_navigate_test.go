package evologsplit

import (
	"testing"

	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/tui/state"
)

func TestPerformSplitNavigateCmd_stepwiseSkippedWhenPlanPreviewOpen(t *testing.T) {
	cfg := &config.Config{AIEvologMultiSplitMode: "stepwise"}
	entries := []jj.EvologEntry{
		{CommitID: "tip", CommitIDShort: "tip"},
		{CommitID: "mid", CommitIDShort: "mid"},
		{CommitID: "old", CommitIDShort: "old"},
	}
	m := Model{
		suggestCfg:          cfg,
		outcomePreviewOpen:  true,
		pendingMultiBaseIDs: []string{"base1", "base2"},
		pendingFilesFirst:   []string{"foo.go"},
		bookmarkName:        "split",
		tipChangeID:         "tipid",
		entries:             entries,
		selectedIdx:         1,
	}
	cmd := m.performSplitNavigateCmd()
	if cmd == nil {
		t.Fatal("expected cmd")
	}
	msg := cmd()
	nav, ok := msg.(state.NavigateMsg)
	if !ok {
		t.Fatalf("expected NavigateMsg, got %T", msg)
	}
	if len(nav.Target.EvologStepwiseRemainder) != 0 {
		t.Fatalf("preview open: want empty remainder, got %#v", nav.Target.EvologStepwiseRemainder)
	}
	if g := nav.Target.EvologMultiBaseCommitIDs; len(g) != 2 || g[0] != "base1" || g[1] != "base2" {
		t.Fatalf("EvologMultiBaseCommitIDs = %#v, want [base1 base2]", g)
	}
	if fs := nav.Target.EvologFilesetsFirst; len(fs) != 1 || fs[0] != "foo.go" {
		t.Fatalf("filesets should be preserved, got %#v", fs)
	}
}

func TestPerformSplitNavigateCmd_stepwiseWhenPreviewClosed(t *testing.T) {
	cfg := &config.Config{AIEvologMultiSplitMode: "stepwise"}
	entries := []jj.EvologEntry{
		{CommitID: "tip", CommitIDShort: "tip"},
		{CommitID: "mid", CommitIDShort: "mid"},
		{CommitID: "old", CommitIDShort: "old"},
	}
	m := Model{
		suggestCfg:          cfg,
		outcomePreviewOpen:  false,
		pendingMultiBaseIDs: []string{"base1", "base2"},
		pendingFilesFirst:   []string{"foo.go"},
		bookmarkName:        "split",
		tipChangeID:         "tipid",
		entries:             entries,
		selectedIdx:         1,
	}
	msg := m.performSplitNavigateCmd()()
	nav := msg.(state.NavigateMsg)
	if rem := nav.Target.EvologStepwiseRemainder; len(rem) != 1 || rem[0] != "base2" {
		t.Fatalf("remainder = %#v, want [base2]", rem)
	}
	if g := nav.Target.EvologMultiBaseCommitIDs; len(g) != 1 || g[0] != "base1" {
		t.Fatalf("EvologMultiBaseCommitIDs = %#v, want [base1]", g)
	}
	if fs := nav.Target.EvologFilesetsFirst; len(fs) != 0 {
		t.Fatalf("filesets deferred with remainder: want empty, got %#v", fs)
	}
}
