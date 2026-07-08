package jj

import (
	"context"
	"fmt"
	"strings"

	"github.com/madicen/jj-tui/internal"
)

// jjMessageArg returns the argument form for a commit message that is safe even when the message
// starts with '-'. Passing "-m"/"--message" followed by a separate value makes jj's (clap) arg
// parser treat a leading-dash message (e.g. a markdown bullet list) as an unknown flag and fail
// with "unexpected argument '-...' found". The "--message=<value>" form binds the whole value to
// the flag, including any leading '-', so descriptions can start with any character.
func jjMessageArg(message string) string {
	return "--message=" + message
}

// CreateNewCommit creates a new commit with the given description
func (s *Service) CreateNewCommit(ctx context.Context, description string) error {
	return s.runJJ(ctx, "commit", jjMessageArg(description))
}

// DescribeCommit sets a new description for a commit (non-interactive)
func (s *Service) DescribeCommit(ctx context.Context, commitID string, message string) error {
	_, err := s.runJJOutput(ctx, "describe", commitID, jjMessageArg(message))
	if err == nil {
		return nil
	}
	if !isJJStaleWorkingCopyError(err) {
		return err
	}
	if uerr := s.runJJ(ctx, "workspace", "update-stale"); uerr != nil {
		return fmt.Errorf("%w\n\nCould not refresh stale working copy (jj workspace update-stale): %v", err, uerr)
	}
	_, err2 := s.runJJOutput(ctx, "describe", commitID, jjMessageArg(message))
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

// ChainCommit is one commit in a `from..to` chain, used to give the LLM
// per-commit context (subject + full description) on top of the cumulative diff.
type ChainCommit struct {
	ChangeIDShort string // 8-char change id, stable across rebases
	Subject       string // first line of the description (or "(no description)")
	Description   string // full description; may be multi-line; may be empty
}

// ListChainCommits returns commits in the revset `fromRev..toRev`, ordered
// oldest → newest (so the AI reads them in the order the work happened).
//
// `fromRev..toRev` is the standard jj revset for "descendants of fromRev that
// are ancestors of (or equal to) toRev, excluding fromRev itself"; it gives the
// chain of work introduced on top of fromRev that ends at toRev. Returns an
// empty slice (no error) when the chain is empty (e.g. toRev is at or below
// fromRev), so callers can cleanly fall back to single-revision context.
func (s *Service) ListChainCommits(ctx context.Context, fromRev, toRev string) ([]ChainCommit, error) {
	fromRev = strings.TrimSpace(fromRev)
	toRev = strings.TrimSpace(toRev)
	if fromRev == "" || toRev == "" {
		return nil, fmt.Errorf("from and to revisions are required")
	}
	// Tab-separated row per commit. Replace embedded tabs/newlines in the
	// description with safe placeholders so a single row is always one line
	// with exactly three tab-separated fields; we restore newlines in the
	// description after parsing (subject is single-line by definition).
	const rowSep = "\x1e" // record separator: ends each commit row
	const fieldSep = "\t"
	const nlMarker = "\x1f"  // unit separator: stand-in for '\n' inside description
	const tabMarker = "\x1d" // group separator: stand-in for '\t' inside description
	template := `change_id.short(8) ++ "` + fieldSep + `" ++ ` +
		`if(description, description.first_line(), "(no description)") ++ "` + fieldSep + `" ++ ` +
		`description.replace("\t", "` + tabMarker + `").replace("\n", "` + nlMarker + `") ++ ` +
		`"` + rowSep + `"`
	// `--reversed` orders oldest → newest; the revset itself doesn't guarantee
	// a particular ordering for chains with merges, but for the common linear
	// case this matches the reading order a human would expect.
	revset := fmt.Sprintf("%s..%s", fromRev, toRev)
	out, err := s.runJJOutput(ctx, "log", "-r", revset, "--no-graph", "--reversed", "-T", template)
	if err != nil {
		return nil, err
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return nil, nil
	}
	var commits []ChainCommit
	for _, row := range strings.Split(out, rowSep) {
		row = strings.TrimSpace(row)
		if row == "" {
			continue
		}
		parts := strings.SplitN(row, fieldSep, 3)
		if len(parts) < 3 {
			continue
		}
		desc := strings.ReplaceAll(parts[2], nlMarker, "\n")
		desc = strings.ReplaceAll(desc, tabMarker, "\t")
		commits = append(commits, ChainCommit{
			ChangeIDShort: strings.TrimSpace(parts[0]),
			Subject:       strings.TrimSpace(parts[1]),
			Description:   strings.TrimRight(desc, "\n"),
		})
	}
	return commits, nil
}

// GetRevisionChangeID gets the change_id for any jj revision (e.g., "main@origin", "@", etc.)
func (s *Service) GetRevisionChangeID(ctx context.Context, revision string) (string, error) {
	out, err := s.runJJOutput(ctx, "log", "-r", revision, "--no-graph", "-T", "change_id")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
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
	return s.runJJ(ctx, "squash", "-r", commitID, jjMessageArg(combinedDesc))
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

// MergeCommit creates a new merge commit whose parents are the target and source commits.
// This is the jj way to "merge from" (e.g. merge main into the current bookmark): the new
// working-copy commit becomes a child of both <target> and <source>.
func (s *Service) MergeCommit(ctx context.Context, targetCommitID, sourceCommitID string) error {
	// jj new <target> <source>
	return s.runJJ(ctx, "new", targetCommitID, sourceCommitID)
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

// DuplicateCommit creates a copy of the given revision. When dest is non-empty
// the duplicate is placed onto that destination (jj duplicate -r X -d DEST);
// otherwise it is duplicated in place onto the revision's existing parents.
func (s *Service) DuplicateCommit(ctx context.Context, rev, dest string) error {
	args := []string{"duplicate", "-r", rev}
	if strings.TrimSpace(dest) != "" {
		args = append(args, "-d", dest)
	}
	return s.runJJ(ctx, args...)
}

// BackoutCommit applies the reverse of the given revision on top of the working
// copy, creating a new commit that undoes it. The underlying jj subcommand is
// `backout` or `revert` depending on the installed jj version (see backoutVerb).
func (s *Service) BackoutCommit(ctx context.Context, rev string) error {
	return s.runJJ(ctx, backoutOrRevertArgs(s.backoutVerb(ctx), rev)...)
}

// RevisionImmutable reports whether the given revision is immutable in jj's config.
func (s *Service) RevisionImmutable(ctx context.Context, revision string) (bool, error) {
	out, err := s.runJJOutputNoHistory(ctx, "log", "-r", revision, "--no-graph", "-T", `if(immutable, "true", "false")`, "--limit", "1")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) == "true", nil
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
