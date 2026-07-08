package jj

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/madicen/jj-tui/internal/integrations/jj/jjout"
)

// CommandHistoryEntry represents a single jj command that was executed
type CommandHistoryEntry struct {
	Command   string    // Full command string (e.g., "jj log -r @")
	Timestamp time.Time // When the command was executed
	Duration  time.Duration
	Success   bool   // Whether the command succeeded
	Error     string // Error message if failed (truncated)
}

// Service handles jujutsu command execution
type Service struct {
	RepoPath       string
	commandHistory []CommandHistoryEntry
	historyMu      sync.RWMutex
	maxHistory     int // Maximum number of commands to keep

	// BookmarkListPreferTracked, when true, makes helpers that need a bookmark
	// listing call `jj bookmark list --tracked` instead of `--all-remotes`. The
	// `--tracked` form is significantly cheaper on colocated repos with hundreds
	// of stale origin/* PR branches; it's sufficient for the graph load's origin
	// divergence enrichment (which only inspects local→@origin pairs) and the
	// branches tab pairs it with a second `remote_bookmarks() & mine()` query so
	// you still see your own untracked branches.
	//
	// Production callers should set this from config.BranchesFilterToTrackedAndMine
	// (see data.InitializeServices). The zero value is false to preserve legacy
	// behavior for tests / direct NewService callers.
	BookmarkListPreferTracked bool

	// cmdRunner executes jj commands. NewService installs an execRunner; tests can
	// inject a fake via NewServiceWithRunner. Hand-constructed Services (e.g.
	// &Service{RepoPath: …} in tests) fall back to a lazily-built execRunner.
	cmdRunner Runner

	// backoutVerb* cache the reverse-revision subcommand for this jj build
	// (`backout` on older jj, renamed to `revert` in newer jj); see caps.go.
	backoutVerbOnce sync.Once
	backoutVerbVal  string
}

// runner returns the installed Runner, lazily building a default execRunner for
// Services constructed without NewService (e.g. in tests that never execute jj).
func (s *Service) runner() Runner {
	if s.cmdRunner == nil {
		s.cmdRunner = newExecRunner(s.RepoPath, s.addToHistory)
	}
	return s.cmdRunner
}

// NewServiceWithRunner builds a Service backed by a custom Runner without
// verifying a jj binary or repository. Intended for fast unit tests that feed
// canned command output through a fake Runner (see internal/mock.FakeRunner).
func NewServiceWithRunner(repoPath string, r Runner) *Service {
	return &Service{
		RepoPath:   repoPath,
		maxHistory: 100,
		cmdRunner:  r,
	}
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

	service := &Service{
		RepoPath:   repoPath,
		maxHistory: 100, // Keep last 100 commands
	}
	service.cmdRunner = newExecRunner(repoPath, service.addToHistory)

	// Test that we can actually run jj commands
	ctx := context.Background()
	if _, err := service.runJJOutput(ctx, "--version"); err != nil {
		return nil, fmt.Errorf("failed to execute jj commands: %w", err)
	}

	return service, nil
}

// GetCommandHistory returns a copy of the command history (most recent first)
func (s *Service) GetCommandHistory() []CommandHistoryEntry {
	s.historyMu.RLock()
	defer s.historyMu.RUnlock()

	// Return a copy in reverse order (most recent first)
	result := make([]CommandHistoryEntry, len(s.commandHistory))
	for i, entry := range s.commandHistory {
		result[len(s.commandHistory)-1-i] = entry
	}
	return result
}

// addToHistory adds a command entry to the history
func (s *Service) addToHistory(entry CommandHistoryEntry) {
	s.historyMu.Lock()
	defer s.historyMu.Unlock()

	s.commandHistory = append(s.commandHistory, entry)

	// Trim history if it exceeds the limit; copy to new slice to release backing array
	if len(s.commandHistory) > s.maxHistory {
		keep := s.commandHistory[len(s.commandHistory)-s.maxHistory:]
		s.commandHistory = append([]CommandHistoryEntry(nil), keep...)
	}
}

// runJJOutputNoHistory runs jj without recording command history (used for graph enrichment).
func (s *Service) runJJOutputNoHistory(ctx context.Context, args ...string) (string, error) {
	return s.runJJOutputNoHistoryWithGlobal(ctx, nil, args...)
}

// runJJOutputNoHistoryWithGlobal is like runJJOutputNoHistory but prepends global jj flags.
func (s *Service) runJJOutputNoHistoryWithGlobal(ctx context.Context, global []string, args ...string) (string, error) {
	return s.runner().RunOutput(ctx, RunOpts{Global: global, NoHistory: true}, args...)
}

// runJJWithGlobal runs jj with optional global flags before subcommand (e.g. --ignore-working-copy).
func (s *Service) runJJWithGlobal(ctx context.Context, global []string, args ...string) error {
	return s.runner().Run(ctx, RunOpts{Global: global}, args...)
}

// runJJOutputWithGlobal is like runJJOutput but prepends global jj flags.
func (s *Service) runJJOutputWithGlobal(ctx context.Context, global []string, args ...string) (string, error) {
	return s.runner().RunOutput(ctx, RunOpts{Global: global}, args...)
}

// runJJ executes a jj command and returns a clean error if it fails
func (s *Service) runJJ(ctx context.Context, args ...string) error {
	return s.runner().Run(ctx, RunOpts{}, args...)
}

// extractErrorMessage extracts the main error message from jj output.
func extractErrorMessage(output string) string {
	return jjout.ExtractErrorMessage(output)
}

// runJJOutput executes a jj command and returns its stdout only
// stderr is captured separately to avoid jj hints/warnings mixing into parsed output
func (s *Service) runJJOutput(ctx context.Context, args ...string) (string, error) {
	return s.runner().RunOutput(ctx, RunOpts{}, args...)
}

// runJJCombined executes a jj command and returns merged stdout+stderr. Used for
// commands whose useful summary is written to stderr (e.g. absorb).
func (s *Service) runJJCombined(ctx context.Context, args ...string) (string, error) {
	return s.runner().RunCombined(ctx, RunOpts{}, args...)
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
