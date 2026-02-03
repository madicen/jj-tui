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

	// Ticket provider selection: "jira" or "codecks"
	TicketProvider string `json:"ticket_provider,omitempty"`

	// Jira settings
	JiraURL   string `json:"jira_url,omitempty"`
	JiraUser  string `json:"jira_user,omitempty"`
	JiraToken string `json:"jira_token,omitempty"`

	// Codecks settings
	CodecksSubdomain string `json:"codecks_subdomain,omitempty"`
	CodecksToken     string `json:"codecks_token,omitempty"`
	CodecksProject   string `json:"codecks_project,omitempty"` // Optional: filter by project name
}

// configDir returns the config directory path
func configDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".config", "jj-tui"), nil
}

// configPath returns the full path to the config file
func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Load reads the config from disk
// Returns an empty config if the file doesn't exist
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return &Config{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// No config file yet - return empty config
			return &Config{}, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

// Save writes the config to disk
func (c *Config) Save() error {
	dir, err := configDir()
	if err != nil {
		return err
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	path, err := configPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write with restrictive permissions (user read/write only)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
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

