package jj

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/madicen/jj-tui/internal/integrations/jj/jjout"
)

// GitFormatDiffForRevision returns a git-format unified diff for the revision against its parents.
// If maxBytes > 0 and the output exceeds maxBytes, the diff is truncated and a trailer is appended.
func (s *Service) GitFormatDiffForRevision(ctx context.Context, revision string, maxBytes int) (string, error) {
	out, err := s.runJJOutput(ctx, "diff", "-r", revision, "--git", "--color", "never")
	if err != nil {
		return "", err
	}
	if maxBytes > 0 && len(out) > maxBytes {
		trailer := "\n\n[diff truncated for AI context]\n"
		keep := maxBytes - len(trailer)
		if keep < 1 {
			keep = maxBytes
			trailer = ""
		}
		return out[:keep] + trailer, nil
	}
	return out, nil
}

// GitFormatDiffFromTo returns a git-format diff between two revisions (same semantics as jj diff --from --to).
func (s *Service) GitFormatDiffFromTo(ctx context.Context, fromRev, toRev string, maxBytes int) (string, error) {
	out, err := s.runJJOutput(ctx, "diff", "--from", fromRev, "--to", toRev, "--git", "--color", "never")
	if err != nil {
		return "", err
	}
	if maxBytes > 0 && len(out) > maxBytes {
		trailer := "\n\n[diff truncated for AI context]\n"
		keep := maxBytes - len(trailer)
		if keep < 1 {
			keep = maxBytes
			trailer = ""
		}
		return out[:keep] + trailer, nil
	}
	return out, nil
}

// ChangedFile represents a file changed in a commit
type ChangedFile struct {
	Path         string // File path
	Status       string // M=modified, A=added, D=deleted, R=renamed
	LinesAdded   int    // meaningful when StatsOK
	LinesRemoved int    // meaningful when StatsOK
	StatsOK      bool   // true when counts came from jj log template (single rev) or parsed git diff (from–to)
	Conflicted   bool   // true when jj resolve --list reports this path at the revision
}

// Template for one revision: per-file path, status char, lines added, lines removed (tab-separated lines).
// Uses one jj invocation vs diff --summary + separate stat work; requires a jj build with Commit.diff().stat().
const changedFilesStatLogTemplate = `self.diff().stat().files().map(|f| f.path().display() ++ "\t" ++ f.status_char() ++ "\t" ++ f.lines_added() ++ "\t" ++ f.lines_removed() ++ "\n")`

// GetChangedFiles gets changed files for a revision vs its parents, with per-file line stats when supported.
func (s *Service) GetChangedFiles(ctx context.Context, commitID string) ([]ChangedFile, error) {
	var files []ChangedFile
	out, err := s.runJJOutput(ctx, "log", "-r", commitID, "--no-graph", "-T", changedFilesStatLogTemplate)
	if err == nil && strings.TrimSpace(out) != "" {
		if parsed, perr := parseChangedFilesStatLogOutput(out); perr == nil && len(parsed) > 0 {
			files = parsed
		}
	}
	if len(files) == 0 {
		var ferr error
		files, ferr = s.getChangedFilesSummaryOnly(ctx, commitID)
		if ferr != nil {
			return nil, ferr
		}
	}
	return s.MarkConflictedChangedFiles(ctx, commitID, files)
}

func parseChangedFilesStatLogOutput(out string) ([]ChangedFile, error) {
	var files []ChangedFile
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) != 4 {
			return nil, fmt.Errorf("expected 4 tab fields, got %d", len(parts))
		}
		path := strings.TrimSpace(parts[0])
		status := strings.TrimSpace(parts[1])
		added, err1 := strconv.Atoi(strings.TrimSpace(parts[2]))
		removed, err2 := strconv.Atoi(strings.TrimSpace(parts[3]))
		if err1 != nil || err2 != nil {
			return nil, fmt.Errorf("invalid line counts")
		}
		if path == "" || status == "" {
			return nil, fmt.Errorf("empty path or status")
		}
		files = append(files, ChangedFile{
			Path:         path,
			Status:       status,
			LinesAdded:   added,
			LinesRemoved: removed,
			StatsOK:      true,
		})
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no files parsed")
	}
	return files, nil
}

func (s *Service) getChangedFilesSummaryOnly(ctx context.Context, commitID string) ([]ChangedFile, error) {
	out, err := s.runJJOutput(ctx, "diff", "--summary", "-r", commitID)
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}
	var files []ChangedFile
	for _, line := range jjout.SplitLines(out) {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) >= 2 {
			files = append(files, ChangedFile{
				Status: parts[0],
				Path:   parts[1],
			})
		}
	}
	return files, nil
}

// DiffSummaryLinesFromTo returns trimmed non-empty lines from `jj diff --from --to --summary`.
// Used for lightweight multi-step summaries (e.g. AI-assisted evolog split hints) without loading full git patches.
func (s *Service) DiffSummaryLinesFromTo(ctx context.Context, fromCommitID, toRev string) ([]string, error) {
	fromCommitID = strings.TrimSpace(fromCommitID)
	toRev = strings.TrimSpace(toRev)
	if fromCommitID == "" || toRev == "" {
		return nil, fmt.Errorf("from and to revisions are required")
	}
	out, err := s.runJJOutputNoHistory(ctx, "diff", "--from", fromCommitID, "--to", toRev, "--summary")
	if err != nil {
		return nil, err
	}
	return jjout.SplitLines(out), nil
}

// DiffNameOnlyLinesFromTo returns trimmed non-empty paths from `jj diff --from --to --name-only`
// (repo-relative, one path per line). Complements DiffSummaryLinesFromTo for callers that need a
// reliable path set regardless of summary line formatting.
func (s *Service) DiffNameOnlyLinesFromTo(ctx context.Context, fromCommitID, toRev string) ([]string, error) {
	fromCommitID = strings.TrimSpace(fromCommitID)
	toRev = strings.TrimSpace(toRev)
	if fromCommitID == "" || toRev == "" {
		return nil, fmt.Errorf("from and to revisions are required")
	}
	out, err := s.runJJOutputNoHistory(ctx, "diff", "--from", fromCommitID, "--to", toRev, "--name-only")
	if err != nil {
		return nil, err
	}
	return jjout.SplitLines(out), nil
}

// DiffChangedFilesFromTo lists paths changed between two revisions (from..to), using jj diff --summary,
// and fills per-file line counts from a git-format diff when possible. The string is the full unified
// git diff (same source as stats) for UI coloring.
func (s *Service) DiffChangedFilesFromTo(ctx context.Context, fromCommitID, toRev string) ([]ChangedFile, string, error) {
	return s.diffChangedFilesFromToWithGit(ctx, fromCommitID, toRev)
}

// diffChangedFilesFromToWithGit is like DiffChangedFilesFromTo but also returns the raw git-format diff output.
func (s *Service) diffChangedFilesFromToWithGit(ctx context.Context, fromCommitID, toRev string) ([]ChangedFile, string, error) {
	fromCommitID = strings.TrimSpace(fromCommitID)
	toRev = strings.TrimSpace(toRev)
	if fromCommitID == "" || toRev == "" {
		return nil, "", fmt.Errorf("from and to revisions are required")
	}
	out, err := s.runJJOutput(ctx, "diff", "--from", fromCommitID, "--to", toRev, "--summary")
	if err != nil {
		return nil, "", err
	}
	var files []ChangedFile
	for _, line := range jjout.SplitLines(out) {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) >= 2 {
			files = append(files, ChangedFile{
				Status: parts[0],
				Path:   parts[1],
			})
		}
	}
	if len(files) == 0 {
		return files, "", nil
	}
	gitOut, gerr := s.runJJOutput(ctx, "diff", "--from", fromCommitID, "--to", toRev, "--git", "--color", "never")
	if gerr != nil {
		return files, "", nil
	}
	if strings.TrimSpace(gitOut) == "" {
		return files, "", nil
	}
	stats := parseGitUnifiedDiffStats(gitOut)
	for i := range files {
		if st, ok := stats[files[i].Path]; ok {
			files[i].LinesAdded = st.added
			files[i].LinesRemoved = st.removed
			files[i].StatsOK = true
		}
	}
	return files, gitOut, nil
}

// DiffChangedFilesEvologStep is like DiffChangedFilesFromTo for one evolog UI row (diff from older snapshot to newer).
// When prevFrom/prevTo are set (older→newer along the list for the row above), files whose git patch text is
// identical to that prior step are omitted so the list only shows new deltas for this step.
// The returned git diff is the full patch from→to (not filtered to the shortened file list).
func (s *Service) DiffChangedFilesEvologStep(ctx context.Context, from, to, prevFrom, prevTo string) ([]ChangedFile, string, error) {
	files, gitCur, err := s.diffChangedFilesFromToWithGit(ctx, from, to)
	if err != nil {
		return nil, "", err
	}
	prevFrom = strings.TrimSpace(prevFrom)
	prevTo = strings.TrimSpace(prevTo)
	if prevFrom == "" || prevTo == "" || len(files) == 0 || strings.TrimSpace(gitCur) == "" {
		return files, gitCur, nil
	}
	gitPrev, gerr := s.runJJOutput(ctx, "diff", "--from", prevFrom, "--to", prevTo, "--git", "--color", "never")
	if gerr != nil || strings.TrimSpace(gitPrev) == "" {
		return files, gitCur, nil
	}
	chunksCur := mapGitUnifiedDiffByPath(gitCur)
	chunksPrev := mapGitUnifiedDiffByPath(gitPrev)
	var kept []ChangedFile
	for _, f := range files {
		cur := chunksCur[f.Path]
		prev := chunksPrev[f.Path]
		curMaterial := materialGitChunk(cur) || (f.StatsOK && (f.LinesAdded > 0 || f.LinesRemoved > 0))
		if !curMaterial {
			continue
		}
		if materialGitChunk(prev) && materialGitChunk(cur) &&
			normalizeGitChunkForCompare(cur) == normalizeGitChunkForCompare(prev) {
			continue
		}
		kept = append(kept, f)
	}
	return kept, gitCur, nil
}

// DiffRevisionFile returns the jj diff for a single path at the given revision vs its parents
// (equivalent to `jj diff -r <rev> -- <path>`).
func (s *Service) DiffRevisionFile(ctx context.Context, revision, path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	rev := strings.TrimSpace(revision)
	if rev == "" {
		return "", fmt.Errorf("revision is required")
	}
	out, err := s.runJJOutputNoHistory(ctx, "diff", "-r", rev, "--git", "--color", "never", "--", path)
	if err != nil {
		return "", err
	}
	return out, nil
}

// RevertFile reverts the changes to a file in a given commit,
// restoring it from the commit's parent.
func (s *Service) RevertFile(ctx context.Context, commitID, filePath string) error {
	// jj restore --to <commit> --from parents(<commit>) -- <file>
	// Using parents() function instead of ~ suffix to avoid revset parsing issues
	parentRev := fmt.Sprintf("parents(%s)", commitID)
	return s.runJJ(ctx, "restore", "--to", commitID, "--from", parentRev, "--", filePath)
}
