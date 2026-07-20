# GPG signing error recovery (design)

Date: 2026-07-20  
Status: approved for implementation planning

## Problem

When `jj` fails to create a commit because GPG cannot unlock a key (typical message: `gpg: signing failed: No pinentry`), the CLI reports:

```text
Internal error: Unexpected error from backend
Caused by:
1: Could not write object of type commit
2: Signing error
3: GPG failed with exit status: 2:
gpg: signing failed: No pinentry
```

jj-tui’s `jjout.ExtractErrorMessage` keeps only the first `Error:` / first meaningful line, so the user sees:

```text
failed to create commit: Internal error: Unexpected error from backend
```

There is no in-app way to recover (e.g. restart a stuck `gpg-agent`).

## Goals

1. Always surface a more descriptive jj error (include the `Caused by` / `gpg:` chain).
2. For recognized signing/pinentry failures, offer **Kill gpg-agent** on the shared error modal.
3. On successful kill, **auto-retry** the original failed command when a retry cmd is stashed.

## Non-goals (v1)

- Disable signing from the modal / session override
- Switch jj signing backend to SSH
- “Unlock in external terminal” helper
- Killing processes other than `gpg-agent`
- Changing jj or GPG configuration permanently

## Approach

Enrich extractor + classify + modal action (existing error modal / retry effects).

## Design

### 1. Richer error extraction (always)

Update `jjout.ExtractErrorMessage`:

1. Prefer the first line beginning with `Error:` or `Internal error:` as the head.
2. Append following lines that are part of the cause chain:
   - `Caused by:`
   - numbered cause lines (`N: …`)
   - bare `gpg:` lines
3. Stop at a blank line, `Hint:`, or `Warning:`.
4. Return a multi-line string suitable for the error modal.

Status-bar consumers that need a single line may truncate separately; do not strip causes at the extractor.

### 2. Signing / pinentry classifier

Pure helper, e.g. `jjout.IsSigningPinentryFailure(text string) bool`:

- Case-insensitive substring match on the full error text for signals such as:
  - `signing error`
  - `no pinentry`
  - `gpg: signing failed`
- No subprocess probes; classification is text-only.

### 3. Error modal: Kill gpg-agent

When the error modal opens and the classifier matches the displayed error text:

- Show an extra button **Kill gpg-agent** (key `k`, plus a mouse zone).
- Unrelated errors keep the existing Copy / Dismiss / Quit (+ Retry when already retryable).

Kill command: `gpgconf --kill gpg-agent` via `exec` (same style as other external tools).

- Success → status `gpg-agent killed`.
- Failure → keep modal; surface the kill error (status or appended message); do **not** auto-retry.

### 4. Auto-retry after successful kill

On successful kill:

- If `pendingRetryCmd` is set → clear the error modal and run it (same path as Retry / `NavigateRetryError`).
- Else → leave the modal open (user can dismiss / copy).

### 5. Make mutation failures retryable

Extend `util.ErrorMsg` with optional `Retry tea.Cmd`.

- Mutation cmds that return `ErrorMsg` (at least `NewCommit`; same pattern for siblings that share the wrapper if cheap) set `Retry` to the same cmd factory so Retry and post-kill auto-retry both work.
- `async_dispatch` maps `ErrorMsg` with non-nil `Retry` to `effShowRetryableError`; plain `ErrorMsg` stays `effShowError`.

### Data flow

1. `jj` fails → runner uses enriched `ExtractErrorMessage` → `util.ErrorMsg{Err, Retry?}`.
2. Main → `effShowRetryableError` if `Retry` set, else `effShowError`; classifier sets “has kill gpg-agent” on the modal from err text.
3. User activates **Kill gpg-agent** → navigate/effect → `gpgconf --kill gpg-agent` → result msg.
4. Success + `pendingRetryCmd` → clear modal + run retry; else update status per rules above.

## Tests

- `jjout`: cause-chain extraction; classifier true/false cases.
- Error modal: kill button visible only when classified; key/zone wired.
- Main/effects: successful kill with pending retry clears modal and schedules retry; kill without retry leaves modal; kill failure does not auto-retry.
- `NewCommit` (or small helper): `ErrorMsg` carries non-nil `Retry`.

## Decisions locked in brainstorming

| Topic | Choice |
| --- | --- |
| UX | Full cause chain always + in-modal actions only for recognized signing/pinentry failures |
| Recovery action (v1) | Kill gpg-agent only |
| After successful kill | Auto-retry when a retry cmd is stashed |
| Implementation shape | Enrich extractor + classify + modal action |
