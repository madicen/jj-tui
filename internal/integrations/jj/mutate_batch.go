package jj

import (
	"context"
	"fmt"
	"strings"
)

// AbandonCommitsBatch abandons multiple revisions in one `jj abandon` invocation so
// jj records a single operation (one undo restores all). commitIDs are revision
// identifiers accepted by jj (change-id, commit-id, or bookmark).
func (s *Service) AbandonCommitsBatch(ctx context.Context, commitIDs []string) error {
	parts := uniqueNonEmpty(commitIDs)
	if len(parts) == 0 {
		return fmt.Errorf("no revisions to abandon")
	}
	if len(parts) == 1 {
		return s.AbandonCommit(ctx, parts[0])
	}
	revset := strings.Join(parts, " | ")
	return s.runJJ(ctx, "abandon", revset)
}

// RebaseCommitsBatch rebases multiple revisions onto destCommitID in one `jj rebase`
// call (`jj rebase -r A -r B -d dest`). All sources must be compatible with jj's
// multi-revision rebase rules (typically contiguous siblings).
func (s *Service) RebaseCommitsBatch(ctx context.Context, sourceCommitIDs []string, destCommitID string) error {
	sources := uniqueNonEmpty(sourceCommitIDs)
	dest := strings.TrimSpace(destCommitID)
	if dest == "" {
		return fmt.Errorf("rebase destination required")
	}
	if len(sources) == 0 {
		return fmt.Errorf("no revisions to rebase")
	}
	if len(sources) == 1 {
		return s.RebaseCommit(ctx, sources[0], dest)
	}
	args := []string{"rebase", "-d", dest}
	for _, src := range sources {
		args = append(args, "-r", src)
	}
	return s.runJJ(ctx, args...)
}

func uniqueNonEmpty(ids []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}
