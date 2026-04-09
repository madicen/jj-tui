package util

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/config"
)

// After a successful editor Start(), wait this long before clearing the loading overlay so the
// spinner is perceptible (Start() returns as soon as the OS accepts the child, which is ~instant).
const externalEditorLoadingMinVisible = 1000 * time.Millisecond

// ExternalEditorOpenedMsg is sent after the external editor process was started successfully (clears loading overlay).
type ExternalEditorOpenedMsg struct {
	FileBase string // basename for status line, optional
}

// RepoAbsPath returns the absolute path for a repo-relative file and ensures it stays under repoRoot.
func RepoAbsPath(repoRoot, relPath string) (string, error) {
	repoRoot = filepath.Clean(repoRoot)
	if repoRoot == "" || repoRoot == "." {
		return "", fmt.Errorf("repository path is not set")
	}
	joined := filepath.Join(repoRoot, filepath.FromSlash(strings.TrimSpace(relPath)))
	cleanJoined := filepath.Clean(joined)
	rel, err := filepath.Rel(repoRoot, cleanJoined)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes repository")
	}
	return cleanJoined, nil
}

// OpenFileInExternalEditorCmd runs the configured editor against an absolute file path (non-blocking for GUI editors).
func OpenFileInExternalEditorCmd(absPath string, cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		if err := openFileInExternalEditor(absPath, cfg); err != nil {
			return ErrorMsg{Err: err, StatusOnly: true}
		}
		time.Sleep(externalEditorLoadingMinVisible)
		return ExternalEditorOpenedMsg{FileBase: filepath.Base(absPath)}
	}
}

func openFileInExternalEditor(absPath string, cfg *config.Config) error {
	preset := config.NormalizeExternalFileEditor(cfg)
	if preset == config.ExternalEditorNone {
		return fmt.Errorf("no external editor configured (Settings → Advanced)")
	}

	absPath = filepath.Clean(absPath)
	if !filepath.IsAbs(absPath) {
		return fmt.Errorf("internal error: path must be absolute")
	}

	switch preset {
	case config.ExternalEditorCursor:
		return startDetached(exec.Command("cursor", absPath))
	case config.ExternalEditorVSCode:
		return startDetached(exec.Command("code", "-g", absPath))
	case config.ExternalEditorZed:
		return startDetached(exec.Command("zed", absPath))
	case config.ExternalEditorNeovim:
		// Requires Neovim remote (nvr) and a listening nvim instance.
		if err := startDetached(exec.Command("nvr", "--remote", absPath)); err != nil {
			return fmt.Errorf("nvr: %w (start nvim with --listen, or use Custom in settings)", err)
		}
		return nil
	case config.ExternalEditorEmacs:
		return startDetached(exec.Command("emacsclient", "-n", "-a", "emacs", absPath))
	case config.ExternalEditorSublime:
		return startDetached(exec.Command("subl", absPath))
	case config.ExternalEditorIntelliJ:
		return startDetached(exec.Command("idea", absPath))
	case config.ExternalEditorCustom:
		tpl := ""
		if cfg != nil {
			tpl = strings.TrimSpace(cfg.ExternalFileEditorCustom)
		}
		if tpl == "" || !strings.Contains(tpl, "{path}") {
			return fmt.Errorf(`custom editor needs a command with {path} (Settings → Advanced)`)
		}
		q := shellQuoteSingle(absPath)
		script := strings.ReplaceAll(tpl, "{path}", q)
		cmd := exec.Command("sh", "-c", script)
		return startDetached(cmd)
	default:
		return fmt.Errorf("unknown editor preset %q", preset)
	}
}

func shellQuoteSingle(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
