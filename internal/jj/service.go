package jj

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/madicen/jj-tui/internal/models"
)

// Service handles jujutsu command execution
type Service struct {
	RepoPath string
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

	service := &Service{RepoPath: repoPath}

	// Test that we can actually run jj commands
	ctx := context.Background()
	if _, err := service.runJJOutput(ctx, "--version"); err != nil {
		return nil, fmt.Errorf("failed to execute jj commands: %w", err)
	}

	return service, nil
}

// GetRepository retrieves the current repository state
func (s *Service) GetRepository(ctx context.Context) (*models.Repository, error) {
	// Before loading the graph, do a quick cleanup of any orphaned empty commits
	// This handles the case where jj auto-created them after a merge
	_ = s.abandonOrphanedEmptyCommits(ctx)

	// Get commit graph (includes working copy)
	graph, err := s.getCommitGraph(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit graph: %w", err)
	}

	// Find working copy from graph
	var workingCopy models.Commit
	for _, c := range graph.Commits {
		if c.IsWorking {
			workingCopy = c
			break
		}
	}

	return &models.Repository{
		Path:        s.RepoPath,
		WorkingCopy: workingCopy,
		Graph:       *graph,
		PRs:         []models.GitHubPR{}, // TODO: populate from GitHub
	}, nil
}

// CreateNewCommit creates a new commit with the given description
func (s *Service) CreateNewCommit(ctx context.Context, description string) error {
	args := []string{"commit", "-m", description}
	return s.runJJ(ctx, args...)
}

// DescribeCommit sets a new description for a commit (non-interactive)
func (s *Service) DescribeCommit(ctx context.Context, commitID string, message string) error {
	args := []string{"describe", commitID, "-m", message}
	_, err := s.runJJOutput(ctx, args...)
	return err
}

// GetCommitDescription gets the full description of a commit
func (s *Service) GetCommitDescription(ctx context.Context, commitID string) (string, error) {
	out, err := s.runJJOutput(ctx, "log", "-r", commitID, "--no-graph", "-T", "description")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// GetRevisionChangeID gets the change_id for any jj revision (e.g., "main@origin", "@", etc.)
func (s *Service) GetRevisionChangeID(ctx context.Context, revision string) (string, error) {
	out, err := s.runJJOutput(ctx, "log", "-r", revision, "--no-graph", "-T", "change_id")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// Undo undoes the last jj operation
func (s *Service) Undo(ctx context.Context) error {
	return s.runJJ(ctx, "undo")
}

// Redo redoes the last undone jj operation (restores the operation before undo)
func (s *Service) Redo(ctx context.Context) error {
	// jj doesn't have a direct "redo" command, but "op restore" can be used
	// to restore to a previous operation. For simplicity, we use "undo" again
	// which effectively undoes the undo (if the last operation was an undo).
	// A more robust solution would track operation IDs, but this works for the common case.
	return s.runJJ(ctx, "undo")
}

// ChangedFile represents a file changed in a commit
type ChangedFile struct {
	Path   string // File path
	Status string // M=modified, A=added, D=deleted, R=renamed
}

// GetChangedFiles gets the list of changed files for a commit
func (s *Service) GetChangedFiles(ctx context.Context, commitID string) ([]ChangedFile, error) {
	// Use jj diff --summary to get a list of changed files
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

		// Format is: "M path/to/file" or "A path/to/file" etc.
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

// IsCommitMutable checks if a commit can be modified
func (s *Service) IsCommitMutable(ctx context.Context, commitID string) bool {
	// Try a no-op describe to see if the commit is mutable
	_, err := s.runJJOutput(ctx, "log", "-r", commitID, "--no-graph", "-T", "if(immutable, \"immutable\", \"mutable\")")
	return err == nil
}

// CheckoutCommit checks out a specific commit (uses jj edit)
func (s *Service) CheckoutCommit(ctx context.Context, commitID string) error {
	args := []string{"edit", commitID}
	return s.runJJ(ctx, args...)
}

// CreateNewBranch creates a new branch at the current commit
func (s *Service) CreateNewBranch(ctx context.Context, branchName string) error {
	args := []string{"branch", "create", branchName}
	return s.runJJ(ctx, args...)
}

// CreateBranchFromMain creates a bookmark for a ticket, handling existing work intelligently.
// If the user has existing work based on main (main -> A -> B...), the bookmark is added
// to the first commit after main (A). Otherwise, a new empty commit is created and the
// bookmark is placed on it. This is the standard jj workflow.
func (s *Service) CreateBranchFromMain(ctx context.Context, bookmarkName string) error {
	// Find the first mutable commit after main in our ancestry
	// This handles: main -> A -> B -> @ by finding A
	// Revset: ancestors of @ that are mutable AND whose parent is main@origin
	rootCommitID, err := s.runJJOutput(ctx, "log", "-r", "ancestors(@) & mutable() & children(main@origin)", "--no-graph", "-T", "change_id", "--limit", "1")
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
	if err := s.runJJ(ctx, "new", "main@origin"); err != nil {
		return fmt.Errorf("failed to create new commit from main: %w", err)
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
	args := []string{"bookmark", "create", bookmarkName, "-r", commitID}
	return s.runJJ(ctx, args...)
}

// MoveBookmark moves an existing bookmark to a different commit
func (s *Service) MoveBookmark(ctx context.Context, bookmarkName, commitID string) error {
	// jj bookmark set <name> -r <revision>
	args := []string{"bookmark", "set", bookmarkName, "-r", commitID}
	return s.runJJ(ctx, args...)
}

// DeleteBookmark deletes a bookmark
func (s *Service) DeleteBookmark(ctx context.Context, bookmarkName string) error {
	args := []string{"bookmark", "delete", bookmarkName}
	return s.runJJ(ctx, args...)
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
	args := []string{"squash", "-r", commitID, "-m", combinedDesc}
	return s.runJJ(ctx, args...)
}

// NewCommit creates a new commit. If parentCommitID is provided, creates a child of that commit.
// Otherwise creates a new commit on top of the current working copy (@).
// Note: This creates an empty commit initially. To avoid unnecessary placeholder commits during
// branch creation, use CreateBranchFromMain instead. NewCommit is useful for creating commits
// at specific parent points in the graph.
func (s *Service) NewCommit(ctx context.Context, parentCommitID string) error {
	args := []string{"new"}
	if parentCommitID != "" {
		args = append(args, parentCommitID)
	}
	return s.runJJ(ctx, args...)
}

// AbandonCommit abandons a commit, removing it from the repository
func (s *Service) AbandonCommit(ctx context.Context, commitID string) error {
	args := []string{"abandon", commitID}
	return s.runJJ(ctx, args...)
}

// RebaseCommit rebases a commit and all its descendants onto a destination commit
func (s *Service) RebaseCommit(ctx context.Context, sourceCommitID, destCommitID string) error {
	// jj rebase -s <source> -d <destination>
	// Using -s (source) instead of -r (revision) so descendants follow along
	args := []string{"rebase", "-s", sourceCommitID, "-d", destCommitID}
	return s.runJJ(ctx, args...)
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
	if err := s.runJJ(ctx, "squash", "--from", commitID, "--", filePath); err != nil {
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
	// jj squash --from <parent> -- <file>
	if err := s.runJJ(ctx, "squash", "--from", commitID, "--", filePath); err != nil {
		return fmt.Errorf("failed to move file to new commit: %w", err)
	}

	return nil
}

// RevertFile reverts the changes to a file in a given commit,
// restoring it from the commit's parent.
func (s *Service) RevertFile(ctx context.Context, commitID, filePath string) error {
	// jj restore --to <commit> --from parents(<commit>) -- <file>
	// Using parents() function instead of ~ suffix to avoid revset parsing issues
	parentRev := fmt.Sprintf("parents(%s)", commitID)
	args := []string{"restore", "--to", commitID, "--from", parentRev, "--", filePath}
	return s.runJJ(ctx, args...)
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
	args := []string{"git", "push", "--bookmark", branch, "--allow-new"}
	pushOut, err := s.runJJOutput(ctx, args...)
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

// FetchFromGit fetches updates from the remote git repository
func (s *Service) FetchFromGit(ctx context.Context) (string, error) {
	// Use jj git fetch to update remote bookmarks
	args := []string{"git", "fetch"}
	out, err := s.runJJOutput(ctx, args...)
	if err != nil {
		return out, fmt.Errorf("fetch failed: %w", err)
	}

	// Also run git fetch directly to ensure we get latest remote refs
	gitFetchCmd := exec.CommandContext(ctx, "git", "fetch", "origin")
	gitFetchCmd.Dir = s.RepoPath
	gitOut, gitErr := gitFetchCmd.CombinedOutput()
	if gitErr != nil {
		// Fetch failures are usually not fatal (e.g., no new changes)
		// Only return error if it's a real network/permission issue
		if !strings.Contains(string(gitOut), "Fetching from") && !strings.Contains(string(gitOut), "up-to-date") {
			out += "\nGit fetch output: " + string(gitOut)
		}
	}

	// After fetch, clean up the working copy state and any orphaned empty commits
	_ = s.cleanupAfterFetch(ctx)

	return out, nil
}

// cleanupAfterFetch handles post-fetch cleanup:
// 1. Moves working copy if it's on an immutable commit
// 2. Abandons orphaned empty commits that don't have bookmarks or content
func (s *Service) cleanupAfterFetch(ctx context.Context) error {
	// First, move working copy if it's immutable
	isImmutable, _ := s.runJJOutput(ctx, "log", "-r", "@", "--no-graph", "-T", "if(immutable, \"true\", \"false\")")
	if strings.TrimSpace(isImmutable) == "true" {
		// Working copy is immutable (e.g., after a merge). Create a new mutable descendant.
		_ = s.runJJ(ctx, "new", "@")
	}

	// Then abandon empty commits that are orphaned (have no bookmarks, no content, and are not working copy)
	// These are commits created by jj when keeping the graph valid after merges
	return s.abandonOrphanedEmptyCommits(ctx)
}

// abandonOrphanedEmptyCommits removes empty commits that have no bookmarks
// These are commits auto-created by jj to keep the working copy valid
func (s *Service) abandonOrphanedEmptyCommits(ctx context.Context) error {
	// Find empty, mutable commits with no bookmarks
	// Exclude: the working copy (@), commits on bookmarks
	orphans, _ := s.runJJOutput(ctx, "log", "-r", "empty() & mutable() & ~bookmarks() & ~@", "--no-graph", "-T", "change_id")
	if strings.TrimSpace(orphans) == "" {
		return nil
	}

	// Abandon each orphaned empty commit
	for _, changeID := range strings.Split(strings.TrimSpace(orphans), "\n") {
		changeID = strings.TrimSpace(changeID)
		if changeID != "" {
			_ = s.runJJ(ctx, "abandon", changeID)
		}
	}

	return nil
}

// getCommitGraph retrieves the commit graph with real jj data
func (s *Service) getCommitGraph(ctx context.Context) (*models.CommitGraph, error) {
	// Use a custom template with a unique marker to separate graph prefix from data
	// The marker "<<<COMMIT>>>" lets us identify where the graph ends and data begins
	// Format after marker: change_id|commit_id|author|date|description|parents|bookmarks|is_working|has_conflict|immutable
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
		if(immutable, "true", "false"),
		"\n"
	)`

	// Run WITH the graph to get ASCII art (no --reversed, keep natural newest-first order)
	// Revset: mutable commits (new work) and bookmarks (local and remote)
	// Try with main@origin first (real repos), fall back without it for test/fresh repos
	out, err := s.runJJOutput(ctx, "log", "-r", "mutable() | bookmarks() | main@origin", "-T", template)
	if err != nil {
		// main@origin doesn't exist (test repo or no remote), try without it
		out, err = s.runJJOutput(ctx, "log", "-r", "mutable() | bookmarks()", "-T", template)
		if err != nil {
			// If template fails, try simpler approach
			return s.getCommitGraphSimple(ctx)
		}
	}

	commits := []models.Commit{}
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
		if len(parts) < 10 {
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

		// Parse parents
		var parents []string
		if parentsStr != "" {
			parents = strings.Split(parentsStr, ",")
		}

		// Parse branches/bookmarks
		// Strip @remote suffixes (e.g., "main@origin" -> "main")
		// Strip * suffix (indicates current bookmark)
		var branches []string
		if branchesStr != "" {
			for _, b := range strings.Split(branchesStr, ",") {
				b = strings.TrimSpace(b)
				// Remove * suffix (current bookmark indicator)
				b = strings.TrimSuffix(b, "*")
				// Remove @remote suffix if present (e.g., "feature@origin" -> "feature")
				if idx := strings.Index(b, "@"); idx > 0 {
					b = b[:idx]
				}
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
				}
			}
		}

		// Parse date
		var date time.Time
		if dateStr != "" {
			date, _ = time.Parse(time.RFC3339, dateStr)
		}

		commit := models.Commit{
			ID:          commitID,
			ShortID:     commitID,
			ChangeID:    changeID,
			Author:      author,
			Email:       author,
			Date:        date,
			Summary:     description,
			Description: description,
			Parents:     parents,
			Branches:    branches,
			IsWorking:   isWorking,
			Conflicts:   hasConflict,
			Immutable:   isImmutable,
			GraphPrefix: graphPrefix,
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

	return &models.CommitGraph{
		Commits:     commits,
		Connections: connections,
	}, nil
}

// getCommitGraphSimple is a fallback that uses simpler parsing
func (s *Service) getCommitGraphSimple(ctx context.Context) (*models.CommitGraph, error) {
	out, err := s.runJJOutput(ctx, "log", "-r", "mutable() | bookmarks()", "--no-graph")
	if err != nil {
		return nil, err
	}

	commits := []models.Commit{}

	// Parse the default jj log output
	lines := strings.Split(out, "\n")
	var currentCommit *models.Commit

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
				currentCommit = &models.Commit{
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

	return &models.CommitGraph{
		Commits:     commits,
		Connections: make(map[string][]string),
	}, nil
}

// runJJ executes a jj command and returns a clean error if it fails
func (s *Service) runJJ(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "jj", args...)
	cmd.Dir = s.RepoPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Extract just the main error message
		errMsg := extractErrorMessage(string(out))
		if errMsg != "" {
			return fmt.Errorf("%s", errMsg)
		}
		return fmt.Errorf("command failed: %w", err)
	}
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
	cmd := exec.CommandContext(ctx, "jj", args...)
	cmd.Dir = s.RepoPath

	// Capture stdout and stderr separately
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Include stderr in error message for debugging
		errOutput := stderr.String()
		if errOutput == "" {
			errOutput = stdout.String()
		}
		return "", fmt.Errorf("jj command '%s' failed: %w\nOutput: %s",
			fmt.Sprintf("jj %s", strings.Join(args, " ")), err, errOutput)
	}

	// Return only stdout - hints/warnings go to stderr
	return stdout.String(), nil
}

// ListBranches returns all local and remote branches
// statsLimit controls how many branches get ahead/behind stats calculated (0 = all)
func (s *Service) ListBranches(ctx context.Context, statsLimit int) ([]models.Branch, error) {
	// Get all bookmarks including remote ones
	out, err := s.runJJOutput(ctx, "bookmark", "list", "--all-remotes")
	if err != nil {
		return nil, fmt.Errorf("failed to list bookmarks: %w", err)
	}

	var branches []models.Branch
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

					branches = append(branches, models.Branch{
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
			colonIdx := strings.Index(line, ":")
			if colonIdx > 0 && !isDeleted {
				// Local branch with commit info on same line
				currentBranch = strings.TrimSpace(line[:colonIdx])
				commitInfo := strings.TrimSpace(line[colonIdx+1:])
				changeID, shortID := parseCommitInfo(commitInfo)

				branches = append(branches, models.Branch{
					Name:     currentBranch,
					CommitID: changeID,
					ShortID:  shortID,
					IsLocal:  true,
				})
			} else if isDeleted {
				// Deleted local branch - extract name
				currentBranch = strings.TrimSpace(strings.TrimSuffix(line, " (deleted)"))
			} else {
				// Branch name only (rare case)
				currentBranch = strings.TrimSpace(line)
			}
		} else if strings.HasPrefix(strings.TrimSpace(line), "@") {
			// This is a remote tracking line (indented)
			// Format: "  @origin: change_id commit_id description"
			trimmedLine := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmedLine, "@") {
				continue
			}

			colonIdx := strings.Index(trimmedLine, ":")
			if colonIdx < 0 {
				continue
			}

			remote := trimmedLine[1:colonIdx] // Skip @ prefix
			// Skip git remote
			if remote == "git" {
				continue
			}

			commitInfo := strings.TrimSpace(trimmedLine[colonIdx+1:])
			changeID, shortID := parseCommitInfo(commitInfo)

			// Only add remote branch if we have a current branch name
			if currentBranch != "" {
				// A branch is tracked if it appears under a branch line (even if deleted)
				// Untracked branches appear on a single line as "branch@origin:"
				branches = append(branches, models.Branch{
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

	// Calculate ahead/behind stats for branches (limited for performance)
	// statsLimit of 0 means calculate for all branches
	maxStats := len(branches)
	if statsLimit > 0 && statsLimit < maxStats {
		maxStats = statsLimit
	}

	for i := 0; i < maxStats; i++ {
		if branches[i].IsLocal {
			branches[i].Ahead, branches[i].Behind = s.GetBranchStats(ctx, branches[i].Name, "")
		} else if branches[i].Remote != "" {
			branches[i].Ahead, branches[i].Behind = s.GetBranchStats(ctx, branches[i].Name, branches[i].Remote)
		}
	}

	return branches, nil
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
	args := []string{"bookmark", "track", remoteBranch}
	return s.runJJ(ctx, args...)
}

// UntrackBranch stops tracking a remote branch
func (s *Service) UntrackBranch(ctx context.Context, branchName, remote string) error {
	remoteBranch := fmt.Sprintf("%s@%s", branchName, remote)
	args := []string{"bookmark", "untrack", remoteBranch}
	return s.runJJ(ctx, args...)
}

// RestoreLocalBranch restores a deleted local branch from its tracked remote
func (s *Service) RestoreLocalBranch(ctx context.Context, branchName, commitID string) error {
	// Use jj bookmark set to create/restore the local bookmark at the remote's revision
	args := []string{"bookmark", "set", branchName, "-r", commitID}
	return s.runJJ(ctx, args...)
}

// PushBranch pushes a local branch to remote
func (s *Service) PushBranch(ctx context.Context, branchName string) error {
	args := []string{"git", "push", "--allow-new", "--bookmark", branchName}
	return s.runJJ(ctx, args...)
}

// FetchFromRemote fetches updates from a remote
func (s *Service) FetchFromRemote(ctx context.Context, remote string) error {
	args := []string{"git", "fetch", "--remote", remote}
	return s.runJJ(ctx, args...)
}

// FetchAllRemotes fetches from all configured remotes
func (s *Service) FetchAllRemotes(ctx context.Context) error {
	args := []string{"git", "fetch", "--all-remotes"}
	return s.runJJ(ctx, args...)
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
