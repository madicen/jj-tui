package jj

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/madicen/jj-tui/internal/integrations/jj/jjout"
	"github.com/madicen/jj-tui/internal/tui/util"
)

// BookmarkListRemoteFlag returns the flag to pass to `jj bookmark list`
// (`--tracked` or `--all-remotes`) based on BookmarkListPreferTracked.
func (s *Service) BookmarkListRemoteFlag() string {
	if s.BookmarkListPreferTracked {
		return "--tracked"
	}
	return "--all-remotes"
}

// SanitizeBookmarkName converts a string into a valid bookmark name: Unicode letters
// and numbers, ASCII hyphen and underscore only. Whitespace becomes underscore; any
// other rune is dropped. Consecutive underscores or consecutive hyphens collapse to one.
func SanitizeBookmarkName(name string) string {
	var b strings.Builder
	b.Grow(len(name))
	var lastSep rune // last written separator: '_' or '-'; 0 if last written was not a sep
	for _, r := range name {
		switch {
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			b.WriteRune(r)
			lastSep = 0
		case unicode.IsSpace(r):
			if lastSep != '_' {
				b.WriteRune('_')
				lastSep = '_'
			}
		case r == '-' || r == '_':
			if lastSep != r {
				b.WriteRune(r)
				lastSep = r
			}
		default:
			// drop commas, punctuation, symbols, etc.
		}
	}
	return strings.Trim(b.String(), "-_")
}

// MaxBookmarkNameLen is the hard cap for bookmark / branch names we generate or assign.
// The AI prompt already advertises a length limit; this constant is what we actually
// enforce on every "source" of new names (AI output, Jira ticket titles, final submit).
// 50 is the GitHub-flow-style middle ground: roomy enough for descriptive names but
// short enough that names stay legible in PR titles, status lines, and the modal's
// existing-bookmark list on narrow terminals.
const MaxBookmarkNameLen = 50

// TruncateBookmarkName is TruncateBookmarkNameTo with MaxBookmarkNameLen.
func TruncateBookmarkName(name string) string {
	return TruncateBookmarkNameTo(name, MaxBookmarkNameLen)
}

// TruncateBookmarkNameTo shortens name to at most max runes and trims any trailing
// '-' / '_' / '/' so the result never ends on a dangling separator. Operates on rune
// count rather than byte length so multi-byte Unicode (which Sanitize preserves,
// e.g. "café-…") isn't split mid-codepoint.
//
// Callers that want char-class normalization should pass through SanitizeBookmarkName
// first; this function only enforces length.
func TruncateBookmarkNameTo(name string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(name)
	if len(runes) <= maxLen {
		return name
	}
	return strings.TrimRight(string(runes[:maxLen]), "-_/")
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
	lines := jjout.SplitLines(out)
	ids := make([]string, 0, len(lines))
	sums := make([]string, 0, len(lines))
	whens := make([]string, 0, len(lines))
	for _, line := range lines {
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
	for _, line := range jjout.SplitLines(out) {
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

// followUpOnOriginMessage is the default description for the new commit created by
// MoveBookmarkDeltaOntoOrigin (user can jj describe afterward).
const followUpOnOriginMessage = "Follow-up (local changes on top of origin)"

// MoveBookmarkDeltaOntoOrigin places bookmark@origin as the parent of new work without rewriting the
// revision Git already has: it fetches, creates a new commit on top of bookmark@origin with the same
// tree as the bookmark tip, moves the bookmark there, rebases any non–working-copy children of the
// old tip onto the new bookmark tip (so local stacks stay intact), then abandons the old tip.
// localChangeID identifies the selected row when localCommitID is empty; prefer localCommitID (graph
// commit.ID). Diff / resolve tip always use commit_id — amend-after-push keeps one change ID on both
// the local tip and bookmark@origin, so a bare change ID is divergent and jj rejects it.
func (s *Service) MoveBookmarkDeltaOntoOrigin(ctx context.Context, bookmarkName, localChangeID, localCommitID string) error {
	if strings.TrimSpace(bookmarkName) == "" || (strings.TrimSpace(localChangeID) == "" && strings.TrimSpace(localCommitID) == "") {
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
	// Same commit_id(...) revset as enrichCommitsDeltaVsOrigin — required when change ID is divergent.
	diffOut, err := s.runJJOutput(ctx, "diff", "--from", remoteRef, "--to", revsetCommitID(tipCommitID), "--summary")
	if err != nil {
		return fmt.Errorf("diff vs origin: %w", err)
	}
	if strings.TrimSpace(diffOut) == "" {
		return fmt.Errorf("tree already matches %s; nothing to move", remoteRef)
	}
	if err := s.runJJ(ctx, "new", remoteRef, "-m", followUpOnOriginMessage); err != nil {
		return wrapJjNewColocatedError(err, "then try Align with origin again")
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
