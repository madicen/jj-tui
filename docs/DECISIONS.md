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
