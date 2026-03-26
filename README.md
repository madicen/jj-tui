# Jujutsu TUI

![Demo](screenshots/demo.gif)

A modern Terminal User Interface (TUI) for managing [Jujutsu](https://github.com/martinvonz/jj) repositories. Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss) for an intuitive and beautiful command-line experience.

### Graph
![Graph](screenshots/graph.png)

### Pull Requests
![Pull Requests](screenshots/prs.png)

### Tickets
![Tickets](screenshots/tickets.png)

### Branches
![Branches](screenshots/branches.png)

### Settings
![Settings](screenshots/settings.png)

### Help
![Help](screenshots/help.png)

### Command History
![Command History](screenshots/command_history.png)

### Forgot new commit? and Update PR

If you already pushed a feature bookmark but kept editing on the same jj revision (so your tree diverges from `bookmark@origin`), **Forgot new commit?** (`f`) restacks that work on the remote tip so you can push without `--force`. **Update PR** (`u`) pushes the bookmark and updates the open PR. The clip below walks through opening a PR, editing without a new commit, then `f` and `u`.

![Forgot new commit? and Update PR](screenshots/after-origin.gif)

### Split

**Split** (`z`) opens an **jj evolog** picker for the selected change: pick an older evolution revision as the parent, preview which files differ from the tip, then run the FAQ-style flow (`jj new`, `jj restore`, optional bookmark move, `jj abandon` old tip). The recording below uses a small fixture repo (`fixtures/setup-evolog-split-vhs-repo.sh`) so the modal shows a clear two-file rollout delta.

![Split](screenshots/evolog-split.gif)

### Resolving divergent commits

When the same **change ID** exists on more than one revision, the graph shows **divergent**. Press **`d`** on that row to open the resolver: each option lists metadata and a short **files vs parent** summary so you can pick which revision to keep (the others are abandoned). The clip uses `fixtures/setup-divergent-vhs-repo.sh`.

![Resolve divergent](screenshots/divergent.gif)

## Features

- **Visual Commit Graph**: Navigate and visualize your commit history with tree structure
- **Split-Pane View**: Graph and changed files in separate scrollable panes with click-to-focus
- **Changed Files View**: See files modified in the selected commit; move files to parent/child commits or revert changes
- **Keyboard & Mouse Support**: Full keyboard navigation with zone-based mouse support for clickable UI elements
- **GitHub Integration**: Create and manage GitHub Pull Requests directly from the TUI
- **GitHub Device Flow**: Login to GitHub via browser - no token copying required
- **Ticket Integration**: View assigned tickets from Jira, Codecks, or GitHub Issues; create a bookmark on your current commit with the ticket name
- **Ticket Status Transitions**: Change ticket statuses directly from the TUI (In Progress, Done, etc.)
- **Commit Management**: Edit, squash, describe, abandon, rebase, and manage commits with simple key presses
- **New Commits from Immutable Parents**: Create new commits based on `main` or other immutable commits
- **Bookmark Management**: Create, move, and delete bookmarks on commits
- **Immutable Commit Detection**: Automatically detects and protects immutable commits (pushed to remote)
- **Divergent Commit Resolution**: Detect and resolve divergent commits (same change ID in multiple versions)
- **Conflicted Bookmark Resolution**: Resolve bookmark conflicts when local and remote have diverged
- **Repository Cleanup Tools**: Abandon old commits, delete all bookmarks, track remote branches
- **Auto-Initialize**: Prompt to run `jj git init` when opening a non-jj repository
- **Real-time Updates**: Auto-refresh repository state and see changes immediately
- **Modern UI**: Beautiful styling with colors, borders, and responsive layouts

## Prerequisites

- [Jujutsu (jj)](https://github.com/martinvonz/jj) installed and available in your PATH
- A jujutsu repository to work with

## Installation

### Homebrew (macOS/Linux)

```bash
brew install madicen/tap/jj-tui
```

### Go Install

If you have Go 1.24+ installed:

```bash
go install github.com/madicen/jj-tui@latest
```

### Download Binary (Linux/macOS)

Download and install the latest release with one command:

**Linux (amd64):**
```bash
curl -sL $(curl -s https://api.github.com/repos/madicen/jj-tui/releases/latest | grep browser_download_url | grep linux_amd64 | cut -d '"' -f 4) | tar xz && sudo mv jj-tui /usr/local/bin/
```

**Linux (arm64):**
```bash
curl -sL $(curl -s https://api.github.com/repos/madicen/jj-tui/releases/latest | grep browser_download_url | grep linux_arm64 | cut -d '"' -f 4) | tar xz && sudo mv jj-tui /usr/local/bin/
```

**macOS (Apple Silicon):**
```bash
curl -sL $(curl -s https://api.github.com/repos/madicen/jj-tui/releases/latest | grep browser_download_url | grep darwin_arm64 | cut -d '"' -f 4) | tar xz && sudo mv jj-tui /usr/local/bin/
```

**macOS (Intel):**
```bash
curl -sL $(curl -s https://api.github.com/repos/madicen/jj-tui/releases/latest | grep browser_download_url | grep darwin_amd64 | cut -d '"' -f 4) | tar xz && sudo mv jj-tui /usr/local/bin/
```

Or download manually from the [GitHub Releases](https://github.com/madicen/jj-tui/releases) page.

Available for:
- **macOS** (Intel & Apple Silicon)
- **Linux** (amd64 & arm64)
- **Windows** (amd64 & arm64)

### From Source

Requires Go 1.24+:

```bash
git clone https://github.com/madicen/jj-tui.git
cd jj-tui
go build -o jj-tui .
```

### Running

```bash
# From within a jujutsu repository
jj-tui

# Or specify a repository path
jj-tui /path/to/your/jj/repo

# Demo mode: use a demo repo with mock tickets/PRs (e.g. for screenshots or trying the UI)
cd path/to/jj/repo
jj-tui --demo
```

## Usage

### Global Shortcuts

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

### Commit Graph View

The graph view has two panes: the commit graph (left) and changed files (right). Click on either pane to focus it, or use keyboard navigation.

**Navigation:**
- `↑/↓`, `j/k`: Navigate commits (graph pane) or scroll (files pane)
- `Tab`: Switch focus between graph and files panes
- **Click** on a pane to focus it
- **Mouse scroll** works on the focused pane

**Commit Actions:**
- `e`, `Enter`: Edit selected commit (checkout with `jj edit`)
- `n`: Create new commit (works even on immutable commits like `main`)
- `d`: Edit commit description
- `s`: Squash selected commit into parent
- `r`: Rebase selected commit (with descendants)
- `a`: Abandon commit (or resolve divergent)
- `m`: Create or move bookmark on selected commit
- `x`: Delete bookmark from selected commit
- `c`: Create PR from selected commit (if bookmark exists)
- `u`: Push/update PR (pushes to existing PR branch)
- `f` (graph pane focused): **Forgot New Commit?** — fetch, then create a new commit on top of `bookmark@origin` with the same tree as the bookmark tip, move the bookmark there, **rebase any stacked commits that were on top of the old tip** onto the new tip (excluding the working copy), abandon the old tip, and **drop any leftover divergent duplicates** of the same change ID (jj keeps the revision on your `@` ancestry). Use when you amended after a push so you can `jj git push` without `--force`.
- `z` (graph pane focused, experimental): **Split (evolog)** — same as the inline **split (z)** when it appears on the selected row: open the evolog modal only when the change has evolution history with a real tree diff vs an older revision (and no blocking descendants). See the clip under [Evolog split (experimental)](#evolog-split-experimental).

**File pane (focus files with Tab first):**
- `[`: Move selected file to new parent commit (split out)
- `]`: Move selected file to new child commit (split out)
- `v`: Revert changes to selected file in this commit

### Pull Requests View

- `↑/↓`, `j/k`: Navigate pull requests
- `Enter`, `e`: Open PR in browser
- `Ctrl+r`: Refresh PR list

### Tickets View (Jira / Codecks / GitHub Issues)

- `↑/↓`, `j/k`: Navigate tickets
- `Enter`: Create branch from selected ticket (creates a bookmark on your **current commit** with the ticket name)
- `o`: Open ticket in browser
- `c`: Change ticket status (transitions to In Progress, Done, etc.)
- `Ctrl+r`: Refresh ticket list

### Settings View

- `Ctrl+J`: Previous sub-tab (GitHub, Jira, Codecks, Advanced)
- `Ctrl+K`: Next sub-tab
- `Tab`, `↓`: Move to next field
- `Shift+Tab`, `↑`: Move to previous field
- `Enter`: Move to next field (or save if on last field)
- `Ctrl+S`: Save settings globally
- `Ctrl+L`: Save settings to local `.jj-tui.json`
- `Esc`: Cancel and return to graph
- **Click** on any field or tab to focus/select it

### Advanced Settings

The Advanced tab in Settings provides repository cleanup tools:

- **Delete All Bookmarks**: Remove all bookmarks in the repository
- **Abandon Old Commits**: Abandon all mutable commits (useful for cleaning up after merging PRs)
- **Track origin/main**: Fetch and track the remote main branch

## Settings

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

### Config File Format

```json
{
  "github_token": "ghp_...",
  "ticket_provider": "github_issues",
  "jira_url": "https://company.atlassian.net",
  "jira_user": "user@example.com",
  "jira_token": "...",
  "jira_excluded_statuses": "Done,Closed",
  "codecks_subdomain": "myteam",
  "codecks_token": "...",
  "codecks_project": "Project Name",
  "codecks_excluded_statuses": "done,resolved",
  "github_issues_excluded_statuses": "closed",
  "graph_revset": ""
}
```

### Graph view revset

The commit graph shows commits selected by a **jj revset**. By default jj-tui uses a revset so you see **your local work** (mutable commits that are ancestors or descendants of `@`), **bookmarks**, and **main**. Including descendants of `@` ensures that “move to parent” / “move to child” split commits stay visible.

To use a custom revset, set `graph_revset` in your config. Examples:

- **Main + your branch only** (minimal noise):  
  `"graph_revset": "trunk() | (ancestors(@) - ancestors(trunk()))"`
- **Only your commits** (author = you):  
  `"graph_revset": "mine() | trunk()"`
- **Ancestors of @ only** (stricter than default, may hide new split commits):  
  `"graph_revset": "(mutable() & ancestors(@)) | bookmarks() | main@origin"`
- **All mutable + bookmarks**:  
  `"graph_revset": "mutable() | bookmarks() | main@origin"`

Leave `graph_revset` empty to use the built-in default. See [jj revset docs](https://jj-vcs.github.io/jj/latest/revsets) for more.

### Ticket Provider Options

The `ticket_provider` field can be one of:
- `"jira"` - Use Jira for tickets
- `"codecks"` - Use Codecks for tickets  
- `"github_issues"` - Use GitHub Issues for tickets
- `""` (empty) - Auto-detect based on configured credentials

## Development

### Project Structure

```
jj-tui/
├── main.go                    # Application entry point
├── go.mod
├── internal/
│   ├── config/                # Configuration (config.json, env)
│   │   └── config.go
│   ├── types.go               # Shared types (Commit, Repository, etc.)
│   ├── integrations/
│   │   ├── jj/                # Jujutsu CLI integration
│   │   │   └── service.go
│   │   ├── github/            # GitHub API (PRs, Issues)
│   │   ├── jira/              # Jira API
│   │   └── codecks/           # Codecks API
│   ├── tickets/               # Ticket service interface
│   │   └── interface.go
│   ├── mock/                  # Mock services for demo mode
│   ├── testutil/              # Test mocks and helpers
│   ├── version/               # Update checks
│   └── tui/
│       ├── tui.go             # Public re-exports
│       ├── state/             # App state, view mode, navigation
│       ├── data/              # Load repo, init services, messages
│       ├── styles/            # Lip Gloss styles
│       ├── mouse/             # Zone IDs for clickable elements
│       ├── util/              # Clipboard, helpers
│       ├── model/             # Main TUI model (Update, view, keys, mouse)
│       └── tabs/              # Tab-specific models and views
│           ├── graph/         # Commit graph, keys, file move/revert, actions
│           ├── prs/           # Pull requests list
│           ├── prform/        # Create/update PR modal
│           ├── tickets/       # Tickets list (Jira/Codecks/Issues)
│           ├── branches/      # Branches/bookmarks list
│           ├── bookmark/      # Create bookmark modal
│           ├── descedit/      # Edit commit description modal
│           ├── settings/      # Settings (GitHub, Jira, Codecks, Advanced)
│           ├── help/          # Help + command history
│           ├── conflict/      # Bookmark conflict resolution
│           ├── divergent/     # Divergent commit resolution
│           ├── warning/       # Warning modal (e.g. empty descriptions)
│           ├── error/         # Error overlay
│           ├── initrepo/      # Non-jj repo → jj git init
│           └── githublogin/   # GitHub device flow
├── fixtures/                  # Demo repository for screenshots
│   ├── setup-demo-repo.sh
│   ├── setup-after-origin-vhs-repo.sh
│   ├── setup-evolog-split-vhs-repo.sh
│   ├── setup-divergent-vhs-repo.sh
│   ├── after-origin-vhs-append-and-tui.sh
│   ├── demo-repo/             # Created by setup-demo-repo.sh
│   ├── after-origin-vhs-repo/ # Created by setup-after-origin-vhs-repo.sh (after-origin GIF)
│   ├── evolog-split-vhs-repo/ # Created by setup-evolog-split-vhs-repo.sh (evolog-split GIF)
│   └── divergent-vhs-repo/    # Created by setup-divergent-vhs-repo.sh (divergent GIF)
├── vhs/                       # VHS tapes for screenshot generation
│   ├── all.tape               # Main demo GIF
│   ├── after-origin.tape      # Forgot New Commit? (f) workflow GIF
│   ├── evolog-split.tape      # Evolog split (z) workflow GIF
│   ├── divergent.tape         # Resolve divergent change (d) GIF
│   ├── graph.tape
│   └── ...
├── screenshots/               # Generated (demo.gif, after-origin.gif, evolog-split.gif, divergent.gif, *.png)
└── README.md
```

### Building

```bash
go mod tidy
go build -o jj-tui .
```

### Updating Screenshots

Screenshots are generated using [VHS](https://github.com/charmbracelet/vhs) with mock data for consistent, reproducible images.

**Automatic (CI)**: The [Generate Screenshots](.github/workflows/screenshots.yml) workflow produces `demo.gif` (`all.tape`), `after-origin.gif`, `evolog-split.gif`, `divergent.gif`, and the PNG captures; it runs after releases (via release workflows) and can be triggered manually. Results are committed to `screenshots/` on `main` when they change.

**Manual (Local)**:
```bash
# Generate PNG captures + after-origin.gif + evolog-split.gif + divergent.gif (demo GIF is separate; see make demo-gif)
make screenshots

# Generate demo GIF
make demo-gif

# Regenerate only the after-origin GIF (also included in `make screenshots`)
make after-origin-gif

# Regenerate only the evolog-split GIF (also included in `make screenshots`)
make evolog-split-gif

# Regenerate only the divergent GIF (also included in `make screenshots`)
make divergent-gif

# Or run individual tapes (PNG/GIF outputs live under screenshots/)
vhs vhs/graph.tape
vhs vhs/command_history.tape
vhs vhs/after-origin.tape
vhs vhs/evolog-split.tape
vhs vhs/divergent.tape
```

This creates a demo jj repository and runs the app in `--demo` mode, which uses mock ticket and PR data.

**Demo Mode**: You can also run the app manually in demo mode for testing:

```bash
make demo
# Or: ./jj-tui --demo
```

### Testing

Run all tests:

```bash
go test ./... -v
```

Run TUI unit tests:

```bash
go test ./internal/tui/ -v
```

Run integration tests (requires `jj` installed):

```bash
go test ./integration_tests/ -v
```

### Dependencies

- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)**: Terminal UI framework
- **[Lip Gloss](https://github.com/charmbracelet/lipgloss)**: Styling and layout
- **[Bubblezone](https://github.com/lrstanley/bubblezone)**: Mouse zone management for clickable UI elements
- **[go-github](https://github.com/google/go-github)**: GitHub API client
- **[oauth2](https://golang.org/x/oauth2)**: OAuth2 authentication

## User Stories

The application supports these key user workflows:

### 1. Commit Navigation
- View commit history in a visual graph with split-pane layout (graph + changed files)
- Navigate with keyboard shortcuts or mouse; Tab switches focus between graph and files panes
- See commit details, authors, and timestamps
- View changed files for each commit in the files pane; move files to parent/child commits (`[` / `]`) or revert (`v`)
- Mouse scroll works on the focused pane

### 2. Commit Management
- Edit commits (checkout with `jj edit` — press `e` or Enter on the commit you want to work on)
- Squash commits into their parents
- Rebase commits onto different parents
- Create new commits (including from immutable parents like `main`)
- Move files between commits: split a file into a new parent or child commit (`[` / `]` with files pane focused)
- Revert file changes in a commit (`v` with files pane focused)
- Immutable commit protection (cannot modify pushed commits)

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

### 6. Repository Monitoring
- Real-time repository state updates
- Conflict detection and display
- Divergent commit detection with visual indicator (⑂)
- Bookmark visualization
- Working copy indicator

### 7. Conflict Resolution
- Resolve divergent commits (choose which version to keep)
- Resolve conflicted bookmarks (keep local or reset to remote)
- Visual indicators in Graph and Branches views

### 8. Repository Setup & Cleanup
- Auto-detect non-jj repositories and offer to initialize
- Initialize with `jj git init` and automatically track `main@origin`
- Abandon old commits after merging PRs
- Delete all bookmarks for fresh start
- Track/fetch remote branches

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Run integration tests
5. Submit a pull request

## License

MIT License - see LICENSE file for details.

## Acknowledgments

- [Jujutsu](https://github.com/martinvonz/jj) for the amazing VCS
- [Charm](https://charm.sh/) for the excellent TUI libraries
- The Go community for great tooling and libraries
