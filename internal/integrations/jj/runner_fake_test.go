package jj_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/mock"
)

// TestUndoCapturesCurrentOpID verifies Undo first reads the current operation id
// (via `jj op log`) and returns it for a later Redo, without a real repo.
func TestUndoCapturesCurrentOpID(t *testing.T) {
	fake := &mock.FakeRunner{
		RunOutputFn: func(_ context.Context, _ jj.RunOpts, args ...string) (string, error) {
			if len(args) > 0 && args[0] == "op" {
				return "opabc123\n", nil
			}
			return "", nil
		},
	}
	svc := jj.NewServiceWithRunner("/fake/repo", fake)

	opID, err := svc.Undo(context.Background())
	if err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if opID != "opabc123" {
		t.Fatalf("Undo opID = %q, want %q", opID, "opabc123")
	}
	// Expect an op-log read followed by an `undo` run.
	if !fake.ArgsContain("op log") {
		t.Fatalf("expected an `op log` invocation, calls=%v", fake.Calls)
	}
	if !fake.ArgsContain("undo") {
		t.Fatalf("expected an `undo` invocation, calls=%v", fake.Calls)
	}
}

// TestListBranchesParsesBookmarkList feeds canned `jj bookmark list` output and
// checks the parser produces the expected local/remote branches.
func TestListBranchesParsesBookmarkList(t *testing.T) {
	const sample = `main: qpvuntsm abc12345 initial
  @origin: qpvuntsm abc12345 initial
feature/x: zzzzzzzz def67890 wip
`
	fake := &mock.FakeRunner{
		RunOutputFn: func(_ context.Context, _ jj.RunOpts, args ...string) (string, error) {
			if len(args) >= 2 && args[0] == "bookmark" && args[1] == "list" {
				return sample, nil
			}
			return "", nil
		},
	}
	svc := jj.NewServiceWithRunner("/fake/repo", fake)

	branches, err := svc.ListBranches(context.Background(), 0)
	if err != nil {
		t.Fatalf("ListBranches: %v", err)
	}

	var haveMainLocal, haveMainOrigin, haveFeature bool
	for _, b := range branches {
		switch {
		case b.Name == "main" && b.IsLocal:
			haveMainLocal = true
		case b.Name == "main" && b.Remote == "origin":
			haveMainOrigin = true
		case b.Name == "feature/x" && b.IsLocal:
			haveFeature = true
		}
	}
	if !haveMainLocal || !haveMainOrigin || !haveFeature {
		t.Fatalf("parsed branches missing entries: %+v", branches)
	}
}

// TestErrorMessageExtractionFromCleanError ensures failing jj commands surface a
// cleaned error to callers when routed through the runner.
func TestErrorMessageExtractionFromCleanError(t *testing.T) {
	fake := &mock.FakeRunner{
		RunOutputFn: func(_ context.Context, _ jj.RunOpts, _ ...string) (string, error) {
			return "", errors.New("Error: No such revision 'zzz'")
		},
	}
	svc := jj.NewServiceWithRunner("/fake/repo", fake)

	_, err := svc.ListBranches(context.Background(), 0)
	if err == nil {
		t.Fatal("expected error from ListBranches when runner fails")
	}
	if !strings.Contains(err.Error(), "No such revision") {
		t.Fatalf("error should carry underlying message, got %q", err.Error())
	}
}
