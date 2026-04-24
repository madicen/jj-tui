package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/integrations/llm"
)

// EvologSplitSuggestMsg is sent when the LLM finishes suggesting an evolog split row.
type EvologSplitSuggestMsg struct {
	ReqID     int
	NoSplit   bool
	PickIndex int // evolog list row index (same as UI): 1 .. len-1 when !NoSplit
	Rationale string
	// FilesForFirstCommit, when non-empty, triggers jj split -r @ with these filesets after the FAQ row-split (optional second phase).
	FilesForFirstCommit []string
	// HunkPrefixFirstCommit maps path → first k hunks into the first child commit (optional second phase; preferred over files when both set).
	HunkPrefixFirstCommit map[string]int
	// MultiSplitBaseCommitIDs is an ordered deepest-first list of base commit ids for sequential FAQ splits (length capped by settings and evolog row count).
	MultiSplitBaseCommitIDs []string
	Err                     error
}

const evologSplitMaxDiffSteps = 72
const evologSplitMaxPromptRunes = 14_000

// evologSplitStepDiffConcurrency limits parallel jj processes while building step summaries.
const evologSplitStepDiffConcurrency = 8

// evologSplitPromptBuildTimeout is the max time allowed to run all jj diff --summary calls for the user prompt.
// This is separate from AITimeout so a slow repo does not consume the LLM budget.
const evologSplitPromptBuildTimeout = 5 * time.Minute

// evologSplitFileValidateTimeout caps jj diff for post-LLM file-path validation.
const evologSplitFileValidateTimeout = 45 * time.Second

// evologSplitMaxMultiIDsToParse bounds how many split_base_commit_ids we accept from JSON (safety + matches evolog depth).
func evologSplitMaxMultiIDsToParse(evologRowCount int) int {
	if evologRowCount < 2 {
		return 1
	}
	v := evologRowCount - 1
	const hard = 256
	if v > hard {
		return hard
	}
	return v
}

const evologSplitSystem = `You help developers split one jj change using an "evolve log" (evolog).

The evolog is listed newest-first: index 0 is the current tip, index 1 is its parent along that rewrite history, etc.

You may recommend a single split boundary (BASE row index between 1 and N-1), recommend NO split, optionally suggest filesets OR per-file hunk prefixes for a second-phase split of the working copy after the row split, and optionally suggest as many sequential FAQ row splits as are justified using commit ids from the table (see split_base_commit_ids).

You receive each row's short id, full commit id, and first-line description, plus for some rows a jj diff --summary for the step from row i to row i-1. When hunk excerpts are included, they are git unified diffs for early steps — use them to count hunks per path (0-based in order of @@ blocks for that path).

Reply with a single JSON object only (no markdown fences, no commentary):
{"no_split": <bool>, "recommended_index": <int>, "rationale": "<short plain text>", "confidence": "high"|"medium"|"low",
 "files_first_commit": ["<path>", ...],
 "hunk_prefix_first_commit": {"<path>": <int>, ...},
 "split_base_commit_ids": ["<full commit id from table>", ...]}

Rules:
- If a single row split is not warranted (e.g. noise-only evolog, or one coherent change), set "no_split": true and "recommended_index": 0. Still provide rationale.
- If you recommend a split, set "no_split": false and set "recommended_index" to an integer from 1 through N-1 (N = number of evolog rows). That row is the BASE for the FAQ split.
- "split_base_commit_ids": optional array of commit_id values copied EXACTLY from the evolog rows' commit ids, for running multiple FAQ splits in order. When you need more than one peel to isolate distinct changes, list every base commit id required: deepest first (larger row index / older work toward smaller index / newer work), so each step applies cleanly after the previous one. You may list up to N-1 ids (N = row count in the user message) when that many separations are warranted; otherwise use fewer. Each id MUST appear in the "## Rows" list. Omit or use a one-element array equal to the chosen row's commit id when only one FAQ split is needed.
- "files_first_commit": optional list of repo-relative paths for ONE extra peel of the working copy after the first FAQ split, using only paths that appear in the diff for the chosen recommended_index step. Paths must be a strict subset of changed paths: NEVER list every file in that step (at least one changed path must stay on @). Omit or [] if not needed. Do not use together with "hunk_prefix_first_commit" when hunks give a finer plan — the UI prefers hunks and will ignore file lists if both are set.
- "hunk_prefix_first_commit": optional object mapping repo-relative path → integer k. For that path in the chosen recommended_index step diff, the first k hunks (indices 0..k-1) go into the FIRST child commit after jj split; remaining hunks for that path stay on @. k must be strictly less than the number of hunks for any path you include. Across the whole change, at least one path must still have hunks left on @ (do not peel every hunk of every file). Omit or {} when not needed, when excerpts were not provided, or when file-level split is enough.
`

// EvologSplitJJStepTotal returns how many jj diff --summary steps are used when building the AI suggest prompt.
func EvologSplitJJStepTotal(entryCount int) int {
	if entryCount < 2 {
		return 0
	}
	return min(evologSplitMaxDiffSteps, entryCount-1)
}

// evologSuggestPrepBatchTimeout bounds one batch of parallel jj diff calls.
const evologSuggestPrepBatchTimeout = 3 * time.Minute

// evologSplitStepDiffTimeout caps one jj diff --summary for a single evolog step. Without this, one
// stuck jj process keeps the whole batch (and UI progress) blocked until evologSuggestPrepBatchTimeout.
const evologSplitStepDiffTimeout = 2 * time.Minute

// EvologSuggestPrepProgressMsg is sent after each jj diff batch while building the suggest prompt (small payload only).
type EvologSuggestPrepProgressMsg struct {
	ReqID   int
	JJDone  int
	JJTotal int
}

// EvologSuggestPrepDoneMsg ends JJ prep. UserPrompt is trimmed and rune-capped on the worker goroutine before send.
type EvologSuggestPrepDoneMsg struct {
	ReqID      int
	JJTotal    int // step count used for JJ prep (for LLM-phase overlay)
	UserPrompt string
	Err        error
}

// evologSuggestMaxPromptBytes aborts prep if the accumulated prompt grows beyond this (avoids OOM / multi-minute trims).
const evologSuggestMaxPromptBytes = 12 << 20

// EvologSuggestPrepChainStartCmd runs all jj diff batches in a tea.Sequence chain so the UI never receives
// multi-megabyte PromptSoFar strings on each step (that was freezing the TUI when combined with TrimEvologUserPrompt).
func EvologSuggestPrepChainStartCmd(reqID int, jjSvc *jj.Service, cfg *config.Config, entries []jj.EvologEntry) tea.Cmd {
	return func() tea.Msg {
		var acc strings.Builder
		nextLow := 0
		jjTotal := 0
		var step func() tea.Msg
		step = func() tea.Msg {
			if jjSvc == nil || cfg == nil {
				return EvologSuggestPrepDoneMsg{ReqID: reqID, Err: fmt.Errorf("AI suggest: missing service or config")}
			}
			if len(entries) < 2 {
				return EvologSuggestPrepDoneMsg{ReqID: reqID, Err: fmt.Errorf("need at least two evolog rows")}
			}
			if nextLow == 0 {
				prefix, lim, err := buildEvologSplitPromptPrefix(entries)
				if err != nil {
					return EvologSuggestPrepDoneMsg{ReqID: reqID, Err: err}
				}
				acc.WriteString(prefix)
				jjTotal = lim
				if jjTotal < 1 {
					out := TrimEvologUserPrompt(strings.TrimSpace(acc.String()))
					return EvologSuggestPrepDoneMsg{ReqID: reqID, JJTotal: 0, UserPrompt: out}
				}
				nextLow = 1
			}
			batchEnd := min(nextLow+evologSplitStepDiffConcurrency-1, jjTotal)
			prepCtx, cancelPrep := context.WithTimeout(context.Background(), evologSuggestPrepBatchTimeout)
			defer cancelPrep()
			chunk, err := fetchEvologStepDiffLinesRange(prepCtx, jjSvc, entries, nextLow, batchEnd)
			if err != nil {
				return EvologSuggestPrepDoneMsg{ReqID: reqID, JJTotal: jjTotal, Err: err}
			}
			appendStepsToPrompt(&acc, entries, nextLow, batchEnd, chunk)
			if batchEnd == jjTotal && len(entries)-1 > evologSplitMaxDiffSteps {
				fmt.Fprintf(&acc, "\n(Only the first %d step summaries are included; you may still choose any valid index 1..%d using row descriptions.)\n", evologSplitMaxDiffSteps, len(entries)-1)
			}
			if batchEnd == jjTotal && cfg.EvologAIHunkPhaseEnabled() {
				hintCtx, hintCancel := context.WithTimeout(context.Background(), 90*time.Second)
				hint, herr := buildEvologHunkHintBlock(hintCtx, jjSvc, entries, jjTotal)
				hintCancel()
				if herr != nil {
					return EvologSuggestPrepDoneMsg{ReqID: reqID, JJTotal: jjTotal, Err: herr}
				}
				if strings.TrimSpace(hint) != "" {
					acc.WriteString(hint)
				}
			}
			if acc.Len() > evologSuggestMaxPromptBytes {
				return EvologSuggestPrepDoneMsg{ReqID: reqID, JJTotal: jjTotal, Err: fmt.Errorf("AI suggest: prompt exceeded internal size limit (%d bytes)", evologSuggestMaxPromptBytes)}
			}
			if batchEnd == jjTotal {
				out := TrimEvologUserPrompt(strings.TrimSpace(acc.String()))
				return EvologSuggestPrepDoneMsg{ReqID: reqID, JJTotal: jjTotal, UserPrompt: out}
			}
			nextLow = batchEnd + 1
			return tea.Sequence(
				func() tea.Msg {
					return EvologSuggestPrepProgressMsg{ReqID: reqID, JJDone: batchEnd, JJTotal: jjTotal}
				},
				func() tea.Msg { return step() },
			)()
		}
		return step()
	}
}

// EvologSuggestLLMCmd runs the LLM and post-parse validation after jj prep is finished.
func EvologSuggestLLMCmd(reqID int, jjSvc *jj.Service, cfg *config.Config, entries []jj.EvologEntry, userPrompt string) tea.Cmd {
	return func() tea.Msg {
		return runEvologSuggestLLM(reqID, jjSvc, cfg, entries, userPrompt)
	}
}

func runEvologSuggestLLM(reqID int, jjSvc *jj.Service, cfg *config.Config, entries []jj.EvologEntry, userPrompt string) EvologSplitSuggestMsg {
	msg := EvologSplitSuggestMsg{ReqID: reqID}
	if jjSvc == nil || cfg == nil || !cfg.AIConfiguredForGeneration() {
		msg.Err = fmt.Errorf("AI is disabled or no API key (Settings → Advanced, or %s)", config.EnvAIAPIKey)
		return msg
	}
	if len(entries) < 2 {
		msg.Err = fmt.Errorf("need at least two evolog rows")
		return msg
	}
	provider, err := llm.NewProviderForConfig(cfg)
	if err != nil {
		msg.Err = err
		return msg
	}
	llmCtx, cancelLLM := context.WithTimeout(context.Background(), cfg.AITimeout())
	defer cancelLLM()
	raw, err := provider.Complete(llmCtx, evologSplitSystem, userPrompt)
	if err != nil {
		msg.Err = err
		return msg
	}
	maxPick := len(entries) - 1
	maxParse := evologSplitMaxMultiIDsToParse(len(entries))
	res, err := parseEvologSplitJSON(raw, maxPick, entries, maxParse)
	if err != nil {
		msg.Err = err
		return msg
	}
	msg.NoSplit = res.NoSplit
	msg.PickIndex = res.PickIndex
	msg.Rationale = res.Rationale
	msg.FilesForFirstCommit = res.FilesForFirstCommit
	msg.HunkPrefixFirstCommit = res.HunkPrefixFirstCommit
	msg.MultiSplitBaseCommitIDs = res.MultiSplitBaseCommitIDs
	capMulti := cfg.EvologAIMultiSplitMaxCap()
	if capMulti < 1 {
		capMulti = 1
	}
	if rowBound := len(entries) - 1; rowBound >= 1 && capMulti > rowBound {
		capMulti = rowBound
	}
	if len(msg.MultiSplitBaseCommitIDs) > capMulti {
		msg.MultiSplitBaseCommitIDs = msg.MultiSplitBaseCommitIDs[:capMulti]
	}
	if !cfg.EvologAIHunkPhaseEnabled() {
		msg.HunkPrefixFirstCommit = nil
	}
	if !cfg.EvologAIFilePhaseEnabled() {
		msg.FilesForFirstCommit = nil
	}
	if len(msg.HunkPrefixFirstCommit) > 0 && len(msg.FilesForFirstCommit) > 0 {
		msg.FilesForFirstCommit = nil
		if strings.TrimSpace(msg.Rationale) != "" {
			msg.Rationale = msg.Rationale + " — file list ignored because hunk_prefix_first_commit is set"
		} else {
			msg.Rationale = "file list ignored because hunk_prefix_first_commit is set"
		}
	}
	if cfg.EvologAIHunkPhaseEnabled() && !res.NoSplit && len(msg.HunkPrefixFirstCommit) > 0 {
		valCtx, cancelVal := context.WithTimeout(context.Background(), evologSplitFileValidateTimeout)
		defer cancelVal()
		clean, note, herr := ValidateEvologHunkPrefixAgainstStep(valCtx, jjSvc, entries, res.PickIndex, msg.HunkPrefixFirstCommit)
		if herr != nil {
			msg.Err = herr
			return msg
		}
		msg.HunkPrefixFirstCommit = clean
		if note != "" {
			if strings.TrimSpace(msg.Rationale) != "" {
				msg.Rationale = msg.Rationale + " — " + note
			} else {
				msg.Rationale = note
			}
		}
	}
	if cfg.EvologAIFilePhaseEnabled() && !res.NoSplit && len(msg.FilesForFirstCommit) > 0 {
		valCtx, cancelVal := context.WithTimeout(context.Background(), evologSplitFileValidateTimeout)
		defer cancelVal()
		filtered, note, ferr := ValidateAndFilterEvologSplitFiles(valCtx, jjSvc, entries, res.PickIndex, msg.FilesForFirstCommit)
		if ferr != nil {
			msg.Err = ferr
			return msg
		}
		msg.FilesForFirstCommit = filtered
		if note != "" {
			if strings.TrimSpace(msg.Rationale) != "" {
				msg.Rationale = msg.Rationale + " — " + note
			} else {
				msg.Rationale = note
			}
		}
	}
	return msg
}

// TrimEvologUserPrompt applies the same rune cap as the monolithic prompt builder without allocating []rune(out)
// for the full string (that can freeze the UI thread on multi-megabyte prompts).
func TrimEvologUserPrompt(s string) string {
	out := strings.TrimSpace(s)
	const max = evologSplitMaxPromptRunes
	n := 0
	for range out {
		n++
		if n > max {
			return truncateEvologUserPromptRunes(out, max)
		}
	}
	return out
}

func truncateEvologUserPromptRunes(s string, max int) string {
	var b strings.Builder
	n := 0
	for _, r := range s {
		if n >= max {
			b.WriteString("\n…(truncated for size)")
			return b.String()
		}
		b.WriteRune(r)
		n++
	}
	return s
}

// buildEvologSplitPromptPrefix returns the static evolog user prompt (rows + "## Steps" header) and the step count.
func buildEvologSplitPromptPrefix(entries []jj.EvologEntry) (prefix string, stepLimit int, err error) {
	n := len(entries)
	if n < 2 {
		return "", 0, fmt.Errorf("need at least two evolog rows")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "N=%d evolog rows (newest first). Valid recommended_index: 0 (no split) or 1 through %d inclusive.\n\n", n, n-1)
	fmt.Fprintf(&b, "## Rows (use exact commit_id values for split_base_commit_ids)\n")
	for i := 0; i < n; i++ {
		e := entries[i]
		short := strings.TrimSpace(e.CommitIDShort)
		if short == "" && len(e.CommitID) >= 8 {
			short = e.CommitID[:8]
		}
		cid := strings.TrimSpace(e.CommitID)
		sum := strings.TrimSpace(e.Summary)
		if sum == "" {
			sum = "(empty)"
		}
		sum = strings.ReplaceAll(sum, "\n", " ")
		fmt.Fprintf(&b, "- index %d: commit_id=%s short=%s — %s\n", i, cid, short, sum)
	}
	b.WriteString("\n## Steps (jj diff --summary from row i to row i-1, newer neighbor above)\n")
	stepLimit = min(evologSplitMaxDiffSteps, n-1)
	return b.String(), stepLimit, nil
}

func appendStepsToPrompt(b *strings.Builder, entries []jj.EvologEntry, lo, hi int, stepLines [][]string) {
	for i := lo; i <= hi; i++ {
		lines := stepLines[i]
		fmt.Fprintf(b, "\n### Boundary after base row index %d (diff from that row up to the row above)\n", i)
		if len(lines) == 0 {
			b.WriteString("(no file changes in summary)\n")
			continue
		}
		for _, ln := range lines {
			b.WriteString(ln)
			b.WriteByte('\n')
		}
	}
}

// fetchEvologStepDiffLinesRange runs jj diff --summary for steps lo..hi inclusive (1-based indices).
func fetchEvologStepDiffLinesRange(ctx context.Context, jjSvc *jj.Service, entries []jj.EvologEntry, lo, hi int) ([][]string, error) {
	if lo < 1 || hi < lo {
		return nil, nil
	}
	out := make([][]string, hi+1)
	sem := make(chan struct{}, evologSplitStepDiffConcurrency)
	batchCtx, cancelBatch := context.WithCancel(ctx)
	defer cancelBatch()

	var wg sync.WaitGroup
	var errMu sync.Mutex
	var firstErr error
	setErr := func(err error) {
		if err == nil {
			return
		}
		errMu.Lock()
		if firstErr == nil {
			firstErr = err
			cancelBatch()
		}
		errMu.Unlock()
	}

	for i := lo; i <= hi; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-batchCtx.Done():
				return
			}
			defer func() { <-sem }()
			if batchCtx.Err() != nil {
				return
			}
			from := strings.TrimSpace(entries[i].CommitID)
			to := strings.TrimSpace(entries[i-1].CommitID)
			if from == "" || to == "" {
				return
			}
			stepCtx, cancelStep := context.WithTimeout(batchCtx, evologSplitStepDiffTimeout)
			defer cancelStep()
			lines, err := jjSvc.DiffSummaryLinesFromTo(stepCtx, from, to)
			if err != nil {
				setErr(fmt.Errorf("diff summary step %d: %w", i, err))
				return
			}
			errMu.Lock()
			if firstErr == nil {
				out[i] = lines
			}
			errMu.Unlock()
		}(i)
	}
	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// fetchEvologStepDiffLines runs jj diff --summary for each step i (from row i to row i-1) concurrently.
// Returns a slice indexed by step number (1..stepLimit); index 0 is unused.
func fetchEvologStepDiffLines(ctx context.Context, jjSvc *jj.Service, entries []jj.EvologEntry, stepLimit int) ([][]string, error) {
	return fetchEvologStepDiffLinesRange(ctx, jjSvc, entries, 1, stepLimit)
}

func buildEvologSplitUserPrompt(ctx context.Context, jjSvc *jj.Service, entries []jj.EvologEntry) (string, error) {
	prefix, stepLimit, err := buildEvologSplitPromptPrefix(entries)
	if err != nil {
		return "", err
	}
	stepLines, err := fetchEvologStepDiffLines(ctx, jjSvc, entries, stepLimit)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString(prefix)
	appendStepsToPrompt(&b, entries, 1, stepLimit, stepLines)
	n := len(entries)
	if n-1 > evologSplitMaxDiffSteps {
		fmt.Fprintf(&b, "\n(Only the first %d step summaries are included; you may still choose any valid index 1..%d using row descriptions.)\n", evologSplitMaxDiffSteps, n-1)
	}
	out := strings.TrimSpace(b.String())
	return TrimEvologUserPrompt(out), nil
}

type evologSplitJSON struct {
	NoSplit                 bool            `json:"no_split"`
	RecommendedIndex        int             `json:"recommended_index"`
	Rationale               string          `json:"rationale"`
	Confidence              string          `json:"confidence"`
	FilesFirstCommit        []string        `json:"files_first_commit"`
	HunkPrefixFirstCommit   map[string]int  `json:"hunk_prefix_first_commit"`
	SplitBaseCommitIDs      []string        `json:"split_base_commit_ids"`
}

// EvologSplitParseResult is the parsed LLM output for evolog split suggestions.
type EvologSplitParseResult struct {
	NoSplit                 bool
	PickIndex               int
	Rationale               string
	FilesForFirstCommit       []string
	HunkPrefixFirstCommit   map[string]int
	MultiSplitBaseCommitIDs []string
}

// normalizeEvologCommitID maps a model-supplied id (full or long unambiguous prefix) to the table's commit id.
func normalizeEvologCommitID(id string, entries []jj.EvologEntry) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	var match string
	for _, e := range entries {
		ec := strings.TrimSpace(e.CommitID)
		if ec == "" {
			continue
		}
		if ec == id {
			return ec
		}
		if len(id) >= 8 && strings.HasPrefix(ec, id) {
			if match != "" {
				return ""
			}
			match = ec
		}
	}
	return match
}

func parseEvologSplitJSON(raw string, maxPick int, entries []jj.EvologEntry, maxMultiIDs int) (EvologSplitParseResult, error) {
	var zero EvologSplitParseResult
	s := strings.TrimSpace(raw)
	if s == "" {
		return zero, fmt.Errorf("empty model response")
	}
	if i := strings.Index(s, "{"); i >= 0 {
		if j := strings.LastIndex(s, "}"); j > i {
			s = s[i : j+1]
		}
	}
	var parsed evologSplitJSON
	if err := json.Unmarshal([]byte(s), &parsed); err != nil {
		return zero, fmt.Errorf("parse JSON: %w", err)
	}
	noSplit := parsed.NoSplit || parsed.RecommendedIndex == 0
	if noSplit {
		r := strings.TrimSpace(parsed.Rationale)
		if r == "" {
			r = "(no rationale)"
		}
		if c := strings.TrimSpace(parsed.Confidence); c != "" {
			r = r + " [" + c + "]"
		}
		return EvologSplitParseResult{NoSplit: true, Rationale: r}, nil
	}
	idx := parsed.RecommendedIndex
	if idx < 1 || idx > maxPick {
		return zero, fmt.Errorf("recommended_index %d out of range (need 0 for no split or 1..%d)", parsed.RecommendedIndex, maxPick)
	}
	r := strings.TrimSpace(parsed.Rationale)
	if r == "" {
		r = "(no rationale)"
	}
	if c := strings.TrimSpace(parsed.Confidence); c != "" {
		r = r + " [" + c + "]"
	}
	if maxMultiIDs < 1 {
		maxMultiIDs = 1
	}
	var multi []string
	for _, id := range parsed.SplitBaseCommitIDs {
		canon := normalizeEvologCommitID(id, entries)
		if canon == "" {
			return zero, fmt.Errorf("unknown split_base_commit_id %q (must match evolog row)", strings.TrimSpace(id))
		}
		multi = append(multi, canon)
		if len(multi) >= maxMultiIDs {
			break
		}
	}
	if len(multi) == 0 {
		multi = []string{strings.TrimSpace(entries[idx].CommitID)}
	}
	var files []string
	for _, p := range parsed.FilesFirstCommit {
		p = strings.TrimSpace(p)
		if p != "" {
			files = append(files, p)
		}
	}
	hunkPrefix := make(map[string]int)
	for k, v := range parsed.HunkPrefixFirstCommit {
		kk := normalizeRepoPathForDiff(k)
		if kk != "" {
			hunkPrefix[kk] = v
		}
	}
	return EvologSplitParseResult{
		NoSplit:                 false,
		PickIndex:               idx,
		Rationale:               r,
		FilesForFirstCommit:     files,
		HunkPrefixFirstCommit:   hunkPrefix,
		MultiSplitBaseCommitIDs: multi,
	}, nil
}
