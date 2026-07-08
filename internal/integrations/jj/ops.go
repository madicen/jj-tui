package jj

import (
	"context"
	"fmt"
	"strings"
)

// Operation describes a single entry from `jj op log` (see ListOperations).
type Operation struct {
	ID          string // operation id (short form)
	Description string // first line of the operation description (e.g. "describe commit …")
	Time        string // operation end time, formatted "2006-01-02 15:04:05"
	User        string // operation user (e.g. "alice@host")
	IsCurrent   bool   // true for the operation the repo is currently at (top of the log)
}

// opLogMarker separates the graph/leading columns from our templated data on
// each `jj op log` row. Mirrors the graph.go "<<<COMMIT>>>" approach so we can
// parse a flat, machine-readable payload while ignoring any graph decoration.
const opLogMarker = "<<<OP>>>"

// opLogTemplate renders one marker-prefixed, pipe-separated row per operation:
// id | end-time | user | description-first-line. Description is placed last and
// reduced to its first line so embedded pipes/newlines in a description body
// can't break field parsing.
const opLogTemplate = `concat(
	"` + opLogMarker + `",
	id.short(12), "|",
	self.time().end().format("%Y-%m-%d %H:%M:%S"), "|",
	self.user(), "|",
	if(self.description(), self.description().first_line(), "(no description)"),
	"\n"
)`

// ListOperations returns recent operations from the jj operation log, newest
// first. limit <= 0 lists all operations. The first entry (newest) is flagged
// IsCurrent since that is the operation the working copy is currently at.
func (s *Service) ListOperations(ctx context.Context, limit int) ([]Operation, error) {
	args := []string{"op", "log", "--no-graph", "-T", opLogTemplate}
	if limit > 0 {
		args = append(args, "--limit", fmt.Sprintf("%d", limit))
	}
	out, err := s.runJJOutput(ctx, args...)
	if err != nil {
		return nil, err
	}

	var ops []Operation
	for _, line := range strings.Split(out, "\n") {
		idx := strings.Index(line, opLogMarker)
		if idx == -1 {
			continue
		}
		data := line[idx+len(opLogMarker):]
		parts := strings.SplitN(data, "|", 4)
		if len(parts) < 4 {
			continue
		}
		op := Operation{
			ID:          strings.TrimSpace(parts[0]),
			Time:        strings.TrimSpace(parts[1]),
			User:        strings.TrimSpace(parts[2]),
			Description: strings.TrimSpace(parts[3]),
		}
		if op.ID == "" {
			continue
		}
		ops = append(ops, op)
	}
	if len(ops) > 0 {
		ops[0].IsCurrent = true
	}
	return ops, nil
}

// RestoreOperation restores the repository to the given operation id
// (jj op restore <id>). This is the "time travel" primitive behind the
// operation-log browser; it is itself an operation, so it can be undone.
func (s *Service) RestoreOperation(ctx context.Context, opID string) error {
	opID = strings.TrimSpace(opID)
	if opID == "" {
		return fmt.Errorf("no operation ID provided for restore")
	}
	return s.runJJ(ctx, "op", "restore", opID)
}

// LatestOperationDescription returns the first line of the most recent
// operation's description (cheap: `jj op log --limit 1`). Used for the
// "Ctrl+z undoes: <op>" status-line hint after a mutating command.
func (s *Service) LatestOperationDescription(ctx context.Context) (string, error) {
	out, err := s.runJJOutputNoHistory(ctx, "op", "log", "--no-graph", "--limit", "1", "-T",
		`if(self.description(), self.description().first_line(), "(no description)")`)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// GetCurrentOperationID returns the current operation ID
func (s *Service) GetCurrentOperationID(ctx context.Context) (string, error) {
	out, err := s.runJJOutput(ctx, "op", "log", "--no-graph", "--limit", "1", "-T", "id")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// Undo undoes the last jj operation and returns the operation ID that was undone (for Redo)
func (s *Service) Undo(ctx context.Context) (string, error) {
	// Capture current op ID before undoing so we can restore to it if needed
	opID, err := s.GetCurrentOperationID(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get current op id: %w", err)
	}
	return opID, s.runJJ(ctx, "undo")
}

// Redo restores to a specific operation ID
func (s *Service) Redo(ctx context.Context, opID string) error {
	if opID == "" {
		return fmt.Errorf("no operation ID provided for redo")
	}
	return s.runJJ(ctx, "op", "restore", opID)
}
