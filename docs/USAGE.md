# jj-tui Usage Guide

Per-tab keys, integrations, and configuration for [jj-tui](../README.md). For a
feature overview, install instructions, and screenshots, see the
[README](../README.md). For contributing and build/test details, see
[DEVELOPMENT.md](DEVELOPMENT.md).

## Table of Contents

- [Global Shortcuts](#global-shortcuts)
- [Welcome screen (non-jj directories)](#welcome-screen-non-jj-directories)
- [Commit graph](#commit-graph)
- [Help tab (`h` / `?`)](#help-tab-h--)
- [Pull Requests view](#pull-requests-view)
- [Tickets view (Jira / Codecks / GitHub Issues)](#tickets-view-jira--codecks--github-issues)
- [Settings view](#settings-view)
  - [GitHub settings tab](#github-settings-tab)
  - [AI settings tab](#ai-settings-tab)
  - [Advanced settings](#advanced-settings)
- [Settings storage (in-app vs env)](#settings-storage-in-app-vs-env)
- [GitHub Integration](#github-integration)
- [Jira Integration](#jira-integration)
- [GitHub Issues Integration](#github-issues-integration)
- [Codecks Integration](#codecks-integration)
- [Configuration](#configuration)
  - [Config File Locations](#config-file-locations)
  - [Per-Repo Configuration](#per-repo-configuration)
  - [Config file format](#config-file-format)
  - [Optional AI assist](#optional-ai-assist)
  - [Graph view revset](#graph-view-revset)
  - [Ticket Provider Options](#ticket-provider-options)
- [User Stories](#user-stories)

## Global Shortcuts

- `Ctrl+q`: Quit application
- `Ctrl+r`: Refresh current view
- `Ctrl+z`: Undo last jj operation
- `Ctrl+y`: Redo (undo the undo)
- `g`: Switch to commit graph view
- `p`: Switch to pull requests view
- `t`: Switch to tickets view
- `b`: Switch to branches view
- `,`: Open settings
- `h`, `?`: Show help
- `Esc`: Return to graph / Cancel current action

## Welcome screen (non-jj directories)

When you launch `jj-tui` in a directory that isn't a Jujutsu repository, a **Welcome to jj-tui** screen appears with three onboarding paths so you can land in a useful state without leaving the TUI.

### Initialize the repo

Press **`i`** (or click **Initialize Repository**) to run **`jj git init --colocate`** in the current directory. The `--colocate` flag is the default because every downstream flow (`git remote add`, `gh repo create --source=.`, pushing branches, the `Update PR` action) assumes a colocated `.git/` directory exists; running plain `jj git init` consistently produced the *“No git remote named 'origin'”* error on first push.

### Optional: connect an existing remote

If you already have a GitHub / GitLab / Bitbucket / self-hosted repository, paste its URL into the **Remote URL** input on the welcome screen:

- **`Tab`** or **`u`**: Focus the **Remote URL** input (you can also click the input to focus it)
- Paste the URL — `git@github.com:owner/repo.git`, `https://github.com/owner/repo.git`, or any other valid Git remote URL
- **`Enter`** (while focused) or **`i`** (after blurring): Initialize **and** add the URL as `origin` — jj-tui runs `jj git init --colocate`, then `git remote add origin <url>`, then `jj git fetch` so the graph picks up the upstream `main` (and any other remote bookmarks) immediately
- **`Esc`** while focused: Blur the input (the typed URL is preserved); press `Esc` again to dismiss the welcome screen entirely

### Optional: create a brand-new GitHub repo

When the [GitHub CLI (`gh`)](https://cli.github.com/) is installed in your `PATH` and authenticated, the welcome screen also offers a one-shot **Create new GitHub repo** path:

- **`g`** or click **Create new GitHub repo (\<dir\>)**: Initializes the jj repo and runs `gh repo create <dir> --private --source=. --remote=origin` to create the GitHub repository, wire up `origin`, and (silently) `jj git fetch`. The repo name defaults to the current directory name, so initializing inside `~/projects/my-app` creates `my-app` under your GitHub account
- **`Ctrl+v`** or click the **Visibility** button: Toggle between **Private** (default) and **Public** before pressing `g`
- We deliberately omit `--push` because a freshly initialized repo usually has no commits to push yet; push later via the normal **`u`** (Update PR) or branch push flows once you have a commit

If the welcome screen shows **`gh` CLI not found in PATH** in place of the button, install the GitHub CLI and run `gh auth login`. You can also authenticate from inside jj-tui later via **Settings → GitHub → Log in with `gh`**.

### Soft-failure behavior

If `jj git init --colocate` succeeds but a follow-up step fails (e.g. `gh` isn't authenticated, the URL is malformed, the GitHub repo already exists), the welcome screen closes anyway — the directory **is** a valid jj repository at that point — and an error modal surfaces the follow-up failure with **Dismiss** / **Copy** / **Quit** buttons. You can fix the remote manually afterward (e.g. `git remote add origin <url>` in your shell, then **`Ctrl+r`** to refresh).

After any successful path the welcome screen also runs a best-effort `jj bookmark track main@origin`, so if the remote already has a `main` branch the graph picks it up without further action.

## Commit graph

The graph view has two panes: the commit graph (left) and changed files (right). Click on either pane to focus it, or use keyboard navigation.

**Navigation:**
- `↑/↓`, `j/k`: Navigate commits (graph pane) or scroll (files pane)
- `Tab`: Switch focus between graph and files panes
- **Click** on a pane to focus it
- **Mouse scroll** works on the focused pane

**Commit actions (graph pane focused unless noted):**
- `e`, `Enter`: Edit selected commit (`jj edit`)
- `n`: Create new commit (works from immutable parents like `main`)
- `d`: Edit description; on a **divergent** row, opens the divergent resolver instead
- `s`: Squash into parent (hidden when the parent would be immutable)
- `r`: Rebase mode—pick destination with `Enter`/`e`, or **Esc** to cancel
- **Mouse**: Press on a commit row, drag, release on another commit to rebase (same as `r` + pick destination); **Esc** cancels an in-progress drag
- `M` (shift+m): Merge-from mode—the selected commit is the target; pick a source commit/bookmark to merge in with `Enter`/`e` or click (creates a merge commit via `jj new <target> <source>`); **Esc** to cancel
- `a`: Abandon commit
- `m`: Create or move bookmark
- `x`: Delete bookmark
- `c`: Create PR, or **resolve diverged bookmark** when the row has a conflicted/diverged bookmark (`c` matches Branches-tab behavior)
- `C` (shift+c): **Resolve diverged bookmark** when shown on the row
- `u`: Update PR (push bookmark branch)
- `f`: **Forgot New Commit?** (when the inline control appears)—restack after amending a pushed bookmark so you can push without `--force`
- `z`: **Split (evolog)** when the inline **split (z)** appears—see [Split](../README.md#split)

**Files pane (focus with Tab or click the files side):**
- `o`: Open full **jj** diff for the selected file (modal)
- `O`: Open the selected file in the **external editor** (configure under **Settings → Advanced** → Open in external editor)
- `[` / `]`: Move file to new parent / child commit
- `v`: Revert the file in this commit

## Help tab (`h` / `?`)

- **`Ctrl+j`** / **`Ctrl+k`** (or **`Tab`**): Switch between **Shortcuts** and **Command history**
- **Command history** lists **`jj`** commands the TUI ran (with timing); copy-friendly for debugging or docs
- Mouse **wheel** scrolls the active sub-tab

## Pull Requests view

- `↑/↓`, `j/k`: Navigate pull requests
- `Enter`, `e`: Open PR in browser
- `Ctrl+r`: Refresh PR list

## Tickets view (Jira / Codecks / GitHub Issues)

- `↑/↓`, `j/k`: Navigate tickets
- `Enter`: Create branch from selected ticket (creates a bookmark on your **current commit** with the ticket name)
- `o`: Open ticket in browser
- `c`: Change ticket status (transitions to In Progress, Done, etc.)
- `Ctrl+r`: Refresh ticket list

## Settings view

Sub-tabs (use **`Ctrl+j`** / **`Ctrl+k`**, click the tab bar, or **`Tab`** through fields):

1. **GitHub** — token / device login, PR list filters, refresh interval, **repository remote** (origin URL / `gh repo create`; see [GitHub settings tab](#github-settings-tab))  
2. **Jira** — URL, user, token, projects, JQL, filters  
3. **Codecks** — subdomain, token, project filter  
4. **Tickets** — active provider (None / Jira / Codecks / GitHub Issues), auto “In Progress” on branch-from-ticket, GitHub Issues status excludes  
5. **Branches** — how many branches to load for the Branches tab (`0` = all)  
6. **Theme** — primary, secondary, muted accent colors (click swatches or **Save** to persist)  
7. **AI** — LLM provider, credentials, and optional **evolog split** defaults (see [AI settings tab](#ai-settings-tab))  
8. **Advanced** — external editor, default graph revset, bookmark sanitize, destructive maintenance (see [Advanced settings](#advanced-settings))  

**Keys:**

- **`Ctrl+j`** / **`Ctrl+k`**: Previous / next settings sub-tab  
- **`Tab`** / **`Shift+Tab`**: Next / previous field (and tab bar navigation where applicable)  
- **`Ctrl+s`**: Save globally (`~/.config/jj-tui/config.json`)  
- **`Ctrl+l`**: Save **local** (`.jj-tui.json` in the repo)  
- **`Esc`**: Cancel and return to the graph (or dismiss in-tab overlays first)  
- **Click** fields, tabs, toggles, and theme swatches

### GitHub settings tab

The **GitHub** sub-tab hosts both the API/auth configuration (token / device login / PR filters / refresh interval — covered above) **and** a dedicated **Repository remote** section that lets you manage the git `origin` remote for the current repo without leaving the TUI. Use this when you initialized a repo earlier without an origin (or want to re-point an existing one) and would otherwise hit `Failed to push branch: No git remote named 'origin'` on first push.

#### Common workflows at a glance

| You want to… | Inputs | Press / Click |
|---|---|---|
| **Add an existing remote** (e.g. `git@github.com:owner/repo.git`) | Paste URL into **Remote URL** | **`Enter`** while focused, or **Apply** |
| **Update an existing origin** (typo / wrong account / SSH→HTTPS) | Edit **Remote URL** (pre-filled) | **`Enter`** while focused, or **Apply** |
| **Remove origin** | — | **`Ctrl+x`**, or **Remove origin** |
| **Push current bookmark** to origin | — | **`p`**, or **Push current bookmark** |
| **Push all local bookmarks** to origin (after Apply / set-url) | — | **`P`**, or **Push all bookmarks** |
| **Create a new GitHub repo + push everything in one shot** | optionally toggle visibility | **`Ctrl+v`** to flip Private/Public, then **`g`** or **Create new GitHub repo** |

#### Field reference

- **Current origin**: shows the live URL from `jj git remote list` (or **`(none configured)`** if the remote isn't set yet); refreshed whenever you open the Settings view or finish a remote operation.
- **Remote URL** input: pre-filled with the existing origin URL when present so you can edit it; **`Tab`** / **`down`** focuses the field, then paste the new URL.
  - **`Enter`** while focused (or click **Apply**): runs `jj git remote add origin <url>` (when no origin yet) or `jj git remote set-url origin <url>` (when changing it), then a best-effort `jj git fetch` so any remote bookmarks (`main@origin`, etc.) appear in the graph immediately. Empty URL + Apply when an origin already exists routes to **Remove** instead.
  - **`Ctrl+x`** or click **Remove origin**: deletes the existing `origin` (no-op if none configured).
- **Push current bookmark (`p`)** / **Push all bookmarks (`P`)**: run `jj git push --bookmark <name>` for the bookmark on `@` (current) or once per local bookmark (all) against the configured origin. Naming each bookmark explicitly creates new remote bookmarks without the deprecated `--allow-new` flag, and the "all" path enumerates bookmarks and passes each explicitly so it stays compatible across jj versions (some currently-supported builds reject `--all-bookmarks`). Both buttons are disabled with a hint to set up origin first when none is configured. Use these after **Apply** to push existing work to a freshly-pointed remote, or anytime you want a one-click push that doesn't require switching to the Branches tab.
- **Create new GitHub repo (`g`)**: when the [GitHub CLI (`gh`)](https://cli.github.com/) is installed and authenticated, runs `gh repo create <dir> --private/--public --source=. --remote=origin`, then **automatically pushes all local bookmarks** to the new origin in the same action — so the most common workflow ("create the repo and push my work") is a single click. The repo name defaults to the current directory name. Requires no existing origin (Apply / Remove first if you want to replace).
  - **`Ctrl+v`** or click **Visibility**: toggles between **Private** (default) and **Public** before pressing `g`.
  - **No bookmarks yet?** The auto-push step is skipped silently and the status reads `Created GitHub repo (no bookmarks to push yet)`. Make a commit / bookmark and use **Push all bookmarks** when you're ready.
  - **Push fails after a successful create?** The new origin is preserved (the GitHub repo really exists), the failure surfaces in the error modal, and you can retry just the push via the **Push all bookmarks** button without re-creating anything.
  - When `gh` isn't on `PATH`, the section shows a hint to install / run `gh auth login` instead of the button.

These actions take effect immediately and aren't part of **Save** — there's nothing to persist because the remote is a property of the repo, not jj-tui config.

If a push attempt fails with `No git remote named 'origin'`, the status / error message includes a one-line pointer to **Settings → GitHub → Repository remote** so you can jump straight to the fix.

### AI settings tab

Configure optional **AI assist** from **Settings → AI** (or merge the same keys in JSON—see [Optional AI assist](#optional-ai-assist)).

- **Enable AI**: Turns on the purple **✧ ^g** chip and **`Ctrl+G`** generation in the commit description editor, **Create PR**, **Create ticket**, and **new bookmark** modals. You always review generated text before saving.  
- **Provider**: **OpenAI-compatible** (Chat Completions), **Google Gemini**, or **Ollama** (local). Picking **Ollama** applies sensible defaults for base URL and model when those fields are empty.  
- **API base URL**, **Model**, **API key**: Same semantics as the `ai_base_url`, `ai_model`, and `ai_api_key` config fields; **`JJ_TUI_AI_API_KEY`** overrides the saved key when set. Gemini ignores custom base URL (Google endpoint).  
- **AI evolog split** (graph **`z`** when evolog split is available): Defaults for post-split describe, honoring AI file/hunk suggestions, stepwise multi-split, and the cap on multi-split “FAQ bases”—stored as `aievolog_*` options in config (see `internal/config` for names).

Use **Save** (**`Ctrl+s`** global, **`Ctrl+l`** local) after changing AI settings. Sending a diff to a cloud API exposes code to that provider; local **Ollama** avoids that.

### Advanced settings

- **Open in external editor**: Presets (Cursor, VS Code, Zed, Neovim/`nvr`, Emacs, Sublime, JetBrains) or **Custom** (`sh -c` with `{path}` → absolute file path). Used from the graph **files** pane with **`O`**.  
- **Default graph revset**: Optional `jj` revset for the commit list; empty = built-in default (see [Graph view revset](#graph-view-revset)).  
- **Sanitize bookmark names**: Auto-fix invalid bookmark characters when creating/moving names.  
- **Delete all bookmarks** / **Abandon old commits**: Destructive maintenance (with confirmation).

## Settings storage (in-app vs env)

You can configure your API credentials in two ways:

### Option 1: In-App Settings (Recommended)

1. Press `,` or click the **Settings** tab
2. Enter your credentials in the form fields
3. Press `Ctrl+S` or click **Save** to apply

Settings are saved to `~/.config/jj-tui/config.json` and persist across sessions.

### Option 2: Environment Variables

You can also set credentials via environment variables; see the GitHub, Jira, and Codecks sections below for the relevant variable names. Environment variables are applied when the app starts and can be overridden by in-app settings.

## GitHub Integration

There are two ways to authenticate with GitHub:

### Option 1: Browser Login (Recommended)

1. Press `,` to open Settings
2. Click **Login with GitHub**
3. Your browser will open to GitHub's authorization page
4. Enter the code shown in the TUI
5. Authorize the application

Your token is automatically saved and persists across sessions.

### Option 2: Personal Access Token

Set your GitHub token as an environment variable:

```bash
export GITHUB_TOKEN=your_github_personal_access_token
```

Or enter it manually in Settings.

The application will automatically detect GitHub remotes and enable PR functionality.

### PR Workflow

1. Select a commit with a bookmark in the graph view
2. Press `c` to create a PR, or `u` to update an existing PR
3. Fill in the PR title and description
4. Press `Ctrl+S` to submit

**Note:** You can create/update PRs from descendant commits - the bookmark will automatically be moved to the selected commit.

## Jira Integration

To use Jira features, set your Jira credentials:

```bash
export JIRA_URL=https://your-domain.atlassian.net
export JIRA_USER=your-email@example.com
export JIRA_TOKEN=your_api_token
```

Get your API token from: https://id.atlassian.com/manage-profile/security/api-tokens

### Jira Workflow

1. Press `t` to open the Tickets view
2. Navigate through your assigned tickets with `j/k` or arrow keys
3. Press `Enter` to create a branch from the selected ticket
   - Creates a **bookmark on your current commit** with a sanitized name (e.g., `PROJ-123-ticket-summary`)
   - Keeps your existing work and commit description intact
   - Optional: if "In Progress on branch" is enabled in settings, transitions the ticket to In Progress
   - When you create a PR, the title is pre-populated with "PROJ-123 - Ticket Summary"

## GitHub Issues Integration

If you're using GitHub Issues for task tracking, they work automatically with your GitHub authentication:

1. Login to GitHub (via browser or token)
2. Select "GitHub Issues" as your ticket provider in Settings
3. Issues assigned to you will appear in the Tickets tab

### GitHub Issues Workflow

1. Press `t` to open the Tickets view
2. Navigate through your assigned issues with `j/k` or arrow keys
3. Press `Enter` to create a branch from the selected issue
   - Creates a **bookmark on your current commit** with a sanitized name (e.g., `123-issue-summary`)
   - When you create a PR, the title is pre-populated with "#123 - Issue Summary"
4. Press `c` to change issue status (Open ↔ Closed)
5. Press `o` to open the issue in your browser

## Codecks Integration

[Codecks](https://www.codecks.io/) is a project management tool designed for game developers. To use Codecks features, set your credentials:

```bash
export CODECKS_SUBDOMAIN=your-account-name
export CODECKS_TOKEN=your_auth_token
export CODECKS_PROJECT=Optional-Project-Name  # Optional: filter cards by project
```

### Getting Your Codecks Token

1. Log in to Codecks in your browser
2. Open browser Developer Tools (F12)
3. Go to Application → Cookies → `https://your-account.codecks.io`
4. Copy the value of the `at` cookie - this is your auth token

### Codecks Workflow

1. Press `t` to open the Tickets view
2. Navigate through your assigned cards with `j/k` or arrow keys
3. Press `o` to open the card in your browser
4. Press `Enter` to create a branch from the selected card
   - Creates a **bookmark on your current commit** with the card title (or short ID) as the name
   - Automatically prepopulates commit descriptions with the card's short ID (e.g., `$12u`) when editing
   - When you create a PR, the title is pre-populated with "$12u - Card Title"

### Codecks Features

- **Short IDs**: Cards display their Codecks short ID (e.g., `$12u`) for easy reference
- **Project Filtering**: Optionally filter cards to a specific project
- **Archive Filtering**: Archived and deleted cards are automatically hidden
- **Direct Links**: Open cards directly in Codecks from the TUI

## Configuration

The application automatically detects:
- Current jujutsu repository
- GitHub remote configuration
- User preferences from jj config
- Credentials from environment variables or in-app settings

### Config File Locations

jj-tui supports multiple configuration files with the following priority (highest to lowest):

1. **`JJ_TUI_CONFIG` environment variable** - Custom config file path
2. **`.jj-tui.json`** - Per-repo config in current directory
3. **`~/.config/jj-tui/config.json`** - Global config

Local config values **merge with and override** global config values. This allows you to:
- Keep sensitive tokens (GitHub, Jira, Codecks) in the global config
- Override project-specific settings (like `codecks_project`) per-repo

### Per-Repo Configuration

Create a `.jj-tui.json` in your repository root to customize settings for that repo:

```json
{
  "ticket_provider": "codecks",
  "codecks_project": "My Project Name"
}
```

You can also share configs across similar repos using the environment variable:

```bash
export JJ_TUI_CONFIG=/path/to/shared-config.json
jj-tui
```

### Config file format

```json
{
  "github_token": "ghp_...",
  "ticket_provider": "github_issues",
  "ticket_auto_in_progress": true,
  "jira_url": "https://company.atlassian.net",
  "jira_user": "user@example.com",
  "jira_token": "...",
  "jira_excluded_statuses": "Done,Closed",
  "codecks_subdomain": "myteam",
  "codecks_token": "...",
  "codecks_project": "Project Name",
  "codecks_excluded_statuses": "done,resolved",
  "github_issues_excluded_statuses": "closed",
  "branch_limit": 50,
  "sanitize_bookmark_names": true,
  "graph_revset": "",
  "external_file_editor": "cursor",
  "external_file_editor_custom": "cursor -g {path}",
  "theme_primary": "#7E00AF",
  "theme_secondary": "#FF79C6",
  "theme_muted": "#6272A4",
  "ai_enabled": false,
  "ai_provider": "openai_compatible",
  "ai_api_key": "",
  "ai_base_url": "https://api.openai.com/v1",
  "ai_model": "gpt-4o-mini",
  "ai_timeout_seconds": 60
}
```

Omit keys you do not need. See `internal/config/config.go` for the full schema and merge rules.

### Optional AI assist

Use **[Settings → AI](#ai-settings-tab)** to toggle generation and set provider/credentials in the TUI; the fields below correspond to the same JSON keys.

When **`ai_enabled`** is true and LLM credentials are available, jj-tui can call a provider to draft text:

- The purple **✧ ^g** chip beside the title in the commit description editor, **Create PR**, **Create ticket**, or **bookmark** modal (new name only), or **Ctrl+G** in those modals, runs generation; you always review before saving or submitting.
- **`ai_provider`**:
  - **`openai_compatible`** (default): Chat Completions at **`ai_base_url`** (default `https://api.openai.com/v1`). Requires an API key unless **`ai_base_url`** is a typical local Ollama URL (`http://127.0.0.1:11434/v1` or `http://localhost:11434/v1`), in which case a placeholder Bearer token is sent automatically.
  - **`gemini`**: Google Generative Language API (requires a real API key).
  - **`ollama`**: Local Ollama OpenAI-compatible API. Defaults: **`ai_base_url`** empty → `http://127.0.0.1:11434/v1`, **`ai_model`** empty → `qwen2.5:1.5b`. **API key is optional** (jj-tui sends a harmless placeholder if unset). Pull the model first (`ollama pull qwen2.5:1.5b` or change **`ai_model`** to a tag you have).
- **`ai_api_key`** / **`JJ_TUI_AI_API_KEY`**: For cloud OpenAI-compatible hosts and Gemini, set a real key (env wins). For **`ollama`** or the local Ollama **`ai_base_url`** above, you may leave the key empty.
- **`ai_base_url`**: API root without a trailing slash. Ignored for Gemini (Google endpoint).
- **`ai_model`**: Model id. Defaults: `gpt-4o-mini` (OpenAI-compatible), `gemini-2.5-flash` (Gemini), or `qwen2.5:1.5b` (**`ollama`**) when empty.
- **`ai_timeout_seconds`**: HTTP timeout (optional; default 60). Local models may need more on **first request after idle** (model load); try **120** or higher if requests time out.

Configure from **Settings → AI** (recommended) or JSON below. Diffs are included in prompts; treat this as sending code to the provider unless you use a local endpoint.

**Cursor / in-IDE models:** Cursor’s chat models are not exposed as a stable HTTP API for third-party apps to call from the TUI. Practical options are **your own API keys** (OpenAI, Google AI Studio for Gemini, Anthropic via an OpenAI-compatible gateway, local Ollama, etc.).

### Graph view revset

The commit graph shows commits selected by a **jj revset**. By default (empty `graph_revset`) jj-tui uses a built-in revset (see below) that limits **mutable** rows to the **working copy's DAG neighborhood** — `ancestors(@)`, `descendants(@)`, and the **sibling subtree** `(parents(@)+)::` (children of `@`'s parents, then their descendants) so split-off branches stay visible — and unions in **`bookmarks()`** and **`main@origin`** so named tips (including the immutable trunk anchor) are always shown. **`mutable()` alone** (without that intersection) selects **every** mutable revision in the repository — including unrelated colocated history — and is a bad default for large team repos.

Built-in default (same as `jj.DefaultGraphRevset` in code):

```text
(mutable() & (ancestors(@) | descendants(@) | (parents(@)+)::)) | bookmarks() | main@origin
```

Graph load still uses capped per-commit probes and parallel bookmark fetch so a deep `ancestors(@)` is less punishing than before.

To use a custom revset, set `graph_revset` in your config. Examples:

- **All mutable everywhere** (can be hundreds of irrelevant rows in big repos):  
  `"graph_revset": "mutable() | bookmarks() | main@origin"`
- **First-parent ancestry only** (fewer merge side-branches on `jj git` repos; see jj `first_ancestors` docs):  
  `"graph_revset": "(mutable() & (first_ancestors(@, 150) | descendants(@))) | bookmarks() | main@origin"`
- **Main + your branch only** (minimal noise):  
  `"graph_revset": "trunk() | (ancestors(@) - ancestors(trunk()))"`
- **Only your commits** (author = you):  
  `"graph_revset": "mine() | trunk()"`
- **Ancestors of @ only** (may hide split children and sibling branches):  
  `"graph_revset": "(mutable() & ancestors(@)) | bookmarks() | main@origin"`
- **Without sibling subtree** (drop `(parents(@)+)::` if split-off branches feel noisy):  
  `"graph_revset": "(mutable() & (ancestors(@) | descendants(@))) | bookmarks() | main@origin"`
- **Verbose bookmark-aware default** (seeds mutable bookmark tips, their mutable ancestors, and per-anchor immutable bases so merge-heavy lines don't elide rows with `~`):  
  `"graph_revset": "(mutable() & (ancestors(@) | descendants(@))) | (bookmarks() & mutable()) | parents(bookmarks() & mutable()) | bookmarks() | main@origin | (::(bookmarks() & mutable()) & mutable()) | heads(::(bookmarks() & mutable()) & immutable()) | heads(::(@) & immutable()) | (heads(::(bookmarks() & mutable()) & immutable())::(bookmarks() & mutable())) | (heads(::(@) & immutable())::@)"`
- **Also show parents of immutable bookmark tips** (e.g. parent of `main` when the bookmark sits on an immutable commit): append `| parents(bookmarks())` to the built-in default in Settings / JSON.

Leave `graph_revset` empty to use the built-in default. See [jj revset docs](https://jj-vcs.github.io/jj/latest/revsets) for more.

### Ticket Provider Options

The `ticket_provider` field can be one of:
- `""` (empty) — pick explicitly in **Settings → Tickets** or auto-detect from credentials
- `"jira"` — Jira
- `"codecks"` — Codecks
- `"github_issues"` — GitHub Issues (uses the same auth as the GitHub tab)

## User Stories

The application supports these key user workflows:

### 1. Commit Navigation
- View commit history in a visual graph with split-pane layout (graph + changed files)
- Navigate with keyboard shortcuts or mouse; Tab switches focus between graph and files panes
- See commit details, authors, and timestamps
- View changed files for each commit in the files pane; move files to parent/child commits (`[` / `]`) or revert (`v`)
- Mouse scroll works on the focused pane

### 2. Commit management
- Edit commits (`jj edit` with **`e`** / **Enter**)
- Squash, describe, abandon; rebase with **`r`** or **mouse drag** between commit rows
- New commits from immutable parents (**`n`**)
- Move files between commits (**`[`** / **`]`** in files pane) or revert (**`v`**)
- Full-file diff modal (**`o`**) and external editor (**`O`**) from the files pane
- Immutable commits are protected from destructive actions

### 3. Bookmark Management
- Create bookmarks on any mutable commit (`m` in graph view)
- Create a bookmark from a ticket on your **current commit** (Tickets tab → Enter)
- Move existing bookmarks to different commits
- Delete bookmarks when no longer needed

### 4. Pull Request Workflow
- Create GitHub PRs from commits with bookmarks
- View existing PRs with status and descriptions
- Update PRs by pushing new commits
- Push from descendant commits (bookmark auto-moves)
- Login to GitHub via browser (Device Flow) - no token copying needed

### 5. Ticket Integration (Jira, Codecks & GitHub Issues)
- View assigned tickets from Jira, Codecks cards, or GitHub Issues
- Create a **bookmark on your current commit** from a ticket (Enter) — keeps your work and description intact
- PR titles and commit description placeholders auto-populated from ticket info
- Change ticket status directly from the TUI (In Progress, Done, etc.)
- Open tickets in the browser
- Consistent layout with description placeholders

### 6. Repository state
- Refresh the repo with **`Ctrl+r`** and after jj operations (checkout, rebase, etc.)
- Conflict detection and display
- Divergent commit detection with visual indicator (≠)
- Bookmark visualization
- Working copy indicator

### 7. Conflict Resolution
- Resolve divergent commits (choose which version to keep)
- Resolve conflicted bookmarks (keep local or reset to remote)
- Visual indicators in Graph and Branches views

### 8. Repository Setup & Cleanup
- Auto-detect non-jj repositories and show the [Welcome screen](#welcome-screen-non-jj-directories) with three onboarding paths:
  - Plain init (`jj git init --colocate`)
  - Init + paste a remote URL (adds `origin`, runs `jj git fetch`)
  - Init + create a brand-new GitHub repo via `gh repo create` (Private by default; toggle with **`Ctrl+v`**)
- Best-effort `jj bookmark track main@origin` after init so the graph picks up the upstream `main` if it exists
- Abandon old commits after merging PRs
- Delete all bookmarks for fresh start
- Track/fetch remote branches
