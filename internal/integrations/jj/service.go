package jj

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/tui/util"
)

// CommandHistoryEntry represents a single jj command that was executed
type CommandHistoryEntry struct {
	Command   string    // Full command string (e.g., "jj log -r @")
	Timestamp time.Time // When the command was executed
	Duration  time.Duration
	Success   bool   // Whether the command succeeded
	Error     string // Error message if failed (truncated)
}

// Service handles jujutsu command execution
type Service struct {
	RepoPath       string
	commandHistory []CommandHistoryEntry
	historyMu      sync.RWMutex
	maxHistory     int // Maximum number of commands to keep
}

// SanitizeBookmarkName converts a string into a valid bookmark name
// Replaces spaces with hyphens, removes invalid characters, etc.
func SanitizeBookmarkName(name string) string {
	// Replace common problematic characters
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "\t", "_")
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	name = strings.ReplaceAll(name, ":", "-")
	name = strings.ReplaceAll(name, "~", "-")
	name = strings.ReplaceAll(name, "^", "-")
	name = strings.ReplaceAll(name, "?", "-")
	name = strings.ReplaceAll(name, "*", "-")
	name = strings.ReplaceAll(name, "[", "-")
	name = strings.ReplaceAll(name, "]", "-")
	name = strings.ReplaceAll(name, "@", "-")
	name = strings.ReplaceAll(name, "'", "")
	name = strings.ReplaceAll(name, ".", "-")

	// Collapse multiple hyphens into one
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}

	for strings.Contains(name, "__") {
		name = strings.ReplaceAll(name, "__", "_")
	}

	// Trim leading/trailing hyphens
	name = strings.Trim(name, "-")

	// Ensure it doesn't start with a dot
	name = strings.TrimPrefix(name, ".")

	return name
}

// NewService creates a new jj service
func NewService(repoPath string) (*Service, error) {
	// Verify jj is installed
	if _, err := exec.LookPath("jj"); err != nil {
		return nil, fmt.Errorf("jj command not found - please install jujutsu: %w", err)
	}

	// If no repo path provided, use current directory
	if repoPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
		repoPath = cwd
	}

	// Verify it's a jj repository
	if !isJJRepo(repoPath) {
		return nil, fmt.Errorf("not a jujutsu repository: %s\nHint: Run 'jj git init' or 'jj init --git' to initialize a repository", repoPath)
	}

	service := &Service{
		RepoPath:   repoPath,
		maxHistory: 100, // Keep last 100 commands
	}

	// Test that we can actually run jj commands
	ctx := context.Background()
	if _, err := service.runJJOutput(ctx, "--version"); err != nil {
		return nil, fmt.Errorf("failed to execute jj commands: %w", err)
	}

	return service, nil
}

// GetCommandHistory returns a copy of the command history (most recent first)
func (s *Service) GetCommandHistory() []CommandHistoryEntry {
	s.historyMu.RLock()
	defer s.historyMu.RUnlock()

	// Return a copy in reverse order (most recent first)
	result := make([]CommandHistoryEntry, len(s.commandHistory))
	for i, entry := range s.commandHistory {
		result[len(s.commandHistory)-1-i] = entry
	}
	return result
}

// addToHistory adds a command entry to the history
func (s *Service) addToHistory(entry CommandHistoryEntry) {
	s.historyMu.Lock()
	defer s.historyMu.Unlock()

	s.commandHistory = append(s.commandHistory, entry)

	// Trim history if it exceeds the limit; copy to new slice to release backing array
	if len(s.commandHistory) > s.maxHistory {
		keep := s.commandHistory[len(s.commandHistory)-s.maxHistory:]
		s.commandHistory = append([]CommandHistoryEntry(nil), keep...)
	}
}

// GetRepository retrieves the current repository state.
// revset: optional jj revset for the graph; if empty, a default is used that focuses on
// your work (mutable ancestors of @), bookmarks, and main to reduce noise from others' merges.
// Graph jj log invocations are recorded in Help → Command history.
func (s *Service) GetRepository(ctx context.Context, revset string) (*internal.Repository, error) {
	return s.getRepository(ctx, revset, true)
}

// GetRepositoryQuiet is the same as GetRepository but does not append the main graph jj log
// (and its fallbacks) to command history. Used for periodic background refresh so history stays readable.
func (s *Service) GetRepositoryQuiet(ctx context.Context, revset string) (*internal.Repository, error) {
	return s.getRepository(ctx, revset, false)
}

func (s *Service) getRepository(ctx context.Context, revset string, recordGraphInHistory bool) (*internal.Repository, error) {
	graph, err := s.getCommitGraph(ctx, revset, recordGraphInHistory)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit graph: %w", err)
	}

	var workingCopy internal.Commit
	for _, c := range graph.Commits {
		if c.IsWorking {
			workingCopy = c
			break
		}
	}

	return &internal.Repository{
		Path:        s.RepoPath,
		WorkingCopy: workingCopy,
		Graph:       *graph,
		PRs:         []internal.GitHubPR{}, // TODO: populate from GitHub
	}, nil
}

// CreateNewCommit creates a new commit with the given description
func (s *Service) CreateNewCommit(ctx context.Context, description string) error {
	return s.runJJ(ctx, "commit", "-m", description)
}

// DescribeCommit sets a new description for a commit (non-interactive)
func (s *Service) DescribeCommit(ctx context.Context, commitID string, message string) error {
	_, err := s.runJJOutput(ctx, "describe", commitID, "-m", message)
	if err == nil {
		return nil
	}
	if !isJJStaleWorkingCopyError(err) {
		return err
	}
	if uerr := s.runJJ(ctx, "workspace", "update-stale"); uerr != nil {
		return fmt.Errorf("%w\n\nCould not refresh stale working copy (jj workspace update-stale): %v", err, uerr)
	}
	_, err2 := s.runJJOutput(ctx, "describe", commitID, "-m", message)
	return err2
}

func isJJStaleWorkingCopyError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "working copy is stale")
}

// GetCommitDescription gets the full description of a commit
func (s *Service) GetCommitDescription(ctx context.Context, commitID string) (string, error) {
	out, err := s.runJJOutput(ctx, "log", "-r", commitID, "--no-graph", "-T", "description")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

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

// GetRevisionChangeID gets the change_id for any jj revision (e.g., "main@origin", "@", etc.)
func (s *Service) GetRevisionChangeID(ctx context.Context, revision string) (string, error) {
	out, err := s.runJJOutput(ctx, "log", "-r", revision, "--no-graph", "-T", "change_id")
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

// ChangedFile represents a file changed in a commit
type ChangedFile struct {
	Path         string // File path
	Status       string // M=modified, A=added, D=deleted, R=renamed
	LinesAdded   int    // meaningful when StatsOK
	LinesRemoved int    // meaningful when StatsOK
	StatsOK      bool   // true when counts came from jj log template (single rev) or parsed git diff (from–to)
}

// DivergentVersion is one visible revision for a divergent jj change ID.
type DivergentVersion struct {
	CommitID      string // full id for jj abandon / compare
	CommitIDShort string
	Summary       string
	Author        string
	WhenDisplay   string        // compact local time for UI
	ParentsShort  string        // short parent commit id(s), comma-separated
	Bookmarks     string        // local bookmark names on this revision (may be empty)
	Immutable     bool          // when true, other heads usually cannot be resolved by abandoning this row
	ChangedFiles  []ChangedFile // vs parent (jj diff --summary -r); nil if listing failed, non-nil when loaded (may be empty)
	FilesLine     string        // compact summary for logs / fallback
}

// plainDiffStatsSuffix is an unstyled variant of the TUI diff stats suffix (+n / −m only when non-zero).
func plainDiffStatsSuffix(added, removed int, ok bool) string {
	if !ok {
		return ""
	}
	var parts []string
	if added > 0 {
		parts = append(parts, fmt.Sprintf("+%d", added))
	}
	if removed > 0 {
		parts = append(parts, fmt.Sprintf("-%d", removed))
	}
	if len(parts) == 0 {
		return ""
	}
	return " " + strings.Join(parts, " ")
}

// formatDivergentFilesLine formats changed-file rows for divergent-resolution UI.
func formatDivergentFilesLine(files []ChangedFile) string {
	if len(files) == 0 {
		return "(no changes vs parent)"
	}
	const maxShow = 4
	var b strings.Builder
	nMore := 0
	for i, f := range files {
		if i >= maxShow {
			nMore = len(files) - maxShow
			break
		}
		if i > 0 {
			b.WriteString(", ")
		}
		p := f.Path
		if len(p) > 40 {
			p = p[:18] + "…" + p[len(p)-18:]
		}
		b.WriteString(f.Status)
		b.WriteByte(' ')
		b.WriteString(p)
		b.WriteString(plainDiffStatsSuffix(f.LinesAdded, f.LinesRemoved, f.StatsOK))
	}
	if nMore > 0 {
		fmt.Fprintf(&b, " (+%d more)", nMore)
	}
	return b.String()
}

func compactWhenDisplay(ts string) string {
	ts = strings.TrimSpace(ts)
	if ts == "" {
		return ""
	}
	layouts := []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05.999999999 -0700 MST"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, ts); err == nil {
			return t.Local().Format("Jan 02 2006 15:04")
		}
	}
	if len(ts) > 20 {
		return ts[:20]
	}
	return ts
}

// Template for one revision: per-file path, status char, lines added, lines removed (tab-separated lines).
// Uses one jj invocation vs diff --summary + separate stat work; requires a jj build with Commit.diff().stat().
const changedFilesStatLogTemplate = `self.diff().stat().files().map(|f| f.path().display() ++ "\t" ++ f.status_char() ++ "\t" ++ f.lines_added() ++ "\t" ++ f.lines_removed() ++ "\n")`

// GetChangedFiles gets changed files for a revision vs its parents, with per-file line stats when supported.
func (s *Service) GetChangedFiles(ctx context.Context, commitID string) ([]ChangedFile, error) {
	out, err := s.runJJOutput(ctx, "log", "-r", commitID, "--no-graph", "-T", changedFilesStatLogTemplate)
	if err == nil && strings.TrimSpace(out) != "" {
		if files, perr := parseChangedFilesStatLogOutput(out); perr == nil && len(files) > 0 {
			return files, nil
		}
	}
	// Older jj or template parse issues: fall back to summary only (no line counts).
	return s.getChangedFilesSummaryOnly(ctx, commitID)
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
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
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
	var lines []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, nil
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
	var paths []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			paths = append(paths, line)
		}
	}
	return paths, nil
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
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
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

// IsCommitMutable checks if a commit can be modified
func (s *Service) IsCommitMutable(ctx context.Context, commitID string) bool {
	// Try a no-op describe to see if the commit is mutable
	_, err := s.runJJOutput(ctx, "log", "-r", commitID, "--no-graph", "-T", "if(immutable, \"immutable\", \"mutable\")")
	return err == nil
}

// CheckoutCommit checks out a specific commit (uses jj edit)
func (s *Service) CheckoutCommit(ctx context.Context, commitID string) error {
	return s.runJJ(ctx, "edit", commitID)
}

// CreateNewBranch creates a new branch at the current commit
func (s *Service) CreateNewBranch(ctx context.Context, branchName string) error {
	return s.runJJ(ctx, "branch", "create", branchName)
}

// CreateBranchFromMain creates a bookmark for a ticket, handling existing work intelligently.
// If the user has existing work based on main (main -> A -> B...), the bookmark is added
// to the first commit after main (A). Otherwise, a new empty commit is created and the
// bookmark is placed on it. This is the standard jj workflow.
func (s *Service) CreateBranchFromMain(ctx context.Context, bookmarkName string) error {
	// Determine the main branch reference (prefer main@origin, fall back to main)
	mainRef := "main@origin"
	if _, err := s.runJJOutput(ctx, "log", "-r", mainRef, "--no-graph", "-T", "change_id", "--limit", "1"); err != nil {
		// main@origin doesn't exist (demo repo or no remote), fall back to main
		mainRef = "main"
	}

	// Find the first mutable commit after main in our ancestry
	// This handles: main -> A -> B -> @ by finding A
	// Revset: ancestors of @ that are mutable AND whose parent is main
	rootCommitID, err := s.runJJOutput(ctx, "log", "-r", fmt.Sprintf("ancestors(@) & mutable() & children(%s)", mainRef), "--no-graph", "-T", "change_id", "--limit", "1")
	if err == nil && strings.TrimSpace(rootCommitID) != "" {
		rootCommitID = strings.TrimSpace(rootCommitID)

		// Check if this root commit is non-empty (has actual changes)
		emptyCheck, _ := s.runJJOutput(ctx, "log", "-r", rootCommitID, "--no-graph", "-T", "empty")
		isRootEmpty := strings.TrimSpace(emptyCheck) == "true"

		if !isRootEmpty {
			// We have existing non-empty work based on main - add bookmark to the root commit
			if err := s.runJJ(ctx, "bookmark", "create", bookmarkName, "-r", rootCommitID); err != nil {
				return fmt.Errorf("failed to create bookmark: %w", err)
			}
			return nil
		}
	}

	// No existing non-empty work based on main. Create a new empty commit and place the bookmark on it.
	// This is the standard jj workflow since bookmarks must be on mutable commits.
	if err := s.runJJ(ctx, "new", mainRef); err != nil {
		return fmt.Errorf("failed to create new commit from %s: %w", mainRef, err)
	}

	// Create the bookmark on this new mutable commit
	if err := s.runJJ(ctx, "bookmark", "create", bookmarkName); err != nil {
		return fmt.Errorf("failed to create bookmark: %w", err)
	}

	return nil
}

// CreateBookmarkOnCommit creates a bookmark on a specific commit
func (s *Service) CreateBookmarkOnCommit(ctx context.Context, bookmarkName, commitID string) error {
	// jj bookmark create <name> -r <revision>
	return s.runJJ(ctx, "bookmark", "create", bookmarkName, "-r", commitID)
}

// MoveBookmark moves an existing bookmark to a different commit
func (s *Service) MoveBookmark(ctx context.Context, bookmarkName, commitID string) error {
	// jj bookmark set <name> -r <revision> (--allow-backwards: target may be an ancestor of the current tip)
	return s.runJJ(ctx, "bookmark", "set", util.BookmarkArgForSetMove(bookmarkName), "-r", commitID, "--allow-backwards")
}

// DeleteBookmark deletes a bookmark
func (s *Service) DeleteBookmark(ctx context.Context, bookmarkName string) error {
	return s.runJJ(ctx, "bookmark", "delete", util.JJExactBookmarkPattern(bookmarkName))
}

// ResolveBookmarkConflictKeepLocal resolves a diverged/conflicted bookmark by collapsing the
// local bookmark to the non-remote tip, then jj git push (no --force; current jj uses lease-style safety).
func (s *Service) ResolveBookmarkConflictKeepLocal(ctx context.Context, bookmarkName string) error {
	bookmarkName = util.BookmarkNameForRevset(bookmarkName)
	bookmarkName = util.LocalBookmarkName(bookmarkName)
	if bookmarkName == "" {
		return fmt.Errorf("bookmark name is required")
	}
	pat := util.RevsetExactPattern(bookmarkName)
	toRev := fmt.Sprintf(
		"latest(heads(bookmarks(%s) ~ latest(remote_bookmarks(%s, %s))))",
		pat, pat, util.RevsetExactPattern("origin"),
	)
	if err := s.runJJ(ctx, "bookmark", "set", util.BookmarkArgForSetMove(bookmarkName), "-r", toRev, "--allow-backwards"); err != nil {
		return fmt.Errorf("bookmark set (keep local): %w", err)
	}
	if err := s.runJJ(ctx, "git", "push", "--bookmark", util.JJExactBookmarkPattern(bookmarkName), "--remote", "origin"); err != nil {
		return fmt.Errorf("git push: %w", err)
	}
	// Colocated git refs update on push, but jj's remote_bookmarks for list/HasConflict can lag until fetch.
	// Ignore errors so a resolve+push success is not reported as failure if fetch is unavailable.
	_ = s.FetchFromRemote(ctx, "origin")
	return nil
}

// ResolveBookmarkConflictResetToRemote resolves a diverged bookmark by resetting local to remote
func (s *Service) ResolveBookmarkConflictResetToRemote(ctx context.Context, bookmarkName string) error {
	bookmarkName = util.BookmarkNameForRevset(bookmarkName)
	bookmarkName = util.LocalBookmarkName(bookmarkName)
	if bookmarkName == "" {
		return fmt.Errorf("bookmark name is required")
	}
	// Set the local bookmark to the tip remembered for origin (handles conflicted remote bookmarks).
	remoteRev := fmt.Sprintf("latest(remote_bookmarks(%s, %s))",
		util.RevsetExactPattern(bookmarkName), util.RevsetExactPattern("origin"))
	return s.runJJ(ctx, "bookmark", "set", util.BookmarkArgForSetMove(bookmarkName), "-r", remoteRev)
}

// joinConflictTabLog parses jj log lines as change_id\tsummary\ttimestamp (tab-separated).
func joinConflictTabLog(out string) (idJoined, summaryJoined, whenJoined string) {
	var ids, sums, whens []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		ids = append(ids, strings.TrimSpace(parts[0]))
		sum := ""
		if len(parts) >= 2 {
			sum = strings.TrimSpace(parts[1])
		}
		sums = append(sums, sum)
		w := ""
		if len(parts) >= 3 {
			w = compactWhenDisplay(strings.TrimSpace(parts[2]))
		}
		whens = append(whens, w)
	}
	if len(ids) == 0 {
		return "", "", ""
	}
	if len(ids) == 1 {
		return ids[0], sums[0], whens[0]
	}
	return strings.Join(ids, ", "), strings.Join(sums, " · "), strings.Join(whens, " · ")
}

// GetBookmarkConflictInfo retrieves information about a conflicted bookmark
// Returns local commit ID, remote commit ID, local summary, remote summary, and compact timestamps when available.
func (s *Service) GetBookmarkConflictInfo(ctx context.Context, bookmarkName string) (localID, remoteID, localSummary, remoteSummary, localWhen, remoteWhen string, err error) {
	bookmarkName = util.BookmarkNameForRevset(bookmarkName)
	bookmarkName = util.LocalBookmarkName(bookmarkName)
	if bookmarkName == "" {
		return "", "", "", "", "", "", fmt.Errorf("bookmark name is required")
	}
	logT := `change_id.short(8) ++ "\t" ++ if(description, description.first_line(), "(no description)") ++ "\t" ++ author.timestamp()`
	// Conflicted bookmarks need bookmarks()/remote_bookmarks(), not a bare symbol (slashes, multi-target).
	localRev := fmt.Sprintf("bookmarks(%s)", util.RevsetExactPattern(bookmarkName))
	remoteRev := fmt.Sprintf("remote_bookmarks(%s, %s)",
		util.RevsetExactPattern(bookmarkName), util.RevsetExactPattern("origin"))
	localOut, err := s.runJJOutput(ctx, "log", "-r", localRev, "--no-graph", "-T", logT)
	if err != nil {
		return "", "", "", "", "", "", fmt.Errorf("failed to get local bookmark info: %w", err)
	}
	localID, localSummary, localWhen = joinConflictTabLog(localOut)

	remoteOut, err := s.runJJOutput(ctx, "log", "-r", remoteRev, "--no-graph", "-T", logT)
	if err != nil {
		return localID, "", localSummary, "", localWhen, "", fmt.Errorf("failed to get remote bookmark info: %w", err)
	}
	remoteID, remoteSummary, remoteWhen = joinConflictTabLog(remoteOut)

	return localID, remoteID, localSummary, remoteSummary, localWhen, remoteWhen, nil
}

// GetDivergentCommitDetails returns one entry per visible revision with the same change ID,
// including metadata and a compact file list vs parent for comparison in the UI.
func (s *Service) GetDivergentCommitDetails(ctx context.Context, changeID string) ([]DivergentVersion, error) {
	changeID = strings.TrimSpace(changeID)
	if changeID == "" {
		return nil, fmt.Errorf("change ID is required")
	}
	const template = `commit_id ++ "\t" ++ commit_id.short(12) ++ "\t" ++ if(description, description.first_line(), "(no description)") ++ "\t" ++ author.email() ++ "\t" ++ author.timestamp() ++ "\t" ++ parents.map(|p| p.commit_id().short(8)).join(",") ++ "\t" ++ bookmarks.join(",") ++ "\t" ++ if(immutable, "true", "false") ++ "\n"`
	out, err := s.runJJOutput(ctx, "log", "-r", fmt.Sprintf("change_id(%s)", changeID), "--no-graph", "-T", template)
	if err != nil {
		return nil, fmt.Errorf("failed to get divergent commit info: %w", err)
	}

	var versions []DivergentVersion
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 5 {
			continue
		}
		fullID := strings.TrimSpace(parts[0])
		if fullID == "" {
			continue
		}
		short := strings.TrimSpace(parts[1])
		summary := strings.TrimSpace(parts[2])
		author := strings.TrimSpace(parts[3])
		tsRaw := strings.TrimSpace(parts[4])
		parents := ""
		bookmarks := ""
		if len(parts) > 5 {
			parents = strings.TrimSpace(parts[5])
		}
		if len(parts) > 6 {
			bookmarks = strings.TrimSpace(parts[6])
		}
		immutable := false
		if len(parts) > 7 {
			immutable = strings.TrimSpace(parts[7]) == "true"
		}

		files, ferr := s.GetChangedFiles(ctx, fullID)
		filesLine := formatDivergentFilesLine(files)
		var storedFiles []ChangedFile
		if ferr != nil {
			filesLine = "(could not list files vs parent)"
			storedFiles = nil
		} else if len(files) == 0 {
			storedFiles = []ChangedFile{}
		} else {
			storedFiles = files
		}

		versions = append(versions, DivergentVersion{
			CommitID:      fullID,
			CommitIDShort: short,
			Summary:       summary,
			Author:        author,
			WhenDisplay:   compactWhenDisplay(tsRaw),
			ParentsShort:  parents,
			Bookmarks:     bookmarks,
			Immutable:     immutable,
			ChangedFiles:  storedFiles,
			FilesLine:     filesLine,
		})
	}

	if len(versions) < 2 {
		return nil, fmt.Errorf("commit is not divergent (only %d version found)", len(versions))
	}

	return versions, nil
}

// GetDivergentCommitInfo retrieves short commit id and summary per version (legacy shape).
func (s *Service) GetDivergentCommitInfo(ctx context.Context, changeID string) (commitIDs []string, summaries []string, err error) {
	versions, err := s.GetDivergentCommitDetails(ctx, changeID)
	if err != nil {
		return nil, nil, err
	}
	for _, v := range versions {
		commitIDs = append(commitIDs, v.CommitIDShort)
		summaries = append(summaries, v.Summary)
	}
	return commitIDs, summaries, nil
}

// ResolveDivergentCommit resolves a divergent commit by keeping one version and abandoning others
// keepCommitID is the commit hash (not change ID) to keep
func (s *Service) ResolveDivergentCommit(ctx context.Context, changeID, keepCommitID string) error {
	versions, err := s.GetDivergentCommitDetails(ctx, changeID)
	if err != nil {
		return err
	}

	for _, v := range versions {
		if commitIDsEquivalent(v.CommitID, keepCommitID) {
			continue
		}
		if err := s.runJJ(ctx, "abandon", v.CommitID); err != nil {
			return fmt.Errorf("failed to abandon commit %s: %w", v.CommitID, err)
		}
	}

	return nil
}

// SquashCommit squashes a commit into its parent
func (s *Service) SquashCommit(ctx context.Context, commitID string) error {
	// Get the description of the commit being squashed
	sourceDesc, err := s.runJJOutput(ctx, "log", "-r", commitID, "--no-graph", "-T", "description")
	if err != nil {
		return err
	}
	sourceDesc = strings.TrimSpace(sourceDesc)

	// Get the description of the parent (destination)
	parentDesc, err := s.runJJOutput(ctx, "log", "-r", fmt.Sprintf("parents(%s)", commitID), "--no-graph", "-T", "description")
	if err != nil {
		return err
	}
	parentDesc = strings.TrimSpace(parentDesc)

	// Combine descriptions - prefer parent's if it exists, otherwise use source's
	// If both have descriptions, combine them with the parent first
	var combinedDesc string
	if parentDesc != "" && sourceDesc != "" {
		combinedDesc = parentDesc + "\n\n" + sourceDesc
	} else if parentDesc != "" {
		combinedDesc = parentDesc
	} else {
		combinedDesc = sourceDesc
	}

	// Squash the commit into its parent with explicit message to avoid interactive editor
	return s.runJJ(ctx, "squash", "-r", commitID, "-m", combinedDesc)
}

// NewCommit creates a new commit. If parentCommitID is provided, creates a child of that commit.
// Otherwise creates a new commit on top of the current working copy (@).
// Note: This creates an empty commit initially. To avoid unnecessary placeholder commits during
// branch creation, use CreateBranchFromMain instead. NewCommit is useful for creating commits
// at specific parent points in the graph.
func (s *Service) NewCommit(ctx context.Context, parentCommitID string) error {
	if parentCommitID != "" {
		return s.runJJ(ctx, "new", parentCommitID)
	}
	return s.runJJ(ctx, "new")
}

// AbandonCommit abandons a commit, removing it from the repository
func (s *Service) AbandonCommit(ctx context.Context, commitID string) error {
	return s.runJJ(ctx, "abandon", commitID)
}

// AbandonOldCommitsBatch runs one `jj abandon` over every mutable commit in the **current graph**
// (except the working-copy row and the main@origin change id), matching the original settings
// behavior. A revset like `mutable() & ~ancestors(main@origin)` was wrong: most local mutable
// commits on trunk are still *in* ancestors(main@origin), so almost nothing was abandoned.
// Divergent commits: each graph row uses its unique **commit** id in the union revset so all
// versions of a change can be abandoned together; if jj rejects the batch (e.g. ordering), use
// the divergent resolver (d) first, then retry cleanup.
func (s *Service) AbandonOldCommitsBatch(ctx context.Context, repo *internal.Repository) (abandoned int, err error) {
	if repo == nil {
		return 0, fmt.Errorf("repository required")
	}
	mainChangeID, err := s.GetRevisionChangeID(ctx, "main@origin")
	if err != nil || strings.TrimSpace(mainChangeID) == "" {
		return 0, fmt.Errorf("could not find main@origin - make sure to track it first")
	}
	mainKey := changeIDRootKey(mainChangeID)

	// Index by commit ID, not change ID: divergent rows share one change (same change_id.short(8) in
	// the graph) but have different commit IDs — deduping by change would drop every extra version
	// and jj would only abandon one revision.
	seen := make(map[string]bool)
	var revParts []string
	for _, commit := range repo.Graph.Commits {
		if commit.IsWorking || commit.Immutable {
			continue
		}
		ch := strings.TrimSpace(commit.ChangeID)
		if ch == "" {
			continue
		}
		if mainKey != "" && changeIDRootKey(ch) == mainKey {
			continue
		}
		rev := strings.TrimSpace(commit.ID)
		if rev == "" {
			rev = strings.TrimSpace(commit.ShortID)
		}
		if rev == "" {
			rev = ch
		}
		if seen[rev] {
			continue
		}
		seen[rev] = true
		revParts = append(revParts, rev)
	}
	if len(revParts) == 0 {
		return 0, nil
	}
	revset := strings.Join(revParts, " | ")
	if err := s.runJJ(ctx, "abandon", revset); err != nil {
		return 0, err
	}
	return len(revParts), nil
}

// RebaseCommit rebases a commit and all its descendants onto a destination commit
func (s *Service) RebaseCommit(ctx context.Context, sourceCommitID, destCommitID string) error {
	// jj rebase -s <source> -d <destination>
	// Using -s (source) instead of -r (revision) so descendants follow along
	return s.runJJ(ctx, "rebase", "-s", sourceCommitID, "-d", destCommitID)
}

// SplitFileToParent moves a single file from a commit to a new parent commit.
// This creates a new commit between the current commit and its parent,
// then moves just the file's changes to that new commit.
func (s *Service) SplitFileToParent(ctx context.Context, commitID, filePath string) error {
	// Step 1: Create a new commit inserted BEFORE the target commit
	// This automatically rebases the target to be a child of the new commit
	if err := s.runJJ(ctx, "new", "--insert-before", commitID, "-m", "(split)"); err != nil {
		return fmt.Errorf("failed to create new parent commit: %w", err)
	}

	// Step 2: Move the file from the original commit to the new commit (now @)
	// Using squash --from moves changes from the source to the current commit
	if err := s.runJJ(ctx, "squash", "--from", commitID, "-m", "(split)", "--", filePath); err != nil {
		return fmt.Errorf("failed to move file to new parent: %w", err)
	}

	return nil
}

// MoveFileToChild moves a single file from a commit to a new child commit.
// This creates a new commit AFTER the specified commit (between it and its children),
// then moves just the file's changes from the parent to the new child.
func (s *Service) MoveFileToChild(ctx context.Context, commitID, filePath string) error {
	// Step 1: Create a new commit inserted AFTER the target commit
	// Using --insert-after automatically rebases existing children onto the new commit
	// Example: A -> B -> C becomes A -> NewCommit -> B -> C
	if err := s.runJJ(ctx, "new", "--insert-after", commitID, "-m", "(split)"); err != nil {
		return fmt.Errorf("failed to create new child commit: %w", err)
	}

	// Step 2: Squash just the specified file from the parent commit to the new commit
	// jj squash --from <parent> -m "(split)" -- <file> ( -m avoids opening editor )
	if err := s.runJJ(ctx, "squash", "--from", commitID, "-m", "(split)", "--", filePath); err != nil {
		return fmt.Errorf("failed to move file to new commit: %w", err)
	}

	return nil
}

// followUpOnOriginMessage is the default description for the new commit created by
// MoveBookmarkDeltaOntoOrigin (user can jj describe afterward).
const followUpOnOriginMessage = "Follow-up (local changes on top of origin)"

// MoveBookmarkDeltaOntoOrigin places bookmark@origin as the parent of new work without rewriting the
// revision Git already has: it fetches, creates a new commit on top of bookmark@origin with the same
// tree as the bookmark tip, moves the bookmark there, rebases any non–working-copy children of the
// old tip onto the new bookmark tip (so local stacks stay intact), then abandons the old tip.
// localChangeID is the selected revision’s change ID (for diff). localCommitID is the git commit id
// (short or full) for revsets where the change ID may be divergent; pass commit.ID from the graph.
func (s *Service) MoveBookmarkDeltaOntoOrigin(ctx context.Context, bookmarkName, localChangeID, localCommitID string) error {
	if strings.TrimSpace(bookmarkName) == "" || strings.TrimSpace(localChangeID) == "" {
		return fmt.Errorf("bookmark name and local revision are required")
	}
	revForSel := strings.TrimSpace(localCommitID)
	if revForSel == "" {
		revForSel = localChangeID
	}
	remoteRef := bookmarkName + "@origin"
	if _, err := s.FetchFromGit(ctx); err != nil {
		return fmt.Errorf("fetch before comparing to origin: %w", err)
	}
	if _, err := s.runJJOutput(ctx, "log", "-r", remoteRef, "--no-graph", "-T", "commit_id", "--limit", "1"); err != nil {
		return fmt.Errorf("no revision %s (track the bookmark or run jj git fetch)", remoteRef)
	}
	tipCommitID, err := s.runJJOutputNoHistory(ctx, "log", "-r", bookmarkName, "--no-graph", "-T", "commit_id", "--limit", "1")
	if err != nil {
		return fmt.Errorf("bookmark %q: %w", bookmarkName, err)
	}
	tipCommitID = strings.TrimSpace(tipCommitID)
	selCommitID, err := s.runJJOutputNoHistory(ctx, "log", "-r", revForSel, "--no-graph", "-T", "commit_id", "--limit", "1")
	if err != nil {
		return fmt.Errorf("selected revision: %w", err)
	}
	selCommitID = strings.TrimSpace(selCommitID)
	if tipCommitID != selCommitID {
		return fmt.Errorf("select the bookmark tip (%s) to align with origin", bookmarkName)
	}
	childrenRev := fmt.Sprintf("children(%s) ~ @", tipCommitID)
	childOut, err := s.runJJOutput(ctx, "log", "-r", childrenRev, "--no-graph", "-T", "commit_id", "--limit", "50")
	if err != nil {
		return fmt.Errorf("check descendants: %w", err)
	}
	var rebaseChildRoots []string
	for _, line := range strings.Split(childOut, "\n") {
		id := strings.TrimSpace(line)
		if id != "" {
			rebaseChildRoots = append(rebaseChildRoots, id)
		}
	}
	diffOut, err := s.runJJOutput(ctx, "diff", "--from", remoteRef, "--to", localChangeID, "--summary")
	if err != nil {
		return fmt.Errorf("diff vs origin: %w", err)
	}
	if strings.TrimSpace(diffOut) == "" {
		return fmt.Errorf("tree already matches %s; nothing to move", remoteRef)
	}
	if err := s.runJJ(ctx, "new", remoteRef, "-m", followUpOnOriginMessage); err != nil {
		return fmt.Errorf("jj new: %w", err)
	}
	if err := s.runJJ(ctx, "restore", "--into", "@", "--from", tipCommitID); err != nil {
		return fmt.Errorf("jj restore: %w", err)
	}
	if err := s.runJJ(ctx, "bookmark", "set", util.BookmarkArgForSetMove(bookmarkName), "-r", "@", "--allow-backwards"); err != nil {
		return fmt.Errorf("jj bookmark set: %w", err)
	}
	if len(rebaseChildRoots) > 0 {
		src := rebaseChildRoots[0]
		if len(rebaseChildRoots) > 1 {
			src = strings.Join(rebaseChildRoots, " | ")
		}
		if err := s.runJJ(ctx, "rebase", "-s", src, "-o", "@"); err != nil {
			return fmt.Errorf("jj rebase (stack on new tip): %w", err)
		}
	}
	if err := s.runJJ(ctx, "abandon", tipCommitID); err != nil {
		return fmt.Errorf("jj abandon old bookmark tip: %w", err)
	}
	// Rebasing children can leave two visible commits for the same change ID (temporary divergence,
	// same situation as the jj FAQ “split work” flow). Keep the newest version on the path to @.
	if err := s.abandonDivergentDuplicateCommitsOffWCPath(ctx); err != nil {
		return err
	}
	return nil
}

// abandonDivergentDuplicateCommitsOffWCPath abandons mutable divergent duplicates, keeping the head
// revision for each change on the ancestry of @ (jj log order: newest first, so limit 1 is @-side).
func (s *Service) abandonDivergentDuplicateCommitsOffWCPath(ctx context.Context) error {
	out, err := s.runJJOutputNoHistory(ctx, "log", "-r", "divergent() & mutable()", "--no-graph", "-T", "change_id ++ \"\\t\" ++ commit_id ++ \"\\n\"", "--limit", "200")
	if err != nil || strings.TrimSpace(out) == "" {
		return nil
	}
	byChange := make(map[string][]string)
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), "\t", 2)
		if len(parts) != 2 {
			continue
		}
		chID, commitID := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		if chID == "" || commitID == "" {
			continue
		}
		byChange[chID] = append(byChange[chID], commitID)
	}
	for chID, cids := range byChange {
		if len(cids) < 2 {
			continue
		}
		keepOut, err := s.runJJOutputNoHistory(ctx, "log", "-r", fmt.Sprintf("change_id(%s) & ::(@)", chID), "--no-graph", "-T", "commit_id", "--limit", "1")
		if err != nil {
			continue
		}
		keep := strings.TrimSpace(keepOut)
		if keep == "" {
			continue
		}
		for _, cid := range cids {
			if commitIDsEquivalent(cid, keep) {
				continue
			}
			if err := s.runJJ(ctx, "abandon", cid); err != nil {
				return fmt.Errorf("jj abandon divergent duplicate after restack: %w", err)
			}
		}
	}
	return nil
}

// EvologListMaxEntries is the `-n` limit for `jj evolog` in the split modal and AI prep (deep rewrite chains).
const EvologListMaxEntries = 128

// EvologEntry is one revision line from jj evolog (newest first).
type EvologEntry struct {
	CommitIDShort string
	CommitID      string
	Summary       string
}

// ListEvolog returns evolution history for a revision (change or commit id), newest first.
func (s *Service) ListEvolog(ctx context.Context, rev string) ([]EvologEntry, error) {
	return s.listEvolog(ctx, rev, false)
}

func (s *Service) listEvolog(ctx context.Context, rev string, noHistory bool) ([]EvologEntry, error) {
	rev = strings.TrimSpace(rev)
	if rev == "" {
		return nil, fmt.Errorf("revision is required")
	}
	const tmpl = `commit.commit_id().short(8) ++ "\t" ++ commit.commit_id() ++ "\t" ++ if(commit.description(), commit.description().first_line(), "(empty)") ++ "\n"`
	var out string
	var err error
	if noHistory {
		out, err = s.runJJOutputNoHistory(ctx, "evolog", "-r", rev, "-G", "-n", strconv.Itoa(EvologListMaxEntries), "-T", tmpl)
	} else {
		out, err = s.runJJOutput(ctx, "evolog", "-r", rev, "-G", "-n", strconv.Itoa(EvologListMaxEntries), "-T", tmpl)
	}
	if err != nil {
		return nil, err
	}
	var entries []EvologEntry
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 2 {
			continue
		}
		summary := ""
		if len(parts) >= 3 {
			summary = parts[2]
		}
		entries = append(entries, EvologEntry{
			CommitIDShort: parts[0],
			CommitID:      parts[1],
			Summary:       summary,
		})
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no evolog entries for %s", rev)
	}
	return entries, nil
}

// EvologSplitDefaultMessage is the placeholder description used during FAQ-style evolog split.
const EvologSplitDefaultMessage = "Follow-up (split via evolog)"

// EvologSplitFilePeelMessage is the jj split -m text for file-level peels (distinct from hunk peels in the graph).
const EvologSplitFilePeelMessage = "Follow-up (evolog file peel)"

// EvologSplitHunkPeelMessage is the jj split -m text for each hunk-level peel round.
const EvologSplitHunkPeelMessage = "Follow-up (evolog hunk peel)"

// jjEvologSplitPrepareGlobals is prepended to read-only jj calls and to `jj git export` before `jj new`
// in MoveBookmarkDeltaOntoEvologBase so they do not snapshot the working copy between export and checkout
// (reduces intermittent "Failed to check out commit …" in colocated repos). `jj workspace update-stale`
// cannot use this flag (jj requires a writable working copy for that command).
var jjEvologSplitPrepareGlobals = []string{"--ignore-working-copy"}

func jjMergeGlobalArgs(global, args []string) []string {
	if len(global) == 0 {
		out := make([]string, len(args))
		copy(out, args)
		return out
	}
	out := make([]string, 0, len(global)+len(args))
	out = append(out, global...)
	out = append(out, args...)
	return out
}

// reconcileColocatedGitBeforeEvologSplit syncs colocated Git with jj before `jj new`.
// When Git HEAD and jj disagree (e.g. `git checkout` without `jj git import`), `jj new` can fail with
// "reference HEAD should have content …, actual content was …". `jj git export` updates Git to match jj.
func (s *Service) reconcileColocatedGitBeforeEvologSplit(ctx context.Context) error {
	// update-stale must materialize the working copy; it cannot run with --ignore-working-copy.
	if err := s.runJJ(ctx, "workspace", "update-stale"); err != nil {
		return fmt.Errorf("jj workspace update-stale: %w", err)
	}
	if err := s.runJJWithGlobal(ctx, jjEvologSplitPrepareGlobals, "git", "export"); err != nil {
		return fmt.Errorf("jj git export: %w", err)
	}
	return nil
}

func isJJColocatedHeadContentMismatch(errMsg string) bool {
	es := strings.ToLower(errMsg)
	return strings.Contains(es, "head") && strings.Contains(es, "should have content")
}

// isJJColocatedGitCheckoutFailure matches jj colocated errors when Git cannot materialize a tree
// for `jj new` (often fixable by re-exporting jj state to Git before retry).
func isJJColocatedGitCheckoutFailure(errMsg string) bool {
	es := strings.ToLower(errMsg)
	return strings.Contains(es, "failed to check out")
}

func shouldRetryEvologJjNewAfterColocatedSync(errMsg string) bool {
	return isJJColocatedHeadContentMismatch(errMsg) || isJJColocatedGitCheckoutFailure(errMsg)
}

func wrapEvologJjNewError(err error) error {
	if err == nil {
		return nil
	}
	wrapped := fmt.Errorf("jj new: %w", err)
	es := err.Error()
	if isJJColocatedHeadContentMismatch(es) {
		return fmt.Errorf("%w\n\nIn a colocated repo, Git’s HEAD can drift from jj (often after `git checkout` without jj). Try: `jj git import` to follow Git, or `jj git export` to push jj’s view to Git, then retry the split.", wrapped)
	}
	if isJJColocatedGitCheckoutFailure(es) {
		return fmt.Errorf("%w\n\nGit could not check out a revision while creating the new change (colocated repo). Try: `jj workspace update-stale` then `jj git export`, ensure the working tree is not blocked by another process, then retry the split. If you only moved HEAD with Git, run `jj git import` or `jj git export` to reconcile.\n\nNote: `jj log` may show `~` between commits when intermediate revisions are elided by the revset filter; that is not the same as a non-linear graph.", wrapped)
	}
	return wrapped
}

// evologSplitParentForNewCommit returns the revision to pass to `jj new` for an evolog split.
// If the user-picked base is empty (same tree as its parent), parenting the new commit directly
// under that base would leave a useless no-description spacer in the graph (main → … → empty → B).
// In that case we use the sole parent instead; diff(base, tip) equals diff(parent, tip), so the
// split boundary is unchanged. For merges or missing parents, base is used as-is.
// jjGlobal, when non-nil, is prepended to jj (e.g. --ignore-working-copy during evolog split prep).
func (s *Service) evologSplitParentForNewCommit(ctx context.Context, baseCommitID string, jjGlobal []string) (string, error) {
	baseCommitID = strings.TrimSpace(baseCommitID)
	emptyOut, err := s.runJJOutputNoHistoryWithGlobal(ctx, jjGlobal, "log", "-r", baseCommitID, "--no-graph", "-T", "empty", "--limit", "1")
	if err != nil {
		return "", fmt.Errorf("log empty flag: %w", err)
	}
	if strings.TrimSpace(emptyOut) != "true" {
		return baseCommitID, nil
	}
	parentsOut, err := s.runJJOutputNoHistoryWithGlobal(ctx, jjGlobal, "log", "-r", baseCommitID, "--no-graph", "-T", "parents.map(|p| p.commit_id()).join(\"\\n\")", "--limit", "1")
	if err != nil {
		return "", fmt.Errorf("log parents: %w", err)
	}
	var parents []string
	for _, line := range strings.Split(parentsOut, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			parents = append(parents, line)
		}
	}
	if len(parents) == 1 {
		return parents[0], nil
	}
	return baseCommitID, nil
}

// RevisionImmutable reports whether the given revision is immutable in jj's config.
func (s *Service) RevisionImmutable(ctx context.Context, revision string) (bool, error) {
	out, err := s.runJJOutputNoHistory(ctx, "log", "-r", revision, "--no-graph", "-T", `if(immutable, "true", "false")`, "--limit", "1")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) == "true", nil
}

// SplitRevisionByFilesets runs non-interactive `jj split -r REV -m MSG -- paths...` (filesets go into the first commit).
// Requires a jj version that supports non-interactive split with path arguments (typically jj 0.14+).
func (s *Service) SplitRevisionByFilesets(ctx context.Context, revision, firstMessage string, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	revision = strings.TrimSpace(revision)
	if revision == "" {
		revision = "@"
	}
	args := []string{"split", "-r", revision, "-m", strings.TrimSpace(firstMessage), "--"}
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p != "" {
			args = append(args, p)
		}
	}
	if len(args) <= 5 { // split -r -m msg -- only
		return nil
	}
	args = s.appendSplitInsertBeforeArgs(ctx, args, revision)
	return s.runJJ(ctx, args...)
}

// EvologMultiSplit runs several FAQ-style evolog splits in order, updating the working-copy tip after each step.
// baseCommitIDs should be ordered deepest-first (larger evolog row index first); when len > 1, ids are
// re-sorted using jj evolog for tipCH so shallow-first lists (e.g. from the LLM) still yield a linear stack.
// After the FAQ steps, splitFilesetsFirst runs first (if non-empty), then hunkPeelRounds (if non-empty), so whole-file peels
// (e.g. binaries) can precede @@-level splits on the reduced diff.
func (s *Service) EvologMultiSplit(ctx context.Context, bookmarkName, initialTipChangeID, initialTipCommitHint string, baseCommitIDs []string, splitFilesetsFirst []string, hunkPeelRounds []map[string]int) error {
	tipCH := strings.TrimSpace(initialTipChangeID)
	tipH := strings.TrimSpace(initialTipCommitHint)
	bases := append([]string(nil), baseCommitIDs...)
	if len(bases) > 1 {
		ev, err := s.ListEvolog(ctx, tipCH)
		if err != nil {
			return fmt.Errorf("evolog multi-split: load evolog for base order: %w", err)
		}
		bases = SortEvologMultiSplitBasesDeepestFirst(ev, bases)
	}
	for i, base := range bases {
		base = strings.TrimSpace(base)
		if base == "" {
			continue
		}
		if err := s.MoveBookmarkDeltaOntoEvologBase(ctx, bookmarkName, tipCH, tipH, base, nil, nil); err != nil {
			return fmt.Errorf("evolog multi-split step %d/%d: %w", i+1, len(bases), err)
		}
		var err error
		tipCH, err = s.GetRevisionChangeID(ctx, "@")
		if err != nil {
			return fmt.Errorf("wc change id after step %d/%d: %w", i+1, len(bases), err)
		}
		tipH, err = s.runJJOutputNoHistory(ctx, "log", "-r", "@", "--no-graph", "-T", "commit_id", "--limit", "1")
		if err != nil {
			return fmt.Errorf("wc commit id after step %d/%d: %w", i+1, len(bases), err)
		}
		tipH = strings.TrimSpace(tipH)
		if i+1 < len(bases) {
			nextBase := strings.TrimSpace(bases[i+1])
			// After an FAQ step, jj evolog for the tip may no longer list older intermediate ids even
			// though the commit is still addressable; require only that the next base revision resolves.
			out, exErr := s.runJJOutputNoHistoryWithGlobal(ctx, jjEvologSplitPrepareGlobals, "log", "-r", nextBase, "--no-graph", "-T", "commit_id", "--limit", "1")
			if exErr != nil || strings.TrimSpace(out) == "" {
				return fmt.Errorf("evolog multi-split step %d/%d: next base %q does not resolve after step %d (graph changed); aborting: %v", i+1, len(bases), nextBase, i+1, exErr)
			}
		}
	}
	if len(splitFilesetsFirst) > 0 {
		if err := s.SplitRevisionByFilesets(ctx, "@", EvologSplitFilePeelMessage, splitFilesetsFirst); err != nil {
			return fmt.Errorf("jj split (by file): %w", err)
		}
	}
	if len(hunkPeelRounds) > 0 {
		if err := s.SplitRevisionByHunkPeelRounds(ctx, "@", EvologSplitHunkPeelMessage, hunkPeelRounds); err != nil {
			return fmt.Errorf("jj split (by hunk): %w", err)
		}
	}
	return nil
}

// MoveBookmarkDeltaOntoEvologBase is the FAQ-style split: jj new <parent>, restore tree from the selected
// tip revision, optionally jj bookmark set, then abandon the old tip. Parent comes from the evolog base row
// the user picked in the UI (FAQ “move work onto an ancestor along this change’s evolog”), not from the
// main bookmark or remote: main stays where it is unless that picked row is the same commit as main.
// When the base is an empty revision, the parent used for jj new is that row’s parent (see evologSplitParentForNewCommit).
// A final describe @ reapplies EvologSplitDefaultMessage so the working copy never ends up with no
// description (e.g. jj metadata quirks after restore/abandon).
// If bookmarkName is empty, the selected revision is the tip (no bookmark move). If non-empty, the
// bookmark must point at the same commit as the selection (same rule as stack-on-origin flow).
// splitFilesetsFirst, when non-empty, runs `jj split -r @ -- <paths>` after the FAQ steps (before any hunk peels).
// hunkPeelRounds, when non-empty, runs hunk-scoped jj split(s) after filesets (if both are set, file peel runs first).
func (s *Service) MoveBookmarkDeltaOntoEvologBase(ctx context.Context, bookmarkName, localChangeID, localCommitID, baseCommitID string, splitFilesetsFirst []string, hunkPeelRounds []map[string]int) error {
	if strings.TrimSpace(localChangeID) == "" {
		return fmt.Errorf("local revision is required")
	}
	// Export jj commits to the Git backend early so `jj log` / `jj diff` / `jj new` all see the same
	// trees Git can check out (avoids intermittent "Failed to check out commit …" in colocated repos).
	if err := s.reconcileColocatedGitBeforeEvologSplit(ctx); err != nil {
		return fmt.Errorf("prepare evolog split (initial colocated git sync): %w", err)
	}
	baseCommitID = strings.TrimSpace(baseCommitID)
	if baseCommitID == "" {
		return fmt.Errorf("base revision is required")
	}
	revForSel := strings.TrimSpace(localCommitID)
	if revForSel == "" {
		revForSel = localChangeID
	}
	selCommitID, err := s.runJJOutputNoHistoryWithGlobal(ctx, jjEvologSplitPrepareGlobals, "log", "-r", revForSel, "--no-graph", "-T", "commit_id", "--limit", "1")
	if err != nil {
		return fmt.Errorf("selected revision: %w", err)
	}
	selCommitID = strings.TrimSpace(selCommitID)

	var tipCommitID string
	bookmarkName = strings.TrimSpace(bookmarkName)
	if bookmarkName != "" {
		tipCommitID, err = s.runJJOutputNoHistoryWithGlobal(ctx, jjEvologSplitPrepareGlobals, "log", "-r", bookmarkName, "--no-graph", "-T", "commit_id", "--limit", "1")
		if err != nil {
			return fmt.Errorf("bookmark %q: %w", bookmarkName, err)
		}
		tipCommitID = strings.TrimSpace(tipCommitID)
		if tipCommitID != selCommitID {
			return fmt.Errorf("select the bookmark tip (%s) to split", bookmarkName)
		}
	} else {
		tipCommitID = selCommitID
	}
	if commitIDsEquivalent(tipCommitID, baseCommitID) {
		return fmt.Errorf("pick an older evolog row as the split point (not the current tip)")
	}
	childrenRev := fmt.Sprintf("children(%s) ~ @", tipCommitID)
	childLines, err := s.runJJOutputWithGlobal(ctx, jjEvologSplitPrepareGlobals, "log", "-r", childrenRev, "--no-graph", "-T", "change_id", "--limit", "20")
	if err != nil {
		return fmt.Errorf("check descendants: %w", err)
	}
	for _, line := range strings.Split(childLines, "\n") {
		if strings.TrimSpace(line) != "" {
			return fmt.Errorf("commit has descendant commits (excluding working copy); rebase or squash the stack first")
		}
	}
	diffOut, err := s.runJJOutputWithGlobal(ctx, jjEvologSplitPrepareGlobals, "diff", "--from", baseCommitID, "--to", localChangeID, "--summary")
	if err != nil {
		return fmt.Errorf("diff vs base: %w", err)
	}
	if strings.TrimSpace(diffOut) == "" {
		return fmt.Errorf("tree already matches base; nothing to split")
	}
	parentForNew, err := s.evologSplitParentForNewCommit(ctx, baseCommitID, jjEvologSplitPrepareGlobals)
	if err != nil {
		return err
	}
	if commitIDsEquivalent(tipCommitID, parentForNew) {
		return fmt.Errorf("split parent would be the tip; pick a different evolog row")
	}
	const maxEvologJjNewAttempts = 3
	var newErr error
	for attempt := 0; attempt < maxEvologJjNewAttempts; attempt++ {
		if err := s.reconcileColocatedGitBeforeEvologSplit(ctx); err != nil {
			return fmt.Errorf("prepare evolog split (colocated git sync): %w", err)
		}
		newErr = s.runJJ(ctx, "new", parentForNew, "-m", EvologSplitDefaultMessage)
		if newErr == nil {
			break
		}
		if attempt+1 >= maxEvologJjNewAttempts || !shouldRetryEvologJjNewAfterColocatedSync(newErr.Error()) {
			return wrapEvologJjNewError(newErr)
		}
		time.Sleep(150 * time.Millisecond)
	}
	if err := s.runJJ(ctx, "restore", "--into", "@", "--from", tipCommitID); err != nil {
		return fmt.Errorf("jj restore: %w", err)
	}
	if bookmarkName != "" {
		if err := s.runJJ(ctx, "bookmark", "set", util.BookmarkArgForSetMove(bookmarkName), "-r", "@", "--allow-backwards"); err != nil {
			return fmt.Errorf("jj bookmark set: %w", err)
		}
	}
	_ = s.runJJ(ctx, "abandon", tipCommitID)
	if err := s.DescribeCommit(ctx, "@", EvologSplitDefaultMessage); err != nil {
		return fmt.Errorf("jj describe: %w", err)
	}
	if len(splitFilesetsFirst) > 0 {
		if err := s.SplitRevisionByFilesets(ctx, "@", EvologSplitFilePeelMessage, splitFilesetsFirst); err != nil {
			return fmt.Errorf("jj split (by file): %w", err)
		}
	}
	if len(hunkPeelRounds) > 0 {
		if err := s.SplitRevisionByHunkPeelRounds(ctx, "@", EvologSplitHunkPeelMessage, hunkPeelRounds); err != nil {
			return fmt.Errorf("jj split (by hunk): %w", err)
		}
	}
	return nil
}

func commitIDsEquivalent(a, b string) bool {
	a, b = strings.TrimSpace(a), strings.TrimSpace(b)
	if a == b {
		return true
	}
	if len(a) >= 8 && len(b) >= 8 && (strings.HasPrefix(a, b) || strings.HasPrefix(b, a)) {
		return true
	}
	return false
}

// RevertFile reverts the changes to a file in a given commit,
// restoring it from the commit's parent.
func (s *Service) RevertFile(ctx context.Context, commitID, filePath string) error {
	// jj restore --to <commit> --from parents(<commit>) -- <file>
	// Using parents() function instead of ~ suffix to avoid revset parsing issues
	parentRev := fmt.Sprintf("parents(%s)", commitID)
	return s.runJJ(ctx, "restore", "--to", commitID, "--from", parentRev, "--", filePath)
}

// GetGitRemoteURL returns the URL of the git remote (origin)
func (s *Service) GetGitRemoteURL(ctx context.Context) (string, error) {
	out, err := s.runJJOutput(ctx, "git", "remote", "list")
	if err != nil {
		return "", fmt.Errorf("failed to list git remotes: %w", err)
	}

	// Parse the output - format is "remote_name url"
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			// Prefer "origin" remote, but take the first one if no origin
			if parts[0] == "origin" {
				return parts[1], nil
			}
		}
	}

	// Return first remote if no origin found
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			return parts[1], nil
		}
	}

	return "", fmt.Errorf("no git remotes found")
}

// GetCurrentBranch returns the current bookmark/branch being tracked
func (s *Service) GetCurrentBranch(ctx context.Context) (string, error) {
	// Get bookmarks that point to the current working copy or its ancestors
	out, err := s.runJJOutput(ctx, "log", "-r", "@", "--no-graph", "-T", "bookmarks")
	if err != nil {
		return "", err
	}

	branch := strings.TrimSpace(out)
	if branch == "" {
		// No bookmark on @, check parent
		out, err = s.runJJOutput(ctx, "log", "-r", "@-", "--no-graph", "-T", "bookmarks")
		if err != nil {
			return "", err
		}
		branch = strings.TrimSpace(out)
	}

	if branch == "" {
		return "main", nil // Default to main
	}

	// If multiple bookmarks, take the first one
	parts := strings.Split(branch, " ")
	return parts[0], nil
}

// PushToGit pushes the current branch to the git remote
// Returns the push output for debugging
func (s *Service) PushToGit(ctx context.Context, branch string) (string, error) {
	// First verify the bookmark exists
	out, err := s.runJJOutput(ctx, "bookmark", "list", "--all")
	if err != nil {
		return "", fmt.Errorf("failed to list bookmarks: %w", err)
	}

	// Check if our bookmark is in the list
	bookmarkExists := false
	for _, line := range strings.Split(out, "\n") {
		// Bookmark list format: "bookmarkname: revision"
		// or just "bookmarkname" if it's at the working copy
		// May have * suffix for current bookmark
		parts := strings.SplitN(line, ":", 2)
		if len(parts) > 0 {
			name := strings.TrimSpace(parts[0])
			// Strip * suffix (indicates current bookmark)
			name = strings.TrimSuffix(name, "*")
			// Strip any @remote suffix
			if idx := strings.Index(name, "@"); idx > 0 {
				name = name[:idx]
			}
			if name == branch {
				bookmarkExists = true
				break
			}
		}
	}

	if !bookmarkExists {
		return "", fmt.Errorf("bookmark '%s' does not exist. Create it first with 'm' (Bookmark)", branch)
	}

	// --allow-new permits creating new remote bookmarks
	// Use runJJOutput to capture any output/errors
	pushOut, err := s.runJJOutput(ctx, "git", "push", "--bookmark", util.JJExactBookmarkPattern(branch), "--allow-new")
	if err != nil {
		return pushOut, fmt.Errorf("push failed: %w", err)
	}

	// Also run a direct git push to ensure the branch is synced
	// This helps when jj's git integration has timing issues
	gitPushCmd := exec.CommandContext(ctx, "git", "push", "origin", branch)
	gitPushCmd.Dir = s.RepoPath
	gitOut, gitErr := gitPushCmd.CombinedOutput()
	if gitErr != nil {
		// If git push fails with "up to date", that's fine
		if !strings.Contains(string(gitOut), "up-to-date") && !strings.Contains(string(gitOut), "Everything up-to-date") {
			pushOut += "\nGit push output: " + string(gitOut)
		}
	}

	return pushOut, nil
}

// FetchFromGit fetches updates from the remote git repository.
// When jj git fetch fails (e.g. "Failed to update refs" with many remotes), we fall back to
// git fetch origin so callers can still compare to bookmark@origin without a blocking error.
func (s *Service) FetchFromGit(ctx context.Context) (string, error) {
	out, err := s.runJJOutput(ctx, "git", "fetch")
	if err != nil {
		gitOut, gitErr := s.runGitFetchOrigin(ctx)
		if gitErr == nil {
			_ = s.cleanupAfterFetch(ctx)
			return out + string(gitOut), nil
		}
		return out, fmt.Errorf("fetch failed: %w", err)
	}

	gitOut, gitErr := s.runGitFetchOrigin(ctx)
	if gitErr != nil {
		// Fetch failures are usually not fatal (e.g., no new changes)
		// Only append output if it's a real network/permission issue
		sGit := string(gitOut)
		if !strings.Contains(sGit, "Fetching from") && !strings.Contains(sGit, "up-to-date") {
			out += "\nGit fetch output: " + sGit
		}
	}

	_ = s.cleanupAfterFetch(ctx)

	return out, nil
}

func (s *Service) runGitFetchOrigin(ctx context.Context) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", "fetch", "origin")
	cmd.Dir = s.RepoPath
	return cmd.CombinedOutput()
}

// cleanupAfterFetch handles post-fetch cleanup:
// 1. Moves working copy if it's on an immutable commit
func (s *Service) cleanupAfterFetch(ctx context.Context) error {
	// First, move working copy if it's immutable
	isImmutable, _ := s.runJJOutput(ctx, "log", "-r", "@", "--no-graph", "-T", "if(immutable, \"true\", \"false\")")
	if strings.TrimSpace(isImmutable) == "true" {
		// Working copy is immutable (e.g., after a merge). Create a new mutable descendant.
		_ = s.runJJ(ctx, "new", "@")
	}
	return nil
}

// DefaultGraphRevset is used when config graph_revset is empty.
//
// We intersect mutable() with (ancestors(@) | descendants(@)) so the graph is tied to the
// working copy's DAG neighborhood. Bare mutable() | bookmarks() | main@origin matches every
// mutable revision repo-wide (including unrelated merged branches, stale divergent pairs, and
// teammates' old lines), which is overwhelming in large colocated repos like access.
// descendants(@) keeps move-to-parent / move-to-child splits visible.
const DefaultGraphRevset = `(mutable() & (ancestors(@) | descendants(@))) | bookmarks() | main@origin`

// Caps for per-commit jj subprocess work during getCommitGraph. After the main jj log, we run
// enrichCommitsDeltaVsOrigin and enrichCommitsEvologSplitViable; each mutable commit with a feature
// bookmark can trigger several jj log/diff/evolog calls. Large revsets (deep ancestors(@), many
// bookmarks) can list 100+ rows and make startup feel hung without these limits.
const (
	graphLoadMaxDeltaVsOriginProbes = 64
	graphLoadMaxEvologSplitProbes   = 36
)

// jjLogWithGraphTemplate runs jj log with the graph ASCII template; recordInHistory controls command history.
func (s *Service) jjLogWithGraphTemplate(ctx context.Context, recordInHistory bool, revsetArg, template string) (string, error) {
	if recordInHistory {
		return s.runJJOutput(ctx, "log", "-r", revsetArg, "-T", template)
	}
	return s.runJJOutputNoHistory(ctx, "log", "-r", revsetArg, "-T", template)
}

// getCommitGraph retrieves the commit graph with real jj data.
// revset: if non-empty, used as the -r revset; if empty, a default is used.
// recordGraphInHistory: when false, the primary (and fallback) jj log calls are not added to command history.
func (s *Service) getCommitGraph(ctx context.Context, revset string, recordGraphInHistory bool) (*internal.CommitGraph, error) {
	// Use a custom template with a unique marker to separate graph prefix from data
	// The marker "<<<COMMIT>>>" lets us identify where the graph ends and data begins
	// Format after marker: change_id|commit_id|author|date|description|parents|bookmarks|is_working|has_conflict|immutable|divergent
	template := `concat(
		"<<<COMMIT>>>",
		change_id.short(8), "|",
		commit_id.short(8), "|",
		author.email(), "|",
		author.timestamp(), "|",
		if(description, description.first_line(), "(no description)"), "|",
		parents.map(|p| p.commit_id().short(8)).join(","), "|",
		bookmarks.join(","), "|",
		if(self.current_working_copy(), "true", "false"), "|",
		if(self.conflict(), "true", "false"), "|",
		if(immutable, "true", "false"), "|",
		if(divergent, "true", "false"),
		"\n"
	)`

	// Run bookmark list concurrently with log; enrichment needs it later and it does not depend on log output.
	var bmOut string
	var bmErr error
	var bmWG sync.WaitGroup
	bmWG.Add(1)
	go func() {
		defer bmWG.Done()
		bmOut, bmErr = s.runJJOutputNoHistory(ctx, "bookmark", "list", "--all-remotes")
	}()

	// Run WITH the graph to get ASCII art (no --reversed, keep natural newest-first order)
	var revsetArg string
	if revset != "" {
		revsetArg = revset
	} else {
		revsetArg = DefaultGraphRevset
	}
	out, err := s.jjLogWithGraphTemplate(ctx, recordGraphInHistory, revsetArg, template)
	if err != nil {
		if revset != "" {
			// Custom revset failed; try a broad safe revset so the app still loads
			out, err = s.jjLogWithGraphTemplate(ctx, recordGraphInHistory, "mutable() | bookmarks()", template)
		} else {
			// Default may fail if main@origin is missing; omit trunk tip from the revset
			out, err = s.jjLogWithGraphTemplate(ctx, recordGraphInHistory, "mutable() | bookmarks()", template)
		}
	}
	bmWG.Wait()
	if err != nil {
		return s.getCommitGraphSimple(ctx, revset, recordGraphInHistory)
	}

	commits := []internal.Commit{}
	connections := make(map[string][]string)
	var pendingGraphLines []string // Graph lines between commits

	lines := strings.Split(out, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		// Check if this line contains commit data (has our marker)
		markerIdx := strings.Index(line, "<<<COMMIT>>>")
		if markerIdx == -1 {
			// This is a graph-only line (connector between commits)
			// Store it to attach to the previous commit (connects it to the next one below)
			graphLine := strings.TrimRight(line, " ")
			if graphLine != "" {
				pendingGraphLines = append(pendingGraphLines, graphLine)
			}
			continue
		}

		// Attach pending graph lines to the previous commit first
		if len(commits) > 0 && len(pendingGraphLines) > 0 {
			commits[len(commits)-1].GraphLines = pendingGraphLines
			pendingGraphLines = nil
		}

		// Extract the graph prefix (everything before the marker)
		graphPrefix := line[:markerIdx]

		// Extract the data (everything after the marker)
		data := line[markerIdx+len("<<<COMMIT>>>"):]

		parts := strings.Split(data, "|")
		if len(parts) < 11 {
			continue
		}

		changeID := strings.TrimSpace(parts[0])
		commitID := strings.TrimSpace(parts[1])
		author := strings.TrimSpace(parts[2])
		dateStr := strings.TrimSpace(parts[3])
		description := strings.TrimSpace(parts[4])
		parentsStr := strings.TrimSpace(parts[5])
		branchesStr := strings.TrimSpace(parts[6])
		isWorking := strings.TrimSpace(parts[7]) == "true"
		hasConflict := strings.TrimSpace(parts[8]) == "true"
		isImmutable := strings.TrimSpace(parts[9]) == "true"
		isDivergent := strings.TrimSpace(parts[10]) == "true"

		// Parse parents
		var parents []string
		if parentsStr != "" {
			parents = strings.Split(parentsStr, ",")
		}

		// Parse branches/bookmarks
		// Strip @remote suffixes (e.g., "main@origin" -> "main")
		// Strip * suffix (indicates current bookmark)
		// Track ? suffix (indicates conflicted/diverged bookmark)
		var branches []string
		var conflictedBranches []string
		if branchesStr != "" {
			for _, raw := range strings.Split(branchesStr, ",") {
				b, isConflicted := util.NormalizeBookmarkListToken(raw)
				// Keep @remote suffixes (e.g. feature@origin) so the graph can distinguish
				// local bookmark tips from remote-tracking positions on different commits.
				// Avoid duplicates
				found := false
				for _, existing := range branches {
					if existing == b {
						found = true
						break
					}
				}
				if !found && b != "" {
					branches = append(branches, b)
					if isConflicted {
						conflictedBranches = append(conflictedBranches, b)
					}
				}
			}
		}

		// Parse date
		var date time.Time
		if dateStr != "" {
			date, _ = time.Parse(time.RFC3339, dateStr)
		}

		commit := internal.Commit{
			ID:                 commitID,
			ShortID:            commitID,
			ChangeID:           changeID,
			Author:             author,
			Email:              author,
			Date:               date,
			Summary:            description,
			Description:        description,
			Parents:            parents,
			Branches:           branches,
			ConflictedBranches: conflictedBranches,
			IsWorking:          isWorking,
			Conflicts:          hasConflict,
			Immutable:          isImmutable,
			Divergent:          isDivergent,
			GraphPrefix:        graphPrefix,
		}

		commits = append(commits, commit)

		// Build connections
		for _, parent := range parents {
			connections[parent] = append(connections[parent], commitID)
		}
	}

	// Attach any remaining graph lines to the last commit
	if len(commits) > 0 && len(pendingGraphLines) > 0 {
		commits[len(commits)-1].GraphLines = pendingGraphLines
	}

	originDiverged := map[string]bool{}
	var suppressForkAfterAheadBehindList map[string]bool
	if bmErr == nil {
		stated, ahBoth := bookmarkListParseOriginDivergence(bmOut)
		originDiverged = s.originDivergedResolved(ctx, stated, ahBoth)
		// jj may print both ahead and behind after merges while tips stay linear; do not let
		// bookmarkDivergedFromOrigin re-add those as conflicts when the fork check already declined.
		suppressForkAfterAheadBehindList = make(map[string]bool)
		for k := range ahBoth {
			k = strings.TrimSpace(k)
			if k != "" && !originDiverged[k] {
				suppressForkAfterAheadBehindList[k] = true
			}
		}
	}
	s.enrichConflictedBookmarks(ctx, commits, originDiverged, suppressForkAfterAheadBehindList)
	s.enrichCommitsDeltaVsOrigin(ctx, commits)
	s.enrichCommitsEvologSplitViable(ctx, commits)

	return &internal.CommitGraph{
		Commits:     commits,
		Connections: connections,
	}, nil
}

// enrichCommitsEvologSplitViable sets EvologSplitViable for mutable commits (cached per change id).
func (s *Service) enrichCommitsEvologSplitViable(ctx context.Context, commits []internal.Commit) {
	cache := make(map[string]bool)
	probes := 0
	for i := range commits {
		c := &commits[i]
		if c.Immutable || c.Divergent || len(c.ConflictedBranches) > 0 || c.Conflicts {
			continue
		}
		ch := strings.TrimSpace(c.ChangeID)
		if ch == "" {
			continue
		}
		if v, ok := cache[ch]; ok {
			c.EvologSplitViable = v
			continue
		}
		if graphLoadMaxEvologSplitProbes > 0 && probes >= graphLoadMaxEvologSplitProbes {
			c.EvologSplitViable = false
			continue
		}
		probes++
		v := s.commitEvologSplitViable(ctx, *c)
		cache[ch] = v
		c.EvologSplitViable = v
	}
}

// commitEvologSplitViable mirrors MoveBookmarkDeltaOntoEvologBase: no non–working-copy children on the
// tip, jj evolog has at least two rows, and some older row has a non-empty tree diff vs this change.
func (s *Service) commitEvologSplitViable(ctx context.Context, c internal.Commit) bool {
	revForSel := strings.TrimSpace(c.ID)
	if revForSel == "" {
		return false
	}
	selCommitID, err := s.runJJOutputNoHistory(ctx, "log", "-r", revForSel, "--no-graph", "-T", "commit_id", "--limit", "1")
	if err != nil {
		return false
	}
	selCommitID = strings.TrimSpace(selCommitID)
	if selCommitID == "" {
		return false
	}

	bn := eligibleBookmarkForOriginDelta(c.Branches)
	var tipCommitID string
	if bn != "" {
		tipCommitID, err = s.runJJOutputNoHistory(ctx, "log", "-r", bn, "--no-graph", "-T", "commit_id", "--limit", "1")
		if err != nil {
			return false
		}
		tipCommitID = strings.TrimSpace(tipCommitID)
		if !commitIDsEquivalent(tipCommitID, selCommitID) {
			return false
		}
	} else {
		tipCommitID = selCommitID
	}

	childrenRev := fmt.Sprintf("children(%s) ~ @", tipCommitID)
	childLines, err := s.runJJOutputNoHistory(ctx, "log", "-r", childrenRev, "--no-graph", "-T", "change_id", "--limit", "20")
	if err != nil {
		return false
	}
	for _, line := range strings.Split(childLines, "\n") {
		if strings.TrimSpace(line) != "" {
			return false
		}
	}

	entries, err := s.listEvolog(ctx, c.ChangeID, true)
	if err != nil || len(entries) < 2 {
		return false
	}
	tipEvolog := strings.TrimSpace(entries[0].CommitID)
	for i := 1; i < len(entries); i++ {
		baseID := strings.TrimSpace(entries[i].CommitID)
		if baseID == "" || commitIDsEquivalent(baseID, tipEvolog) {
			continue
		}
		ok, err := s.revisionDiffSummaryNonEmptyNoHistory(ctx, baseID, c.ChangeID)
		if err != nil {
			continue
		}
		if ok {
			return true
		}
	}
	return false
}

// eligibleBookmarkForOriginDelta returns the first non–default-branch bookmark name on a commit, for comparing to *@origin.
func eligibleBookmarkForOriginDelta(branches []string) string {
	for _, b := range branches {
		b = strings.TrimSpace(b)
		if b == "" {
			continue
		}
		local := util.LocalBookmarkName(b)
		switch strings.ToLower(local) {
		case "main", "master":
			continue
		}
		return local
	}
	return ""
}

// descendantRevsetForOriginEnrichment is the jj revset for this graph row when probing bookmark@origin
// ancestry and tree diffs. Uses commit_id(...) so divergent changes (multiple revisions, one change ID)
// do not make `jj diff` / `x::y` ambiguous or error.
func descendantRevsetForOriginEnrichment(c internal.Commit) string {
	if id := strings.TrimSpace(c.ID); id != "" {
		return revsetCommitID(id)
	}
	return strings.TrimSpace(c.ChangeID)
}

func (s *Service) enrichCommitsDeltaVsOrigin(ctx context.Context, commits []internal.Commit) {
	probes := 0
	for i := range commits {
		c := &commits[i]
		if c.Immutable {
			continue
		}
		bn := eligibleBookmarkForOriginDelta(c.Branches)
		if bn == "" {
			continue
		}
		descRev := descendantRevsetForOriginEnrichment(*c)
		if descRev == "" {
			continue
		}
		if graphLoadMaxDeltaVsOriginProbes > 0 && probes >= graphLoadMaxDeltaVsOriginProbes {
			c.HasDeltaVsBookmarkOrigin = false
			continue
		}
		probes++
		// Already stacked on bookmark@origin (origin tip is an ancestor of this revision): push
		// updates the remote bookmark; "(f)" restack is redundant (e.g. right after MoveBookmarkDeltaOntoOrigin).
		if s.revisionBookmarkOriginIsAncestorOf(ctx, bn, descRev) {
			c.HasDeltaVsBookmarkOrigin = false
			continue
		}
		// Non-empty tree diff vs bookmark@origin and not in the ancestry chain above → offer Forgot.
		ok, err := s.revisionDiffSummaryNonEmptyNoHistory(ctx, bn+"@origin", descRev)
		if err != nil || !ok {
			c.HasDeltaVsBookmarkOrigin = false
			continue
		}
		c.HasDeltaVsBookmarkOrigin = true
	}
}

// revisionBookmarkOriginIsAncestorOf is true when bn@origin lies on the ancestry of descendantRev
// (jj revset x::y: commits below x and above y). Then the revision is already built on top of the
// remembered remote tip — not the colocated "forgot to stack" case.
// Pass commit_id(...) via descendantRevsetForOriginEnrichment when the row may be divergent.
func (s *Service) revisionBookmarkOriginIsAncestorOf(ctx context.Context, bookmarkLocalName, descendantRev string) bool {
	bookmarkLocalName = strings.TrimSpace(bookmarkLocalName)
	descendantRev = strings.TrimSpace(descendantRev)
	if bookmarkLocalName == "" || descendantRev == "" {
		return false
	}
	originRef := bookmarkLocalName + "@origin"
	rev := fmt.Sprintf("%s::%s", originRef, descendantRev)
	out, err := s.runJJOutputNoHistory(ctx, "log", "-r", rev, "--no-graph", "-T", "commit_id", "--limit", "1")
	return err == nil && strings.TrimSpace(out) != ""
}

func (s *Service) revisionDiffSummaryNonEmptyNoHistory(ctx context.Context, fromRef, toRev string) (bool, error) {
	out, err := s.runJJOutputNoHistory(ctx, "diff", "--from", fromRef, "--to", toRev, "--summary")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// runJJOutputNoHistory runs jj without recording command history (used for graph enrichment).
func (s *Service) runJJOutputNoHistory(ctx context.Context, args ...string) (string, error) {
	return s.runJJOutputNoHistoryWithGlobal(ctx, nil, args...)
}

// runJJOutputNoHistoryWithGlobal is like runJJOutputNoHistory but prepends global jj flags.
func (s *Service) runJJOutputNoHistoryWithGlobal(ctx context.Context, global []string, args ...string) (string, error) {
	merged := jjMergeGlobalArgs(global, args)
	cmd := exec.CommandContext(ctx, "jj", merged...)
	cmd.Dir = s.RepoPath
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		errOut := stderr.String()
		if errOut == "" {
			errOut = stdout.String()
		}
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(errOut))
	}
	return stdout.String(), nil
}

// getCommitGraphSimple is a fallback that uses simpler parsing
func (s *Service) getCommitGraphSimple(ctx context.Context, revset string, recordInHistory bool) (*internal.CommitGraph, error) {
	revsetArg := "mutable() | bookmarks()"
	if revset != "" {
		revsetArg = revset
	}
	var out string
	var err error
	if recordInHistory {
		out, err = s.runJJOutput(ctx, "log", "-r", revsetArg, "--no-graph")
	} else {
		out, err = s.runJJOutputNoHistory(ctx, "log", "-r", revsetArg, "--no-graph")
	}
	if err != nil {
		return nil, err
	}

	commits := []internal.Commit{}

	// Parse the default jj log output
	lines := strings.Split(out, "\n")
	var currentCommit *internal.Commit

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if currentCommit != nil {
				commits = append(commits, *currentCommit)
				currentCommit = nil
			}
			continue
		}

		// Lines starting with @ or ○ indicate a commit
		if strings.HasPrefix(line, "@") || strings.HasPrefix(line, "○") || strings.HasPrefix(line, "◆") {
			if currentCommit != nil {
				commits = append(commits, *currentCommit)
			}

			isWorking := strings.HasPrefix(line, "@")
			// Remove the prefix and parse the rest
			line = strings.TrimLeft(line, "@○◆ ")

			// Try to parse: change_id commit_id author date summary
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				currentCommit = &internal.Commit{
					ChangeID:  parts[0],
					ShortID:   parts[0],
					ID:        parts[0],
					IsWorking: isWorking,
				}

				if len(parts) >= 3 {
					currentCommit.Author = parts[1]
				}
				if len(parts) >= 4 {
					// Rest is the summary
					currentCommit.Summary = strings.Join(parts[3:], " ")
				}
			}
		} else if currentCommit != nil && currentCommit.Summary == "" {
			// This might be a continuation line with the summary
			currentCommit.Summary = line
		}
	}

	if currentCommit != nil {
		commits = append(commits, *currentCommit)
	}

	s.enrichCommitsDeltaVsOrigin(ctx, commits)
	s.enrichCommitsEvologSplitViable(ctx, commits)

	return &internal.CommitGraph{
		Commits:     commits,
		Connections: make(map[string][]string),
	}, nil
}

// runJJWithGlobal runs jj with optional global flags before subcommand (e.g. --ignore-working-copy).
func (s *Service) runJJWithGlobal(ctx context.Context, global []string, args ...string) error {
	merged := jjMergeGlobalArgs(global, args)
	cmdStr := "jj " + strings.Join(merged, " ")
	startTime := time.Now()

	cmd := exec.CommandContext(ctx, "jj", merged...)
	cmd.Dir = s.RepoPath
	out, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	entry := CommandHistoryEntry{
		Command:   cmdStr,
		Timestamp: startTime,
		Duration:  duration,
		Success:   err == nil,
	}
	if err != nil {
		errMsg := extractErrorMessage(string(out))
		if errMsg != "" {
			entry.Error = errMsg
			s.addToHistory(entry)
			return fmt.Errorf("%s", errMsg)
		}
		entry.Error = err.Error()
		s.addToHistory(entry)
		return fmt.Errorf("command failed: %w", err)
	}

	s.addToHistory(entry)
	return nil
}

// runJJOutputWithGlobal is like runJJOutput but prepends global jj flags.
func (s *Service) runJJOutputWithGlobal(ctx context.Context, global []string, args ...string) (string, error) {
	merged := jjMergeGlobalArgs(global, args)
	cmdStr := "jj " + strings.Join(merged, " ")
	startTime := time.Now()

	cmd := exec.CommandContext(ctx, "jj", merged...)
	cmd.Dir = s.RepoPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(startTime)

	entry := CommandHistoryEntry{
		Command:   cmdStr,
		Timestamp: startTime,
		Duration:  duration,
		Success:   err == nil,
	}

	if err != nil {
		errOutput := stderr.String()
		if errOutput == "" {
			errOutput = stdout.String()
		}
		entry.Error = extractErrorMessage(errOutput)
		if entry.Error == "" {
			entry.Error = err.Error()
		}
		s.addToHistory(entry)
		return "", fmt.Errorf("jj command '%s' failed: %w\nOutput: %s",
			cmdStr, err, errOutput)
	}

	s.addToHistory(entry)
	return stdout.String(), nil
}

// runJJ executes a jj command and returns a clean error if it fails
func (s *Service) runJJ(ctx context.Context, args ...string) error {
	cmdStr := "jj " + strings.Join(args, " ")
	startTime := time.Now()

	cmd := exec.CommandContext(ctx, "jj", args...)
	cmd.Dir = s.RepoPath
	out, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	// Log the command to history
	entry := CommandHistoryEntry{
		Command:   cmdStr,
		Timestamp: startTime,
		Duration:  duration,
		Success:   err == nil,
	}
	if err != nil {
		// Extract just the main error message
		errMsg := extractErrorMessage(string(out))
		if errMsg != "" {
			entry.Error = errMsg
			s.addToHistory(entry)
			return fmt.Errorf("%s", errMsg)
		}
		entry.Error = err.Error()
		s.addToHistory(entry)
		return fmt.Errorf("command failed: %w", err)
	}

	s.addToHistory(entry)
	return nil
}

// extractErrorMessage extracts the main error message from jj output
func extractErrorMessage(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Error:") {
			return strings.TrimPrefix(line, "Error: ")
		}
	}
	// Return first non-empty, non-warning line
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "Warning:") && !strings.HasPrefix(line, "Hint:") {
			return line
		}
	}
	return ""
}

// runJJOutput executes a jj command and returns its stdout only
// stderr is captured separately to avoid jj hints/warnings mixing into parsed output
func (s *Service) runJJOutput(ctx context.Context, args ...string) (string, error) {
	cmdStr := "jj " + strings.Join(args, " ")
	startTime := time.Now()

	cmd := exec.CommandContext(ctx, "jj", args...)
	cmd.Dir = s.RepoPath

	// Capture stdout and stderr separately
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(startTime)

	// Log the command to history
	entry := CommandHistoryEntry{
		Command:   cmdStr,
		Timestamp: startTime,
		Duration:  duration,
		Success:   err == nil,
	}

	if err != nil {
		// Include stderr in error message for debugging
		errOutput := stderr.String()
		if errOutput == "" {
			errOutput = stdout.String()
		}
		entry.Error = extractErrorMessage(errOutput)
		if entry.Error == "" {
			entry.Error = err.Error()
		}
		s.addToHistory(entry)
		return "", fmt.Errorf("jj command '%s' failed: %w\nOutput: %s",
			fmt.Sprintf("jj %s", strings.Join(args, " ")), err, errOutput)
	}

	s.addToHistory(entry)
	// Return only stdout - hints/warnings go to stderr
	return stdout.String(), nil
}

// ListBranches returns all local and remote branches
// statsLimit controls how many branches get ahead/behind stats calculated (0 = all)
func (s *Service) ListBranches(ctx context.Context, statsLimit int) ([]internal.Branch, error) {
	// Get all bookmarks including remote ones
	out, err := s.runJJOutput(ctx, "bookmark", "list", "--all-remotes")
	if err != nil {
		return nil, fmt.Errorf("failed to list bookmarks: %w", err)
	}

	var branches []internal.Branch
	lines := strings.Split(out, "\n")

	var currentBranch string
	var isDeleted bool

	for _, line := range lines {
		if line == "" {
			continue
		}

		// Check if this is a new branch line (not indented - doesn't start with space)
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			// Reset state for new branch
			isDeleted = strings.Contains(line, "(deleted)")

			// Check if this is an untracked remote branch (format: "branch@origin: ...")
			// These have @ in the name without space before it
			if strings.Contains(line, "@") && !strings.Contains(line, "(deleted)") {
				// Format: "branch@origin: change_id commit_id description"
				atIdx := strings.Index(line, "@")
				colonIdx := strings.Index(line, ":")
				if atIdx >= 0 && colonIdx > atIdx {
					branchName := line[:atIdx]
					remote := line[atIdx+1 : colonIdx]

					// Skip git remote
					if remote == "git" {
						currentBranch = ""
						continue
					}

					commitInfo := strings.TrimSpace(line[colonIdx+1:])
					changeID, shortID := parseCommitInfo(commitInfo)

					branches = append(branches, internal.Branch{
						Name:      branchName,
						Remote:    remote,
						CommitID:  changeID,
						ShortID:   shortID,
						IsTracked: false, // Untracked remote branch
						IsLocal:   false,
					})
					currentBranch = ""
					continue
				}
			}

			// Check if this is a local branch with commit info
			// Format: "branch-name: change_id commit_id description"
			// or "branch-name (deleted)"
			// May have ? suffix indicating conflict (local/remote diverged)
			colonIdx := strings.Index(line, ":")
			if colonIdx > 0 && !isDeleted {
				// Local branch with commit info on same line
				rawBranchName := strings.TrimSpace(line[:colonIdx])
				lineHead := strings.ToLower(strings.TrimSpace(line[:colonIdx+1]))
				normalizedName, fromQuestionMark := util.NormalizeBookmarkListToken(rawBranchName)
				// Conflict: jj often marks with ? on the name; some versions mention it in the header.
				hasConflict := fromQuestionMark ||
					strings.Contains(lineHead, "conflict") ||
					strings.Contains(lineHead, "diverg")
				currentBranch = normalizedName
				commitInfo := strings.TrimSpace(line[colonIdx+1:])
				changeID, shortID := parseCommitInfo(commitInfo)

				branches = append(branches, internal.Branch{
					Name:        currentBranch,
					CommitID:    changeID,
					ShortID:     shortID,
					IsLocal:     true,
					HasConflict: hasConflict,
				})
			} else if isDeleted {
				// Deleted local branch - extract name
				currentBranch = strings.TrimSpace(strings.TrimSuffix(line, " (deleted)"))
			} else {
				// Branch name only (rare case)
				currentBranch = strings.TrimSpace(line)
			}
		} else if strings.HasPrefix(strings.TrimSpace(line), "@") {
			// Remote tracking line (indented): "  @origin: …" or "  @origin (ahead…behind…): …"
			trimmedLine := strings.TrimSpace(line)
			remote, commitInfo, ok := parseBookmarkListRemoteLine(trimmedLine)
			if !ok {
				continue
			}
			if remote == "git" {
				continue
			}

			changeID, shortID := parseCommitInfo(commitInfo)

			// Only add remote branch if we have a current branch name
			if currentBranch != "" {
				// A branch is tracked if it appears under a branch line (even if deleted)
				// Untracked branches appear on a single line as "branch@origin:"
				branches = append(branches, internal.Branch{
					Name:         currentBranch,
					LocalDeleted: isDeleted, // Track if local copy was deleted
					Remote:       remote,
					CommitID:     changeID,
					ShortID:      shortID,
					IsTracked:    true, // Always tracked if shown as indented @remote: line
					IsLocal:      false,
				})
			}
		}
	}

	// Optimization: Filter remote branches by recency, always keep local branches
	// Also keep remote counterparts of local branches
	if statsLimit > 0 {
		// Build a set of local branch names to keep their remote counterparts
		localBranchNames := make(map[string]bool)
		var localBranches, remoteBranches, remoteCounterparts []internal.Branch

		for _, b := range branches {
			if b.IsLocal {
				localBranches = append(localBranches, b)
				localBranchNames[b.Name] = true
			}
		}

		// Separate remote branches: counterparts of local vs others
		for _, b := range branches {
			if !b.IsLocal {
				if localBranchNames[b.Name] {
					// This is a remote counterpart of a local branch - always keep
					remoteCounterparts = append(remoteCounterparts, b)
				} else {
					remoteBranches = append(remoteBranches, b)
				}
			}
		}

		// Calculate remaining slots for other remote branches
		remoteLimit := max(statsLimit-len(localBranches)-len(remoteCounterparts), 0)

		if len(remoteBranches) > remoteLimit && remoteLimit > 0 {
			// Query timestamps for remote branches using branch ref directly
			// Format: "branch@remote|timestamp\n"
			var revsets []string
			branchToRef := make(map[string]string) // ref -> branch index key
			for _, b := range remoteBranches {
				if b.Remote != "" {
					ref := fmt.Sprintf("%s@%s", b.Name, b.Remote)
					revsets = append(revsets, ref)
					branchToRef[ref] = ref
				}
			}

			if len(revsets) > 0 {
				// Use a different template that includes the branch ref for matching
				revset := strings.Join(revsets, " | ")
				out, err := s.runJJOutput(ctx, "log", "-r", revset, "--no-graph",
					"-T", `if(bookmarks, bookmarks ++ "|" ++ committer.timestamp().utc().format("%s") ++ "\n", "")`)
				if err == nil {
					// Parse timestamps into a map keyed by branch@remote
					timestamps := make(map[string]int64)
					for _, line := range strings.Split(out, "\n") {
						line = strings.TrimSpace(line)
						if line == "" {
							continue
						}
						parts := strings.Split(line, "|")
						if len(parts) == 2 {
							branchRef := strings.TrimSpace(parts[0])
							if ts, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64); err == nil {
								timestamps[branchRef] = ts
							}
						}
					}

					// Sort remote branches by timestamp (most recent first)
					sort.Slice(remoteBranches, func(i, j int) bool {
						refI := fmt.Sprintf("%s@%s", remoteBranches[i].Name, remoteBranches[i].Remote)
						refJ := fmt.Sprintf("%s@%s", remoteBranches[j].Name, remoteBranches[j].Remote)
						tsI, okI := timestamps[refI]
						tsJ, okJ := timestamps[refJ]
						// Branches with timestamps come before those without
						if okI != okJ {
							return okI
						}
						return tsI > tsJ // Descending - most recent first
					})
				}
				// If timestamp query fails, branches stay in original order
			}

			// Keep only the N most recent remote branches
			remoteBranches = remoteBranches[:remoteLimit]
		} else if len(remoteBranches) > remoteLimit {
			remoteBranches = remoteBranches[:remoteLimit]
		}

		// Recombine: local + their remote counterparts + other recent remotes
		branches = append(localBranches, remoteCounterparts...)
		branches = append(branches, remoteBranches...)
	}

	// Calculate ahead/behind stats with parallel fetching
	const maxConcurrent = 10
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	for i := range branches {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			branch := &branches[idx]
			if branch.IsLocal {
				branch.Ahead, branch.Behind = s.GetBranchStats(ctx, branch.Name, "")
			} else if branch.Remote != "" {
				branch.Ahead, branch.Behind = s.GetBranchStats(ctx, branch.Name, branch.Remote)
			}
		}(i)
	}
	wg.Wait()

	stated, ahBoth := bookmarkListParseOriginDivergence(out)
	originDiverged := s.originDivergedResolved(ctx, stated, ahBoth)
	suppressForkAfterAheadBehindList := make(map[string]bool)
	for k := range ahBoth {
		k = strings.TrimSpace(k)
		if k != "" && !originDiverged[k] {
			suppressForkAfterAheadBehindList[k] = true
		}
	}
	for i := range branches {
		b := &branches[i]
		if !b.IsLocal {
			continue
		}
		switch strings.ToLower(b.Name) {
		case "main", "master":
			continue
		}
		// Reconcile HasConflict from bookmark list + DAG. The first-pass parse can leave HasConflict
		// true (e.g. ? on the name) after @origin already matches local; when originDiverged and the
		// fork detector both say no, clear it.
		if originDiverged[b.Name] {
			b.HasConflict = true
			continue
		}
		if suppressForkAfterAheadBehindList[b.Name] {
			continue
		}
		if s.bookmarkDivergedFromOrigin(ctx, b.Name) {
			b.HasConflict = true
		} else {
			b.HasConflict = false
		}
	}

	return branches, nil
}

// parseBookmarkListRemoteLine parses an indented jj bookmark list line such as
// "  @origin: …" or "  @origin (ahead by 1, behind by 1): …". The first ":" in the line is often
// inside the parenthetical, not after the remote name.
func parseBookmarkListRemoteLine(trimmed string) (remote string, info string, ok bool) {
	if !strings.HasPrefix(trimmed, "@") {
		return "", "", false
	}
	rest := strings.TrimSpace(trimmed[1:])
	if rest == "" {
		return "", "", false
	}
	if paren := strings.Index(rest, " ("); paren >= 0 {
		remote = strings.TrimSpace(rest[:paren])
		close := strings.Index(rest, "):")
		if close < 0 {
			return "", "", false
		}
		info = strings.TrimSpace(rest[close+2:])
		return remote, info, true
	}
	colon := strings.Index(rest, ":")
	if colon < 0 {
		return "", "", false
	}
	return strings.TrimSpace(rest[:colon]), strings.TrimSpace(rest[colon+1:]), true
}

var reAheadByJJ = regexp.MustCompile(`(?i)ahead by\s+(\d+)`)
var reBehindByJJ = regexp.MustCompile(`(?i)behind by\s+(\d+)`)

// jjOriginQualifierAheadBehind parses the parenthetical after @origin on a bookmark list line
// (e.g. "(ahead by 1 commits, behind by 1 commits)") and returns ahead/behind counts when both appear.
func jjOriginQualifierAheadBehind(originRemoteLine string) (ahead, behind int, ok bool) {
	lo := strings.ToLower(originRemoteLine)
	idx := strings.Index(lo, "@origin")
	if idx < 0 {
		return 0, 0, false
	}
	after := strings.TrimSpace(originRemoteLine[idx+len("@origin"):])
	if !strings.HasPrefix(after, "(") {
		return 0, 0, false
	}
	close := strings.Index(after, "):")
	if close < 0 {
		return 0, 0, false
	}
	inner := after[1:close]
	am := reAheadByJJ.FindStringSubmatch(inner)
	bm := reBehindByJJ.FindStringSubmatch(inner)
	if len(am) < 2 || len(bm) < 2 {
		return 0, 0, false
	}
	a, errA := strconv.Atoi(am[1])
	b, errB := strconv.Atoi(bm[1])
	if errA != nil || errB != nil {
		return 0, 0, false
	}
	return a, b, true
}

// bookmarkListParseOriginDivergence parses `jj bookmark list --all-remotes` into two buckets.
// conflictedStated is authoritative (jj says conflicted on the @origin line).
// aheadBehindBothNonZero records "(ahead by N, behind by M)" with N>0 and M>0 — jj sometimes prints
// that after merges even when local and remote tips are still linearly related; callers must confirm
// with originDivergedResolved (DAG fork check) before treating as diverged.
//
// jj prints "(ahead by 0, behind by N)" for behind-only; those do not set aheadBehindBothNonZero.
func bookmarkListParseOriginDivergence(listOutput string) (conflictedStated, aheadBehindBothNonZero map[string]bool) {
	conflictedStated = make(map[string]bool)
	aheadBehindBothNonZero = make(map[string]bool)
	var pendingLocal string
	for _, line := range strings.Split(listOutput, "\n") {
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			pendingLocal = ""
			colonIdx := strings.Index(line, ":")
			if colonIdx <= 0 {
				continue
			}
			head := strings.TrimSpace(line[:colonIdx])
			if strings.Contains(head, "@") {
				continue
			}
			if strings.Contains(strings.ToLower(line), "(deleted)") {
				continue
			}
			norm, _ := util.NormalizeBookmarkListToken(head)
			if norm != "" {
				pendingLocal = norm
			}
			continue
		}
		t := strings.TrimSpace(line)
		r, info, ok := parseBookmarkListRemoteLine(t)
		if !ok || r != "origin" {
			continue
		}
		if pendingLocal == "" {
			continue
		}
		full := strings.ToLower(t)
		infoLower := strings.ToLower(info)
		if strings.Contains(infoLower, "conflicted") || strings.Contains(full, "conflicted") {
			conflictedStated[pendingLocal] = true
			continue
		}
		if ah, bh, ok := jjOriginQualifierAheadBehind(t); ok && ah > 0 && bh > 0 {
			aheadBehindBothNonZero[pendingLocal] = true
		}
	}
	return conflictedStated, aheadBehindBothNonZero
}

// originDivergedResolved turns bookmark list parse output into "needs diverged resolver" names:
// always includes conflictedStated; includes ahead/behind candidates only when bookmarkDivergedFromOrigin
// confirms a real DAG fork (neither tip is an ancestor of the other).
func (s *Service) originDivergedResolved(ctx context.Context, conflictedStated, aheadBehindBothNonZero map[string]bool) map[string]bool {
	d := make(map[string]bool)
	for k := range conflictedStated {
		if strings.TrimSpace(k) != "" {
			d[k] = true
		}
	}
	for k := range aheadBehindBothNonZero {
		k = strings.TrimSpace(k)
		if k == "" || d[k] {
			continue
		}
		if s.bookmarkDivergedFromOrigin(ctx, k) {
			d[k] = true
		}
	}
	return d
}

// commitIDAtRevision returns the commit_id for rev (e.g. bookmark name or name@origin).
func (s *Service) commitIDAtRevision(ctx context.Context, rev string) (string, error) {
	out, err := s.runJJOutputNoHistory(ctx, "log", "-r", rev, "--no-graph", "-T", "commit_id", "--limit", "1")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// revsetCommitID forces jj to treat a token as a git commit id (not a change id prefix).
func revsetCommitID(commitID string) string {
	commitID = strings.TrimSpace(commitID)
	if commitID == "" {
		return ""
	}
	return fmt.Sprintf("commit_id(%s)", commitID)
}

// changeIDRootKey normalizes a jj change_id template value for comparison (strip /N divergent suffix).
func changeIDRootKey(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return ""
	}
	if i := strings.Index(s, "/"); i > 0 {
		return s[:i]
	}
	return s
}

// commitIDsHaveAncestorDescendantRelationship is true when either commit is an ancestor of the other
// in the jj DAG (pure ahead/behind). Used so we do not treat "N commits ahead of origin" as a
// diverged bookmark: that case should offer "Forgot New Commit?" (HasDeltaVsBookmarkOrigin), not resolve.
//
// We use x::y ("descendants of x that are also ancestors of y") with commit_id() so jj does not
// interpret a short hex as a change id. Hidden remote tips still participate once both ends are named.
func (s *Service) commitIDsHaveAncestorDescendantRelationship(ctx context.Context, a, b string) bool {
	a, b = strings.TrimSpace(a), strings.TrimSpace(b)
	if a == "" || b == "" {
		return false
	}
	ra, rb := revsetCommitID(a), revsetCommitID(b)
	for _, pair := range [2][2]string{{ra, rb}, {rb, ra}} {
		x, y := pair[0], pair[1]
		out, err := s.runJJOutputNoHistory(ctx, "log", "-r", fmt.Sprintf("%s::%s", x, y), "--no-graph", "-T", "commit_id", "--limit", "1")
		if err == nil && strings.TrimSpace(out) != "" {
			return true
		}
	}
	return false
}

// bookmarkDivergedFromOrigin is true when the local bookmark tip and origin's tip are on a true fork:
// different commits and neither is an ancestor of the other. Simple ahead (or behind) shares
// ancestry, so we return false — bookmark list + graph still mark (conflicted) and "(ahead>0 behind>0)".
// We compare commit_id, not change_id, because jj amends can keep the same change_id while the git commit differs.
// Bare names with '/' are invalid revsets (change-offset syntax); conflicted bookmarks need bookmarks()/remote_bookmarks().
func (s *Service) bookmarkDivergedFromOrigin(ctx context.Context, localName string) bool {
	if strings.TrimSpace(localName) == "" {
		return false
	}
	pat := util.RevsetExactPattern(localName)
	localRev := fmt.Sprintf("latest(bookmarks(%s))", pat)
	remoteRev := fmt.Sprintf("latest(remote_bookmarks(%s, %s))", pat, util.RevsetExactPattern("origin"))
	localID, errL := s.commitIDAtRevision(ctx, localRev)
	remoteID, errR := s.commitIDAtRevision(ctx, remoteRev)
	if errL != nil || errR != nil {
		return false
	}
	if localID == "" || remoteID == "" || localID == remoteID {
		return false
	}
	if s.commitIDsHaveAncestorDescendantRelationship(ctx, localID, remoteID) {
		return false
	}
	// Do not bail out on same jj change_id alone: amend-after-push keeps one change_id while local
	// and @origin tips are sibling commits (ahead+behind on the bookmark list). That must stay a
	// diverged bookmark until resolved; only a linear ancestor/descendant relationship is "not a fork".
	return true
}

// bookmarkNeedsDivergedResolver is true when the user should see "Resolve diverged bookmark", not
// "Forgot New Commit?". jj graph templates append "?" for any local vs remote tip mismatch, including
// a linear stack ahead of origin (forgot to push); those share ancestry and must not block (f).
//
// suppressForkAfterAheadBehindList names bookmarks where jj listed (ahead>0, behind>0) but
// originDivergedResolved did not mark diverged (tips are linear). The fork detector must not override.
func (s *Service) bookmarkNeedsDivergedResolver(ctx context.Context, name string, originDiverged map[string]bool, forkCache map[string]bool, suppressForkAfterAheadBehindList map[string]bool) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	if suppressForkAfterAheadBehindList != nil && suppressForkAfterAheadBehindList[name] {
		return false
	}
	if originDiverged != nil && originDiverged[name] {
		return true
	}
	if v, ok := forkCache[name]; ok {
		return v
	}
	v := s.bookmarkDivergedFromOrigin(ctx, name)
	forkCache[name] = v
	return v
}

// enrichConflictedBookmarks adds bookmarks that need the diverged resolver (jj may omit ? in graph output).
// originDiverged comes from bookmark list plus DAG confirmation for (ahead>0, behind>0) lines.
// The fallback uses bookmarkDivergedFromOrigin when the list did not run or did not classify the name.
func (s *Service) enrichConflictedBookmarks(ctx context.Context, commits []internal.Commit, originDiverged map[string]bool, suppressForkAfterAheadBehindList map[string]bool) {
	forkCache := make(map[string]bool)
	for i := range commits {
		c := &commits[i]
		seen := make(map[string]bool)
		for _, x := range c.ConflictedBranches {
			seen[x] = true
		}
		for _, b := range c.Branches {
			raw, _ := util.NormalizeBookmarkListToken(b)
			name := util.LocalBookmarkName(strings.TrimSpace(raw))
			if name == "" {
				continue
			}
			switch strings.ToLower(name) {
			case "main", "master":
				continue
			}
			if s.bookmarkNeedsDivergedResolver(ctx, name, originDiverged, forkCache, suppressForkAfterAheadBehindList) && !seen[name] {
				c.ConflictedBranches = append(c.ConflictedBranches, name)
				seen[name] = true
			}
		}
	}
	s.pruneSpuriousGraphConflictMarks(ctx, commits, originDiverged, forkCache, suppressForkAfterAheadBehindList)
}

// pruneSpuriousGraphConflictMarks drops bookmark names that were marked conflicted only because jj's
// graph added "?" on a linear ahead/behind relationship. Without this, the TUI hides "Forgot New
// Commit?" and blocks (f), pushing users toward "keep local" resolve + git push (often a force-style
// update on the PR branch).
func (s *Service) pruneSpuriousGraphConflictMarks(ctx context.Context, commits []internal.Commit, originDiverged map[string]bool, forkCache map[string]bool, suppressForkAfterAheadBehindList map[string]bool) {
	if forkCache == nil {
		forkCache = make(map[string]bool)
	}
	for i := range commits {
		c := &commits[i]
		if len(c.ConflictedBranches) == 0 {
			continue
		}
		kept := make([]string, 0, len(c.ConflictedBranches))
		seen := make(map[string]bool)
		for _, raw := range c.ConflictedBranches {
			n := strings.TrimSpace(raw)
			if n == "" || seen[n] {
				continue
			}
			if s.bookmarkNeedsDivergedResolver(ctx, n, originDiverged, forkCache, suppressForkAfterAheadBehindList) {
				kept = append(kept, n)
				seen[n] = true
			}
		}
		c.ConflictedBranches = kept
	}
}

// parseCommitInfo extracts change_id and short commit id from jj output
// Format: "change_id commit_id description"
func parseCommitInfo(info string) (changeID, shortID string) {
	parts := strings.Fields(info)
	if len(parts) >= 2 {
		changeID = parts[0]
		shortID = parts[1]
	} else if len(parts) == 1 {
		changeID = parts[0]
		shortID = parts[0]
	}
	return
}

// countRevisions counts the number of revisions matching a revset
func (s *Service) countRevisions(ctx context.Context, revset string) int {
	out, err := s.runJJOutput(ctx, "log", "-r", revset, "--no-graph", "-T", `"x"`)
	if err != nil {
		return 0
	}
	// Count 'x' characters (one per revision)
	return strings.Count(out, "x")
}

// GetBranchStats calculates ahead/behind counts for a branch relative to trunk
// For local branches, pass empty string for remoteName
// For remote branches, pass the remote name (e.g., "origin")
func (s *Service) GetBranchStats(ctx context.Context, branchName string, remoteName string) (ahead, behind int) {
	// Use trunk() as the base reference (usually main@origin)
	var branchRef string
	if remoteName == "" {
		// Local branch
		branchRef = branchName
	} else {
		// Remote branch: use name@remote format
		branchRef = fmt.Sprintf("%s@%s", branchName, remoteName)
	}

	// Commits in branch that are not in trunk (ahead)
	ahead = s.countRevisions(ctx, fmt.Sprintf("(%s)..(%s)", "trunk()", branchRef))

	// Commits in trunk that are not in branch (behind)
	behind = s.countRevisions(ctx, fmt.Sprintf("(%s)..(%s)", branchRef, "trunk()"))

	return ahead, behind
}

// TrackBranch starts tracking a remote branch
func (s *Service) TrackBranch(ctx context.Context, branchName, remote string) error {
	remoteBranch := fmt.Sprintf("%s@%s", branchName, remote)
	return s.runJJ(ctx, "bookmark", "track", remoteBranch)
}

// UntrackBranch stops tracking a remote branch
func (s *Service) UntrackBranch(ctx context.Context, branchName, remote string) error {
	remoteBranch := fmt.Sprintf("%s@%s", branchName, remote)
	return s.runJJ(ctx, "bookmark", "untrack", remoteBranch)
}

// RestoreLocalBranch restores a deleted local branch from its tracked remote
func (s *Service) RestoreLocalBranch(ctx context.Context, branchName, commitID string) error {
	// Use jj bookmark set to create/restore the local bookmark at the remote's revision
	return s.runJJ(ctx, "bookmark", "set", util.BookmarkArgForSetMove(branchName), "-r", commitID)
}

// PushBranch pushes a local branch to remote
func (s *Service) PushBranch(ctx context.Context, branchName string) error {
	return s.runJJ(ctx, "git", "push", "--allow-new", "--bookmark", util.JJExactBookmarkPattern(branchName))
}

// FetchFromRemote fetches updates from a remote
func (s *Service) FetchFromRemote(ctx context.Context, remote string) error {
	return s.runJJ(ctx, "git", "fetch", "--remote", remote)
}

// FetchAllRemotes fetches from all configured remotes
func (s *Service) FetchAllRemotes(ctx context.Context) error {
	return s.runJJ(ctx, "git", "fetch", "--all-remotes")
}

// isJJRepo checks if a directory is a jj repository
func isJJRepo(path string) bool {
	jjDir := filepath.Join(path, ".jj")
	if info, err := os.Stat(jjDir); err == nil && info.IsDir() {
		return true
	}
	// Check parent directories recursively
	parent := filepath.Dir(path)
	if parent != path {
		return isJJRepo(parent)
	}
	return false
}
