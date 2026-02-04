package actions

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/jj-tui/internal/jj"
)

// NewCommit creates a new commit as a child of the given parent
func NewCommit(svc *jj.Service, parentCommitID string) tea.Cmd {
	return func() tea.Msg {
		if err := svc.NewCommit(context.Background(), parentCommitID); err != nil {
			return ErrorMsg{Err: fmt.Errorf("failed to create commit: %w", err)}
		}
		repo, err := svc.GetRepository(context.Background())
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return RepositoryLoadedMsg{Repository: repo}
	}
}

// Checkout checks out (edits) the specified commit
func Checkout(svc *jj.Service, changeID string) tea.Cmd {
	return func() tea.Msg {
		if err := svc.CheckoutCommit(context.Background(), changeID); err != nil {
			return ErrorMsg{Err: fmt.Errorf("failed to checkout: %w", err)}
		}
		repo, err := svc.GetRepository(context.Background())
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return EditCompletedMsg{Repository: repo}
	}
}

// Squash squashes the specified commit into its parent
func Squash(svc *jj.Service, changeID string) tea.Cmd {
	return func() tea.Msg {
		if err := svc.SquashCommit(context.Background(), changeID); err != nil {
			return ErrorMsg{Err: fmt.Errorf("failed to squash: %w", err)}
		}
		repo, err := svc.GetRepository(context.Background())
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return RepositoryLoadedMsg{Repository: repo}
	}
}

// Abandon abandons the specified commit
func Abandon(svc *jj.Service, changeID string) tea.Cmd {
	return func() tea.Msg {
		if err := svc.AbandonCommit(context.Background(), changeID); err != nil {
			return ErrorMsg{Err: fmt.Errorf("failed to abandon: %w", err)}
		}
		repo, err := svc.GetRepository(context.Background())
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return RepositoryLoadedMsg{Repository: repo}
	}
}

// Rebase rebases the source commit onto the destination
func Rebase(svc *jj.Service, sourceChangeID, destChangeID string) tea.Cmd {
	return func() tea.Msg {
		if err := svc.RebaseCommit(context.Background(), sourceChangeID, destChangeID); err != nil {
			return ErrorMsg{Err: fmt.Errorf("failed to rebase: %w", err)}
		}
		repo, err := svc.GetRepository(context.Background())
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return RepositoryLoadedMsg{Repository: repo}
	}
}

