# jj-tui Development Guide

How the codebase is laid out and how to build, test, and regenerate screenshots.
For end-user docs see the [README](../README.md) and [USAGE.md](USAGE.md); for the
testing philosophy, fixtures, and mocks see [TESTING.md](TESTING.md).

## Table of Contents

- [Project Structure](#project-structure)
- [Building](#building)
- [Testing](#testing)
- [Updating Screenshots](#updating-screenshots)
- [Dependencies](#dependencies)
- [Contributing](#contributing)

## Project Structure

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
│       ├── util/              # Clipboard, external editor, helpers
│       ├── model/             # Main TUI model (Update, view, keys, mouse, effects)
│       ├── tab/               # Tab interfaces (Renderer + target Tab/hook contracts)
│       └── tabs/              # Tab-specific models and views
│           ├── graph/         # Commit graph, keys, file move/revert, actions
│           ├── prs/           # Pull requests list
│           ├── prform/        # Create/update PR modal
│           ├── tickets/       # Tickets list (Jira/Codecks/Issues)
│           ├── branches/      # Branches/bookmarks list
│           ├── bookmark/      # Create bookmark modal
│           ├── descedit/      # Edit commit description modal
│           ├── settings/      # Settings tabs (GitHub, Jira, Codecks, tickets, branches, theme, ai, advanced)
│           ├── help/          # Help (shortcuts + jj command history)
│           ├── filediff/      # Full-file diff modal (jj diff)
│           ├── evologsplit/   # Evolog split wizard
│           ├── conflict/      # Bookmark conflict resolution
│           ├── divergent/     # Divergent commit resolution
│           ├── warning/       # Warning modal (e.g. empty descriptions)
│           ├── error/         # Error overlay
│           ├── initrepo/      # Non-jj repo → jj git init
│           └── githublogin/   # GitHub device flow
├── integration_tests/         # End-to-end tests against a real jj binary
├── fixtures/                  # Demo repository for screenshots
│   ├── setup-demo-repo.sh
│   ├── setup-after-origin-vhs-repo.sh
│   ├── setup-evolog-split-vhs-repo.sh
│   ├── setup-divergent-vhs-repo.sh
│   ├── setup-bookmark-conflict-vhs-repo.sh
│   ├── setup-op-log-vhs-repo.sh
│   ├── setup-revset-filter-vhs-repo.sh
│   ├── setup-multiselect-vhs-repo.sh
│   ├── after-origin-vhs-append-and-tui.sh
│   ├── demo-repo/             # Created by setup-demo-repo.sh
│   ├── after-origin-vhs-repo/ # Created by setup-after-origin-vhs-repo.sh (after-origin GIF)
│   ├── evolog-split-vhs-repo/ # Created by setup-evolog-split-vhs-repo.sh (evolog-split GIF)
│   ├── divergent-vhs-repo/    # Created by setup-divergent-vhs-repo.sh (divergent GIF)
│   ├── bookmark-conflict-vhs-repo/ # setup-bookmark-conflict-vhs-repo.sh (bookmark-conflict GIF)
│   ├── op-log-vhs-repo/       # setup-op-log-vhs-repo.sh (op-log GIF)
│   ├── revset-filter-vhs-repo/ # setup-revset-filter-vhs-repo.sh (revset-filter GIF)
│   └── multiselect-vhs-repo/  # setup-multiselect-vhs-repo.sh (multiselect GIF)
├── vhs/                       # VHS tapes for screenshot generation
│   ├── all.tape               # Main demo GIF
│   ├── after-origin.tape      # Forgot New Commit? (f) workflow GIF
│   ├── evolog-split.tape      # Evolog split (z) workflow GIF
│   ├── divergent.tape         # Resolve divergent change (d) GIF
│   ├── bookmark-conflict.tape # Diverged bookmark resolver (Branches c) GIF
│   ├── op-log.tape            # Operation log browser (Ctrl+o) GIF
│   ├── revset-filter.tape     # Revset search/filter (/) GIF
│   ├── multiselect.tape       # Multi-select batch abandon GIF
│   ├── graph.tape
│   └── ...
├── screenshots/               # Generated (demo.gif, after-origin.gif, evolog-split.gif, divergent.gif, bookmark-conflict.gif, op-log.gif, revset-filter.gif, multiselect.gif, *.png)
└── README.md
```

## Tabs and the root model

The root model (`internal/tui/model`) owns each tab. Cross-cutting orchestration
and tab wiring were consolidated (plan item P2.3) so that `model.go` no longer
imports any concrete tab package: those imports are now construction-time-only
(in `model/init.go`), and the runtime `Update`/`View` paths reach tabs only
through the `tab.Tab` interface and small typed hook interfaces.

- **Effects (`model/effects.go`).** Message handlers that need to touch another
  component's state (the error modal, cross-tab PR reconciliation, follow-up
  repository/branch reloads, bookmark-conflict sources) return typed `effect`
  values applied by one dispatcher (`applyEffects`). Prefer adding a new
  `eff*` type + `applyEffect` case over reaching into another component inline.
- **Tab registry (`model/init.go` `initTabRegistry`).** The six primary content
  tabs (graph, PRs, branches, tickets, settings, help) are registered in
  `tabRegistry` / `tabOrder` behind the `tab.Tab` interface (`internal/tui/tab`).
  The window-resize fan-out, content-render dispatch, and generic input
  forwarding (keys, mouse, zone clicks) iterate the registry instead of naming
  each concrete field.
- **Adapters (`model/tab_adapters.go`).** Each concrete tab is wrapped by a small
  adapter that satisfies `tab.Tab` (`Update(msg, *AppState) (Tab, tea.Cmd)`,
  `View(app)`, `SetDimensions`) and applies any per-tab command wrapping. Tab
  specific queries the model still needs (e.g. tickets' status-change mode,
  settings' esc handling) are exposed via typed hook interfaces
  (`model/tab_hooks.go`) and looked up with a type assertion — the base
  `tab.Tab` interface is deliberately kept narrow.
- **Async/handler routing (`model/async_dispatch.go`, `model/model_handlers.go`).**
  The concrete-typed message cases (background `*LoadedMsg` routing, effects,
  form submits) live in `dispatchAsyncMsg` — the `default` arm of the root
  `Update` switch — and the tab-driving `*Model` methods live in
  `model_handlers.go`. These sibling files hold the concrete tab-package imports
  so `model.go` stays free of them.

**Adding a primary tab touches ≤3 places:** (1) add the concrete field to the
`Model` struct in `model/model_state.go`; (2) register a `tab.Tab` adapter for it
in `model/init.go` `initTabRegistry` (and add the adapter struct in
`model/tab_adapters.go`); (3) wire any async result messages it emits into
`dispatchAsyncMsg` (`model/async_dispatch.go`). Optional tab-specific queries go
through a hook interface in `model/tab_hooks.go` rather than widening `tab.Tab`.

## Building

```bash
go mod tidy
go build -o jj-tui .
```

The [Makefile](../Makefile) wraps the common tasks: `make build`, `make test`,
`make cover` (tests + coverage summary), and the screenshot targets below.

## Testing

```bash
# All tests (unit + integration)
make test          # go test ./...

# Tests with a coverage summary
make cover         # go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out

# Integration tests only (requires jj installed)
go test ./integration_tests/... -v
```

See [TESTING.md](TESTING.md) for the unit-vs-integration boundary, fixture
scripts, table-driven conventions, and where mocks live.

## Updating Screenshots

New VHS fixture repos should follow the pattern in existing `fixtures/setup-*-vhs-repo.sh` scripts (create an isolated jj repo, seed the scenario, then point a `.tape` at it).

Screenshots are generated using [VHS](https://github.com/charmbracelet/vhs) with mock data for consistent, reproducible images.

**Automatic (CI)**: The [Generate Screenshots](../.github/workflows/screenshots.yml) workflow produces `demo.gif` (`all.tape`), `after-origin.gif`, `evolog-split.gif`, `divergent.gif`, `bookmark-conflict.gif`, `op-log.gif`, `revset-filter.gif`, `multiselect.gif`, and the PNG captures; it runs after releases (via release workflows) and can be triggered manually. Results are committed to `screenshots/` on `main` when they change.

**Manual (Local)**:
```bash
# Generate PNG captures + workflow GIFs (demo GIF is separate; see make demo-gif)
make screenshots

# Generate demo GIF
make demo-gif

# Regenerate individual workflow GIFs (also included in `make screenshots`)
make after-origin-gif
make evolog-split-gif
make divergent-gif
make bookmark-conflict-gif
make op-log-gif
make revset-filter-gif
make multiselect-gif

# Or run individual tapes (PNG/GIF outputs live under screenshots/)
vhs vhs/graph.tape
vhs vhs/command_history.tape
vhs vhs/after-origin.tape
vhs vhs/evolog-split.tape
vhs vhs/divergent.tape
vhs vhs/op-log.tape
vhs vhs/revset-filter.tape
vhs vhs/multiselect.tape
```

This creates a demo jj repository and runs the app in `--demo` mode, which uses mock ticket and PR data.

**Demo Mode**: You can also run the app manually in demo mode for testing:

```bash
make demo
# Or: ./jj-tui --demo
```

## Dependencies

- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)**, **[Bubbles](https://github.com/charmbracelet/bubbles)** (e.g. spinners, inputs)
- **[Lip Gloss](https://github.com/charmbracelet/lipgloss)**: Styling and layout
- **[Bubblezone](https://github.com/lrstanley/bubblezone)**: Mouse hit targets
- **[bubble-overlay](https://github.com/madicen/bubble-overlay)**, **[bubble-color-picker](https://github.com/madicen/bubble-color-picker)**: Centered modals and theme UI
- **[go-github](https://github.com/google/go-github)**, **[oauth2](https://golang.org/x/oauth2)**: GitHub REST + device flow
- **[githubv4](https://github.com/shurcooL/githubv4)**: GitHub GraphQL (issues)

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Run integration tests
5. Submit a pull request
