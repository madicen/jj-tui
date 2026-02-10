package models

import (
	"time"
)

// Commit represents a jujutsu commit
type Commit struct {
	ID          string    `json:"id"`
	ShortID     string    `json:"short_id"`
	ChangeID    string    `json:"change_id"`
	Author      string    `json:"author"`
	Email       string    `json:"email"`
	Date        time.Time `json:"date"`
	Summary     string    `json:"summary"`
	Description string    `json:"description"`
	Parents     []string  `json:"parents"`
	Children    []string  `json:"children"`
	Branches    []string  `json:"branches"`
	Tags        []string  `json:"tags"`
	IsWorking   bool      `json:"is_working"`
	Conflicts   bool      `json:"conflicts"`
	Immutable   bool      `json:"immutable"`
	GraphPrefix string    `json:"graph_prefix"` // ASCII art graph prefix from jj (e.g., "│ ○  ")
	GraphLines  []string  `json:"graph_lines"`  // Connector lines after this commit (e.g., ["│", "├─╯"])
}

// CommitGraph represents the visual structure of commits
type CommitGraph struct {
	Commits     []Commit           `json:"commits"`
	Connections map[string][]string `json:"connections"` // commit_id -> connected_commit_ids
}

// CheckStatus represents the CI check status of a PR
type CheckStatus string

const (
	CheckStatusPending CheckStatus = "pending"
	CheckStatusSuccess CheckStatus = "success"
	CheckStatusFailure CheckStatus = "failure"
	CheckStatusNone    CheckStatus = "none"
)

// ReviewStatus represents the review status of a PR
type ReviewStatus string

const (
	ReviewStatusApproved         ReviewStatus = "approved"
	ReviewStatusChangesRequested ReviewStatus = "changes_requested"
	ReviewStatusPending          ReviewStatus = "pending"
	ReviewStatusNone             ReviewStatus = "none"
)

// GitHubPR represents a GitHub Pull Request
type GitHubPR struct {
	Number       int          `json:"number"`
	Title        string       `json:"title"`
	Body         string       `json:"body"`
	URL          string       `json:"url"`
	State        string       `json:"state"`
	BaseBranch   string       `json:"base_branch"`
	HeadBranch   string       `json:"head_branch"`
	CommitIDs    []string     `json:"commit_ids"`
	CheckStatus  CheckStatus  `json:"check_status"`  // CI check status
	ReviewStatus ReviewStatus `json:"review_status"` // Review status
}

// Repository represents the current jj repository state
type Repository struct {
	Path        string      `json:"path"`
	WorkingCopy Commit      `json:"working_copy"`
	Graph       CommitGraph `json:"graph"`
	PRs         []GitHubPR  `json:"prs"`
}

// CreatePRRequest represents a request to create a pull request
type CreatePRRequest struct {
	Title      string   `json:"title"`
	Body       string   `json:"body"`
	HeadBranch string   `json:"head_branch"`
	BaseBranch string   `json:"base_branch"`
	CommitIDs  []string `json:"commit_ids"`
	Draft      bool     `json:"draft"`
}

// UpdatePRRequest represents a request to update a pull request
type UpdatePRRequest struct {
	Title     string   `json:"title,omitempty"`
	Body      string   `json:"body,omitempty"`
	CommitIDs []string `json:"commit_ids,omitempty"`
}

// Branch represents a git branch/bookmark
type Branch struct {
	Name         string `json:"name"`          // Branch name (e.g., "main", "feature-x")
	Remote       string `json:"remote"`        // Remote name if remote branch (e.g., "origin"), empty for local
	CommitID     string `json:"commit_id"`     // Commit this branch points to
	ShortID      string `json:"short_id"`      // Short commit ID
	IsTracked    bool   `json:"is_tracked"`    // True if this remote branch is being tracked locally
	IsLocal      bool   `json:"is_local"`      // True if this is a local branch
	LocalDeleted bool   `json:"local_deleted"` // True if local was deleted but remote is still tracked
	IsCurrent    bool   `json:"is_current"`    // True if this is the current branch
	Ahead        int    `json:"ahead"`         // Commits ahead of remote (for local branches)
	Behind       int    `json:"behind"`        // Commits behind remote (for local branches)
	HasConflict  bool   `json:"has_conflict"`  // True if local and remote have diverged
}
