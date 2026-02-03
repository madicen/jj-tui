package jj

import (
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
	if err != nil {
		return false
	}
	return true
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

// CreateBranchFromMain creates a new branch from main, bookmarks it, and rebases
// the current working copy onto it. This is used when creating a new branch for a Jira ticket.
func (s *Service) CreateBranchFromMain(ctx context.Context, bookmarkName string) error {
	// Get the current working copy's change ID before we move away from it
	currentChangeID, err := s.runJJOutput(ctx, "log", "-r", "@", "--no-graph", "-T", "change_id")
	if err != nil {
		return fmt.Errorf("failed to get current working copy: %w", err)
	}
	currentChangeID = strings.TrimSpace(currentChangeID)

	// Check if the current working copy is empty (no changes)
	// If it's empty, we'll just create the new branch without rebasing
	status, _ := s.runJJOutput(ctx, "log", "-r", "@", "--no-graph", "-T", "empty")
	isEmpty := strings.TrimSpace(status) == "true"

	// Create a new commit from main
	if err := s.runJJ(ctx, "new", "main"); err != nil {
		return fmt.Errorf("failed to create new commit from main: %w", err)
	}

	// Create bookmark on the new commit
	if err := s.runJJ(ctx, "bookmark", "create", bookmarkName); err != nil {
		return fmt.Errorf("failed to create bookmark: %w", err)
	}

	// If the old working copy had changes, rebase it onto the new branch
	if !isEmpty && currentChangeID != "" {
		// Rebase the old working copy onto the new branch
		if err := s.runJJ(ctx, "rebase", "-s", currentChangeID, "-d", "@"); err != nil {
			return fmt.Errorf("failed to rebase current work onto new branch: %w", err)
		}

		// Edit the rebased commit (move working copy to it)
		if err := s.runJJ(ctx, "edit", currentChangeID); err != nil {
			return fmt.Errorf("failed to edit rebased commit: %w", err)
		}
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
// After squashing, it moves to the squash result (the parent that received the changes)
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
	if err := s.runJJ(ctx, args...); err != nil {
		return err
	}

	// After squash, @ is a new empty commit on top of the squash result
	// Use `jj edit @-` to move directly to the parent (squash result)
	// This abandons the empty commit automatically
	return s.runJJ(ctx, "edit", "@-")
}

// NewCommit creates a new empty commit after the current working copy
func (s *Service) NewCommit(ctx context.Context) error {
	args := []string{"new"}
	return s.runJJ(ctx, args...)
}

// AbandonCommit abandons a commit, removing it from the repository
func (s *Service) AbandonCommit(ctx context.Context, commitID string) error {
	args := []string{"abandon", commitID}
	return s.runJJ(ctx, args...)
}

// RebaseCommit rebases a commit onto a destination commit
func (s *Service) RebaseCommit(ctx context.Context, sourceCommitID, destCommitID string) error {
	// jj rebase -r <source> -d <destination>
	args := []string{"rebase", "-r", sourceCommitID, "-d", destCommitID}
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
	// Revset: all mutable (unpushed) commits plus bookmarks - this catches sibling branches
	out, err := s.runJJOutput(ctx, "log", "-r", "mutable() | bookmarks()", "-T", template)
	if err != nil {
		// If template fails, try simpler approach
		return s.getCommitGraphSimple(ctx)
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

// runJJOutput executes a jj command and returns its output
func (s *Service) runJJOutput(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "jj", args...)
	cmd.Dir = s.RepoPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("jj command '%s' failed: %w\nOutput: %s",
			fmt.Sprintf("jj %s", strings.Join(args, " ")), err, string(out))
	}
	// Filter out warning lines
	return filterWarnings(string(out)), nil
}

// filterWarnings removes jj warning messages from output
func filterWarnings(output string) string {
	lines := strings.Split(output, "\n")
	var filtered []string
	skip := false
	for _, line := range lines {
		// Skip Warning: and Hint: lines and their continuations
		if strings.HasPrefix(line, "Warning:") || strings.HasPrefix(line, "Hint:") {
			skip = true
			continue
		}
		if skip && strings.HasPrefix(line, "  ") {
			continue
		}
		skip = false
		filtered = append(filtered, line)
	}
	return strings.Join(filtered, "\n")
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
