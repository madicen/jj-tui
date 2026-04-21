package ai

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/config"
	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/integrations/llm"
)

const diffBytesCommit = 120_000
const diffBytesPR = 120_000
const diffBytesBookmark = 80_000

// GenerateCommitDescriptionCmd runs the LLM and returns TextGeneratedMsg.
func GenerateCommitDescriptionCmd(reqID int, jjSvc *jj.Service, cfg *config.Config, commitID, changeShort, currentDesc string) tea.Cmd {
	return func() tea.Msg {
		msg := TextGeneratedMsg{ReqID: reqID, Kind: KindCommitDescription, CommitID: commitID}
		if jjSvc == nil || cfg == nil || !cfg.AIConfiguredForGeneration() {
			msg.Err = fmt.Errorf("AI is disabled or no API key (Settings → Advanced, or %s)", config.EnvAIAPIKey)
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
		out, err := provider.Complete(ctx, CommitDescriptionSystem, CommitDescriptionUser(changeShort, currentDesc, diff))
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
			msg.Err = fmt.Errorf("AI is disabled or no API key (Settings → Advanced, or %s)", config.EnvAIAPIKey)
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
		out, err := provider.Complete(ctx, prSystem, PRUser(baseBranch, headBranch, hintTitle, diff))
		if err != nil {
			msg.Err = err
			return msg
		}
		msg.Title, msg.Body = ParsePRTitleBody(out)
		return msg
	}
}

// GenerateBookmarkNameCmd suggests a bookmark name from the working-copy parent's diff or the given revision.
func GenerateBookmarkNameCmd(reqID int, jjSvc *jj.Service, cfg *config.Config, revision, ticketHint string) tea.Cmd {
	return func() tea.Msg {
		msg := TextGeneratedMsg{ReqID: reqID, Kind: KindBookmark}
		if jjSvc == nil || cfg == nil || !cfg.AIConfiguredForGeneration() {
			msg.Err = fmt.Errorf("AI is disabled or no API key (Settings → Advanced, or %s)", config.EnvAIAPIKey)
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
		out, err := provider.Complete(ctx, bookmarkSystem, BookmarkUser(ticketHint, diff))
		if err != nil {
			msg.Err = err
			return msg
		}
		line := strings.TrimSpace(strings.Split(out, "\n")[0])
		msg.Text = line
		return msg
	}
}
