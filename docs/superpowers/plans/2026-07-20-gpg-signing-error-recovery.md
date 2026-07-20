# GPG Signing Error Recovery Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Surface jj `Caused by` / `gpg:` chains in errors, and let users kill `gpg-agent` from the error modal with auto-retry when a retry cmd is stashed.

**Architecture:** Enrich `jjout.ExtractErrorMessage`; add text classifier; extend error modal + navigate/effects; optional `Retry` on `util.ErrorMsg` so `NewCommit` can auto-retry after kill.

**Tech Stack:** Go, bubbletea, existing error modal / `effShowRetryableError`.

## Global Constraints

- v1 recovery action is kill gpg-agent only (`gpgconf --kill gpg-agent`).
- No disable-signing, SSH switch, or unlock-in-terminal helper.
- Classifier is text-only (no subprocess probes).
- Spec: `docs/superpowers/specs/2026-07-20-gpg-signing-error-recovery-design.md`

## File map

| File | Role |
| --- | --- |
| `internal/integrations/jj/jjout/jjout.go` | Richer extract + `IsSigningPinentryFailure` |
| `internal/integrations/jj/jjout/jjout_test.go` | Unit tests |
| `internal/tui/util/messages.go` | `ErrorMsg.Retry tea.Cmd` |
| `internal/tui/util/gpg.go` | `KillGPGAgentCmd` + result msg |
| `internal/tui/mouse/zones.go` | `ZoneActionKillGPGAgent` |
| `internal/tui/state/navigate.go` | `NavigateKillGPGAgent` |
| `internal/tui/tabs/error/*` | Modal UI + key/zone for kill |
| `internal/tui/model/effects.go` | Set kill flag when showing error |
| `internal/tui/model/async_dispatch.go` | Route `ErrorMsg.Retry`; handle kill result |
| `internal/tui/model/navigate_repo.go` | Handle `NavigateKillGPGAgent` |
| `internal/tui/tabs/graph/actions.go` | `NewCommit` sets `Retry` |

---

### Task 1: jjout extract + classifier

**Files:**
- Modify: `internal/integrations/jj/jjout/jjout.go`
- Modify: `internal/integrations/jj/jjout/jjout_test.go`

**Interfaces:**
- Produces: `ExtractErrorMessage(output string) string` (multi-line cause chain)
- Produces: `IsSigningPinentryFailure(text string) bool`

- [ ] **Step 1: Extend tests for cause chain + classifier**
- [ ] **Step 2: Implement extract + classifier**
- [ ] **Step 3: `go test ./internal/integrations/jj/jjout/ -count=1`**
- [ ] **Step 4: Commit**

### Task 2: Kill cmd + ErrorMsg.Retry + navigate/zone

**Files:**
- Create: `internal/tui/util/gpg.go`
- Modify: `internal/tui/util/messages.go`
- Modify: `internal/tui/mouse/zones.go`
- Modify: `internal/tui/state/navigate.go`

- [ ] **Step 1: Add `KillGPGAgentResultMsg`, `KillGPGAgentCmd`**
- [ ] **Step 2: Add `ErrorMsg.Retry`, zone id, navigate kind**
- [ ] **Step 3: Commit**

### Task 3: Error modal kill button

**Files:**
- Modify: `internal/tui/tabs/error/model.go`
- Modify: `internal/tui/tabs/error/view_helpers.go`
- Modify: `internal/tui/tabs/error/view_helpers_test.go` (+ model tests as needed)

- [ ] **Step 1: `hasKillGPG` field; SetError classifies via `jjout.IsSigningPinentryFailure`**
- [ ] **Step 2: Render button; key `k`; zone click → `NavigateKillGPGAgent`**
- [ ] **Step 3: Tests for visibility / key**
- [ ] **Step 4: Commit**

### Task 4: Main wiring + NewCommit retry + auto-retry

**Files:**
- Modify: `internal/tui/model/async_dispatch.go`
- Modify: `internal/tui/model/navigate_repo.go`
- Modify: `internal/tui/model/effects.go` (if needed)
- Modify: `internal/tui/tabs/graph/actions.go`
- Test: `internal/tui/model/retry_error_test.go` or new `gpg_kill_test.go`

- [ ] **Step 1: `util.ErrorMsg` with Retry → `effShowRetryableError`**
- [ ] **Step 2: Handle `NavigateKillGPGAgent` → run kill cmd**
- [ ] **Step 3: Handle `KillGPGAgentResultMsg` → status; on success + pending retry → clear + retry**
- [ ] **Step 4: `NewCommit` self-referential `Retry`**
- [ ] **Step 5: Tests**
- [ ] **Step 6: build/vet/test; commit**

---

## Spec coverage

| Spec item | Task |
| --- | --- |
| Richer ExtractErrorMessage | 1 |
| IsSigningPinentryFailure | 1 |
| Kill button on modal | 3 |
| gpgconf --kill gpg-agent | 2, 4 |
| Auto-retry when pendingRetryCmd | 4 |
| ErrorMsg.Retry / NewCommit | 4 |
| Out of scope items | omitted |
