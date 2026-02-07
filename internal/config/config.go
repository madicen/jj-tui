// Package config handles persistent configuration for jj-tui
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the persistent configuration
type Config struct {
	GitHubToken string `json:"github_token,omitempty"`

	// GitHub filter settings
	GitHubShowMerged      *bool `json:"github_show_merged,omitempty"`       // nil = true (show by default)
	GitHubShowClosed      *bool `json:"github_show_closed,omitempty"`       // nil = true (show by default)
	GitHubOnlyMine        *bool `json:"github_only_mine,omitempty"`         // nil = false (show all by default)
	GitHubPRLimit         *int  `json:"github_pr_limit,omitempty"`          // nil = 100 (default limit)
	GitHubRefreshInterval *int  `json:"github_refresh_interval,omitempty"` // nil = 120 seconds (2 min default), 0 = disabled

	// Ticket provider selection: "jira" or "codecks"
	TicketProvider string `json:"ticket_provider,omitempty"`

	// Jira settings
	JiraURL              string `json:"jira_url,omitempty"`
	JiraUser             string `json:"jira_user,omitempty"`
	JiraToken            string `json:"jira_token,omitempty"`
	JiraExcludedStatuses string `json:"jira_excluded_statuses,omitempty"` // Comma-separated statuses to hide

	// Codecks settings
	CodecksSubdomain        string `json:"codecks_subdomain,omitempty"`
	CodecksToken            string `json:"codecks_token,omitempty"`
	CodecksProject          string `json:"codecks_project,omitempty"`          // Optional: filter by project name
	CodecksExcludedStatuses string `json:"codecks_excluded_statuses,omitempty"` // Comma-separated statuses to hide

	// Ticket workflow settings
	TicketAutoInProgress *bool `json:"ticket_auto_in_progress,omitempty"` // nil = true (auto-set "In Progress" when creating branch)

	// Internal: tracks where the config was loaded from
	loadedFrom string `json:"-"`
}

// LocalConfigFileName is the name of the per-repo config file
const LocalConfigFileName = ".jj-tui.json"

// configDir returns the global config directory path
func configDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".config", "jj-tui"), nil
}

// globalConfigPath returns the full path to the global config file
func globalConfigPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// localConfigPath returns the path to the local config file in the current directory
func localConfigPath() string {
	return LocalConfigFileName
}

// loadFromFile loads config from a specific file path
func loadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // File doesn't exist
		}
		return nil, fmt.Errorf("failed to read config from %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config from %s: %w", path, err)
	}
	cfg.loadedFrom = path
	return &cfg, nil
}

// mergeConfig merges source config into dest, only overwriting non-empty values
func mergeConfig(dest, source *Config) {
	if source == nil {
		return
	}
	if source.GitHubToken != "" {
		dest.GitHubToken = source.GitHubToken
	}
	if source.GitHubShowMerged != nil {
		dest.GitHubShowMerged = source.GitHubShowMerged
	}
	if source.GitHubShowClosed != nil {
		dest.GitHubShowClosed = source.GitHubShowClosed
	}
	if source.GitHubOnlyMine != nil {
		dest.GitHubOnlyMine = source.GitHubOnlyMine
	}
	if source.GitHubPRLimit != nil {
		dest.GitHubPRLimit = source.GitHubPRLimit
	}
	if source.GitHubRefreshInterval != nil {
		dest.GitHubRefreshInterval = source.GitHubRefreshInterval
	}
	if source.TicketProvider != "" {
		dest.TicketProvider = source.TicketProvider
	}
	if source.JiraURL != "" {
		dest.JiraURL = source.JiraURL
	}
	if source.JiraUser != "" {
		dest.JiraUser = source.JiraUser
	}
	if source.JiraToken != "" {
		dest.JiraToken = source.JiraToken
	}
	if source.JiraExcludedStatuses != "" {
		dest.JiraExcludedStatuses = source.JiraExcludedStatuses
	}
	if source.CodecksSubdomain != "" {
		dest.CodecksSubdomain = source.CodecksSubdomain
	}
	if source.CodecksToken != "" {
		dest.CodecksToken = source.CodecksToken
	}
	if source.CodecksProject != "" {
		dest.CodecksProject = source.CodecksProject
	}
	if source.CodecksExcludedStatuses != "" {
		dest.CodecksExcludedStatuses = source.CodecksExcludedStatuses
	}
	if source.TicketAutoInProgress != nil {
		dest.TicketAutoInProgress = source.TicketAutoInProgress
	}
}

// Load reads config with the following priority (highest to lowest):
// 1. JJ_TUI_CONFIG env var (specific config file path)
// 2. .jj-tui.json in current directory (local/repo config)
// 3. ~/.config/jj-tui/config.json (global config)
// Local config values override global config values.
func Load() (*Config, error) {
	cfg := &Config{}

	// Check for JJ_TUI_CONFIG env var first
	if envPath := os.Getenv("JJ_TUI_CONFIG"); envPath != "" {
		envCfg, err := loadFromFile(envPath)
		if err != nil {
			return nil, err
		}
		if envCfg != nil {
			return envCfg, nil
		}
		// If env var is set but file doesn't exist, return empty config
		cfg.loadedFrom = envPath
		return cfg, nil
	}

	// Load global config first
	globalPath, err := globalConfigPath()
	if err == nil {
		globalCfg, err := loadFromFile(globalPath)
		if err != nil {
			return nil, err
		}
		if globalCfg != nil {
			cfg = globalCfg
		}
	}

	// Load local config and merge (local overrides global)
	localPath := localConfigPath()
	localCfg, err := loadFromFile(localPath)
	if err != nil {
		return nil, err
	}
	if localCfg != nil {
		mergeConfig(cfg, localCfg)
		cfg.loadedFrom = localPath // Mark as loaded from local
	} else if cfg.loadedFrom == "" {
		cfg.loadedFrom = globalPath
	}

	return cfg, nil
}

// Save writes the config to disk
// By default, saves to the global config. Use SaveLocal() for local config.
func (c *Config) Save() error {
	return c.SaveTo("")
}

// SaveLocal saves the config to the local .jj-tui.json file in the current directory
func (c *Config) SaveLocal() error {
	return c.SaveTo(localConfigPath())
}

// SaveTo saves the config to a specific path, or global config if path is empty
func (c *Config) SaveTo(path string) error {
	if path == "" {
		// Save to global config
		dir, err := configDir()
		if err != nil {
			return err
		}

		// Create config directory if it doesn't exist
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		path, err = globalConfigPath()
		if err != nil {
			return err
		}
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write with restrictive permissions (user read/write only)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	c.loadedFrom = path
	return nil
}

// HasLocalConfig returns true if a local .jj-tui.json exists in the current directory
func HasLocalConfig() bool {
	_, err := os.Stat(localConfigPath())
	return err == nil
}

// LoadedFrom returns the path the config was loaded from
func (c *Config) LoadedFrom() string {
	return c.loadedFrom
}

// IsLocal returns true if the config was loaded from a local .jj-tui.json file
func (c *Config) IsLocal() bool {
	return c.loadedFrom == localConfigPath()
}

// ApplyToEnvironment sets environment variables from config values
// Only sets variables that are not already set (env takes precedence)
func (c *Config) ApplyToEnvironment() {
	if c.GitHubToken != "" && os.Getenv("GITHUB_TOKEN") == "" {
		os.Setenv("GITHUB_TOKEN", c.GitHubToken)
	}
	if c.JiraURL != "" && os.Getenv("JIRA_URL") == "" {
		os.Setenv("JIRA_URL", c.JiraURL)
	}
	if c.JiraUser != "" && os.Getenv("JIRA_USER") == "" {
		os.Setenv("JIRA_USER", c.JiraUser)
	}
	if c.JiraToken != "" && os.Getenv("JIRA_TOKEN") == "" {
		os.Setenv("JIRA_TOKEN", c.JiraToken)
	}
	if c.CodecksSubdomain != "" && os.Getenv("CODECKS_SUBDOMAIN") == "" {
		os.Setenv("CODECKS_SUBDOMAIN", c.CodecksSubdomain)
	}
	if c.CodecksToken != "" && os.Getenv("CODECKS_TOKEN") == "" {
		os.Setenv("CODECKS_TOKEN", c.CodecksToken)
	}
	if c.CodecksProject != "" && os.Getenv("CODECKS_PROJECT") == "" {
		os.Setenv("CODECKS_PROJECT", c.CodecksProject)
	}
}

// UpdateFromEnvironment updates config with current environment values
func (c *Config) UpdateFromEnvironment() {
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		c.GitHubToken = token
	}
	if url := os.Getenv("JIRA_URL"); url != "" {
		c.JiraURL = url
	}
	if user := os.Getenv("JIRA_USER"); user != "" {
		c.JiraUser = user
	}
	if token := os.Getenv("JIRA_TOKEN"); token != "" {
		c.JiraToken = token
	}
	if subdomain := os.Getenv("CODECKS_SUBDOMAIN"); subdomain != "" {
		c.CodecksSubdomain = subdomain
	}
	if token := os.Getenv("CODECKS_TOKEN"); token != "" {
		c.CodecksToken = token
	}
	if project := os.Getenv("CODECKS_PROJECT"); project != "" {
		c.CodecksProject = project
	}
}

// HasGitHub returns true if GitHub is configured
func (c *Config) HasGitHub() bool {
	return c.GitHubToken != ""
}

// ShowMergedPRs returns whether to show merged PRs (defaults to true)
func (c *Config) ShowMergedPRs() bool {
	if c.GitHubShowMerged == nil {
		return true
	}
	return *c.GitHubShowMerged
}

// ShowClosedPRs returns whether to show closed PRs (defaults to true)
func (c *Config) ShowClosedPRs() bool {
	if c.GitHubShowClosed == nil {
		return true
	}
	return *c.GitHubShowClosed
}

// OnlyMyPRs returns whether to show only the user's own PRs (defaults to false)
func (c *Config) OnlyMyPRs() bool {
	if c.GitHubOnlyMine == nil {
		return false
	}
	return *c.GitHubOnlyMine
}

// PRLimit returns the maximum number of PRs to load (defaults to 100)
func (c *Config) PRLimit() int {
	if c.GitHubPRLimit == nil {
		return 100
	}
	return *c.GitHubPRLimit
}

// PRRefreshInterval returns the PR auto-refresh interval in seconds
// Returns 0 if auto-refresh is disabled, defaults to 120 (2 minutes)
func (c *Config) PRRefreshInterval() int {
	if c.GitHubRefreshInterval == nil {
		return 120 // Default: 2 minutes
	}
	return *c.GitHubRefreshInterval
}

// AutoInProgressOnBranch returns true if tickets should auto-transition to "In Progress" when creating a branch
// Defaults to true (enabled)
func (c *Config) AutoInProgressOnBranch() bool {
	if c.TicketAutoInProgress == nil {
		return true // Default: enabled
	}
	return *c.TicketAutoInProgress
}

// HasJira returns true if Jira is fully configured
func (c *Config) HasJira() bool {
	return c.JiraURL != "" && c.JiraUser != "" && c.JiraToken != ""
}

// HasCodecks returns true if Codecks is fully configured
func (c *Config) HasCodecks() bool {
	return c.CodecksSubdomain != "" && c.CodecksToken != ""
}

// GetTicketProvider returns the configured ticket provider, defaulting based on what's configured
func (c *Config) GetTicketProvider() string {
	if c.TicketProvider != "" {
		return c.TicketProvider
	}
	// Auto-detect based on what's configured
	if c.HasCodecks() {
		return "codecks"
	}
	if c.HasJira() {
		return "jira"
	}
	return ""
}

