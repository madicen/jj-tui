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
		diff, err := jjSvc.GitFormatDiffFromTo(ctx, fromRev, changeID, diffBytesPR)
		if err != nil {
			diff, err = jjSvc.GitFormatDiffForRevision(ctx, changeID, diffBytesPR)
			if err != nil {
				msg.Err = fmt.Errorf("diff: %w", err)
				return msg
			}
		}
		provider, err := llm.NewProviderForConfig(cfg)
		if err != nil {
			msg.Err = err
			return msg
		}
		out, err := provider.Complete(ctx, aiprompts.PRSystem, aiprompts.PRUser(baseBranch, headBranch, hintTitle, diff))
		if err != nil {
			msg.Err = err
			return msg
		}
		msg.Title, msg.Body = aiprompts.ParsePRTitleBody(out)
		msg.Title = aiprompts.MergeGeneratedPRTitle(hintTitle, msg.Title)
		return msg
	}
}

// GenerateTicketFormCmd generates issue title and body from a revision's diff (ticket the change would close).
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
		diff, err := jjSvc.GitFormatDiffForRevision(ctx, rev, diffBytesTicket)
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
		out, err := provider.Complete(ctx, aiprompts.TicketSystem, aiprompts.TicketUser(short, hintSummary, hintDescription, diff))
		if err != nil {
			msg.Err = err
			return msg
		}
		msg.Title, msg.Body = aiprompts.ParsePRTitleBody(out)
		msg.Title = aiprompts.MergeGeneratedPRTitle(hintSummary, msg.Title)
		return msg
	}
}

// GenerateBookmarkNameCmd suggests a bookmark name from the working-copy parent's diff or the given revision.
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
		diff, err := jjSvc.GitFormatDiffForRevision(ctx, rev, diffBytesBookmark)
		if err != nil {
			msg.Err = fmt.Errorf("diff: %w", err)
			return msg
		}
		provider, err := llm.NewProviderForConfig(cfg)
		if err != nil {
			msg.Err = err
			return msg
		}
		out, err := provider.Complete(ctx, aiprompts.BookmarkSystem, aiprompts.BookmarkUser(ticketHint, diff))
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
