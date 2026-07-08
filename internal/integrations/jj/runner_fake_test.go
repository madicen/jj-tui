package jj_test

import (
	"context"
	"errors"
	"reflect"
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

// TestListOperationsParsesOpLog feeds canned `jj op log` output (marker +
// pipe-separated template) and checks the parser produces ordered operations
// with the newest flagged current.
func TestListOperationsParsesOpLog(t *testing.T) {
	const sample = "<<<OP>>>de64f4977189|2026-07-07 22:56:26|alice@host|describe commit abc\n" +
		"<<<OP>>>fb10d44b8106|2026-07-07 22:56:15|alice@host|new empty commit\n" +
		"<<<OP>>>3fd04c6d83e2|2026-07-07 22:56:00|alice@host|(no description)\n"
	fake := &mock.FakeRunner{
		RunOutputFn: func(_ context.Context, _ jj.RunOpts, args ...string) (string, error) {
			if len(args) >= 2 && args[0] == "op" && args[1] == "log" {
				return sample, nil
			}
			return "", nil
		},
	}
	svc := jj.NewServiceWithRunner("/fake/repo", fake)

	ops, err := svc.ListOperations(context.Background(), 5)
	if err != nil {
		t.Fatalf("ListOperations: %v", err)
	}
	if len(ops) != 3 {
		t.Fatalf("expected 3 operations, got %d: %+v", len(ops), ops)
	}
	if ops[0].ID != "de64f4977189" || ops[0].Description != "describe commit abc" || ops[0].User != "alice@host" {
		t.Fatalf("op[0] parsed wrong: %+v", ops[0])
	}
	if ops[0].Time != "2026-07-07 22:56:26" {
		t.Fatalf("op[0] time parsed wrong: %q", ops[0].Time)
	}
	if !ops[0].IsCurrent {
		t.Fatalf("newest op should be flagged current: %+v", ops[0])
	}
	if ops[1].IsCurrent || ops[2].IsCurrent {
		t.Fatalf("only the newest op should be current: %+v", ops)
	}
	if !fake.ArgsContain("--limit 5") {
		t.Fatalf("expected a --limit 5 flag on op log, calls=%v", fake.Calls)
	}
}

// TestRestoreOperationArgs verifies RestoreOperation shells out to
// `jj op restore <id>` and rejects an empty id.
func TestRestoreOperationArgs(t *testing.T) {
	var ranArgs []string
	fake := &mock.FakeRunner{
		RunFn: func(_ context.Context, _ jj.RunOpts, args ...string) error {
			ranArgs = append([]string(nil), args...)
			return nil
		},
	}
	svc := jj.NewServiceWithRunner("/fake/repo", fake)

	if err := svc.RestoreOperation(context.Background(), "  abc123  "); err != nil {
		t.Fatalf("RestoreOperation: %v", err)
	}
	if got, want := ranArgs, []string{"op", "restore", "abc123"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("RestoreOperation args = %v, want %v", got, want)
	}
	if err := svc.RestoreOperation(context.Background(), "   "); err == nil {
		t.Fatal("expected error for empty operation id")
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

// TestBackoutVerbDetection verifies BackoutCommit probes `jj backout --help`
// and picks the right subcommand + argv: legacy jj keeps `backout` (defaults
// onto @), while newer jj that renamed it uses `revert -r X --onto @`.
func TestBackoutVerbDetection(t *testing.T) {
	cases := []struct {
		name           string
		backoutHelpErr error
		wantArgs       []string
	}{
		{
			name:           "old jj has backout",
			backoutHelpErr: nil,
			wantArgs:       []string{"backout", "-r", "abc"},
		},
		{
			name:           "new jj renamed to revert",
			backoutHelpErr: errors.New("error: unrecognized subcommand 'backout'"),
			wantArgs:       []string{"revert", "-r", "abc", "--onto", "@"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var ranArgs []string
			helpProbes := 0
			fake := &mock.FakeRunner{
				RunOutputFn: func(_ context.Context, _ jj.RunOpts, args ...string) (string, error) {
					if len(args) >= 2 && args[0] == "backout" && args[1] == "--help" {
						helpProbes++
						return "", tc.backoutHelpErr
					}
					return "", nil
				},
				RunFn: func(_ context.Context, _ jj.RunOpts, args ...string) error {
					ranArgs = append([]string(nil), args...)
					return nil
				},
			}
			svc := jj.NewServiceWithRunner("/fake/repo", fake)

			if err := svc.BackoutCommit(context.Background(), "abc"); err != nil {
				t.Fatalf("BackoutCommit: %v", err)
			}
			if !reflect.DeepEqual(ranArgs, tc.wantArgs) {
				t.Fatalf("ran args = %v, want %v", ranArgs, tc.wantArgs)
			}

			// A second call must reuse the cached verb rather than probing again.
			if err := svc.BackoutCommit(context.Background(), "abc"); err != nil {
				t.Fatalf("BackoutCommit (second): %v", err)
			}
			if helpProbes != 1 {
				t.Fatalf("expected the backout capability to be probed once, got %d", helpProbes)
			}
		})
	}
}

// TestDuplicateArgs verifies DuplicateCommit passes -r, and -d only when a
// destination is supplied.
func TestDuplicateArgs(t *testing.T) {
	run := func(dest string) []string {
		var ranArgs []string
		fake := &mock.FakeRunner{
			RunFn: func(_ context.Context, _ jj.RunOpts, args ...string) error {
				ranArgs = append([]string(nil), args...)
				return nil
			},
		}
		svc := jj.NewServiceWithRunner("/fake/repo", fake)
		if err := svc.DuplicateCommit(context.Background(), "abc", dest); err != nil {
			t.Fatalf("DuplicateCommit(dest=%q): %v", dest, err)
		}
		return ranArgs
	}

	if got, want := run(""), []string{"duplicate", "-r", "abc"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("no-dest args = %v, want %v", got, want)
	}
	if got, want := run("main"), []string{"duplicate", "-r", "abc", "-d", "main"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("with-dest args = %v, want %v", got, want)
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
