package data

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/integrations/jj"
)

// RemoteOp identifies which operation a RemoteOpResultMsg corresponds to. Main uses this to
// route the result (refresh repo, update status text, surface errors).
type RemoteOp int

const (
	// RemoteOpApply added or updated `origin` to a user-supplied URL.
	RemoteOpApply RemoteOp = iota
	// RemoteOpCreateGh created a brand-new GitHub repo via `gh repo create` and wired up `origin`.
	RemoteOpCreateGh
	// RemoteOpRemove deleted the `origin` remote.
	RemoteOpRemove
)

// RemoteOpResultMsg is sent when an Apply / CreateGh / Remove command finishes. Err is non-nil
// on failure; PreviousURL/NewURL describe what changed for the success status line.
//
// When Op == RemoteOpCreateGh and Err == nil, the optional fields PushedCount / PushedNames /
// PushOutput / PushErr describe the inline post-create push attempt. PushErr being non-nil
// while Err is nil is the soft-failure case: the GitHub repo was created and origin is wired
// up, but pushing local bookmarks failed (e.g. network blip, auth issue). The user can press
// the Push all bookmarks button to retry without losing the new repo.
type RemoteOpResultMsg struct {
	Op          RemoteOp
	NewURL      string
	PreviousURL string
	Err         error

	PushedCount int
	PushedNames []string
	PushOutput  string
	PushErr     error
}

// PushResultMsg is sent when the standalone Push current / Push all buttons finish. Separate
// from RemoteOpResultMsg so the two flows have independent status text and so the auto-push
// after Create can be styled distinctly from a deliberate user-initiated push.
type PushResultMsg struct {
	// All records user intent: true => "Push all bookmarks" (we enumerated and pushed each via
	// --bookmark <name>), false => "Push current bookmark" (default jj behavior, no flag).
	All         bool
	PushedCount int
	PushedNames []string
	Output      string
	Err         error
}

// ApplyOriginCmd adds `origin` if missing or updates its URL otherwise. Empty url is rejected
// here; callers wanting to remove the remote must use RemoveOriginCmd. After mutating the remote
// we run a best-effort `jj git fetch` so the caller can refresh and immediately see remote
// bookmarks (e.g. main@origin) without a second user action.
func ApplyOriginCmd(svc *jj.Service, url string) tea.Cmd {
	url = strings.TrimSpace(url)
	return func() tea.Msg {
		msg := RemoteOpResultMsg{Op: RemoteOpApply, NewURL: url}
		if url == "" {
			msg.Err = fmt.Errorf("remote URL is empty (use Remove to delete origin)")
			return msg
		}
		ctx := context.Background()
		current, _ := readOriginURL(ctx, svc)
		msg.PreviousURL = current
		if current == "" {
			if err := runJJRemote(ctx, svc, "add", "origin", url); err != nil {
				msg.Err = err
				return msg
			}
		} else if current != url {
			if err := runJJRemote(ctx, svc, "set-url", "origin", url); err != nil {
				msg.Err = err
				return msg
			}
		}
		// Best-effort: ignore errors so an unreachable URL still leaves origin configured.
		_ = runJJ(ctx, svc, "git", "fetch")
		return msg
	}
}

// CreateGhRepoCmd creates a new GitHub repository via `gh repo create`, wires up `origin`, and
// then attempts an inline push of every local bookmark (`jj git push --allow-new --bookmark
// <name>` repeated per name) so the user's existing work reaches the new remote in a single
// user action. We use jj's push (not `gh repo create --push`) because gh's --push runs raw
// `git push -u origin <branch>` which skips jj's import/export step and can leave colocated
// repos slightly out of sync; pushing every bookmark explicitly also handles stacked work and
// stays compatible across jj versions that renamed `--all-bookmarks` (the older flag is
// rejected on some currently-supported builds, so we avoid it).
//
// The push step is intentionally soft-failing: if create succeeds but push doesn't (e.g. auth,
// network, or "no bookmarks yet"), the GitHub repo is preserved and the result message carries
// PushErr so the UI can surface the partial outcome and offer a retry via the Push all button.
func CreateGhRepoCmd(svc *jj.Service, name string, private bool) tea.Cmd {
	return func() tea.Msg {
		msg := RemoteOpResultMsg{Op: RemoteOpCreateGh}
		if _, err := exec.LookPath("gh"); err != nil {
			msg.Err = fmt.Errorf("gh CLI not found in PATH (install gh and run `gh auth login`): %w", err)
			return msg
		}
		name = strings.TrimSpace(name)
		if name == "" {
			cwd, _ := os.Getwd()
			name = filepath.Base(cwd)
		}
		visibility := "--public"
		if private {
			visibility = "--private"
		}
		ctx := context.Background()
		current, _ := readOriginURL(ctx, svc)
		msg.PreviousURL = current
		if current != "" {
			msg.Err = fmt.Errorf("origin already configured (%s); remove it first or use Apply to change the URL", current)
			return msg
		}
		out, err := exec.Command("gh", "repo", "create", name, visibility, "--source=.", "--remote=origin").CombinedOutput()
		if err != nil {
			msg.Err = fmt.Errorf("gh repo create %s failed: %s", name, strings.TrimSpace(string(out)))
			return msg
		}
		newURL, _ := readOriginURL(ctx, svc)
		msg.NewURL = newURL
		_ = runJJ(ctx, svc, "git", "fetch")

		// Auto-push: only attempt when there's at least one local bookmark to push. Empty repos
		// (no commits, no bookmarks) hit a clean "nothing to push" status without surfacing an
		// error modal — that's the welcome-screen-style first-init case where the user simply
		// has nothing yet.
		names := listLocalBookmarks(ctx, svc)
		if len(names) > 0 {
			pushOut, pushErr := pushBookmarks(ctx, svc, names)
			msg.PushOutput = pushOut
			msg.PushErr = pushErr
			if pushErr == nil {
				msg.PushedCount = len(names)
				msg.PushedNames = names
			}
		}
		return msg
	}
}

// PushBookmarksCmd runs `jj git push --allow-new` against the configured origin. The "all"
// branch enumerates local bookmarks and pushes each via `--bookmark <name>` so we don't depend
// on `--all-bookmarks`, which is unrecognized on some currently-supported jj versions; the
// "current" branch passes no bookmark flag and lets jj's default selection apply (the bookmark
// on @). Used by the standalone Push current / Push all buttons in the Repository remote
// panel. Independent of CreateGhRepoCmd's auto-push so users can retry / push later without
// re-creating the GitHub repo.
func PushBookmarksCmd(svc *jj.Service, all bool) tea.Cmd {
	return func() tea.Msg {
		msg := PushResultMsg{All: all}
		ctx := context.Background()
		// Fail fast with a clear error if the user clicks Push before configuring origin.
		// Without this guard the underlying jj command emits a less-friendly error.
		if origin, _ := readOriginURL(ctx, svc); origin == "" {
			msg.Err = fmt.Errorf("no `origin` remote configured (use Apply or Create new GitHub repo above first)")
			return msg
		}
		if all {
			names := listLocalBookmarks(ctx, svc)
			if len(names) == 0 {
				// Match the auto-push-after-create no-op: don't pop an error modal for a
				// genuinely empty repo. Caller decides whether to surface a status line.
				return msg
			}
			out, err := pushBookmarks(ctx, svc, names)
			msg.Output = out
			if err != nil {
				msg.Err = err
				return msg
			}
			msg.PushedNames = names
			msg.PushedCount = len(names)
			return msg
		}
		out, err := pushBookmarks(ctx, svc, nil)
		msg.Output = out
		if err != nil {
			msg.Err = err
			return msg
		}
		// Resolve the bookmark on @ for the status line. Best-effort; failure here just leaves
		// PushedNames empty and PushedCount=1 (we successfully pushed something).
		if name, derr := svc.GetCurrentBranch(ctx); derr == nil && strings.TrimSpace(name) != "" {
			msg.PushedNames = []string{strings.TrimSpace(name)}
		}
		msg.PushedCount = 1
		return msg
	}
}

// pushBookmarks runs `jj git push --allow-new` against origin and returns its combined output.
// When names is empty, no `--bookmark` flag is passed and jj's default selection applies (the
// bookmark on @). When names is non-empty, each entry is forwarded as `--bookmark <name>`, which
// is the version-portable way to push multiple bookmarks at once: jj historically renamed the
// "all bookmarks" shorthand (e.g. `--all-bookmarks`), and some currently-shipping jj builds
// reject it outright, while every bookmark-era jj accepts the singular `--bookmark` flag. The
// helper is shared between CreateGhRepoCmd's auto-push step and PushBookmarksCmd so both entry
// points produce identical jj invocations.
func pushBookmarks(ctx context.Context, svc *jj.Service, names []string) (string, error) {
	if svc == nil {
		return "", fmt.Errorf("jj service unavailable")
	}
	args := []string{"git", "push", "--allow-new"}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		args = append(args, "--bookmark", name)
	}
	cmd := exec.CommandContext(ctx, "jj", args...)
	if svc.RepoPath != "" {
		cmd.Dir = svc.RepoPath
	}
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))
	if err != nil {
		return output, fmt.Errorf("jj %s: %s", strings.Join(args, " "), output)
	}
	return output, nil
}

// listLocalBookmarks returns the names of local bookmarks (no @remote suffix). Used to decide
// whether the auto-push step has anything to do; an empty result means the repo is fresh and we
// should skip push silently rather than emit a confusing error.
func listLocalBookmarks(ctx context.Context, svc *jj.Service) []string {
	if svc == nil {
		return nil
	}
	cmd := exec.CommandContext(ctx, "jj", "bookmark", "list", "--template", "name ++ \"\\n\"")
	if svc.RepoPath != "" {
		cmd.Dir = svc.RepoPath
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil
	}
	var names []string
	seen := map[string]struct{}{}
	for _, raw := range strings.Split(string(out), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		// `jj bookmark list --template name` includes both local and remote-tracking entries
		// (the latter formatted as "name@remote"); we only want bookmarks the user owns
		// locally, since those are what the per-bookmark push targets.
		if strings.Contains(line, "@") {
			continue
		}
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		names = append(names, line)
	}
	return names
}

// RemoveOriginCmd deletes the `origin` remote. Returns an error if origin doesn't exist (so the
// caller can surface a clear message rather than silently no-oping).
func RemoveOriginCmd(svc *jj.Service) tea.Cmd {
	return func() tea.Msg {
		msg := RemoteOpResultMsg{Op: RemoteOpRemove}
		ctx := context.Background()
		current, _ := readOriginURL(ctx, svc)
		msg.PreviousURL = current
		if current == "" {
			msg.Err = fmt.Errorf("no `origin` remote to remove")
			return msg
		}
		if err := runJJRemote(ctx, svc, "remove", "origin"); err != nil {
			msg.Err = err
			return msg
		}
		return msg
	}
}

// readOriginURL returns the current `origin` URL or "" if no origin is configured. Errors are
// returned as-is for diagnostics; callers typically treat any error as "no origin yet".
func readOriginURL(ctx context.Context, svc *jj.Service) (string, error) {
	if svc == nil {
		return "", fmt.Errorf("jj service unavailable")
	}
	url, err := svc.GetGitRemoteURL(ctx)
	if err != nil {
		// GetGitRemoteURL returns "no git remotes found" when nothing is configured; treat that
		// as a clean empty state rather than an error so callers can render "(none)" without a
		// special-case branch.
		if strings.Contains(err.Error(), "no git remotes") {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(url), nil
}

// runJJRemote runs `jj git remote <args>` against the active jj service. Wrapping at this level
// keeps the cmds in this file from each repeating the same error formatting and stdout/stderr
// capture machinery.
func runJJRemote(ctx context.Context, svc *jj.Service, args ...string) error {
	full := append([]string{"git", "remote"}, args...)
	return runJJ(ctx, svc, full...)
}

// runJJ executes a jj subcommand by name and surfaces stderr in the returned error so the user
// sees the actual git-remote / fetch / network message rather than just `exit status 1`. The jj
// service exposes runJJOutput as a method but it's package-private; we call into the public
// surface (CombinedOutput via os/exec) so this file stays independent of the service internals.
func runJJ(ctx context.Context, svc *jj.Service, args ...string) error {
	if svc == nil {
		return fmt.Errorf("jj service unavailable")
	}
	cmd := exec.CommandContext(ctx, "jj", args...)
	if svc.RepoPath != "" {
		cmd.Dir = svc.RepoPath
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("jj %s: %s", strings.Join(args, " "), strings.TrimSpace(string(out)))
	}
	return nil
}
