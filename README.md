# Jujutsu TUI

A modern Terminal User Interface (TUI) for managing [Jujutsu](https://github.com/martinvonz/jj) repositories. Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss) for an intuitive and beautiful command-line experience.

## Features

- **Visual Commit Graph**: Navigate and visualize your commit history with tree structure
- **Changed Files View**: See files modified in the selected commit with a nested folder structure
- **Keyboard & Mouse Support**: Full keyboard navigation with zone-based mouse support for clickable UI elements
- **GitHub Integration**: Create and manage GitHub Pull Requests directly from the TUI
- **Jira Integration**: View assigned tickets and create branches from Jira issues with auto-populated names
- **Commit Management**: Edit, squash, describe, abandon, rebase, and manage commits with simple key presses
- **Bookmark Management**: Create, move, and delete bookmarks on commits
- **Immutable Commit Detection**: Automatically detects and protects immutable commits (pushed to remote)
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

If you have Go 1.21+ installed:

```bash
go install github.com/madicen/jj-tui@latest
```

### Download Binary

Download pre-built binaries from the [GitHub Releases](https://github.com/madicen/jj-tui/releases) page.

Available for:
- **macOS** (Intel & Apple Silicon)
- **Linux** (amd64 & arm64)
- **Windows** (amd64 & arm64)

### From Source

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
```

## Usage

### Global Shortcuts

- `Ctrl+q`: Quit application
- `Ctrl+r`: Refresh current view
- `g`: Switch to commit graph view
- `p`: Switch to pull requests view
- `t`: Switch to tickets (Jira) view
- `,`: Open settings
- `h`: Show help
- `Esc`: Return to main view / Cancel current action

### Commit Graph View

- `↑/↓`, `j/k`: Navigate commits
- `e`, `Enter`: Edit selected commit (checkout with `jj edit`)
- `s`: Squash selected commit into parent
- `n`: Create new commit
- `d`: Edit commit description
- `a`: Abandon commit
- `r`: Rebase selected commit
- `b`: Create or move bookmark on selected commit
- `x`: Delete bookmark from selected commit
- `c`: Create PR from selected commit (if bookmark exists)
- `u`: Push/update PR (pushes to existing PR branch)

### Pull Requests View

- `↑/↓`, `j/k`: Navigate pull requests
- `Enter`, `e`: Open PR in browser
- `Ctrl+r`: Refresh PR list

### Tickets (Jira) View

- `↑/↓`, `j/k`: Navigate tickets
- `Enter`: Create branch from selected ticket
- `Ctrl+r`: Refresh ticket list

### Settings View

- `Tab`, `↓`: Move to next field
- `Shift+Tab`, `↑`: Move to previous field
- `Enter`: Move to next field (or save if on last field)
- `Ctrl+S`: Save settings
- `Esc`: Cancel and return to graph
- **Click** on any field to focus it

## Settings

You can configure your API credentials in two ways:

### Option 1: In-App Settings (Recommended)

1. Press `,` or click the **Settings** tab
2. Enter your credentials in the form fields
3. Press `Ctrl+S` or click **Save** to apply

Settings are saved to `~/.config/jj-tui/config.json` and persist across sessions.

### Option 2: Environment Variables

## GitHub Integration

To use GitHub features, set your GitHub token:

```bash
export GITHUB_TOKEN=your_github_personal_access_token
```

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
   - Creates a new commit branched from main
   - Creates a bookmark with a sanitized name (e.g., `PROJ-123-ticket-summary`)
   - Rebases your current work onto the new branch
   - Pre-populates PR title with "PROJ-123 - Ticket Summary" when you create a PR

## Configuration

The application automatically detects:
- Current jujutsu repository
- GitHub remote configuration
- User preferences from jj config
- Credentials from environment variables or in-app settings

## Development

### Project Structure

```
jj-tui/
├── main.go                 # Application entry point
├── internal/
│   ├── jj/                 # Jujutsu command integration
│   │   └── service.go
│   ├── github/             # GitHub API integration
│   │   └── service.go
│   ├── jira/               # Jira API integration
│   │   └── service.go
│   ├── models/             # Data models
│   │   └── commit.go
│   └── tui/                # Terminal UI components
│       ├── model.go        # Main application model
│       ├── view.go         # View rendering
│       ├── keys.go         # Keyboard handlers
│       ├── mouse.go        # Mouse handlers
│       ├── actions.go      # Business logic actions
│       ├── messages.go     # Event message types
│       ├── zones.go        # Clickable zone ID constants
│       ├── styles.go       # UI styling with lipgloss
│       └── view/           # View renderers
│           ├── renderer.go
│           ├── graph.go
│           ├── prs.go
│           ├── jira.go
│           ├── bookmark.go
│           └── help.go
├── integration_tests/      # Integration tests
│   └── main_test.go
└── README.md
```

### Building

```bash
go mod tidy
go build -o jj-tui .
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
- View commit history in a visual graph
- Navigate with keyboard shortcuts
- See commit details, authors, and timestamps
- View changed files for each commit

### 2. Commit Management
- Edit commits (checkout with `jj edit`)
- Squash commits into their parents
- Rebase commits onto different parents
- Create new commits
- Immutable commit protection (cannot modify pushed commits)

### 3. Bookmark Management
- Create bookmarks on any mutable commit
- Move existing bookmarks to different commits
- Delete bookmarks when no longer needed

### 4. Pull Request Workflow
- Create GitHub PRs from commits with bookmarks
- View existing PRs with status
- Update PRs by pushing new commits
- Push from descendant commits (bookmark auto-moves)

### 5. Jira Integration
- View assigned Jira tickets
- Create branches from tickets with auto-named bookmarks
- PR titles auto-populated from Jira ticket info

### 6. Repository Monitoring
- Real-time repository state updates
- Conflict detection and display
- Bookmark visualization
- Working copy indicator

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
