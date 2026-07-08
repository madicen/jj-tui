# jj-tui Testing Guide

How the test suite is organized, what needs a real `jj` binary, and the
conventions to follow when adding tests. See [DEVELOPMENT.md](DEVELOPMENT.md) for
build/screenshot tooling and the [README](../README.md) for the feature overview.

## Table of Contents

- [Running the tests](#running-the-tests)
- [Unit vs integration](#unit-vs-integration)
- [Where mocks live](#where-mocks-live)
- [Fixtures](#fixtures)
- [Running against a real jj](#running-against-a-real-jj)
- [Table-driven conventions](#table-driven-conventions)
- [TUI model test helpers](#tui-model-test-helpers)

## Running the tests

```bash
# Everything (unit + integration), same as CI
make test                       # go test ./...

# With a coverage summary
make cover                      # go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out

# A single package
go test ./internal/tui/model/... -v

# Integration tests only (need a real jj on PATH)
go test ./integration_tests/... -v
```

CI runs the full-package suite with coverage on both `ubuntu-latest` and
`macos-latest`, and lints with `golangci-lint` (see
[`.github/workflows/ci.yml`](../.github/workflows/ci.yml)).

## Unit vs integration

The suite has two layers with a deliberate boundary:

- **Unit tests** live next to the code they test as `*_test.go` files under
  `internal/...` (55+ files). They must run **without** a `jj` binary, network
  access, or on-disk repositories. They cover pure logic (parsers, revset
  building, name sanitizing, view rendering) and TUI message handling using the
  in-memory model helpers and mock services described below.
- **Integration tests** live in [`integration_tests/`](../integration_tests) and
  exercise the real [`internal/integrations/jj`](../internal/integrations/jj)
  service against a **real `jj` binary** in a throwaway repository created per
  test (`NewTestRepository`). They cover behaviors that only emerge when driving
  the actual CLI: `GetRepository`, `CreateNewCommit`, `MoveFileToChild`, evolog
  hunk-split flows, chained commits, etc.

Rule of thumb: if a behavior depends on `jj`'s actual output or side effects,
it belongs in `integration_tests/`; if it can be expressed with canned
input/mocks, keep it as a fast unit test beside the code.

Any test that shells out to `jj` **must guard** on availability so the suite
degrades gracefully where `jj` is absent:

```go
if _, err := exec.LookPath("jj"); err != nil {
    t.Skip("jj command not available")
}
```

## Where mocks live

- **[`internal/mock`](../internal/mock)** — runtime mock services used by
  **demo mode** (`jj-tui --demo`) and by tests that need a stand-in GitHub or
  ticket backend: `mock/github.go`, `mock/tickets.go`. These implement the same
  interfaces the real integrations satisfy so the TUI can run with mock PRs and
  tickets.
- **[`internal/testutil`](../internal/testutil)** — test-only helpers and mocks
  (`testutil/mocks.go`): `NewMockJiraService()`, `MockTicketService`,
  `MockJJService`, etc. Prefer these in unit and TUI tests instead of hitting a
  real service.

When you add a new integration interface, add its mock alongside the existing
ones so both demo mode and tests can use it.

## Fixtures

Shell scripts under [`fixtures/`](../fixtures) build reproducible jj repositories
for the VHS screenshot recordings (see [DEVELOPMENT.md](DEVELOPMENT.md#updating-screenshots)):

| Script | Builds | Used by |
|---|---|---|
| `setup-demo-repo.sh` | `fixtures/demo-repo/` | the main demo GIF / most tapes |
| `setup-after-origin-vhs-repo.sh` | `fixtures/after-origin-vhs-repo/` | "Forgot new commit?" (`f`) GIF |
| `setup-evolog-split-vhs-repo.sh` | `fixtures/evolog-split-vhs-repo/` | evolog split (`z`) GIF |
| `setup-divergent-vhs-repo.sh` | `fixtures/divergent-vhs-repo/` | divergent resolver GIF |
| `setup-bookmark-conflict-vhs-repo.sh` | `fixtures/bookmark-conflict-vhs-repo/` (+ fake origin) | diverged-bookmark resolver GIF |

These generated repositories are **not tracked** (they are `.gitignore`d) — a
script regenerates them on demand. A test that needs one of these fixtures should
invoke the script from `TestMain`/setup rather than relying on tracked output;
the bookmark-conflict fixture test in `internal/tui/model` does exactly this by
running `setup-bookmark-conflict-vhs-repo.sh`.

Integration tests generally do **not** use these VHS fixtures; they create their
own minimal repo per test via `NewTestRepository`, which runs `jj git init` and
sets a local `user.name` / `user.email`.

## Running against a real jj

Integration tests and jj-backed unit tests require `jj` on `PATH` and a usable
git identity (jj reads git config for author info). Locally:

```bash
jj --version                    # confirm jj is installed
git config --global user.name   # confirm an identity exists (CI sets a dummy one)
go test ./integration_tests/... -v
```

In CI the workflow installs the latest `jj` release (via `brew` on macOS, the
musl tarball on Linux) and configures a dummy `user.name` / `user.email` before
running `go test ./...`.

## Table-driven conventions

Prefer table-driven subtests for anything with multiple input/output cases —
parsers, sanitizers, and formatters especially. Use a slice of named-case
structs and drive each with `t.Run` so failures point at the exact case:

```go
func TestSanitizeBookmarkName(t *testing.T) {
    cases := []struct {
        name string
        in   string
        want string
    }{
        {"spaces to underscores", "Implement auth", "Implement_auth"},
        {"strip conflicted suffix", "feat (conflicted)", "feat"},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            if got := SanitizeBookmarkName(tc.in); got != tc.want {
                t.Errorf("SanitizeBookmarkName(%q) = %q, want %q", tc.in, got, tc.want)
            }
        })
    }
}
```

Parser tests such as `changed_files_parse_test.go` define the contract for the
corresponding parser — treat them as the spec when refactoring parsing code.

## TUI model test helpers

The TUI is a Bubble Tea program, so tests drive it by feeding `tea.Msg`s to
`Model.Update` and asserting on `Model.View()` output or exported getters.
`integration_tests/main_test.go` provides reusable helpers:

- `newTestModel()` — builds a `*tui.Model` with fixed dimensions and a small
  in-memory repository (no `jj` required).
- `updateModel(m, msg)` — runs `Update` once and returns the new `*tui.Model`.
- `updateModelWithCmd(m, msg)` — runs `Update`, then runs the returned `tea.Cmd`
  once and feeds its message back (for request/response flows like dismissing an
  error modal).
- `runPendingCmds(m, msg, maxRounds)` — drains a chain of commands (expanding
  `tea.BatchMsg`) the way the real runtime would, for multi-step flows like
  "Save Local" or "Create Bookmark from Ticket".

Assert on rendered output with substring checks (`strings.Contains`) for
user-visible text, and on exported getters (`GetViewMode`, `GetSelectedPR`, …)
for state.
