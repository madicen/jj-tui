package jj

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/madicen/jj-tui/internal"
	"github.com/madicen/jj-tui/internal/tui/util"
)

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

// DefaultGraphRevset is used when config graph_revset is empty.
//
// We intersect mutable() with (ancestors(@) | descendants(@) | (parents(@)+)::) so the graph
// is tied to the working copy's DAG neighborhood. Bare mutable() | bookmarks() | main@origin
// matches every mutable revision repo-wide (including unrelated merged branches, stale
// divergent pairs, and teammates' old lines), which is overwhelming in large colocated repos.
//
//   - ancestors(@) and descendants(@) give the working copy's own line plus any splits.
//   - (parents(@)+):: adds sibling branches: parents(@)+ is the children of @'s parents
//     (i.e. @ and its siblings), and `::` then includes their full descendant subtree so
//     split-off branches stay visible.
//
// (bookmarks() & mine()) | trunk() keeps your named tips and the immutable trunk anchor
// visible even when they fall outside the @ neighborhood above. We deliberately do NOT
// union plain bookmarks() here: in colocated repos `jj git import` materializes a local
// bookmark for every origin/* PR branch and would balloon the graph to 1000+ rows.
const DefaultGraphRevset = `(mutable() & (ancestors(@) | descendants(@) | (parents(@)+)::)) | (bookmarks() & mine()) | trunk()`

// graphMineFilterAncestors is the ancestor depth used when intersecting the configured
// graph revset with the "mine() | trunk() | @" pin set (see ApplyMineFilterToRevset).
// 2 matches the upstream jj `revsets.log` default (`ancestors(immutable_heads().., 2)`)
// so rows authored by others still get up to 2 generations of parent context.
const graphMineFilterAncestors = 2

// ApplyMineFilterToRevset wraps an arbitrary graph revset with a "mine-or-context" filter:
//
//	(<base>) & ancestors(mine() | trunk() | @, 2) | trunk() | @
//
// The intersection drops rows authored by other contributors, while the trailing pins
// guarantee the working copy and trunk tip stay visible even if the user's authored
// commits don't intersect the base revset (e.g. fresh clone, no local work yet).
// Callers that want the legacy "show everyone" behavior should skip this wrapper —
// see config.GraphFilterToMine().
func ApplyMineFilterToRevset(base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = DefaultGraphRevset
	}
	return fmt.Sprintf(
		"((%s) & ancestors(mine() | trunk() | @, %d)) | trunk() | @",
		base, graphMineFilterAncestors,
	)
}

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
	// Uses --tracked when BookmarkListPreferTracked is set: divergence enrichment only inspects
	// local→@origin pairs (see bookmarkListParseOriginDivergence), so untracked origin/* entries
	// add no signal and cost ~1.3k extra rows to parse on big colocated repos.
	var bmOut string
	var bmErr error
	var bmWG sync.WaitGroup
	bmWG.Add(1)
	go func() {
		defer bmWG.Done()
		bmOut, bmErr = s.runJJOutputNoHistory(ctx, "bookmark", "list", s.BookmarkListRemoteFlag())
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
		// Fall back to a broad, safe revset so the app still loads. This covers both a
		// failing custom revset and the default failing when main@origin is missing.
		out, err = s.jjLogWithGraphTemplate(ctx, recordGraphInHistory, "mutable() | bookmarks()", template)
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
