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

// GitHubPR represents a GitHub Pull Request
type GitHubPR struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	URL       string `json:"url"`
	State     string `json:"state"`
	BaseBranch string `json:"base_branch"`
	HeadBranch string `json:"head_branch"`
	CommitIDs []string `json:"commit_ids"`
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
