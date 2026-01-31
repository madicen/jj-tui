# Jujutsu TUI

A modern Terminal User Interface (TUI) for managing [Jujutsu](https://github.com/martinvonz/jj) repositories. Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss) for an intuitive and beautiful command-line experience.

## Features

- **Visual Commit Graph**: Navigate and visualize your commit history with tree structure
- **Keyboard & Mouse Support**: Full keyboard navigation with zone-based mouse support for clickable UI elements
- **GitHub Integration**: Create and manage GitHub Pull Requests directly from the TUI
- **Jira Integration**: View assigned tickets and create branches from Jira issues
- **Commit Management**: Edit, squash, describe, abandon, and manage commits with simple key presses
- **Immutable Commit Detection**: Automatically detects and protects immutable commits (pushed to remote)
- **Real-time Updates**: Auto-refresh repository state and see changes immediately
- **Modern UI**: Beautiful styling with colors, borders, and responsive layouts

## Prerequisites

- [Jujutsu (jj)](https://github.com/martinvonz/jj) installed and available in your PATH
- Go 1.21+ for building from source
- A jujutsu repository to work with

## Installation

### From Source

```bash
cd jj-tui
go mod tidy
go build -o jj-tui .
```

### Running

```bash
# From within a jujutsu repository
./jj-tui

# Or specify a repository path
./jj-tui /path/to/your/jj/repo
```

## Usage

### Global Shortcuts

- `q`, `Ctrl+C`: Quit application
- `g`: Switch to commit graph view
- `p`: Switch to pull requests view
- `i`: Switch to Jira issues view
- `,`: Open settings
- `h`, `?`: Show help
- `r`: Refresh current view
- `Esc`: Return to main view

### Commit Graph View

- `↑/↓`, `j/k`: Navigate commits
- `e`, `Enter`: Edit selected commit (checkout with `jj edit`)
- `s`: Squash selected commit into parent
- `n`: Create new commit
- `d`: Edit commit description
- `a`: Abandon commit

### Pull Requests View

- `↑/↓`, `j/k`: Navigate pull requests
- `Enter`, `e`: Open PR in browser
- `r`: Refresh PR list

### Jira View

- `↑/↓`, `j/k`: Navigate tickets
- `Enter`: Create branch from selected ticket
- `r`: Refresh ticket list

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

The settings are stored for the current session. To persist them across sessions, use environment variables.

### Option 2: Environment Variables

## GitHub Integration

To use GitHub features, set your GitHub token:

```bash
export GITHUB_TOKEN=your_github_personal_access_token
```

The application will automatically detect GitHub remotes and enable PR functionality.

## Jira Integration

To use Jira features, set your Jira credentials:

```bash
export JIRA_URL=https://your-domain.atlassian.net
export JIRA_USER=your-email@example.com
export JIRA_TOKEN=your_api_token
```

Get your API token from: https://id.atlassian.com/manage-profile/security/api-tokens

### Jira Workflow

1. Press `i` to open the Jira view
2. Navigate through your assigned tickets with `j/k` or arrow keys
3. Press `Enter` to create a branch from the selected ticket
   - Creates a new commit
   - Creates a bookmark named after the ticket key (e.g., `proj-123`)
   - Sets the commit description to the ticket summary

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
│       ├── model.go        # Main application model (all views)
│       ├── model_test.go   # Event-based unit tests
│       ├── styles.go       # UI styling with lipgloss
│       ├── messages.go     # Event message types
│       └── zones.go        # Clickable zone ID constants
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

Run benchmarks:

```bash
go test ./integration_tests/ -bench=. -v
```

### Dependencies

- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)**: Terminal UI framework
- **[Lip Gloss](https://github.com/charmbracelet/lipgloss)**: Styling and layout
- **[Bubblezone](https://github.com/lrstanley/bubblezone)**: Mouse zone management for clickable UI elements
- **[go-github](https://github.com/google/go-github)**: GitHub API client
- **[go-jira](https://github.com/andygrunwald/go-jira)**: Jira API client
- **[oauth2](https://golang.org/x/oauth2)**: OAuth2 authentication

## User Stories

The application supports these key user workflows:

### 1. Commit Navigation
- View commit history in a visual graph
- Navigate with keyboard shortcuts
- See commit details, authors, and timestamps

### 2. Commit Management
- Edit commits (checkout with `jj edit`)
- Squash commits into their parents
- Create new commits
- Immutable commit protection (cannot modify pushed commits)

### 3. Pull Request Workflow
- Create GitHub PRs from current branch
- View existing PRs with status
- Update PRs with new commits
- Link commits to PRs automatically

### 4. Repository Monitoring
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
