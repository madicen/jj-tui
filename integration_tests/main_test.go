package integration_tests

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/madicen/jj-tui/internal/jj"
	"github.com/madicen/jj-tui/internal/models"
)

// TestRepository manages a test jj repository
type TestRepository struct {
	Path string
	t    *testing.T
}

// NewTestRepository creates a new test repository
func NewTestRepository(t *testing.T) *TestRepository {
	tempDir, err := os.MkdirTemp("", "jj-tui-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	repo := &TestRepository{
		Path: tempDir,
		t:    t,
	}

	// Initialize jj repository
	if err := repo.runCommand("jj", "git", "init"); err != nil {
		t.Fatalf("Failed to initialize jj repository: %v", err)
	}

	// Set up basic git config
	repo.runCommand("jj", "config", "set", "--repo", "user.name", "Test User")
	repo.runCommand("jj", "config", "set", "--repo", "user.email", "test@example.com")

	return repo
}

// Cleanup removes the test repository
func (r *TestRepository) Cleanup() {
	if r.Path != "" {
		os.RemoveAll(r.Path)
	}
}

// runCommand executes a command in the repository directory
func (r *TestRepository) runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = r.Path
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// writeFile writes content to a file in the repository
func (r *TestRepository) writeFile(filename, content string) error {
	fullPath := filepath.Join(r.Path, filename)
	return os.WriteFile(fullPath, []byte(content), 0644)
}

// commitFile creates a file and commits it
func (r *TestRepository) commitFile(filename, content, message string) error {
	if err := r.writeFile(filename, content); err != nil {
		return err
	}
	return r.runCommand("jj", "commit", "-m", message)
}

// TestJJServiceBasicOperations tests basic jj service operations
func TestJJServiceBasicOperations(t *testing.T) {
	// Skip if jj is not available
	if _, err := exec.LookPath("jj"); err != nil {
		t.Skip("jj command not available")
	}

	repo := NewTestRepository(t)
	defer repo.Cleanup()

	ctx := context.Background()
	service, err := jj.NewService(repo.Path)
	if err != nil {
		t.Fatalf("Failed to create jj service: %v", err)
	}

	t.Run("GetRepository", func(t *testing.T) {
		repository, err := service.GetRepository(ctx)
		if err != nil {
			t.Fatalf("Failed to get repository: %v", err)
		}

		if repository.Path != repo.Path {
			t.Errorf("Expected path %s, got %s", repo.Path, repository.Path)
		}

		if len(repository.Graph.Commits) == 0 {
			t.Error("Expected at least one commit (working copy)")
		}
	})

	t.Run("CreateCommit", func(t *testing.T) {
		// Create a file and commit it
		if err := repo.writeFile("test.txt", "Hello, World!"); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		if err := service.CreateNewCommit(ctx, "Add test file"); err != nil {
			t.Fatalf("Failed to create commit: %v", err)
		}

		// Verify the commit was created
		repository, err := service.GetRepository(ctx)
		if err != nil {
			t.Fatalf("Failed to get repository after commit: %v", err)
		}

		// Should have at least 2 commits now (initial + our commit)
		if len(repository.Graph.Commits) < 2 {
			t.Errorf("Expected at least 2 commits, got %d", len(repository.Graph.Commits))
		}

		// Find our commit
		found := false
		for _, commit := range repository.Graph.Commits {
			if commit.Summary == "Add test file" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Could not find the created commit")
		}
	})
}

// TestCommitGraphVisualization tests commit graph visualization
func TestCommitGraphVisualization(t *testing.T) {
	if _, err := exec.LookPath("jj"); err != nil {
		t.Skip("jj command not available")
	}

	repo := NewTestRepository(t)
	defer repo.Cleanup()

	ctx := context.Background()
	service, err := jj.NewService(repo.Path)
	if err != nil {
		t.Fatalf("Failed to create jj service: %v", err)
	}

	// Create a series of commits
	commits := []struct {
		filename string
		content  string
		message  string
	}{
		{"file1.txt", "Content 1", "First commit"},
		{"file2.txt", "Content 2", "Second commit"},
		{"file3.txt", "Content 3", "Third commit"},
	}

	for _, commit := range commits {
		if err := repo.commitFile(commit.filename, commit.content, commit.message); err != nil {
			t.Fatalf("Failed to create commit %s: %v", commit.message, err)
		}
	}

	// Get repository state
	repository, err := service.GetRepository(ctx)
	if err != nil {
		t.Fatalf("Failed to get repository: %v", err)
	}

	// Verify we have the expected commits
	if len(repository.Graph.Commits) < len(commits) {
		t.Errorf("Expected at least %d commits, got %d", len(commits), len(repository.Graph.Commits))
	}

	// Verify commit messages are present
	commitMessages := make(map[string]bool)
	for _, commit := range repository.Graph.Commits {
		commitMessages[commit.Summary] = true
	}

	for _, expected := range commits {
		if !commitMessages[expected.message] {
			t.Errorf("Expected commit message '%s' not found", expected.message)
		}
	}
}

// TestPRWorkflow tests the PR creation workflow
func TestPRWorkflow(t *testing.T) {
	// This test focuses on the data structures and workflow
	// without actually creating GitHub PRs

	t.Run("CreatePRRequest", func(t *testing.T) {
		req := &models.CreatePRRequest{
			Title:      "Test PR",
			Body:       "This is a test PR",
			BaseBranch: "main",
			HeadBranch: "feature-branch",
			CommitIDs:  []string{"commit1", "commit2"},
			Draft:      false,
		}

		if req.Title != "Test PR" {
			t.Errorf("Expected title 'Test PR', got '%s'", req.Title)
		}

		if len(req.CommitIDs) != 2 {
			t.Errorf("Expected 2 commit IDs, got %d", len(req.CommitIDs))
		}
	})

	t.Run("UpdatePRRequest", func(t *testing.T) {
		req := &models.UpdatePRRequest{
			Title:     "Updated PR Title",
			CommitIDs: []string{"commit1", "commit2", "commit3"},
		}

		if req.Title != "Updated PR Title" {
			t.Errorf("Expected title 'Updated PR Title', got '%s'", req.Title)
		}

		if len(req.CommitIDs) != 3 {
			t.Errorf("Expected 3 commit IDs, got %d", len(req.CommitIDs))
		}
	})
}

// TestRepositoryState tests repository state management
func TestRepositoryState(t *testing.T) {
	repo := &models.Repository{
		Path: "/test/path",
		WorkingCopy: models.Commit{
			ID:        "working",
			ShortID:   "work",
			Summary:   "Working copy",
			IsWorking: true,
		},
		Graph: models.CommitGraph{
			Commits: []models.Commit{
				{
					ID:      "commit1",
					ShortID: "com1",
					Summary: "First commit",
					Parents: []string{},
				},
				{
					ID:      "commit2",
					ShortID: "com2",
					Summary: "Second commit",
					Parents: []string{"commit1"},
				},
			},
			Connections: map[string][]string{
				"commit1": {"commit2"},
			},
		},
		PRs: []models.GitHubPR{},
	}

	if repo.Path != "/test/path" {
		t.Errorf("Expected path '/test/path', got '%s'", repo.Path)
	}

	if !repo.WorkingCopy.IsWorking {
		t.Error("Expected working copy to be marked as working")
	}

	if len(repo.Graph.Commits) != 2 {
		t.Errorf("Expected 2 commits, got %d", len(repo.Graph.Commits))
	}

	if len(repo.Graph.Connections["commit1"]) != 1 {
		t.Errorf("Expected commit1 to have 1 connection, got %d", len(repo.Graph.Connections["commit1"]))
	}
}

// BenchmarkRepositoryLoad benchmarks repository loading
func BenchmarkRepositoryLoad(b *testing.B) {
	if _, err := exec.LookPath("jj"); err != nil {
		b.Skip("jj command not available")
	}

	repo := NewTestRepository(&testing.T{})
	defer repo.Cleanup()

	// Create some commits for a more realistic benchmark
	for i := 0; i < 10; i++ {
		filename := fmt.Sprintf("file%d.txt", i)
		content := fmt.Sprintf("Content %d", i)
		message := fmt.Sprintf("Commit %d", i)
		repo.commitFile(filename, content, message)
	}

	ctx := context.Background()
	service, err := jj.NewService(repo.Path)
	if err != nil {
		b.Fatalf("Failed to create jj service: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := service.GetRepository(ctx)
		if err != nil {
			b.Fatalf("Failed to get repository: %v", err)
		}
	}
}
