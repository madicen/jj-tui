package jj

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/madicen/jj-tui/internal/integrations/jj/jjout"
	"github.com/madicen/jj-tui/internal/tui/util"
)

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
	for _, line := range jjout.SplitLines(out) {
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

// wrapJjNewColocatedError augments jj new failures in colocated repos when Git HEAD and jj disagree.
// retryPhrase is appended after the import/export guidance (e.g. "then retry the split", "then try Align with origin again").
func wrapJjNewColocatedError(err error, retryPhrase string) error {
	if err == nil {
		return nil
	}
	wrapped := fmt.Errorf("jj new: %w", err)
	es := err.Error()
	if isJJColocatedHeadContentMismatch(es) {
		return fmt.Errorf("%w\n\nIn a colocated repo, Git’s HEAD can drift from jj (often after `git checkout` without jj). Try: `jj git import` to follow Git, or `jj git export` to push jj’s view to Git, %s", wrapped, retryPhrase)
	}
	if isJJColocatedGitCheckoutFailure(es) {
		return fmt.Errorf("%w\n\nGit could not check out a revision while creating the new change (colocated repo). Try: `jj workspace update-stale` then `jj git export`, ensure the working tree is not blocked by another process, %s. If you only moved HEAD with Git, run `jj git import` or `jj git export` to reconcile.\n\nNote: `jj log` may show `~` between commits when intermediate revisions are elided by the revset filter; that is not the same as a non-linear graph", wrapped, retryPhrase)
	}
	return wrapped
}

func wrapEvologJjNewError(err error) error {
	return wrapJjNewColocatedError(err, "then retry the split")
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
	parents := jjout.SplitLines(parentsOut)
	if len(parents) == 1 {
		return parents[0], nil
	}
	return baseCommitID, nil
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
	args := []string{"split", "-r", revision, jjMessageArg(strings.TrimSpace(firstMessage)), "--"}
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
// Steps after the first are skipped when jj reports an empty tree diff vs that base (already satisfied
// after a prior FAQ move, redundant LLM base ids, or duplicate bases after sort) so the run can continue
// to file/hunk peels instead of failing with "tree already matches base".
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
		if i > 0 {
			diffOut, derr := s.runJJOutputWithGlobal(ctx, jjEvologSplitPrepareGlobals, "diff", "--from", base, "--to", tipCH, "--summary")
			if derr != nil {
				return fmt.Errorf("evolog multi-split step %d/%d: diff vs base: %w", i+1, len(bases), derr)
			}
			if strings.TrimSpace(diffOut) == "" {
				continue
			}
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
	// Chained splits (esp. with --insert-before) can briefly surface two commits for one change_id on
	// the path to @; same cleanup as stack-on-origin after rebase (see abandonDivergentDuplicateCommitsOffWCPath).
	if err := s.abandonDivergentDuplicateCommitsOffWCPath(ctx); err != nil {
		return fmt.Errorf("cleanup divergent duplicates after evolog multi-split: %w", err)
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
	if err := s.abandonDivergentDuplicateCommitsOffWCPath(ctx); err != nil {
		return fmt.Errorf("cleanup divergent duplicates after evolog split: %w", err)
	}
	return nil
}
