# P2.3 Tab-Migration Test Safety Net

This document describes the behavioral / golden test safety net added to de-risk
the **P2.3 all-or-nothing tab migration** (moving the six primary tabs and modal
views onto a generic `Tab` interface with generic message forwarding). It lists
what is and isn't locked down so the migration worker knows the guarantees.

> **Goal:** if the migration changes ANY user-visible output (a moved line, a
> dropped word, a changed color), drops a command, mis-routes a background
> message, or opens/closes a modal differently, a test fails loudly.

## How to run / update

```sh
# Verify the whole net:
make build && make test && go test ./integration_tests/... && golangci-lint run

# Golden tests only:
go test ./internal/tui/model ./internal/tui/tabs/settings -run Golden

# Regenerate goldens after an INTENTIONAL, reviewed change:
go test ./internal/tui/model -run TestGoldenView -update
go test ./internal/tui/model -run TestGoldenModal -update
go test ./internal/tui/tabs/settings -run TestGoldenSettings -update
```

The golden helper lives at `internal/testutil/golden.go`
(`testutil.AssertGolden(t, name, actual)` + `testutil.ForceDeterministicRendering()`).
Golden files are stored under each package's `testdata/` directory.

## Determinism strategy

Golden output is byte-exact **including ANSI escapes**. To keep it stable across
machines / CI / TTY vs non-TTY:

- `testutil.ForceDeterministicRendering()` pins the lipgloss color profile
  (`termenv.TrueColor`) and dark-background flag.
- Each golden test also pins the global theme via `styles.SetTheme(...)`.
- Fixtures use fixed dates/IDs and stable ordering.
- The **Settings** tab is golden-locked in its own package (not through the root
  `Model.View()`) because its root-model render calls `exec.LookPath("gh")` and
  reads integration env vars; the settings golden test clears those env vars and
  pins `GhAvailable=false`.

## Coverage inventory — GOLDEN-LOCKED (full `View()` string)

Root-model composite (`internal/tui/model/golden_view_test.go`, `golden_modal_test.go`):

| Area | States locked |
|---|---|
| Graph | loaded+selection (with changed-files tree), working-copy selected, immutable selected, empty/loading |
| PRs | loaded (with GitHub), no-GitHub connect screen |
| Branches | loaded + selection (local/remote/tracked/conflict rows) |
| Tickets | loaded (with service), no-service screen |
| Help | shortcuts pane |
| Error modal | no-retry, with-retry |
| Warning modal | with commit list |
| Bookmark-conflict modal | via `BookmarkConflictInfoMsg`, Branches underlay |
| GitHub-login modal | device-flow, gh-CLI mode |
| Workspaces modal | loaded list |
| Describe modal | populated textarea |
| Create-PR modal | opened for a bookmarked commit |
| Create-bookmark modal | opened |
| Create-ticket modal | opened (mock Jira service) |
| Graph rebase mode | destination-select UI |
| Graph merge mode | source-select UI |
| PR context menu | long-press menu open (representative list-tab menu) |

Settings (`internal/tui/tabs/settings/golden_view_test.go`):

| Area | States locked |
|---|---|
| Every sub-tab | GitHub, Jira, Codecks, Tickets, Branches, Theme, AI, Advanced |
| Focused field | Jira field 3, AI field 1 |

## Coverage inventory — BEHAVIORAL (state + emitted commands, not golden)

Background message routing while a DIFFERENT tab is active
(`internal/tui/model/background_routing_test.go`) — the behavior most at risk in
the migration (today routed by concrete type in `model.go`):

- `PrsLoadedMsg`, `TicketsLoadedMsg`, `BranchesLoadedMsg`, `ChangedFilesLoadedMsg`
  delivered while inactive still reach their tab.
- `PrsLoadedMsg` emits the resolve-open-PRs effect; `BranchActionMsg` success
  emits reload effects, failure emits none; `BranchesLoadedMsg` in the
  create-bookmark view keeps the modal open; `LoadErrorMsg` routes to the error
  modal.
- `WindowSizeMsg` fans out to every tab.

Model dispatch / modal-flow / effects
(`internal/tui/model/dispatch_flow_test.go`):

- Tab-switch key routing (`g/p/t/b/,/h`, `esc`→graph).
- Per-modal-kind `chromedSlot` gating (open key + cleared when back on a tab).
- `NavigateBackToGraph` / `NavigateDismissError` reset to graph.
- Create-bookmark open→Esc→close flow.
- Effect dispatcher outcomes: show/clear error, reload repo, load branches,
  resolve open PRs, set bookmark-conflict sources, `applyEffects` batching.
- Additional `chromedSlot` z-order combos (error/warning alone, workspaces
  overlay, error/warning over workspaces). The committed
  `chromed_slot_zorder_test.go` covers the base priority matrix.

Per-tab key/request mapping and selection clamping are covered by the existing
`*characterization*` tests under `internal/tui/tabs/*` (kept green across the
migration).

## Known GAPS (not locked; call out if you touch these)

- **Graph commit/file context menus** are exercised behaviorally elsewhere but
  are NOT golden-locked at the root level (they require a mouse-press + long-press
  tick sequence with unexported graph arming fields). The PR context menu golden
  is the representative long-press-menu lock; branches/tickets menus are not
  golden-locked.
- **Divergent, evolog-split, and file-diff modals** are locked at the z-order /
  gating level (`chromedSlot` key) but do NOT have full-body `View()` goldens.
- **Settings GitHub/Jira/Codecks live values** are rendered from cleared env in
  the golden (so secrets/host state don't leak); real populated states are not
  golden-locked.
- **AI-profile popover** over form modals is not captured (form-modal goldens are
  taken with the popover closed / no injected profiles).
- **Time/relative-time rendering**: none of the golden-locked views render wall
  clock time (command-history timestamps are empty in tests); if a future view
  renders `time.Now()`-derived text, it must inject a fixed clock before it can
  be golden-locked.

## Guidance for the migration worker

1. Run the full verification protocol before AND after the migration. A golden
   diff is expected to be **empty**; if `AssertGolden` fails, the migration
   changed rendered output — investigate before running `-update`.
2. The background-routing tests are the canary for generic message forwarding:
   if you replace concrete-type routing with "forward to active tab", these fail.
   Preserve routing of `*LoadedMsg` to the owning tab regardless of active tab.
3. Only regenerate goldens with `-update` after a human-reviewed, intentional
   rendering change — never to "make the build green".
