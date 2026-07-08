package jj

import (
	"context"
	"fmt"
	"strings"
)

// ConflictFile is an unresolved merge conflict at a revision.
type ConflictFile struct {
	Path string
	Kind string // e.g. "2-sided conflict"
}

// ListUnresolvedConflicts runs `jj resolve --list` for a revision and returns
// paths that still need resolution. An empty slice means no conflicts remain.
func (s *Service) ListUnresolvedConflicts(ctx context.Context, revision string) ([]ConflictFile, error) {
	rev := strings.TrimSpace(revision)
	if rev == "" {
		rev = "@"
	}
	out, err := s.runJJCombined(ctx, "resolve", "-r", rev, "--list")
	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "no conflicts found") {
			return nil, nil
		}
		return nil, err
	}
	return parseResolveListOutput(out), nil
}

// ResolveFileWithTool runs `jj resolve --tool <tool> -r <rev> <path>` for one file.
// tool may be a configured merge tool name or jj built-ins `:ours` / `:theirs`.
func (s *Service) ResolveFileWithTool(ctx context.Context, revision, path, tool string) error {
	rev := strings.TrimSpace(revision)
	if rev == "" {
		rev = "@"
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("file path required")
	}
	tool = strings.TrimSpace(tool)
	args := []string{"resolve", "-r", rev, path}
	if tool != "" {
		args = append(args, "--tool", tool)
	}
	return s.runJJ(ctx, args...)
}

func parseResolveListOutput(out string) []ConflictFile {
	var files []ConflictFile
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Warning:") {
			continue
		}
		if strings.Contains(strings.ToLower(line), "no conflicts found") {
			continue
		}
		path, kind, ok := strings.Cut(line, "    ")
		if !ok {
			// Some jj versions use two spaces; fall back to fields split.
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				path = fields[0]
				kind = strings.Join(fields[1:], " ")
				ok = true
			}
		}
		if !ok || strings.TrimSpace(path) == "" {
			continue
		}
		files = append(files, ConflictFile{
			Path: strings.TrimSpace(path),
			Kind: strings.TrimSpace(kind),
		})
	}
	return files
}

// MarkConflictedChangedFiles sets ChangedFile.Conflicted for paths returned by
// jj resolve --list on the same revision.
func (s *Service) MarkConflictedChangedFiles(ctx context.Context, revision string, files []ChangedFile) ([]ChangedFile, error) {
	conflicts, err := s.ListUnresolvedConflicts(ctx, revision)
	if err != nil {
		return files, err
	}
	if len(conflicts) == 0 {
		return files, nil
	}
	set := make(map[string]bool, len(conflicts))
	for _, c := range conflicts {
		set[c.Path] = true
	}
	out := make([]ChangedFile, len(files))
	copy(out, files)
	for i := range out {
		if set[out[i].Path] {
			out[i].Conflicted = true
			delete(set, out[i].Path)
		}
	}
	for path := range set {
		out = append(out, ChangedFile{Path: path, Status: "C", Conflicted: true})
	}
	return out, nil
}
