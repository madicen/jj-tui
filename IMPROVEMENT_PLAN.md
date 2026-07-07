# jj-tui Improvement Plan

> **Audience:** an AI coding agent (Claude Opus 4.8) executing this plan work-item by work-item, plus human reviewers.
> **Generated:** 2026-07-06 from a full-codebase analysis (~52k Go LOC).

---

## How to execute this plan

Rules for the executing agent:

1. **One work item per changeset.** Each item below has an ID (e.g. `P1.2`). Complete it, verify it, commit it (`jj describe` / commit message must reference the ID), then move on. Never batch unrelated items.
2. **Verification protocol (run after every item):**
   ```sh
   make build            # go build -o jj-tui .
   make test             # unit tests
   go test ./integration_tests/...   # requires a real `jj` binary on PATH
   golangci-lint run
   ```
   An item is not done until all four pass.
3. **Behavior-preserving first.** Phases 0–3 must not change user-visible behavior (except where the item explicitly says so). If a refactor forces a behavior choice, keep the current behavior and leave a `// PLAN(P<id>):` comment.
4. **Respect dependency order.** Each item lists `Depends:`. Do not start an item whose dependencies are incomplete.
5. **Tests move with code.** When extracting a package, move/adapt its tests in the same changeset and add tests for the new shared package itself.
6. **When in doubt about jj semantics**, consult `README.md` (usage) and `integration_tests/` (expected flows) before changing `internal/integrations/jj`.
7. **Effort key:** S = <½ day, M = ½–2 days, L = 3–5 days, XL = >1 week.

---

## Current-state summary (verified)

- **Stack:** Go 1.25, bubbletea v1.3.10, bubbles v1.0.0, lipgloss v1.1.0, bubblezone (used in 47 files), custom `madicen/bubble-{overlay,dropdown,color-picker}`, go-github/v66 + githubv4, hand-rolled Jira/Codecks HTTP clients, OpenAI-compatible/Gemini/Ollama AI client.
- **God files:** `internal/integrations/jj/service.go` (3,358 LOC, ~130 public methods), `internal/tui/model/model.go` (2,576 LOC; the `Model` has ~134 methods across 9 files), `internal/config/config.go` (1,351 LOC, one flat struct with 50+ fields).
- **Tab wiring is ad-hoc:** no `Tab` interface; root model holds each tab as a concrete field and makes 115+ direct method calls into tabs; tabs import each other (`graph/actions.go` imports descedit/bookmark/branches tabs); cross-tab communication is a 40+-kind `NavigateMsg` handled by a ~330-line switch in `model.go`.
- **Duplication clusters:** 7 settings sub-tabs re-implement the same form/focus/dropdown plumbing; branches/prs/tickets tabs re-implement list selection + scrolling + long-press context menus; 7 `view_helpers.go` files repeat truncation/width/zone-mark helpers. Estimated 1,400–1,900 LOC removable.
- **Testing/CI:** 59 test files, but `ci.yml` only tests a subset of packages, ubuntu-only, no coverage reporting. `.golangci.yml` is permissive.
- **Hygiene:** `fixtures/_tmp-bookmark-conflict-origin.git/` (a raw git dir incl. hook samples) is tracked; local clutter (`tmp.evolog-test-*`, `.tmp-jj-split*`, 17MB built binary, `.DS_Store`) exists but is untracked; README is 45KB with no TOC.
- **Feature gaps vs jj:** no operation-log browser (undo/redo exist via `jj undo` / `jj op restore` at `service.go:380-402` but ops are never shown), no `absorb`, no file-level conflict resolution UI (the conflict tab is bookmark-divergence only), no revset search/filter in graph, no `duplicate`/`backout`, no `file annotate` (blame), no multi-select, no workspaces UI, hardcoded keybindings, manual-only refresh (`Ctrl+r`).

---

## Phase 0 — Hygiene & engineering health (do first; all independent)

### P0.1 Remove tracked temp fixture and tighten .gitignore
- **Effort:** S · **Depends:** none
- `git rm -r --cached fixtures/_tmp-bookmark-conflict-origin.git` (verify first that fixture setup scripts — `fixtures/setup-bookmark-conflict-vhs-repo.sh` — regenerate it; if a test needs it, regenerate in TestMain instead of tracking it).
- Add to `.gitignore`: `.DS_Store`, `tmp.*`, `.tmp-*`, `fixtures/_tmp-*`, `*.prof`.
- **Accept:** `git ls-files | grep -E '_tmp|^tmp\.|\.DS_Store'` returns nothing; fixture-dependent tests still pass.

### P0.2 CI: full package coverage, coverage report, OS matrix
- **Effort:** M · **Depends:** none
- In `.github/workflows/ci.yml`: replace the per-package test list with `go test ./...` (excluding fixtures), add `-coverprofile` + upload artifact (or Codecov), add `macos-latest` to the matrix (Windows optional — gate integration tests on jj availability).
- Add a `make cover` target.
- **Accept:** CI green on both OSes; coverage artifact produced; no package silently untested.

### P0.3 Stricter golangci-lint
- **Effort:** M · **Depends:** none
- Enable explicitly: `govet, staticcheck, errcheck, gosec, gocritic, revive, misspell, unconvert, prealloc`. Fix or `//nolint` (with reason) all findings.
- **Accept:** `golangci-lint run` clean with the stricter config.

### P0.4 README restructure
- **Effort:** M · **Depends:** none
- Add a TOC; split into `README.md` (features, install, quickstart, screenshots), `docs/USAGE.md` (per-tab keys and workflows), `docs/DEVELOPMENT.md` (structure, building, tests, fixtures/VHS, release). Keep anchors used by links.
- **Accept:** README < 15KB; all screenshots/links resolve.

### P0.5 CI badge + TESTING.md
- **Effort:** S · **Depends:** P0.2
- Add CI badge to README. Write `docs/TESTING.md`: unit vs integration boundary, fixture scripts, how to run with a real jj, table-driven conventions, where mocks live (`internal/mock`, `internal/testutil`).
- **Accept:** files exist, referenced from README.

---

## Phase 1 — Extract shared internal libraries (kill duplication)

Goal: net −1,400 LOC or better, no behavior change. These establish the packages later phases build on.

### P1.1 `internal/tui/render` — shared view utilities
- **Effort:** M · **Depends:** none
- Audit the seven `view_helpers.go` files (`tabs/settings` 859 LOC, `tabs/graph` 527, `model` 352, plus tickets/prs/branches/help/error) and extract the repeated pure helpers:
  - `Mark(zm *zone.Manager, id, content string) string` (nil-safe zone marking — currently copy-pasted per tab)
  - `TruncateEllipsis(s string, max int) string` — **must be rune/ANSI-safe**, replacing the byte-slicing `desc[:150]+"..."` pattern (use `x/ansi` truncate)
  - `Separator(width int) string` with the shared min-width clamp
  - `SafeWidth(total, padding int) int`
- Replace all call sites; delete local copies.
- **Accept:** no tab-local duplicate of these helpers remains (`grep -rn "func mark(" internal/tui/tabs` empty); unit tests for the new package incl. wide-rune truncation.

### P1.2 `internal/tui/form` — shared settings-form base
- **Effort:** L · **Depends:** P1.1
- The eight settings sub-tabs (`tabs/settings/{github,jira,codecks,tickets,branches,ai,theme,advanced}/model.go`, 138–640 LOC each) all hand-roll: `focusedField` int + clamping, j/k/up/down focus cycling, focus/blur of `textinput.Model`s, `SetInputWidth` fan-out, and getter/setter pairs.
- Create `form.Model`: ordered field list (text inputs + dropdown + toggle field kinds), focus management, navigation key handling, width propagation, zone-manager wiring.
- Migrate sub-tabs one per commit, smallest first (theme → branches → tickets → codecks → advanced → github → ai → jira).
- **Accept:** each migrated sub-tab shrinks ≥40%; existing settings tests (`github-settings_test.go`, `ai/*_test.go`, `ai_profile_flow_test.go`) pass unchanged or with mechanical updates only.

### P1.3 `internal/tui/form/dropdown` — dropdown helper
- **Effort:** S · **Depends:** P1.2 (or fold into it)
- Extract the 4× duplicated pattern (accent-color syncing, nil guards, `wasOpen` + `ItemChosenMsg` index dispatch) from `settings/{github,tickets,ai,advanced}` into a `DropdownField` with an `onSelect(i int)` callback.
- **Accept:** no sub-tab defines its own `UpdateDropdown`.

### P1.4 `internal/tui/listnav` — shared list-tab base
- **Effort:** L · **Depends:** P1.1
- `tabs/{branches,prs,tickets}/model.go` each re-implement: `selectedIndex`, `listYOffset`, `SetDimensions`, scroll-to-selected, long-press state (`longPressItemIndex/PressID/MouseX/Y`), context-menu open state, double-click tracking.
- Create `listnav.Model` embedding that state + `HandleNav(key)`, `HandleLongPress(msg)`, `ScrollToSelected()`, `VisibleRange()`. Tabs keep their own rendering and context-menu *actions*.
- Migrate branches → prs → tickets, one per commit. Evaluate graph tab last (its viewport is more complex; only adopt if clean).
- **Accept:** the three tabs share the base; context-menu behavior identical (existing `context_menu_test.go` files pass).

### P1.5 `internal/tui/confirm` — reusable confirm modal
- **Effort:** M · **Depends:** none
- Consolidate the repeated y/n confirm-dialog implementations (per-tab `actions.go` files, `evologsplit/nosplit_confirm.go`, `model/overlay_helpers.go`) into one component: title, message, yes/no callbacks returning `tea.Cmd`, mouse zones for buttons.
- **Accept:** all confirm flows route through it; adds the hook Phase 5 needs for destructive-op confirmation (P5.3).

### P1.6 `internal/integrations/httpapi` — shared HTTP service base
- **Effort:** M · **Depends:** none
- Extract from `jira/service.go` and `codecks/service.go` (and github where not covered by go-github): base-URL normalization, request building with auth decorator func, status→error mapping (esp. 401), JSON decode, context timeouts.
- **Accept:** jira + codecks services use it; their service tests pass; net LOC down.

### P1.7 `internal/integrations/jj/jjout` — jj output parsing helpers
- **Effort:** S · **Depends:** none
- Extract `SplitLines` (trim + drop empties) and the repeated regex field-extraction used by `parseChangeIDFromOutput`, `parseCommitInfo`, evolog/bookmark parsers, plus `extractErrorMessage` (`service.go:2525-2541`).
- **Accept:** parsers in service.go call the shared helpers; existing parse tests (`changed_files_parse_test.go`, etc.) pass.

---

## Phase 2 — Architectural refactor

Goal: make the codebase safe to extend. Order matters here.

### P2.1 `jj.Runner` — command-runner interface
- **Effort:** M · **Depends:** P1.7
- Define in `internal/integrations/jj`:
  ```go
  type Runner interface {
      Run(ctx context.Context, args ...string) error
      RunOutput(ctx context.Context, args ...string) (string, error)
  }
  ```
  Note the *six* current variants that must collapse into it: `runJJ` (:2491), `runJJOutput` (:2545), `runJJWithGlobal` (:2417), `runJJOutputWithGlobal` (:2450), `runJJOutputNoHistory` (:2313), `runJJOutputNoHistoryWithGlobal` (:2318). Model global args and history-logging as options (`Run(ctx, opts, args...)` or functional options) rather than six methods. Move command-history logging into the `execRunner`. `Service` takes a `Runner` (default: execRunner). Add a `fakeRunner` in `internal/mock` for canned-output unit tests.
- **Accept:** all jj invocations go through `Runner`; at least 3 previously-integration-only behaviors get fast unit tests using `fakeRunner` (e.g. error-message extraction, undo op-id capture, bookmark list parsing).

### P2.2 Split `jj/service.go` into domain services
- **Effort:** XL · **Depends:** P2.1
- Split by domain, keeping `Service` as a thin façade so ~130 call sites don't all change at once:
  - `jj/graph.go` → repo/log loading, changed files (`GetRepository`, `getCommitGraph`, `GetChangedFiles`)
  - `jj/mutate.go` → describe/new/squash/rebase/merge/abandon
  - `jj/bookmarks.go` → bookmark CRUD + diverged-bookmark resolution
  - `jj/evolog.go` + existing `service_evolog_hunk_split.go` → evolog list + split flows
  - `jj/ops.go` → op log/undo/redo (extended in P4.1)
  - `jj/remote.go` → git push/fetch/remote list
- Mechanical moves first (same package, new files), then extract sub-structs (`Service.Graph`, `Service.Mutate`, …) with the façade delegating. Deprecate façade methods gradually.
- **Accept:** no file in `internal/integrations/jj` exceeds ~800 LOC; all jj tests + integration tests pass; façade keeps API compatibility for tabs.

### P2.3 `Tab` interface + lifecycle hooks
- **Effort:** L · **Depends:** P1.2, P1.4
- Define in `internal/tui/state` (or new `internal/tui/tab`):
  ```go
  type Tab interface {
      Update(msg tea.Msg, app *state.AppState) (Tab, tea.Cmd)
      View(app *state.AppState) string
      SetDimensions(w, h int)
  }
  // optional hooks, checked by type assertion:
  type RepositoryAware interface{ OnRepositoryLoaded(*internal.Repository) }
  type Activatable interface{ OnActivated(); OnDeactivated() }
  ```
- Convert tabs one per commit; root model stores `map[state.ViewMode]Tab` + ordered list instead of 6 concrete fields; replace the 115+ direct calls with interface calls/hooks.
- **Accept:** `model.go` imports of concrete tab packages drop to construction-time only (`init.go`); adding a hypothetical new tab requires touching ≤3 places (documented in DEVELOPMENT.md).

### P2.4 Centralized modal/overlay state machine
- **Effort:** L · **Depends:** P2.3
- Replace the ~15 scattered booleans on `Model` (`model.go:49-149`: `evologDescribePreviewActive`, `bookmarkConflictReturnValid`, `modalUnderlayValid`, …) with a `ModalStack`:
  ```go
  type ModalKind int
  type Modal struct { Kind ModalKind; Model tea.Model }
  type ModalStack struct { stack []Modal }  // Push/Pop/Top/Has
  ```
  `chromedSlot()` (`modal_layer.go`) and `applyFormModalsOverlay()` read the stack top instead of testing booleans in priority order.
- **Accept:** zero modal-state booleans left on `Model`; modal stacking/priority behavior unchanged (verify against `chrome_minimize_test.go`, `modal_frame` tests, and VHS fixture tests).

### P2.5 Remove inter-tab imports
- **Effort:** M · **Depends:** P2.3
- `tabs/graph/actions.go` imports descedit/bookmark/branches/prs tabs directly. Route these through `NavigateMsg` / the modal stack instead.
- **Accept:** `grep -rn "tabs/" internal/tui/tabs/*/ --include='*.go' | grep import`-style check shows no tab importing a sibling tab.

### P2.6 Config sub-structs + change notification
- **Effort:** L · **Depends:** P1.2
- Restructure `config.Config` into `GitHub`, `Jira`, `Codecks`, `Tickets`, `AI`, `Theme`, `UI`, `Advanced` sub-structs. **Keep JSON keys identical** (tags stay `github_token` etc.) so existing `~/.config/jj-tui/config.json` and `.jj-tui.json` files load unchanged; add a round-trip compatibility test using a fixture of today's schema (the repo's own `.jj-tui.json` is a sample).
- Add a `config.ChangedMsg` broadcast after save so tabs re-read instead of holding stale snapshots.
- **Accept:** old config files load byte-compatibly (golden-file test); settings save triggers live re-style (e.g. theme change applies without restart — it already does; must not regress).

### P2.7 Shrink the `NavigateMsg` mega-switch
- **Effort:** M · **Depends:** P2.3, P2.4
- `handleNavigate` (`model.go:576-900`) has 40+ kinds. Split into per-domain handlers (`navigate_graph.go`, `navigate_pr.go`, …) or convert modal-opening kinds into `ModalStack.Push` calls. No kind may bypass the stack.
- **Accept:** no single navigate handler >100 LOC.

### P2.8 Single source of truth for Repository
- **Effort:** M · **Depends:** P2.3
- Tabs currently cache their own `Repository` copy via `UpdateRepository()` fan-out (`data_handlers.go:70-76`), with manual PR reconciliation (`data_handlers.go:250-256`). Make tabs read `app.Repository` (they already receive `*AppState`) and delete the fan-out + caches; `RepositoryAware` hook remains for tabs needing recompute-on-load.
- **Accept:** `grep -rn "UpdateRepository" internal/tui` only matches the AppState setter; refresh behavior identical.

---

## Phase 3 — Replace hand-rolled code with libraries

Each item: prototype behind the existing tests first; if the library can't reproduce current behavior exactly, keep ours and record why in a `docs/DECISIONS.md` entry.

### P3.1 Unified-diff parsing → `bluekeyes/go-gitdiff` (or `sourcegraph/go-diff`)
- **Effort:** M · **Depends:** P2.2
- `internal/integrations/jj/git_unified_hunks.go` (379 LOC) hand-parses `jj diff --git` output. Both libraries parse git-style unified diffs robustly (rename/binary/mode headers, `\ No newline` cases we may mishandle).
- Port behind the existing tests (`git_unified_hunks_test.go`, `git_unified_diff_stats_test.go`) — they define the contract. Prefer `go-gitdiff` (zero deps, handles git extensions).
- **Accept:** all existing hunk/stat tests pass against the library-backed implementation; hand parser deleted.

### P3.2 AI client cleanup (keep multi-provider, drop boilerplate)
- **Effort:** M · **Depends:** none
- `internal/tui/ai` supports `openai_compatible`, `gemini`, `ollama` (see `config.go:160-163, 1018-1035`) via hand-rolled HTTP. Extract a small `internal/ai/client` package with one `Complete(ctx, req) (string, error)` interface and three thin provider adapters, reusing P1.6's HTTP base. Evaluate `sashabaranov/go-openai` for the openai-compatible path (also covers Ollama's OpenAI-compat endpoint); Gemini via `google.golang.org/genai` only if it doesn't bloat the binary — otherwise keep the thin hand-rolled adapter.
- Preserve: configurable base URL, timeout settings (see `ai_timeout_test.go`), profiles (`profiles_test.go`).
- **Accept:** provider behavior matrix unchanged (existing AI tests pass); JSON request/response code exists in exactly one place per provider.

### P3.3 Keybindings → `bubbles/key` (enabler for P5.1)
- **Effort:** L · **Depends:** P2.3
- Keys are hardcoded string switches in `internal/tui/model/keys.go` and per-tab handlers. Introduce `key.Binding` maps per tab (`type KeyMap struct { Up, Down, Confirm key.Binding … }`) with the current keys as defaults. This is mostly mechanical but touches every tab — do it per-tab after P2.3 lands.
- Bonus: `key.Binding` carries help text → feed the help tab from KeyMaps instead of a hand-maintained list.
- **Accept:** all shortcuts behave identically; help tab content generated from KeyMaps.

### P3.4 Evaluate `charmbracelet/huh` for prform/ticketform — **evaluation only**
- **Effort:** M · **Depends:** P1.2
- After P1.2, our own form base may be sufficient. Spike: rebuild `tabs/prform` on huh in a branch; check overlay composition (bubble-overlay), zone-based mouse support, and theming compatibility. Adopt only if it deletes >300 LOC without regressing mouse behavior. Record the decision in `docs/DECISIONS.md`.

### P3.5 Self-update check → `creativeprojects/go-selfupdate` (optional)
- **Effort:** S · **Depends:** none
- `internal/version` hand-rolls a GitHub release check. Library adds signature verification + actual self-update. Keep "notify only" as default; `--self-update` flag opt-in. Skip if Homebrew is the primary channel and this adds more surface than value — decision entry either way.

### P3.6 Bubbletea/lipgloss/bubbles v2 migration — **explicitly deferred**
- Do not migrate during this plan. The custom `madicen/bubble-*` deps and bubblezone must move in lockstep; do it as its own project after Phase 2 stabilizes. Add a `docs/DECISIONS.md` entry with the trigger condition (all upstream deps have stable v2 releases).

---

## Phase 4 — Missing features (ranked by user value)

All feature UIs must use the Phase 1/2 primitives (Tab interface, ModalStack, listnav, confirm). Each feature: service method(s) + integration test first, UI second.

### P4.1 Operation log browser + safe time-travel
- **Effort:** L · **Depends:** P2.2 (jj/ops.go), P2.3, P2.4
- Undo/redo already shell out to `jj undo`/`jj op restore` (`service.go:380-402`) but users can't see operations. Add:
  - `ops.List(ctx, limit)` parsing `jj op log --no-graph -T <template>` into `{ID, Description, Time, User}`.
  - An **Operations** view (key: `Ctrl+o` or under Help) listing ops via `listnav`; Enter → confirm modal → `op restore` to that op; refresh graph after.
  - Status-line hint after each mutating op: "Ctrl+z undoes: <op description>" (P5.4 delivers the hint UI).
- **Accept:** integration test: perform 3 ops, restore to the first, graph reflects it; op list renders in fixture repo.

### P4.2 `jj absorb`
- **Effort:** M · **Depends:** P2.2
- Graph tab: `a` on working copy (or context menu) → preview modal (run `jj absorb --dry-run`, show which revisions receive which files) → confirm → `jj absorb`. Handle "nothing absorbed" gracefully.
- **Accept:** integration test with a fixture where a WC change lands in an ancestor; dry-run preview matches result.

### P4.3 File-level merge-conflict resolution UI
- **Effort:** XL · **Depends:** P2.2, P2.4, P3.1
- Today `tabs/conflict` handles only diverged *bookmarks*. Add real conflict support:
  - Detect conflicted revisions/files (graph already marks conflicts; extend `GetChangedFiles` to flag conflicted paths).
  - Conflict view per file: parse jj conflict markers (materialized "diff-style" markers), offer per-conflict **take side 1 / side 2 / both**; write result via `jj restore`-style plumbing or direct file write + snapshot. Fallback action: "open in external merge tool" (`jj resolve --tool`).
- Start with the fallback (`jj resolve` + external tool + `jj resolve --list` status view) as an MVP; per-hunk in-TUI picking is a stretch goal.
- **Accept:** integration test: create a conflicted merge in a fixture, resolve one file via the flow, `jj resolve --list` shows it resolved.

### P4.4 Revset search/filter in graph
- **Effort:** L · **Depends:** P2.3
- `/` in graph opens an input overlay: accept either free text (compiled to `description(substring-i:"…") | author(substring-i:"…")`) or a raw revset (prefix `:`). Run `jj log -r <revset>` into a filtered graph state; `Esc` restores; show active filter in the header. Validate revset errors inline (jj's error text).
- **Accept:** filtering by author/text works in the demo fixture; invalid revset shows error without losing graph state.

### P4.5 `jj duplicate` + `jj backout`
- **Effort:** M · **Depends:** P2.2
- Context menu entries on a revision: **Duplicate** (`jj duplicate -r X`, optional `-d` destination via the same destination-picker used by rebase) and **Backout/Revert** (`jj backout -r X` — check jj version: newer jj renamed to `jj revert`; detect via `jj backout --help` exit code and use the right verb).
- **Accept:** integration tests for both; version-detection covered by a unit test on the fake runner.

### P4.6 Blame view (`jj file annotate`)
- **Effort:** M · **Depends:** P2.2, P2.3
- Files pane: `B` on a file → scrollable annotate view (`jj file annotate <path> -r <rev>`), lines prefixed with change-id/author/age, Enter on a line jumps the graph selection to that change.
- **Accept:** renders on fixture repo; jump-to-change works.

### P4.7 Multi-select batch operations
- **Effort:** L · **Depends:** P1.4, P2.4
- Graph: `Space` toggles selection (marker in gutter), `Esc` clears. Batch actions on selection: abandon (with confirm listing all), squash-into-parent chain, rebase selection (`jj rebase -r A -r B -d dest` where contiguous). Keep scope tight: abandon + rebase first.
- **Accept:** select 3 commits → abandon → all gone, single undo op restores all (jj batches per command — verify and document).

### P4.8 Workspaces (view-only MVP)
- **Effort:** M · **Depends:** P2.2
- List `jj workspace list` in a small view; indicate current; actions: add/forget behind confirm. No cross-workspace graph rendering in this plan.
- **Accept:** list + add + forget work in an integration test.

---

## Phase 5 — Quality of life

### P5.1 Configurable keybindings
- **Effort:** M · **Depends:** P3.3, P2.6
- Add `"keys": { "graph.abandon": "x", … }` to config (global + per-repo). Load into the P3.3 KeyMaps at startup; validate collisions per scope and report in an error modal. Document defaults in USAGE.md (generated table if easy).
- **Accept:** rebinding a key via config works; collision produces a clear error, not silent misbehavior.

### P5.2 Auto-refresh
- **Effort:** M · **Depends:** P2.8
- Config `ui.auto_refresh_seconds` (0 = off, default 0). `tea.Tick`-driven silent reload (reuse `SilentRepositoryLoadedMsg` path) that **skips while any modal is open or a jj command is in flight** to avoid clobbering in-progress work. Stretch: watch `.jj/repo/op_heads` mtime instead of polling.
- **Accept:** with it on, an external `jj new` in a shell appears within the interval; no refresh occurs while a describe modal is open.

### P5.3 Destructive-op confirmations
- **Effort:** S · **Depends:** P1.5
- Route abandon, divergent-resolution (abandons losers), backout-free `op restore`, and force-ish pushes through the shared confirm modal, each with a one-line consequence description ("Abandons 2 revisions; undo with Ctrl+z"). Config toggle `ui.confirm_destructive` (default on).
- **Accept:** each destructive path shows confirm; toggle disables.

### P5.4 Undo hint in status line
- **Effort:** S · **Depends:** P4.1
- After every mutating command, fetch latest op description (cheap: `jj op log --limit 1`) and show "Ctrl+z undoes: …" in the footer for a few seconds.
- **Accept:** visible after describe/squash/abandon.

### P5.5 Error modal: extend Retry coverage
- **Effort:** S · **Depends:** P2.4
- A Retry mechanism already exists (`tabs/error/model.go:16` `hasRetry` + `NavigateRetryError`, used by e.g. the PRs tab). Audit all error paths and wire `hasRetry` for every retryable operation (push, fetch, PR/ticket API calls, AI calls); ensure the failing command is consistently carried so replay works.
- **Accept:** inventory of error paths documented; each retryable one shows Retry and replays correctly.

### P5.6 Progress/spinner audit
- **Effort:** S · **Depends:** none
- Audit all >200ms operations (initial load, push/fetch, PR list, AI calls) for a visible loading state; fix gaps using the existing spinner + `state.Loading`.
- **Accept:** no interaction path freezes silently (manual checklist in the PR description).

### P5.7 Paste support in text areas
- **Effort:** S · **Depends:** none
- Ensure bracketed paste works in describe/PR/ticket forms (bubbletea v1 supports it; verify multiline paste into `textarea`/`textinput` fields and strip trailing newline artifacts).
- **Accept:** multi-line paste into PR description preserves lines.

---

## Sequencing

```
Phase 0 (all parallel)
   │
Phase 1:  P1.1 ─→ P1.2 ─→ P1.3        P1.5   P1.6   P1.7
              └─→ P1.4                                │
   │                                                  ▼
Phase 2:  P2.1 (needs P1.7) ─→ P2.2 ─────────────────────────┐
          P2.3 (needs P1.2+P1.4) ─→ P2.4 ─→ P2.5, P2.7       │
          P2.6                    P2.8                        │
   │                                                          ▼
Phase 3:  P3.1 · P3.2 · P3.3 · P3.4 · P3.5   (P3.6 deferred) │
   │                                                          ▼
Phase 4:  P4.1 → P5.4 · P4.2 · P4.3 · P4.4 · P4.5 · P4.6 · P4.7 · P4.8
Phase 5:  P5.1 (after P3.3) · P5.2 · P5.3 · P5.5 · P5.6 · P5.7
```

Suggested execution order for maximum early value:
1. **Week 1:** P0.1–P0.5 (hygiene + CI safety net — everything after this is protected by better CI)
2. **Weeks 2–4:** P1.1–P1.7 (dedup; codebase shrinks and patterns emerge)
3. **Weeks 5–9:** P2.1–P2.8 (architecture; biggest risk, gated by the tests added above)
4. **Weeks 10–11:** P3.1–P3.3 (library swaps now that seams exist)
5. **Weeks 12+:** Phase 4 features interleaved with Phase 5 QoL (each feature is now cheap because of the primitives)

## Risk register

| Risk | Mitigation |
|---|---|
| P2.2 (service split) breaks subtle evolog/split flows | Façade keeps API stable; integration tests + fixture VHS repos run per commit; move code before changing it |
| P2.4 (modal stack) changes overlay z-order | Snapshot current `chromedSlot` priority order into a table test *before* refactoring |
| P2.6 config restructure corrupts user configs | Golden-file round-trip test with today's schema; JSON tags frozen |
| P3.1 diff library differs on edge cases | Existing hunk tests are the contract; add `\ No newline`/rename/binary cases before swapping |
| jj CLI version drift (`backout` vs `revert`, split path args) | Central version-detect helper in `jj` package (P4.5); document minimum jj version in README |
| Bubbletea v2 temptation mid-plan | Explicitly deferred (P3.6); decision file records trigger conditions |

## Definition of done (whole plan)

- `internal/integrations/jj` has no file >800 LOC; `model.go` <1,000 LOC.
- Net LOC reduction from Phases 1–3 ≥1,200 despite added tests.
- CI: `go test ./...` on ubuntu+macos with coverage artifact; stricter lint clean.
- New tabs/features implementable by touching ≤3 wiring points (documented).
- Feature parity additions shipped: op log browser, absorb, conflict-resolution MVP, revset search, duplicate/backout, blame, configurable keys, auto-refresh.

