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
	// HunkPrefixFirstCommit maps path → first k hunks (single peel); ignored when HunkPeelRounds is set.
	HunkPrefixFirstCommit map[string]int
	// HunkPeelRounds is an ordered list of peels (each map is one jj split on @); use for full multi-commit partition.
	HunkPeelRounds []map[string]int
	// MultiSplitBaseCommitIDs is an ordered deepest-first list of base commit ids for sequential FAQ splits (length capped by settings and evolog row count).
	MultiSplitBaseCommitIDs []string
	Err                     error
}

// evologSplitMaxHunkPeelRounds caps how many jj split rounds we run from one AI suggestion (safety).
const evologSplitMaxHunkPeelRounds = 64

// evologSplitMaxDiffSteps limits jj diff --summary sections in the AI user prompt (each step is several lines).
const evologSplitMaxDiffSteps = 120

// evologSplitMaxPromptRunes caps the user prompt sent to the LLM. Large values improve “split everything”
// suggestions on deep evologs; very large strings are trimmed on the prep goroutine (not the UI thread).
const evologSplitMaxPromptRunes = 48_000

// evologSplitStepDiffConcurrency limits parallel jj processes while building step summaries.
const evologSplitStepDiffConcurrency = 8

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

Your job is to propose a COMPLETE automation plan when the working copy or history mixes several logical commits: (1) zero or more FAQ-style row moves via split_base_commit_ids (deepest-first), (2) zero or more hunk-scoped jj splits on @ via hunk_peel_rounds (or a single hunk_prefix_first_commit), (3) optional file paths if hunks are not available. Do not minimize the plan to "one peel" when multiple peels are genuinely needed — the user can confirm once and run the whole sequence.

You receive each row's short id, full commit id, and first-line description, plus for some rows a jj diff --summary for the step from row i to row i-1 (only the first many steps may appear when N is large). When hunk excerpts are included, they are git unified diffs for early steps — use them to count hunks per path (0-based in order of @@ blocks for that path). If the message ends with a truncation note, treat missing older steps as unknown detail but still use the full "## Rows" list: when N is large or summaries show stacked unrelated work, propose as many split_base_commit_ids as are justified (up to the client cap), not a single minimal peel.

Reply with a single JSON object only (no markdown fences, no commentary):
{"no_split": <bool>, "recommended_index": <int>, "rationale": "<short plain text>", "confidence": "high"|"medium"|"low",
 "files_first_commit": ["<path>", ...],
 "hunk_prefix_first_commit": {"<path>": <int>, ...},
 "hunk_peel_rounds": [{"<path>": <int>, ...}, ...],
 "split_base_commit_ids": ["<full commit id from table>", ...]}

Rules:
- If no split is warranted (e.g. noise-only evolog, or one coherent change), set "no_split": true and "recommended_index": 0. Still provide rationale.
- If you recommend work to separate, set "no_split": false and set "recommended_index" to an integer from 1 through N-1 (N = number of evolog rows). Use the shallowest row index among the FAQ bases you intend (closest to tip / smallest index) so the UI previews the first separation from the tip; split_base_commit_ids still lists every FAQ base when more than one FAQ move is needed.
- "split_base_commit_ids": array of commit_id values copied EXACTLY from the evolog rows' "## Rows" list. When multiple FAQ peels are needed to untangle stacked work, list EVERY base id required, ordered deepest-first (first id = largest row index among your bases / oldest separation, last id = smallest row index among your bases / closest to tip). A one-element array is only for a single FAQ move. You may list up to N-1 ids when warranted. If you omit the field or send [], the client defaults to one FAQ base derived from recommended_index.
- "files_first_commit": optional list of repo-relative paths for ONE extra peel of the working copy after FAQ step(s), using only paths that appear in the diff for the chosen recommended_index step. Paths must be a strict subset of changed paths: NEVER list every file in that step (at least one changed path must stay on @). Omit or [] if not needed. Do not use together with hunk fields when hunks give a finer plan — the UI prefers hunks and will ignore file lists if both are set.
- "hunk_prefix_first_commit": optional object for a SINGLE hunk peel after FAQ (same semantics as one element of hunk_peel_rounds). Prefer "hunk_peel_rounds" whenever more than one jj split on @ is needed to finish partitioning.
- "hunk_peel_rounds": optional array of objects; each object maps path → k for ONE jj split on @ after all FAQ step(s). Round 1 uses the same path/k semantics as hunk_prefix_first_commit against the current @ vs @- diff. After each split, @ has a smaller diff; round 2's k values apply to THAT new diff (re-count @@ hunks from the updated working tree). To fully partition into G commits from the current tree slice, use G-1 rounds; each round must be a strict proper subset of the diff at that moment (every path you mention: 0 < k < current hunk count for that path). The final @ holds the last segment (do not add a round that would peel every remaining hunk). Omit or [] when no hunk peel is needed. Do not send both hunk_peel_rounds and hunk_prefix_first_commit — if both appear, hunk_peel_rounds wins.
`

func appendEvologSplitAutomationHint(b *strings.Builder, cfg *config.Config) {
	if b == nil {
		return
	}
	mode := "batch — all split_base_commit_ids in one user confirm, then all hunk_peel_rounds in order."
	if cfg != nil && cfg.EvologAIMultiSplitStepwise() {
		mode = "stepwise — one split_base_commit_id per user confirm; hunk_peel_rounds run after the last FAQ confirm."
	}
	maxBases := "evolog depth and Settings → Advanced (AI evolog multi-split max)"
	if cfg != nil {
		maxBases = fmt.Sprintf("min(evolog depth−1, %d) from Settings → Advanced", cfg.EvologAIMultiSplitMaxCap())
	}
	b.WriteString("\n## Client automation (for planning only; do not echo this heading in your JSON)\n")
	fmt.Fprintf(b, "- FAQ bases: %s\n", mode)
	b.WriteString("- Execution order: run every split_base_commit_id in array order (deepest first), then each hunk_peel_rounds map as its own jj split on @.\n")
	fmt.Fprintf(b, "- Limits: %s FAQ ids; up to %d hunk peel rounds per suggestion.\n", maxBases, evologSplitMaxHunkPeelRounds)
	b.WriteString("- Prefer returning the full split_base_commit_ids and hunk_peel_rounds arrays in one response whenever multiple steps are correct — do not collapse to a single step out of convenience.\n")
}

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
			if batchEnd == jjTotal {
				appendEvologSplitAutomationHint(&acc, cfg)
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
	msg.HunkPeelRounds = res.HunkPeelRounds
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
		msg.HunkPeelRounds = nil
	}
	if !cfg.EvologAIFilePhaseEnabled() {
		msg.FilesForFirstCommit = nil
	}
	if len(msg.HunkPeelRounds) == 0 && len(msg.HunkPrefixFirstCommit) > 0 {
		one := make(map[string]int, len(msg.HunkPrefixFirstCommit))
		for k, v := range msg.HunkPrefixFirstCommit {
			one[k] = v
		}
		msg.HunkPeelRounds = []map[string]int{one}
	}
	if len(msg.HunkPeelRounds) > 0 {
		msg.HunkPrefixFirstCommit = nil
		if len(msg.FilesForFirstCommit) > 0 {
			msg.FilesForFirstCommit = nil
			if strings.TrimSpace(msg.Rationale) != "" {
				msg.Rationale = msg.Rationale + " — file list ignored because hunk peel plan is set"
			} else {
				msg.Rationale = "file list ignored because hunk peel plan is set"
			}
		}
	}
	if cfg.EvologAIHunkPhaseEnabled() && !res.NoSplit && len(msg.HunkPeelRounds) > 0 {
		valCtx, cancelVal := context.WithTimeout(context.Background(), evologSplitFileValidateTimeout)
		defer cancelVal()
		clean, note, herr := ValidateEvologHunkPrefixAgainstStep(valCtx, jjSvc, entries, res.PickIndex, msg.HunkPeelRounds[0])
		if herr != nil {
			msg.Err = herr
			return msg
		}
		msg.HunkPeelRounds[0] = clean
		if note != "" {
			if strings.TrimSpace(msg.Rationale) != "" {
				msg.Rationale = msg.Rationale + " — " + note
			} else {
				msg.Rationale = note
			}
		}
		if res.HunkPeelRoundsTruncated {
			if strings.TrimSpace(msg.Rationale) != "" {
				msg.Rationale = msg.Rationale + fmt.Sprintf(" — hunk_peel_rounds truncated to %d rounds", evologSplitMaxHunkPeelRounds)
			} else {
				msg.Rationale = fmt.Sprintf("hunk_peel_rounds truncated to %d rounds", evologSplitMaxHunkPeelRounds)
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
			b.WriteString("\n(AI: early sections are preserved; tail may be missing. Use full ## Rows; prefer broad split_base_commit_ids + hunk_peel_rounds when the change is mixed.)\n")
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

type evologSplitJSON struct {
	NoSplit               bool             `json:"no_split"`
	RecommendedIndex      int              `json:"recommended_index"`
	Rationale             string           `json:"rationale"`
	Confidence            string           `json:"confidence"`
	FilesFirstCommit      []string         `json:"files_first_commit"`
	HunkPrefixFirstCommit map[string]int   `json:"hunk_prefix_first_commit"`
	HunkPeelRounds        []map[string]int `json:"hunk_peel_rounds"`
	SplitBaseCommitIDs    []string         `json:"split_base_commit_ids"`
}

// EvologSplitParseResult is the parsed LLM output for evolog split suggestions.
type EvologSplitParseResult struct {
	NoSplit                 bool
	PickIndex               int
	Rationale               string
	FilesForFirstCommit     []string
	HunkPrefixFirstCommit   map[string]int
	HunkPeelRounds          []map[string]int
	HunkPeelRoundsTruncated bool // true when hunk_peel_rounds exceeded evologSplitMaxHunkPeelRounds before cap
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

func capAndNormalizeHunkPeelRounds(list []map[string]int) []map[string]int {
	if len(list) == 0 {
		return nil
	}
	if len(list) > evologSplitMaxHunkPeelRounds {
		list = list[:evologSplitMaxHunkPeelRounds]
	}
	var out []map[string]int
	for _, m := range list {
		if len(m) == 0 {
			continue
		}
		nm := make(map[string]int)
		for k, v := range m {
			kk := normalizeRepoPathForDiff(k)
			if kk == "" || v <= 0 {
				continue
			}
			nm[kk] = v
		}
		if len(nm) == 0 {
			continue
		}
		out = append(out, nm)
	}
	if len(out) == 0 {
		return nil
	}
	return out
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
	truncated := len(parsed.HunkPeelRounds) > evologSplitMaxHunkPeelRounds
	peelRounds := capAndNormalizeHunkPeelRounds(parsed.HunkPeelRounds)
	if len(peelRounds) > 0 {
		hunkPrefix = nil
	}
	return EvologSplitParseResult{
		NoSplit:                 false,
		PickIndex:               idx,
		Rationale:               r,
		FilesForFirstCommit:     files,
		HunkPrefixFirstCommit:   hunkPrefix,
		HunkPeelRounds:          peelRounds,
		HunkPeelRoundsTruncated: truncated,
		MultiSplitBaseCommitIDs: multi,
	}, nil
}
