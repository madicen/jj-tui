package jj

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/integrations/jj/jjout"
	"github.com/madicen/jj-tui/internal/tui/util"
)

// listMineUntrackedRemoteBookmarks returns one Branch per (remote_bookmark, remote)
// pair where the tip change was authored by the current user. Used by ListBranches
// in BookmarkListPreferTracked mode to backfill PR branches you opened but haven't
// tracked locally — `jj bookmark list --tracked` omits those, but you almost
// certainly want them visible in the branches tab.
//
// Implementation: one `jj log -r 'remote_bookmarks() & mine()'` call with a template
// that emits "name|remote|change_id_short|commit_id_short" per remote_bookmark
// attached to each row. We can't use the bookmark list parser here because it
// doesn't accept revsets; the log template path is both faster (one query, no parse
// of 1000+ unrelated rows) and richer (we get the change/commit ids inline).
func (s *Service) listMineUntrackedRemoteBookmarks(ctx context.Context) ([]internal.Branch, error) {
	const fieldSep = "\x1f" // unit separator
	const rowSep = "\x1e"   // record separator
	template := `remote_bookmarks.map(|b| ` +
		`b.name() ++ "` + fieldSep + `" ++ ` +
		`b.remote() ++ "` + fieldSep + `" ++ ` +
		`self.change_id().short(8) ++ "` + fieldSep + `" ++ ` +
		`self.commit_id().short(8) ++ "` + rowSep + `"` +
		`).join("")`
	out, err := s.runJJOutputNoHistory(ctx, "log",
		"-r", "remote_bookmarks() & mine()",
		"--no-graph",
		"-T", template,
	)
	if err != nil {
		return nil, err
	}
	var branches []internal.Branch
	for _, row := range strings.Split(out, rowSep) {
		row = strings.TrimSpace(row)
		if row == "" {
			continue
		}
		parts := strings.Split(row, fieldSep)
		if len(parts) < 4 {
			continue
		}
		remote := strings.TrimSpace(parts[1])
		if remote == "" || remote == "git" {
			continue
		}
		branches = append(branches, internal.Branch{
			Name:      strings.TrimSpace(parts[0]),
			Remote:    remote,
			CommitID:  strings.TrimSpace(parts[2]),
			ShortID:   strings.TrimSpace(parts[3]),
			IsTracked: false,
			IsLocal:   false,
		})
	}
	return branches, nil
}

// ListBranches returns all local and remote branches.
//
// statsLimit controls how many branches get ahead/behind stats calculated (0 = all).
//
// When BookmarkListPreferTracked is set the listing uses `jj bookmark list --tracked`
// (cheap; ~tens of rows even on 1000-branch repos) and then augments the result with
// any remote bookmarks whose tip you authored (via a separate `remote_bookmarks() & mine()`
// jj log query) so you don't lose visibility of your own un-tracked PR branches.
func (s *Service) ListBranches(ctx context.Context, statsLimit int) ([]internal.Branch, error) {
	out, err := s.runJJOutput(ctx, "bookmark", "list", s.BookmarkListRemoteFlag())
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

	// When BookmarkListPreferTracked is on, the listing above used `--tracked` and
	// therefore omitted every untracked origin/* bookmark — including PR branches
	// you authored but haven't tracked. Backfill those with one extra jj log so the
	// branches tab still surfaces your own work. The dedup set keys off (name, remote)
	// so we never double-add a branch that is also a tracked counterpart of a local.
	if s.BookmarkListPreferTracked {
		mineBranches, mineErr := s.listMineUntrackedRemoteBookmarks(ctx)
		if mineErr == nil && len(mineBranches) > 0 {
			seen := make(map[string]bool, len(branches))
			for _, b := range branches {
				if !b.IsLocal {
					seen[b.Name+"@"+b.Remote] = true
				}
			}
			for _, b := range mineBranches {
				if seen[b.Name+"@"+b.Remote] {
					continue
				}
				branches = append(branches, b)
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
					for _, line := range jjout.SplitLines(out) {
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

		// Recombine: local + their remote counterparts + other recent remotes.
		//nolint:gocritic // appendAssign is intentional here: localBranches is not reused afterward.
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
		closeIdx := strings.Index(rest, "):")
		if closeIdx < 0 {
			return "", "", false
		}
		info = strings.TrimSpace(rest[closeIdx+2:])
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
	closeIdx := strings.Index(after, "):")
	if closeIdx < 0 {
		return 0, 0, false
	}
	inner := after[1:closeIdx]
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

// parseCommitInfo extracts change_id and short commit id from jj output
// Format: "change_id commit_id description"
func parseCommitInfo(info string) (changeID, shortID string) {
	return jjout.ParseCommitInfo(info)
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

// FetchAndTrackBranch pulls a single bookmark down from a remote and starts tracking it.
// This is the "paste a branch name" path: it works even when the bookmark isn't yet in the
// local view (e.g. a coworker's branch that hasn't been fetched, or one hidden by the
// default --tracked listing). When remote is empty we fetch from every remote and default
// the tracking ref to origin; pass "name@remote" upstream to target a specific remote.
func (s *Service) FetchAndTrackBranch(ctx context.Context, branchName, remote string) error {
	fetchArgs := []string{"git", "fetch", "--branch", branchName}
	if remote != "" {
		fetchArgs = append(fetchArgs, "--remote", remote)
	} else {
		fetchArgs = append(fetchArgs, "--all-remotes")
	}
	if err := s.runJJ(ctx, fetchArgs...); err != nil {
		return fmt.Errorf("failed to fetch bookmark %q: %w", branchName, err)
	}
	trackRemote := remote
	if trackRemote == "" {
		trackRemote = "origin"
	}
	return s.TrackBranch(ctx, branchName, trackRemote)
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
