package jj

import (
	"context"
	"fmt"
	"strings"

	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/tui/util"
)

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
