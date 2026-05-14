package ai

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/integrations/llm"
	"github.com/madicen/jj-tui/internal/tui/aiprompts"
)

const diffBytesCommit = 120_000
const diffBytesPR = 120_000
const diffBytesBookmark = 80_000
const diffBytesTicket = 120_000

// chainContext is the result of trying to enrich an AI prompt with the full
// chain of commits between baseRev and toRev (instead of only the tip's
// vs-parents diff). diff is what should be passed to the prompt builder;
// chainSummary is "" when the chain is empty / could not be loaded, in which
// case the diff falls back to the single-revision diff and the prompt
// behaves like the old per-commit version.
type chainContext struct {
	diff         string
	chainSummary string
}

// loadChainContext returns the cumulative diff baseRev → toRev plus a formatted
// list of the commits in `baseRev..toRev` for inlining in an AI prompt. It
// degrades gracefully:
//
//  1. If listing chain commits succeeds AND returns at least one commit, the
//     cumulative diff and a chain summary are used. This is the new behavior
//     the user requested: the AI sees the whole stack, not just the tip.
//  2. If the chain is empty (toRev is at or below baseRev — e.g. the selected
//     commit IS trunk, or trunk is ahead) OR listing fails (no `trunk()`,
//     missing remote), we fall back to the single-revision diff with no chain
//     summary. The cumulative-diff path is also retried as a single-rev diff if
//     it errors, so a misconfigured `trunk()` never blocks generation.
//
// maxBytes applies to whichever diff path is ultimately used.
func loadChainContext(ctx context.Context, jjSvc *jj.Service, baseRev, toRev string, maxBytes int) (chainContext, error) {
	toRev = strings.TrimSpace(toRev)
	if toRev == "" {
		toRev = "@"
	}
	baseRev = strings.TrimSpace(baseRev)

	// Without a usable base we can't form a chain at all; return the single-rev diff.
	if baseRev == "" {
		diff, err := jjSvc.GitFormatDiffForRevision(ctx, toRev, maxBytes)
		return chainContext{diff: diff}, err
	}

	commits, listErr := jjSvc.ListChainCommits(ctx, baseRev, toRev)
	if listErr != nil || len(commits) == 0 {
		diff, err := jjSvc.GitFormatDiffForRevision(ctx, toRev, maxBytes)
		return chainContext{diff: diff}, err
	}

	cumulativeDiff, diffErr := jjSvc.GitFormatDiffFromTo(ctx, baseRev, toRev, maxBytes)
	if diffErr != nil {
		// Cumulative path broke despite a non-empty chain (rare: e.g. baseRev
		// resolves but jj refuses --from/--to for it). Fall back to single-rev
		// and drop the chain summary so the prompt's wording stays accurate.
		diff, err := jjSvc.GitFormatDiffForRevision(ctx, toRev, maxBytes)
		return chainContext{diff: diff}, err
	}

	summaries := make([]aiprompts.ChainCommitSummary, 0, len(commits))
	for _, c := range commits {
		summaries = append(summaries, aiprompts.ChainCommitSummary{
			ChangeIDShort: c.ChangeIDShort,
			Subject:       c.Subject,
			Description:   c.Description,
		})
	}
	return chainContext{
		diff:         cumulativeDiff,
		chainSummary: aiprompts.FormatChainSummary(summaries),
	}, nil
}

// GenerateCommitDescriptionCmd runs the LLM and returns TextGeneratedMsg.
func GenerateCommitDescriptionCmd(reqID int, jjSvc *jj.Service, cfg *config.Config, commitID, changeShort, currentDesc string) tea.Cmd {
	return func() tea.Msg {
		msg := TextGeneratedMsg{ReqID: reqID, Kind: KindCommitDescription, CommitID: commitID}
		if jjSvc == nil || cfg == nil || !cfg.AIConfiguredForGeneration() {
			msg.Err = fmt.Errorf("AI is disabled or no API key (Settings → AI, or %s)", config.EnvAIAPIKey)
			return msg
		}
		ctx, cancel := context.WithTimeout(context.Background(), cfg.AITimeout())
		defer cancel()
		diff, err := jjSvc.GitFormatDiffForRevision(ctx, commitID, diffBytesCommit)
		if err != nil {
			msg.Err = fmt.Errorf("diff: %w", err)
			return msg
		}
		provider, err := llm.NewProviderForConfig(cfg)
		if err != nil {
			msg.Err = err
			return msg
		}
		out, err := provider.Complete(ctx, aiprompts.CommitDescriptionSystem, aiprompts.CommitDescriptionUser(changeShort, currentDesc, diff))
		if err != nil {
			msg.Err = err
			return msg
		}
		msg.Text = out
		return msg
	}
}

// GeneratePRFormCmd generates PR title and body.
func GeneratePRFormCmd(reqID int, jjSvc *jj.Service, cfg *config.Config, changeID, baseBranch, headBranch, hintTitle string) tea.Cmd {
	return func() tea.Msg {
		msg := TextGeneratedMsg{ReqID: reqID, Kind: KindPR, CommitID: changeID}
		if jjSvc == nil || cfg == nil || !cfg.AIConfiguredForGeneration() {
			msg.Err = fmt.Errorf("AI is disabled or no API key (Settings → AI, or %s)", config.EnvAIAPIKey)
			return msg
		}
		ctx, cancel := context.WithTimeout(context.Background(), cfg.AITimeout())
		defer cancel()
		fromRev := strings.TrimSpace(baseBranch) + "@origin"
		chain, err := loadChainContext(ctx, jjSvc, fromRev, changeID, diffBytesPR)
		if err != nil {
			msg.Err = fmt.Errorf("diff: %w", err)
			return msg
		}
		provider, err := llm.NewProviderForConfig(cfg)
		if err != nil {
			msg.Err = err
			return msg
		}
		out, err := provider.Complete(ctx, aiprompts.PRSystem, aiprompts.PRUser(baseBranch, headBranch, hintTitle, chain.chainSummary, chain.diff))
		if err != nil {
			msg.Err = err
			return msg
		}
		msg.Title, msg.Body = aiprompts.ParsePRTitleBody(out)
		msg.Title = aiprompts.MergeGeneratedPRTitle(hintTitle, msg.Title)
		return msg
	}
}

// GenerateTicketFormCmd generates issue title and body from the chain of
// commits up to the selected revision (the work the ticket would close).
//
// When the selected revision is above trunk(), we send the cumulative
// trunk → revision diff plus a summary of every commit in the chain so the
// AI describes the whole stack. When the chain is empty (or trunk() can't
// be resolved), we fall back to the single-revision diff.
func GenerateTicketFormCmd(reqID int, jjSvc *jj.Service, cfg *config.Config, changeID, changeShort, hintSummary, hintDescription string) tea.Cmd {
	return func() tea.Msg {
		msg := TextGeneratedMsg{ReqID: reqID, Kind: KindTicket, CommitID: changeID}
		if jjSvc == nil || cfg == nil || !cfg.AIConfiguredForGeneration() {
			msg.Err = fmt.Errorf("AI is disabled or no API key (Settings → AI, or %s)", config.EnvAIAPIKey)
			return msg
		}
		ctx, cancel := context.WithTimeout(context.Background(), cfg.AITimeout())
		defer cancel()
		rev := strings.TrimSpace(changeID)
		if rev == "" {
			rev = "@"
		}
		chain, err := loadChainContext(ctx, jjSvc, "trunk()", rev, diffBytesTicket)
		if err != nil {
			msg.Err = fmt.Errorf("diff: %w", err)
			return msg
		}
		provider, err := llm.NewProviderForConfig(cfg)
		if err != nil {
			msg.Err = err
			return msg
		}
		short := strings.TrimSpace(changeShort)
		if short == "" {
			short = rev
		}
		out, err := provider.Complete(ctx, aiprompts.TicketSystem, aiprompts.TicketUser(short, hintSummary, hintDescription, chain.chainSummary, chain.diff))
		if err != nil {
			msg.Err = err
			return msg
		}
		msg.Title, msg.Body = aiprompts.ParsePRTitleBody(out)
		msg.Title = aiprompts.MergeGeneratedPRTitle(hintSummary, msg.Title)
		return msg
	}
}

// GenerateBookmarkNameCmd suggests a bookmark name from the chain of commits
// `trunk()..revision` (oldest → newest), so the suggested name reflects the
// whole stack of work rather than only the tip commit's local changes. Falls
// back to the single-revision diff when the chain is empty (e.g. the selected
// commit IS trunk) or when trunk() can't be resolved.
func GenerateBookmarkNameCmd(reqID int, jjSvc *jj.Service, cfg *config.Config, revision, ticketHint string) tea.Cmd {
	return func() tea.Msg {
		msg := TextGeneratedMsg{ReqID: reqID, Kind: KindBookmark}
		if jjSvc == nil || cfg == nil || !cfg.AIConfiguredForGeneration() {
			msg.Err = fmt.Errorf("AI is disabled or no API key (Settings → AI, or %s)", config.EnvAIAPIKey)
			return msg
		}
		ctx, cancel := context.WithTimeout(context.Background(), cfg.AITimeout())
		defer cancel()
		rev := strings.TrimSpace(revision)
		if rev == "" {
			rev = "@"
		}
		chain, err := loadChainContext(ctx, jjSvc, "trunk()", rev, diffBytesBookmark)
		if err != nil {
			msg.Err = fmt.Errorf("diff: %w", err)
			return msg
		}
		provider, err := llm.NewProviderForConfig(cfg)
		if err != nil {
			msg.Err = err
			return msg
		}
		out, err := provider.Complete(ctx, aiprompts.BookmarkSystem, aiprompts.BookmarkUser(ticketHint, chain.chainSummary, chain.diff))
		if err != nil {
			msg.Err = err
			return msg
		}
		// The bookmark prompt advertises a 50-char ceiling, but local models routinely
		// ignore it and emit two-paragraph branch names. Cap here so the suggestion the
		// user sees in the input field is already operationally sane; SubmitCmd also
		// re-applies the cap as a backstop for any other path that feeds SetBookmarkName.
		line := strings.TrimSpace(strings.Split(out, "\n")[0])
		line = jj.TruncateBookmarkName(line)
		msg.Text = line
		return msg
	}
}
