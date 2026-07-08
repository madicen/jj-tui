package jj

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/madicen/jj-tui/internal/integrations/jj/jjout"
)

// Workspace describes one entry from `jj workspace list`.
type Workspace struct {
	Name        string // workspace name (e.g. "default")
	ChangeID    string // change_id of the workspace's working-copy commit
	CommitID    string // commit_id of the workspace's working-copy commit
	Description string // first line of the working-copy commit description
	Root        string // absolute path to the workspace root
	Current     bool   // true for the workspace this Service operates in
}

// workspaceListTemplate renders one tab-separated row per workspace:
// name, change id, commit id, description first line, root path.
const workspaceListTemplate = `name ++ "\t" ++ target.change_id().short(8) ++ "\t" ++ ` +
	`target.commit_id().short(8) ++ "\t" ++ target.description().first_line() ++ "\t" ++ root ++ "\n"`

// ListWorkspaces returns the repository's workspaces, marking the current one
// (the workspace this Service's repo path resolves to).
func (s *Service) ListWorkspaces(ctx context.Context) ([]Workspace, error) {
	out, err := s.runJJOutput(ctx, "workspace", "list", "-T", workspaceListTemplate)
	if err != nil {
		return nil, err
	}
	// The current workspace root is used to flag which row is "this" workspace;
	// jj workspace list doesn't mark it, so we compare against `workspace root`.
	currentRoot := ""
	if root, rerr := s.runJJOutputNoHistory(ctx, "workspace", "root"); rerr == nil {
		currentRoot = normalizeWorkspacePath(root)
	}

	var workspaces []Workspace
	for _, line := range jjout.SplitLines(out) {
		parts := strings.SplitN(line, "\t", 5)
		if len(parts) < 2 {
			continue
		}
		w := Workspace{Name: strings.TrimSpace(parts[0])}
		if len(parts) >= 2 {
			w.ChangeID = strings.TrimSpace(parts[1])
		}
		if len(parts) >= 3 {
			w.CommitID = strings.TrimSpace(parts[2])
		}
		if len(parts) >= 4 {
			w.Description = strings.TrimSpace(parts[3])
		}
		if len(parts) >= 5 {
			w.Root = normalizeWorkspacePath(parts[4])
		}
		if currentRoot != "" && w.Root != "" && w.Root == currentRoot {
			w.Current = true
		}
		workspaces = append(workspaces, w)
	}
	return workspaces, nil
}

// AddWorkspace creates a new workspace rooted at destPath (jj workspace add PATH).
func (s *Service) AddWorkspace(ctx context.Context, destPath string) error {
	return s.runJJ(ctx, "workspace", "add", destPath)
}

// ForgetWorkspace removes the named workspace from the repo (jj workspace forget
// NAME). The workspace's directory on disk is left untouched.
func (s *Service) ForgetWorkspace(ctx context.Context, name string) error {
	return s.runJJ(ctx, "workspace", "forget", name)
}

// normalizeWorkspacePath trims and, where possible, resolves symlinks so the
// current-workspace comparison is robust to /var vs /private/var style aliases.
func normalizeWorkspacePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		return resolved
	}
	return filepath.Clean(p)
}
