package data

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// InitOptions controls how RunJJInit initializes the working directory. Defaults (zero value)
// reproduce the legacy behavior: plain `jj git init` with a best-effort `jj bookmark track
// main@origin` afterwards.
type InitOptions struct {
	// Colocate runs `jj git init --colocate` so a `.git` directory is created alongside the jj
	// repo. Required for `git remote add` and `gh repo create --source=.` to work, and matches
	// what every fixture/demo in this repo uses.
	Colocate bool
	// RemoteURL, when non-empty, is added as the `origin` remote after init and a best-effort
	// `jj git fetch` is run so jj sees any remote bookmarks. Ignored when GhCreateRepo is true
	// (gh repo create wires up origin itself).
	RemoteURL string
	// GhCreateRepo runs `gh repo create <GhRepoName> --private/--public --source=. --remote=origin`
	// after the jj/git init. Requires the gh CLI installed and authenticated.
	GhCreateRepo bool
	// GhRepoName is the repository name passed to `gh repo create`. Empty falls back to
	// filepath.Base(cwd). Only used when GhCreateRepo is true.
	GhRepoName string
	// GhRepoPrivate selects visibility for `gh repo create`: true => --private, false => --public.
	// Only used when GhCreateRepo is true.
	GhRepoPrivate bool
}

// RunJJInit runs `jj git init` (optionally `--colocate`) in the current directory and, depending
// on opts, wires up an `origin` remote either by URL or by creating a fresh GitHub repo via the
// gh CLI. Returns a cmd that sends JJInitSuccessMsg on success or InitErrorMsg on failure.
func RunJJInit(opts InitOptions) tea.Cmd {
	return func() tea.Msg {
		args := []string{"git", "init"}
		if opts.Colocate {
			args = append(args, "--colocate")
		}
		if output, err := exec.Command("jj", args...).CombinedOutput(); err != nil {
			return InitErrorMsg{
				Err:       fmt.Errorf("failed to initialize repository: %s", strings.TrimSpace(string(output))),
				NotJJRepo: true,
			}
		}

		switch {
		case opts.GhCreateRepo:
			if err := createGitHubRepo(opts); err != nil {
				// jj init already succeeded; flag JJInitialized so main dismisses the init
				// screen and loads the (now-valid) jj repo while still surfacing the gh
				// failure as a user-visible error.
				return InitErrorMsg{Err: err, JJInitialized: true}
			}
		case strings.TrimSpace(opts.RemoteURL) != "":
			if err := addRemoteOrigin(opts); err != nil {
				return InitErrorMsg{Err: err, JJInitialized: true}
			}
		}

		// Best-effort: track main@origin so the user lands in a useful state when the remote has
		// a `main` branch. Silent failure is intentional (no remote yet, or remote has no main).
		_, _ = exec.Command("jj", "bookmark", "track", "main@origin").CombinedOutput()
		return JJInitSuccessMsg{}
	}
}

// createGitHubRepo runs `gh repo create` with non-interactive flags so it succeeds without prompts
// when gh is authed. We deliberately do not pass `--push` because the freshly initialized repo
// usually has no commits yet; the user pushes via the normal jj-tui PR/push flows later.
func createGitHubRepo(opts InitOptions) error {
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("gh CLI not found in PATH (install gh, or paste a remote URL instead): %w", err)
	}
	name := strings.TrimSpace(opts.GhRepoName)
	if name == "" {
		cwd, _ := os.Getwd()
		name = filepath.Base(cwd)
	}
	visibility := "--public"
	if opts.GhRepoPrivate {
		visibility = "--private"
	}
	ghArgs := []string{"repo", "create", name, visibility, "--source=.", "--remote=origin"}
	out, err := exec.Command("gh", ghArgs...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh repo create %s failed: %s", name, strings.TrimSpace(string(out)))
	}
	// gh leaves us in a state where the new remote exists but jj hasn't imported it yet; a fetch
	// is fast and ensures `jj bookmark track main@origin` below has something to bind to.
	_, _ = exec.Command("jj", "git", "fetch").CombinedOutput()
	return nil
}

// addRemoteOrigin wires up `origin` to the user-supplied URL. In a colocated layout we use the
// real `git remote add` so subsequent gh/git commands see the remote; without colocation we fall
// back to `jj git remote add` which is the only way to register a remote in a non-colocated repo.
func addRemoteOrigin(opts InitOptions) error {
	url := strings.TrimSpace(opts.RemoteURL)
	var cmd *exec.Cmd
	if opts.Colocate {
		cmd = exec.Command("git", "remote", "add", "origin", url)
	} else {
		cmd = exec.Command("jj", "git", "remote", "add", "origin", url)
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("add origin %s: %s", url, strings.TrimSpace(string(out)))
	}
	// Best-effort: pull remote bookmarks so the user can immediately see and check out main@origin.
	_, _ = exec.Command("jj", "git", "fetch").CombinedOutput()
	return nil
}
