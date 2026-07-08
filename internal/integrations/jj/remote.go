package jj

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/madicen/jj-tui/internal/tui/util"
)

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

	// Naming the bookmark explicitly with --bookmark is enough for jj to create it on the
	// remote if it's new (the old --allow-new flag is deprecated/removed in current jj).
	// Use runJJOutput to capture any output/errors
	pushOut, err := s.runJJOutput(ctx, "git", "push", "--bookmark", util.JJExactBookmarkPattern(branch))
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

// PushBranch pushes a local branch to remote. Naming the bookmark explicitly with --bookmark is
// enough for jj to create it on the remote if it's new (the old --allow-new flag is deprecated/
// removed in current jj).
func (s *Service) PushBranch(ctx context.Context, branchName string) error {
	return s.runJJ(ctx, "git", "push", "--bookmark", util.JJExactBookmarkPattern(branchName))
}

// FetchFromRemote fetches updates from a remote
func (s *Service) FetchFromRemote(ctx context.Context, remote string) error {
	return s.runJJ(ctx, "git", "fetch", "--remote", remote)
}

// FetchAllRemotes fetches from all configured remotes
func (s *Service) FetchAllRemotes(ctx context.Context) error {
	return s.runJJ(ctx, "git", "fetch", "--all-remotes")
}
