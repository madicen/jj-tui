// Package tui provides the terminal user interface for jj-tui.
// This file re-exports types from the internal model package for backward compatibility.
package tui

import (
	"context"

	"github.com/madicen/jj-tui/internal/github"
	"github.com/madicen/jj-tui/internal/jj"
	"github.com/madicen/jj-tui/internal/tui/model"
)

// Model is the main TUI model - re-exported from model package
type Model = model.Model

// ViewMode constants - re-exported from model package
const (
	ViewCommitGraph     = model.ViewCommitGraph
	ViewPullRequests    = model.ViewPullRequests
	ViewJira            = model.ViewTickets
	ViewSettings        = model.ViewSettings
	ViewHelp            = model.ViewHelp
	ViewEditDescription = model.ViewEditDescription
	ViewCreatePR        = model.ViewCreatePR
	ViewCreateBookmark  = model.ViewCreateBookmark
	ViewGitHubLogin     = model.ViewGitHubLogin
)

// SelectionMode constants - re-exported from model package
const (
	SelectionNormal            = model.SelectionNormal
	SelectionRebaseDestination = model.SelectionRebaseDestination
)

// New creates a new Model - re-exported from model package
func New(ctx context.Context) *Model {
	return model.New(ctx)
}

// NewWithServices creates a new Model with pre-configured services
func NewWithServices(ctx context.Context, jjSvc *jj.Service, ghSvc *github.Service) *Model {
	return model.NewWithServices(ctx, jjSvc, ghSvc)
}

// NewDemo creates a new Model in demo mode with mock services
// This is used for VHS screenshots and visual testing
func NewDemo(ctx context.Context) *Model {
	return model.NewDemo(ctx)
}

// ErrorMsg creates an error message for testing purposes
func ErrorMsg(err error) model.ErrorMsgType {
	return model.ErrorMsg(err)
}

// Message types for external use
type (
	TabSelectedMsg = model.TabSelectedMsg
	ActionMsg      = model.ActionMsg
)

// ActionType constants
const (
	ActionQuit     = model.ActionQuit
	ActionRefresh  = model.ActionRefresh
	ActionNewPR    = model.ActionNewPR
	ActionCheckout = model.ActionCheckout
	ActionEdit     = model.ActionEdit
)
