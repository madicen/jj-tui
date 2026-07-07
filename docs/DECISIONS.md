# Decision log

This file records library-adoption and architecture decisions made while executing
`IMPROVEMENT_PLAN.md`. Each entry states the item ID, the decision (adopt/reject),
and concrete reasoning so a future reader understands why the code is the way it is.

---

## P3.1 — Unified-diff parsing: `bluekeyes/go-gitdiff`

**Decision: REJECT — keep the hand-written parser in `internal/integrations/jj/git_unified_hunks.go`.**

### What was evaluated

`ParseGitUnifiedHunksPerPath` (plus `parseUnifiedHunkHeader` / `parseHunkRange`) hand-parses
`jj diff --git` output into `[]UnifiedHunk` per path, plus a set of binary paths. The plan
proposed replacing this with `github.com/bluekeyes/go-gitdiff` (v0.8.1), which parses git-style
unified diffs, including rename/binary/mode headers and `\ No newline` cases.

The existing tests (`git_unified_hunks_test.go`, `git_unified_diff_stats_test.go`) are the contract
and must not change. I prototyped go-gitdiff behind those tests before touching any production code.

### Why the library cannot reproduce current behavior

go-gitdiff **strictly validates hunk line counts**. When a `@@ -old,N +new,M @@` header declares
more lines than the hunk body actually contains, it treats the next line (the following `@@` header
or `diff --git`) as an invalid hunk line and aborts. Critically, on a parse error it returns **zero
files** (`Parse` returns all files parsed *before* the error, which for a first-hunk mismatch is none),
so the entire diff is silently dropped rather than partially recovered.

The hand parser is deliberately **lenient**: it ignores the declared `@@` counts entirely and simply
collects `+` / `-` / ` ` / `\` lines until the next `@@` or `diff --git`. Several contract-defining
test fixtures rely on this leniency — e.g. `TestParseTwoHunksPrefix` uses:

```
@@ -1,3 +1,4 @@
 package p
+import "fmt"
 const x = 1
```

The header claims 3 old / 4 new lines, but the body has only 2 old-side and 3 new-side lines.
Running this fixture through `gitdiff.Parse` yields:

```
PARSE ERROR: gitdiff: line 8: invalid line operation: '@' (files=0)
```

The following tests all use this same count-mismatched two-hunk fixture and would fail against a
go-gitdiff-backed parser (all return 0 files → empty hunk map):

- `TestParseTwoHunksPrefix`
- `TestSanitizeHunkPrefixMapAgainstDiff_stalePathDropped`
- `TestSanitizeHunkPrefixMapAgainstDiff_staleOnlyReturnsNil`
- `TestSanitizeHunkPrefixMapAgainstDiff_clampsOversizedK`

go-gitdiff exposes no option to relax this validation, so the incompatibility is fundamental, not
a configuration gap. The other fixtures (single hunk, binary marker with/without `index` line,
multi-file) parse fine and map cleanly to `UnifiedHunk`; the count-strictness is the sole blocker.

### Why the leniency matters beyond the tests

The parser feeds the AI-driven evolog hunk-split flow (`internal/tui/ai/evolog_split_hunks.go`),
where hunk plans and diffs are influenced by LLM output and by peeling earlier files off `@`.
Tolerating slightly-inconsistent or truncated diffs (and degrading gracefully instead of dropping
the whole diff to zero files) is desirable there. Adopting a parser that discards the entire diff on
the first count mismatch would be a behavior regression, not just a test failure.

Note also that the patch-application, validation, and sanitization logic
(`ApplyUnifiedHunkPrefix`, `applyHunksToOriginalLines`, `VerifyUnifiedHunksReconstructRight`,
`ValidateHunkPrefixPlan`, `SanitizeHunkPrefixMapAgainstDiff`) is jj-split domain logic, not generic
diff parsing, and is out of scope for any diff-parsing library regardless.

### Outcome

Hand parser retained; `go-gitdiff` **not** added as a dependency. No production code or tests changed.
Per the plan's global rule ("if the library can't reproduce current behavior exactly, keep ours and
record why"), this is the intended successful outcome for P3.1.

---

## P3.2 — AI client cleanup: `sashabaranov/go-openai` and `google.golang.org/genai`

**Decision: REJECT both libraries — keep the hand-rolled multi-provider adapters in
`internal/integrations/llm`. Reuse the shared `internal/integrations/httpapi` base for request
building + auth decoration (implemented).**

### Current shape (already close to the target)

`internal/integrations/llm` already provides what P3.2 asks for:

- a single `Complete(ctx, systemPrompt, userPrompt) (string, error)` interface (`Provider`),
- thin per-provider adapters (`OpenAICompatibleProvider` over `Client`; `GeminiProvider`),
- JSON request/response code in exactly one place per provider (`chatRequest`/`chatResponse` +
  `parseOpenAIChatCompletionBody` in `client.go`; `geminiGenerate*` + `parseGeminiGenerateBody` in
  `gemini.go`),
- shared retry/backoff with `Retry-After` honoring (`retry.go`) and shared error-hint
  classification for throttle/overload/quota (`classify.go`),
- a config-profile-driven factory preserving configurable base URL, timeout, and profiles
  (`factory.go`).

### Why `go-openai` is rejected

1. **The contract tests must pass unchanged and are tightly coupled to our types.**
   `factory_test.go` directly reads `op.client.Model`, `op.client.APIKey`, and `op.client.BaseURL`;
   `client_test.go` marshals the internal `chatResponse` struct; `retry_test.go` asserts our exact
   retry semantics (429 retried honoring `Retry-After`, 401 not retried, attempt counts). go-openai
   has its own client types (no exported `Model`/`APIKey`/`BaseURL` fields to assert on) and performs
   **no retries**, so adopting it would force rewriting these tests — violating the accept criterion.
2. **We would still need our custom retry + failure-hint layer**, so go-openai removes little and
   duplicates request/response modeling we already have in one place.
3. **Binary cost.** Measured with a realistic `CreateChatCompletion` probe: go-openai adds
   **~6.4 MB** over the empty-Go baseline (jj-tui is ~17 MB → roughly +37%). go-openai has zero
   transitive deps, so this is the sole cost, but it buys nothing given (1) and (2).

### Why `google.golang.org/genai` is rejected

Measured with a realistic `GenerateContent` probe, genai adds **~16.7 MB** over baseline and pulls in
100+ transitive modules (`google.golang.org/api`, protobuf, OAuth machinery). That is meaningful
bloat for a single `generateContent` POST that the ~90-line hand-rolled adapter already covers. Keep
the thin hand-rolled Gemini adapter.

### What was implemented

Routed OpenAI-compatible and Gemini request construction + auth decoration through the shared
`httpapi.Client` (P1.6), removing the duplicated `http.NewRequestWithContext` + header-setting
boilerplate from both adapters. Response reading, 2xx handling, retry/backoff, and error
classification remain in the LLM-specific `withLLMHTTPRetry` because `httpapi`'s `EnsureOK`/`DoRead`
treat only 200 as success and perform no retries. All existing AI tests pass unchanged; JSON
request/response code remains in exactly one place per provider.
