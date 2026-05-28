// Package config handles persistent configuration for jj-tui
package config

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DefaultAIProfileName is the synthesized profile name used when a legacy
// (flat) AI config has no `ai_profiles` list on disk.
const DefaultAIProfileName = "default"

// AIProfile is one named bundle of AI inference settings. The user can switch
// between profiles from the long-press menu on generate buttons (one-shot
// override) or from Settings → AI (persistent active profile).
type AIProfile struct {
	Name           string `json:"name"`
	Provider       string `json:"provider,omitempty"`
	BaseURL        string `json:"base_url,omitempty"`
	Model          string `json:"model,omitempty"`
	APIKey         string `json:"api_key,omitempty"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"`
}

// Summary returns a short "provider · model" label for UI rows.
func (p AIProfile) Summary() string {
	prov := strings.TrimSpace(p.Provider)
	if prov == "" {
		prov = "openai_compatible"
	}
	model := strings.TrimSpace(p.Model)
	if model == "" {
		switch prov {
		case "gemini":
			model = "gemini-2.5-flash"
		case "ollama":
			model = OllamaDefaultModel
		default:
			model = "gpt-4o-mini"
		}
	}
	return prov + " · " + model
}

// GitHubAuthMethod represents how the user authenticated with GitHub
type GitHubAuthMethod string

const (
	// GitHubAuthNone means no authentication
	GitHubAuthNone GitHubAuthMethod = ""
	// GitHubAuthDeviceFlow means authentication via Device Flow (needs periodic reauth)
	GitHubAuthDeviceFlow GitHubAuthMethod = "device_flow"
	// GitHubAuthToken means authentication via manual token entry
	GitHubAuthToken GitHubAuthMethod = "token"
	// GitHubAuthGhCLI means authentication via GitHub CLI (`gh auth login`); no token stored in config.
	GitHubAuthGhCLI GitHubAuthMethod = "gh_cli"
)

// GitHub token source: where jj-tui reads the API token (explicit choice; no cross-source fallback).
const (
	GitHubTokenSourceSaved = "saved"   // github_token in jj-tui config (device flow or pasted)
	GitHubTokenSourceEnv   = "env"     // GITHUB_TOKEN environment variable only
	GitHubTokenSourceGhCLI = "gh_cli" // `gh auth token` only
)

// NormalizeGitHubTokenSource returns a valid github_token_source value, defaulting to saved.
func NormalizeGitHubTokenSource(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case GitHubTokenSourceSaved:
		return GitHubTokenSourceSaved
	case GitHubTokenSourceEnv:
		return GitHubTokenSourceEnv
	case GitHubTokenSourceGhCLI, "gh-cli":
		return GitHubTokenSourceGhCLI
	default:
		return GitHubTokenSourceSaved
	}
}

// Config holds the persistent configuration
type Config struct {
	GitHubToken       string           `json:"github_token,omitempty"`
	GitHubTokenSource string           `json:"github_token_source,omitempty"` // saved | env | gh_cli (see constants)
	GitHubAuthMethod  GitHubAuthMethod `json:"github_auth_method,omitempty"`  // How the saved token was obtained

	// GitHub filter settings
	GitHubShowMerged      *bool `json:"github_show_merged,omitempty"`      // nil = true (show by default)
	GitHubShowClosed      *bool `json:"github_show_closed,omitempty"`      // nil = true (show by default)
	GitHubOnlyMine        *bool `json:"github_only_mine,omitempty"`        // nil = false (show all by default)
	GitHubPRLimit         *int  `json:"github_pr_limit,omitempty"`         // nil = 100 (default limit)
	GitHubRefreshInterval *int  `json:"github_refresh_interval,omitempty"` // nil = 120 seconds (2 min default), 0 = disabled

	// Ticket provider selection: "jira" or "codecks"
	TicketProvider string `json:"ticket_provider,omitempty"`

	// Jira settings
	JiraURL              string `json:"jira_url,omitempty"`
	JiraUser             string `json:"jira_user,omitempty"`
	JiraToken            string `json:"jira_token,omitempty"`
	JiraProject          string `json:"jira_project,omitempty"`           // Project key for creating new issues (e.g., "PROJ")
	JiraProjectFilter    string `json:"jira_project_filter,omitempty"`    // Optional: project key(s) to filter ticket list (e.g., "PROJ" or "PROJ,TEAM")
	JiraIssueType        string `json:"jira_issue_type,omitempty"`        // Default issue type when creating issues (e.g., "Task", "Bug", "Story")
	JiraJQL              string `json:"jira_jql,omitempty"`               // Optional: custom JQL to append to query (e.g., "sprint in openSprints()")
	JiraExcludedStatuses string `json:"jira_excluded_statuses,omitempty"` // Comma-separated statuses to hide

	// Codecks settings
	CodecksSubdomain        string `json:"codecks_subdomain,omitempty"`
	CodecksToken            string `json:"codecks_token,omitempty"`
	CodecksProject          string `json:"codecks_project,omitempty"`           // Optional: filter by project name
	CodecksExcludedStatuses string `json:"codecks_excluded_statuses,omitempty"` // Comma-separated statuses to hide

	// GitHub Issues settings (uses existing GitHubToken for auth)
	GitHubIssuesExcludedStatuses string `json:"github_issues_excluded_statuses,omitempty"` // Comma-separated statuses to hide (e.g., "closed")

	// Ticket workflow settings
	TicketAutoInProgress *bool `json:"ticket_auto_in_progress,omitempty"` // nil = true (auto-set "In Progress" when creating branch)

	// Branch settings
	BranchStatsLimit      *int  `json:"branch_limit,omitempty"`            // nil = 50 (default limit for branch stats calculation)
	SanitizeBookmarkNames *bool `json:"sanitize_bookmark_names,omitempty"` // nil = true (auto-fix invalid bookmark names)

	// Graph view: jj revset for which commits to show. Empty = jj.DefaultGraphRevset (see DefaultGraphRevset in jj package: mutable rows in @'s ancestors/descendants plus the (parents(@)+):: sibling subtree, unioned with bookmarks() and main@origin).
	// Example: "trunk() | (ancestors(@) - ancestors(trunk()))" for main + your branch only.
	GraphRevset string `json:"graph_revset,omitempty"`

	// ExternalFileEditor opens the selected changed file from the graph (files pane, key O).
	// Values: none, cursor, vscode, zed, neovim, emacs, sublime, idea, custom (case-insensitive; see NormalizeExternalFileEditor).
	ExternalFileEditor string `json:"external_file_editor,omitempty"`
	// ExternalFileEditorCustom: when ExternalFileEditor is "custom", a shell snippet run as `sh -c` with {path}
	// replaced by a single-quoted absolute path, e.g. `cursor -g {path}` or `alacritty -e nvim {path}`.
	ExternalFileEditorCustom string `json:"external_file_editor_custom,omitempty"`

	// Theme colors (hex, e.g. "#7E00AF"). Empty = use built-in defaults.
	ThemePrimary   string `json:"theme_primary,omitempty"`
	ThemeSecondary string `json:"theme_secondary,omitempty"`
	ThemeMuted     string `json:"theme_muted,omitempty"`

	// Optional generative text. API key: config ai_api_key and/or env JJ_TUI_AI_API_KEY (env wins).
	AIEnabled        *bool  `json:"ai_enabled,omitempty"`         // nil/false = off
	AIBaseURL        string `json:"ai_base_url,omitempty"`        // empty = https://api.openai.com/v1
	AIModel          string `json:"ai_model,omitempty"`           // empty = gpt-4o-mini
	AITimeoutSeconds *int   `json:"ai_timeout_seconds,omitempty"` // nil/0 = 60
	AIProvider       string `json:"ai_provider,omitempty"`        // empty = openai_compatible; allowed: openai_compatible, gemini, ollama
	AIAPIKey         string `json:"ai_api_key,omitempty"`         // optional; env overrides when set

	// AIProfiles is the list of saved named (provider, model, base URL, key, timeout) presets.
	// The active profile's fields are mirrored onto the flat AI* fields above for back-compat
	// with callers like llm.NewProviderForConfig. Empty list = legacy single-profile config:
	// a "default" profile is synthesized from the flat fields on Load.
	AIProfiles []AIProfile `json:"ai_profiles,omitempty"`
	// AIActiveProfile names the entry in AIProfiles that is currently active.
	AIActiveProfile string `json:"ai_active_profile,omitempty"`

	// AI evolog split (optional; nil = defaults below)
	AIEvologDescribeAfterSplitDefault *bool  `json:"ai_evolog_describe_after_split_default,omitempty"` // nil/false = off when opening split modal
	AIEvologFileSplitEnabled          *bool  `json:"ai_evolog_file_split_enabled,omitempty"`           // nil/true = honor LLM files_first_commit; false = ignore
	AIEvologHunkSplitEnabled          *bool  `json:"ai_evolog_hunk_split_enabled,omitempty"`           // nil/true = honor hunk_prefix_first_commit + hunk prompt; false = ignore
	AIEvologMultiSplitMax             *int   `json:"ai_evolog_multi_split_max,omitempty"`              // nil = full cap; clamp 1–EvologAIMultiSplitHardMax
	AIEvologMultiSplitMode            string `json:"ai_evolog_multi_split_mode,omitempty"`             // empty or "batch" = one cmd; "stepwise" = one base per confirm

	// Internal: tracks where the config was loaded from
	loadedFrom string `json:"-"`
}

// EnvAIAPIKey is the environment variable for the LLM API key; when set, it overrides ai_api_key in config.
const EnvAIAPIKey = "JJ_TUI_AI_API_KEY"

// OllamaDefaultChatBaseURL is the default OpenAI-compatible API root for a local Ollama server (no trailing slash).
const OllamaDefaultChatBaseURL = "http://127.0.0.1:11434/v1"

// OllamaDefaultModel is the default model id when ai_provider is ollama and ai_model is empty.
const OllamaDefaultModel = "qwen2.5:1.5b"

// OllamaOpenAICompatiblePlaceholderKey is used as the Bearer token for Ollama; the server ignores it for typical local setups.
const OllamaOpenAICompatiblePlaceholderKey = "ollama"

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
	if source.GitHubAuthMethod != "" {
		dest.GitHubAuthMethod = source.GitHubAuthMethod
	}
	if source.GitHubTokenSource != "" {
		dest.GitHubTokenSource = source.GitHubTokenSource
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
	if source.JiraProject != "" {
		dest.JiraProject = source.JiraProject
	}
	if source.JiraProjectFilter != "" {
		dest.JiraProjectFilter = source.JiraProjectFilter
	}
	if source.JiraIssueType != "" {
		dest.JiraIssueType = source.JiraIssueType
	}
	if source.JiraJQL != "" {
		dest.JiraJQL = source.JiraJQL
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
	if source.GitHubIssuesExcludedStatuses != "" {
		dest.GitHubIssuesExcludedStatuses = source.GitHubIssuesExcludedStatuses
	}
	if source.TicketAutoInProgress != nil {
		dest.TicketAutoInProgress = source.TicketAutoInProgress
	}
	if source.BranchStatsLimit != nil {
		dest.BranchStatsLimit = source.BranchStatsLimit
	}
	if source.SanitizeBookmarkNames != nil {
		dest.SanitizeBookmarkNames = source.SanitizeBookmarkNames
	}
	if source.GraphRevset != "" {
		dest.GraphRevset = source.GraphRevset
	}
	if source.ThemePrimary != "" {
		dest.ThemePrimary = source.ThemePrimary
	}
	if source.ThemeSecondary != "" {
		dest.ThemeSecondary = source.ThemeSecondary
	}
	if source.ThemeMuted != "" {
		dest.ThemeMuted = source.ThemeMuted
	}
	if source.ExternalFileEditor != "" {
		dest.ExternalFileEditor = source.ExternalFileEditor
	}
	if source.ExternalFileEditorCustom != "" {
		dest.ExternalFileEditorCustom = source.ExternalFileEditorCustom
	}
	if source.AIEnabled != nil {
		dest.AIEnabled = source.AIEnabled
	}
	if source.AIBaseURL != "" {
		dest.AIBaseURL = source.AIBaseURL
	}
	if source.AIModel != "" {
		dest.AIModel = source.AIModel
	}
	if source.AITimeoutSeconds != nil {
		dest.AITimeoutSeconds = source.AITimeoutSeconds
	}
	if source.AIProvider != "" {
		dest.AIProvider = source.AIProvider
	}
	if source.AIAPIKey != "" {
		dest.AIAPIKey = source.AIAPIKey
	}
	if len(source.AIProfiles) > 0 {
		dest.AIProfiles = make([]AIProfile, len(source.AIProfiles))
		copy(dest.AIProfiles, source.AIProfiles)
	}
	if source.AIActiveProfile != "" {
		dest.AIActiveProfile = source.AIActiveProfile
	}
	if source.AIEvologDescribeAfterSplitDefault != nil {
		dest.AIEvologDescribeAfterSplitDefault = source.AIEvologDescribeAfterSplitDefault
	}
	if source.AIEvologFileSplitEnabled != nil {
		dest.AIEvologFileSplitEnabled = source.AIEvologFileSplitEnabled
	}
	if source.AIEvologHunkSplitEnabled != nil {
		dest.AIEvologHunkSplitEnabled = source.AIEvologHunkSplitEnabled
	}
	if source.AIEvologMultiSplitMax != nil {
		dest.AIEvologMultiSplitMax = source.AIEvologMultiSplitMax
	}
	if source.AIEvologMultiSplitMode != "" {
		dest.AIEvologMultiSplitMode = source.AIEvologMultiSplitMode
	}
}

// Canonical external editor presets (NormalizeExternalFileEditor).
const (
	ExternalEditorNone     = "none"
	ExternalEditorCursor   = "cursor"
	ExternalEditorVSCode   = "vscode"
	ExternalEditorZed      = "zed"
	ExternalEditorNeovim   = "neovim"
	ExternalEditorEmacs    = "emacs"
	ExternalEditorSublime  = "sublime"
	ExternalEditorIntelliJ = "idea"
	ExternalEditorCustom   = "custom"
)

// NormalizeExternalFileEditor returns a canonical preset string for cfg (nil-safe).
func NormalizeExternalFileEditor(cfg *Config) string {
	if cfg == nil {
		return ExternalEditorNone
	}
	s := strings.ToLower(strings.TrimSpace(cfg.ExternalFileEditor))
	switch s {
	case "", "none", "disabled", "off":
		return ExternalEditorNone
	case "cursor":
		return ExternalEditorCursor
	case "vscode", "code", "vs code", "visual studio code":
		return ExternalEditorVSCode
	case "zed":
		return ExternalEditorZed
	case "neovim", "nvim", "nvr":
		return ExternalEditorNeovim
	case "emacs", "emacsclient":
		return ExternalEditorEmacs
	case "sublime", "subl":
		return ExternalEditorSublime
	case "idea", "intellij", "jetbrains", "webstorm", "goland", "rustrover", "pycharm":
		return ExternalEditorIntelliJ
	case "custom":
		return ExternalEditorCustom
	default:
		return ExternalEditorNone
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
			envCfg.normalizeAIProfiles()
			return envCfg, nil
		}
		// If env var is set but file doesn't exist, return empty config
		cfg.loadedFrom = envPath
		cfg.normalizeAIProfiles()
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

	cfg.normalizeAIProfiles()
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

	// Make the persisted view coherent: ensure the active profile reflects any
	// in-memory edits to the flat AI* fields, then re-mirror the active profile
	// onto the flat fields so both representations agree on disk.
	c.normalizeAIProfiles()
	c.syncActiveAIProfileFromFlat()
	c.applyActiveAIProfile()

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

// ghAuthTokenTimeout bounds how long `gh auth token` may run.
const ghAuthTokenTimeout = 5 * time.Second

// TryGitHubCLIToken returns the token from `gh auth token` when GitHub CLI is installed and logged in.
func TryGitHubCLIToken() (string, bool) {
	if _, err := exec.LookPath("gh"); err != nil {
		return "", false
	}
	ctx, cancel := context.WithTimeout(context.Background(), ghAuthTokenTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", "auth", "token")
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	tok := strings.TrimSpace(string(out))
	if tok == "" {
		return "", false
	}
	return tok, true
}

// GitHubTokenSourceOrDefault returns the configured token source, or a legacy default when unset.
func (c *Config) GitHubTokenSourceOrDefault() string {
	if c == nil {
		return GitHubTokenSourceSaved
	}
	if strings.TrimSpace(c.GitHubTokenSource) == "" {
		if c.GitHubToken != "" {
			return GitHubTokenSourceSaved
		}
		if os.Getenv("GITHUB_TOKEN") != "" {
			return GitHubTokenSourceEnv
		}
		return GitHubTokenSourceSaved
	}
	return NormalizeGitHubTokenSource(c.GitHubTokenSource)
}

// GitHubTokenForAPI returns the token for cfg's chosen github_token_source only (no fallback
// across sources). Pass nil for an in-memory empty config (treated as saved with no token).
func GitHubTokenForAPI(cfg *Config) (token, source string) {
	src := GitHubTokenSourceSaved
	if cfg != nil {
		src = cfg.GitHubTokenSourceOrDefault()
	}
	switch src {
	case GitHubTokenSourceSaved:
		if cfg == nil || cfg.GitHubToken == "" {
			return "", ""
		}
		if from := cfg.LoadedFrom(); from != "" {
			return cfg.GitHubToken, fmt.Sprintf("config:%s", from)
		}
		return cfg.GitHubToken, "config"
	case GitHubTokenSourceEnv:
		if t := os.Getenv("GITHUB_TOKEN"); t != "" {
			return t, "env:GITHUB_TOKEN"
		}
		return "", ""
	case GitHubTokenSourceGhCLI:
		if tok, ok := TryGitHubCLIToken(); ok {
			return tok, "gh:auth token"
		}
		return "", ""
	default:
		return "", ""
	}
}

// ApplyToEnvironment sets environment variables from config values
// Only sets variables that are not already set (env takes precedence)
func (c *Config) ApplyToEnvironment() {
	if c.GitHubTokenSourceOrDefault() == GitHubTokenSourceSaved && c.GitHubToken != "" && os.Getenv("GITHUB_TOKEN") == "" {
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
	if c.JiraProject != "" && os.Getenv("JIRA_PROJECT") == "" {
		os.Setenv("JIRA_PROJECT", c.JiraProject)
	}
	if c.JiraProjectFilter != "" && os.Getenv("JIRA_PROJECT_FILTER") == "" {
		os.Setenv("JIRA_PROJECT_FILTER", c.JiraProjectFilter)
	}
	if c.JiraIssueType != "" && os.Getenv("JIRA_ISSUE_TYPE") == "" {
		os.Setenv("JIRA_ISSUE_TYPE", c.JiraIssueType)
	}
	if c.JiraJQL != "" && os.Getenv("JIRA_JQL") == "" {
		os.Setenv("JIRA_JQL", c.JiraJQL)
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
	// Legacy: only copy GITHUB_TOKEN into config when github_token_source was never set.
	if strings.TrimSpace(c.GitHubTokenSource) == "" {
		if token := os.Getenv("GITHUB_TOKEN"); token != "" {
			c.GitHubToken = token
		}
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
	if project := os.Getenv("JIRA_PROJECT"); project != "" {
		c.JiraProject = project
	}
	if filter := os.Getenv("JIRA_PROJECT_FILTER"); filter != "" {
		c.JiraProjectFilter = filter
	}
	if issueType := os.Getenv("JIRA_ISSUE_TYPE"); issueType != "" {
		c.JiraIssueType = issueType
	}
	if jql := os.Getenv("JIRA_JQL"); jql != "" {
		c.JiraJQL = jql
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

// HasGitHub returns true if the chosen token source yields a non-empty token.
func (c *Config) HasGitHub() bool {
	tok, _ := GitHubTokenForAPI(c)
	return tok != ""
}

// UsedDeviceFlow returns true if the GitHub token was obtained via Device Flow
func (c *Config) UsedDeviceFlow() bool {
	return c.GitHubAuthMethod == GitHubAuthDeviceFlow
}

// UsedGhCLIAuth returns true if GitHub auth is via the GitHub CLI token source flow.
func (c *Config) UsedGhCLIAuth() bool {
	return c.GitHubAuthMethod == GitHubAuthGhCLI || c.GitHubTokenSourceOrDefault() == GitHubTokenSourceGhCLI
}

// SetGitHubToken sets the GitHub token and auth method
func (c *Config) SetGitHubToken(token string, method GitHubAuthMethod) {
	c.GitHubToken = token
	c.GitHubAuthMethod = method
	c.GitHubTokenSource = GitHubTokenSourceSaved
}

// ClearGitHub clears the GitHub token and auth method
func (c *Config) ClearGitHub() {
	c.GitHubToken = ""
	c.GitHubAuthMethod = GitHubAuthNone
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

// BranchLimit returns the maximum number of branches to calculate stats for (defaults to 50)
// Branches beyond this limit will still show but without ahead/behind counts
func (c *Config) BranchLimit() int {
	if c.BranchStatsLimit == nil {
		return 50
	}
	return *c.BranchStatsLimit
}

// ShouldSanitizeBookmarkNames returns whether to auto-fix invalid bookmark names (defaults to true)
func (c *Config) ShouldSanitizeBookmarkNames() bool {
	if c.SanitizeBookmarkNames == nil {
		return true // Default: enabled
	}
	return *c.SanitizeBookmarkNames
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
	// Note: github_issues is not auto-detected since it shares the GitHub token
	// User must explicitly set ticket_provider: "github_issues" to use it
	return ""
}

// HasGitHubIssues returns true if GitHub Issues can be used as a ticket provider
// (requires GitHub to be configured)
func (c *Config) HasGitHubIssues() bool {
	return c.HasGitHub()
}

// GetThemePrimary returns the primary theme color (hex). Defaults to "#7E00AF" if not set.
func (c *Config) GetThemePrimary() string {
	if c == nil || c.ThemePrimary == "" {
		return "#7E00AF"
	}
	return c.ThemePrimary
}

// GetThemeSecondary returns the secondary theme color (hex). Defaults to "#50FA7B" if not set.
func (c *Config) GetThemeSecondary() string {
	if c == nil || c.ThemeSecondary == "" {
		return "#50FA7B"
	}
	return c.ThemeSecondary
}

// GetThemeMuted returns the muted theme color (hex). Defaults to "#6272A4" if not set.
func (c *Config) GetThemeMuted() string {
	if c == nil || c.ThemeMuted == "" {
		return "#6272A4"
	}
	return c.ThemeMuted
}

// AIGenerationEnabled is true when the user turned on AI assist in settings.
func (c *Config) AIGenerationEnabled() bool {
	if c == nil || c.AIEnabled == nil {
		return false
	}
	return *c.AIEnabled
}

// AIBaseURLResolved returns the API base URL (no trailing slash), for OpenAI-compatible clients.
func (c *Config) AIBaseURLResolved() string {
	if c == nil {
		return "https://api.openai.com/v1"
	}
	s := strings.TrimSpace(c.AIBaseURL)
	if s == "" {
		if c.AIProviderOrDefault() == "ollama" {
			return OllamaDefaultChatBaseURL
		}
		return "https://api.openai.com/v1"
	}
	return strings.TrimSuffix(s, "/")
}

// AIModelOrDefault returns the chat model name.
func (c *Config) AIModelOrDefault() string {
	if c == nil {
		return "gpt-4o-mini"
	}
	s := strings.TrimSpace(c.AIModel)
	if s == "" {
		return "gpt-4o-mini"
	}
	return s
}

// AIModelResolved returns the model id to send to the provider, using provider-specific defaults when AIModel is empty.
func (c *Config) AIModelResolved() string {
	if c == nil {
		return "gpt-4o-mini"
	}
	if s := strings.TrimSpace(c.AIModel); s != "" {
		return s
	}
	switch c.AIProviderOrDefault() {
	case "gemini":
		return "gemini-2.5-flash"
	case "ollama":
		return OllamaDefaultModel
	default:
		return "gpt-4o-mini"
	}
}

// AITimeout returns the HTTP timeout for LLM requests (default 60s).
func (c *Config) AITimeout() time.Duration {
	if c == nil || c.AITimeoutSeconds == nil || *c.AITimeoutSeconds <= 0 {
		return 60 * time.Second
	}
	return time.Duration(*c.AITimeoutSeconds) * time.Second
}

// EffectiveAIAPIKey returns the API key from the environment if set, otherwise from the loaded config.
// This lets env override the stored key without needing to re-save config.
func EffectiveAIAPIKey(cfg *Config) string {
	if s := strings.TrimSpace(os.Getenv(EnvAIAPIKey)); s != "" {
		return s
	}
	if cfg == nil {
		return ""
	}
	return strings.TrimSpace(cfg.AIAPIKey)
}

// IsOllamaOpenAICompatibleBaseURL reports whether baseURL points at a typical local Ollama OpenAI-compatible endpoint
// (http(s)://127.0.0.1:11434/v1 or http(s)://localhost:11434/v1). Used to allow an empty API key for local-only setups.
func IsOllamaOpenAICompatibleBaseURL(baseURL string) bool {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return false
	}
	u, err := url.Parse(strings.TrimSuffix(baseURL, "/"))
	if err != nil || u.Host == "" {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if host != "127.0.0.1" && host != "localhost" {
		return false
	}
	if u.Port() != "11434" {
		return false
	}
	path := u.Path
	if path == "" {
		path = "/"
	}
	path = strings.TrimSuffix(path, "/")
	return path == "/v1"
}

// AISupportsGenerationCredentials is true when an API key is set, or when using Ollama provider / local Ollama base URL without a key.
func (c *Config) AISupportsGenerationCredentials() bool {
	if c == nil {
		return false
	}
	if strings.TrimSpace(EffectiveAIAPIKey(c)) != "" {
		return true
	}
	if c.AIProviderOrDefault() == "ollama" {
		return true
	}
	u := strings.TrimSpace(c.AIBaseURL)
	if u == "" {
		return false
	}
	return IsOllamaOpenAICompatibleBaseURL(strings.TrimSuffix(u, "/"))
}

// ResolveOpenAICompatibleBearerKey returns the Bearer token for OpenAI-compatible chat requests.
func (c *Config) ResolveOpenAICompatibleBearerKey() (string, error) {
	if k := strings.TrimSpace(EffectiveAIAPIKey(c)); k != "" {
		return k, nil
	}
	if c == nil {
		return "", fmt.Errorf("missing API key")
	}
	switch c.AIProviderOrDefault() {
	case "ollama":
		return OllamaOpenAICompatiblePlaceholderKey, nil
	default:
		if IsOllamaOpenAICompatibleBaseURL(strings.TrimSuffix(strings.TrimSpace(c.AIBaseURL), "/")) {
			return OllamaOpenAICompatiblePlaceholderKey, nil
		}
	}
	return "", fmt.Errorf("missing API key")
}

// AIConfiguredForGeneration is true when AI is enabled and credentials are available (API key, Ollama preset, or local Ollama base URL).
func (c *Config) AIConfiguredForGeneration() bool {
	return c != nil && c.AIGenerationEnabled() && c.AISupportsGenerationCredentials()
}

// AIProviderOrDefault returns the configured provider.
func (c *Config) AIProviderOrDefault() string {
	if c == nil {
		return "openai_compatible"
	}
	s := strings.ToLower(strings.TrimSpace(c.AIProvider))
	if s == "" {
		return "openai_compatible"
	}
	switch s {
	case "openai_compatible", "gemini", "ollama":
		return s
	default:
		return "openai_compatible"
	}
}

// NormalizeAIProvider returns a canonical provider id (openai_compatible, gemini, ollama).
func NormalizeAIProvider(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "gemini":
		return "gemini"
	case "ollama":
		return "ollama"
	default:
		return "openai_compatible"
	}
}

// snapshotAIProfileFromFlat builds an AIProfile from the current flat AI* fields.
func (c *Config) snapshotAIProfileFromFlat(name string) AIProfile {
	timeout := 0
	if c.AITimeoutSeconds != nil && *c.AITimeoutSeconds > 0 {
		timeout = *c.AITimeoutSeconds
	}
	return AIProfile{
		Name:           strings.TrimSpace(name),
		Provider:       NormalizeAIProvider(c.AIProvider),
		BaseURL:        strings.TrimSpace(c.AIBaseURL),
		Model:          strings.TrimSpace(c.AIModel),
		APIKey:         c.AIAPIKey,
		TimeoutSeconds: timeout,
	}
}

// findAIProfileIndex returns the index of the named profile (case-insensitive on
// the trimmed name) and whether it was found.
func (c *Config) findAIProfileIndex(name string) (int, bool) {
	if c == nil {
		return -1, false
	}
	target := strings.ToLower(strings.TrimSpace(name))
	if target == "" {
		return -1, false
	}
	for i, p := range c.AIProfiles {
		if strings.ToLower(strings.TrimSpace(p.Name)) == target {
			return i, true
		}
	}
	return -1, false
}

// FindAIProfile returns a copy of the named profile and whether it exists.
func (c *Config) FindAIProfile(name string) (AIProfile, bool) {
	idx, ok := c.findAIProfileIndex(name)
	if !ok {
		return AIProfile{}, false
	}
	return c.AIProfiles[idx], true
}

// applyActiveAIProfile copies the active profile's fields onto the flat AI*
// fields so existing callers (llm.NewProviderForConfig) see the right values.
// Called after Load and after SetActiveAIProfile.
func (c *Config) applyActiveAIProfile() {
	if c == nil || len(c.AIProfiles) == 0 {
		return
	}
	idx, ok := c.findAIProfileIndex(c.AIActiveProfile)
	if !ok {
		idx = 0
		c.AIActiveProfile = c.AIProfiles[idx].Name
	}
	p := c.AIProfiles[idx]
	c.AIProvider = NormalizeAIProvider(p.Provider)
	c.AIBaseURL = p.BaseURL
	c.AIModel = p.Model
	c.AIAPIKey = p.APIKey
	if p.TimeoutSeconds > 0 {
		v := p.TimeoutSeconds
		c.AITimeoutSeconds = &v
	} else {
		c.AITimeoutSeconds = nil
	}
}

// syncActiveAIProfileFromFlat copies the flat AI* fields back into the active
// profile slot. Called after Load (legacy migration) and on Save so any in-memory
// flat-field edits stay consistent with the persisted profile list.
func (c *Config) syncActiveAIProfileFromFlat() {
	if c == nil || len(c.AIProfiles) == 0 {
		return
	}
	idx, ok := c.findAIProfileIndex(c.AIActiveProfile)
	if !ok {
		return
	}
	name := c.AIProfiles[idx].Name
	c.AIProfiles[idx] = c.snapshotAIProfileFromFlat(name)
}

// normalizeAIProfiles ensures Profiles+ActiveProfile are coherent and flat
// fields mirror the active profile. Idempotent. Called on Load and Save.
func (c *Config) normalizeAIProfiles() {
	if c == nil {
		return
	}
	if len(c.AIProfiles) == 0 {
		// Legacy migration: synthesize a single default profile from the flat fields.
		c.AIProfiles = []AIProfile{c.snapshotAIProfileFromFlat(DefaultAIProfileName)}
		c.AIActiveProfile = DefaultAIProfileName
		return
	}
	for i := range c.AIProfiles {
		c.AIProfiles[i].Name = strings.TrimSpace(c.AIProfiles[i].Name)
		if c.AIProfiles[i].Name == "" {
			c.AIProfiles[i].Name = fmt.Sprintf("profile-%d", i+1)
		}
		c.AIProfiles[i].Provider = NormalizeAIProvider(c.AIProfiles[i].Provider)
		c.AIProfiles[i].BaseURL = strings.TrimSpace(c.AIProfiles[i].BaseURL)
		c.AIProfiles[i].Model = strings.TrimSpace(c.AIProfiles[i].Model)
	}
	if _, ok := c.findAIProfileIndex(c.AIActiveProfile); !ok {
		c.AIActiveProfile = c.AIProfiles[0].Name
	}
	c.applyActiveAIProfile()
}

// ActiveAIProfile returns a copy of the active AI profile, falling back to a
// profile synthesised from the flat fields when none is configured.
func (c *Config) ActiveAIProfile() AIProfile {
	if c == nil {
		return AIProfile{Name: DefaultAIProfileName, Provider: "openai_compatible"}
	}
	if idx, ok := c.findAIProfileIndex(c.AIActiveProfile); ok {
		return c.AIProfiles[idx]
	}
	if len(c.AIProfiles) > 0 {
		return c.AIProfiles[0]
	}
	return c.snapshotAIProfileFromFlat(DefaultAIProfileName)
}

// AIProfileList returns a copy of the profile list, guaranteeing at least one
// (synthesised) entry. Safe to use for menu rendering.
func (c *Config) AIProfileList() []AIProfile {
	if c == nil {
		return []AIProfile{{Name: DefaultAIProfileName, Provider: "openai_compatible"}}
	}
	if len(c.AIProfiles) == 0 {
		return []AIProfile{c.snapshotAIProfileFromFlat(DefaultAIProfileName)}
	}
	out := make([]AIProfile, len(c.AIProfiles))
	copy(out, c.AIProfiles)
	return out
}

// SetActiveAIProfile switches the active profile and mirrors its fields onto
// the flat AI* fields. Returns an error when name does not match any profile.
func (c *Config) SetActiveAIProfile(name string) error {
	if c == nil {
		return fmt.Errorf("nil config")
	}
	idx, ok := c.findAIProfileIndex(name)
	if !ok {
		return fmt.Errorf("ai profile %q not found", name)
	}
	c.AIActiveProfile = c.AIProfiles[idx].Name
	c.applyActiveAIProfile()
	return nil
}

// AddAIProfile appends p to the profile list. Returns an error when a profile
// with the same name already exists or the name is empty.
func (c *Config) AddAIProfile(p AIProfile) error {
	if c == nil {
		return fmt.Errorf("nil config")
	}
	name := strings.TrimSpace(p.Name)
	if name == "" {
		return fmt.Errorf("profile name is empty")
	}
	if _, ok := c.findAIProfileIndex(name); ok {
		return fmt.Errorf("ai profile %q already exists", name)
	}
	p.Name = name
	p.Provider = NormalizeAIProvider(p.Provider)
	c.AIProfiles = append(c.AIProfiles, p)
	if c.AIActiveProfile == "" {
		c.AIActiveProfile = name
		c.applyActiveAIProfile()
	}
	return nil
}

// UpdateAIProfile replaces the profile identified by name. When the updated
// profile is active, the flat AI* fields are re-synced.
func (c *Config) UpdateAIProfile(name string, p AIProfile) error {
	if c == nil {
		return fmt.Errorf("nil config")
	}
	idx, ok := c.findAIProfileIndex(name)
	if !ok {
		return fmt.Errorf("ai profile %q not found", name)
	}
	newName := strings.TrimSpace(p.Name)
	if newName == "" {
		newName = c.AIProfiles[idx].Name
	}
	if !strings.EqualFold(newName, c.AIProfiles[idx].Name) {
		if other, exists := c.findAIProfileIndex(newName); exists && other != idx {
			return fmt.Errorf("ai profile %q already exists", newName)
		}
	}
	p.Name = newName
	p.Provider = NormalizeAIProvider(p.Provider)
	wasActive := strings.EqualFold(c.AIActiveProfile, c.AIProfiles[idx].Name)
	c.AIProfiles[idx] = p
	if wasActive {
		c.AIActiveProfile = p.Name
		c.applyActiveAIProfile()
	}
	return nil
}

// DeleteAIProfile removes the named profile. The last profile cannot be deleted.
// When the deleted profile was active, the first remaining profile becomes active.
func (c *Config) DeleteAIProfile(name string) error {
	if c == nil {
		return fmt.Errorf("nil config")
	}
	if len(c.AIProfiles) <= 1 {
		return fmt.Errorf("cannot delete the last AI profile")
	}
	idx, ok := c.findAIProfileIndex(name)
	if !ok {
		return fmt.Errorf("ai profile %q not found", name)
	}
	wasActive := strings.EqualFold(c.AIActiveProfile, c.AIProfiles[idx].Name)
	c.AIProfiles = append(c.AIProfiles[:idx], c.AIProfiles[idx+1:]...)
	if wasActive {
		c.AIActiveProfile = c.AIProfiles[0].Name
		c.applyActiveAIProfile()
	}
	return nil
}

// CycleActiveAIProfile moves the active pointer by delta (typically +/-1) through
// the profile list, wrapping at the ends. No-op when fewer than 2 profiles exist.
func (c *Config) CycleActiveAIProfile(delta int) {
	if c == nil || len(c.AIProfiles) < 2 {
		return
	}
	idx, ok := c.findAIProfileIndex(c.AIActiveProfile)
	if !ok {
		idx = 0
	}
	n := len(c.AIProfiles)
	idx = ((idx+delta)%n + n) % n
	c.AIActiveProfile = c.AIProfiles[idx].Name
	c.applyActiveAIProfile()
}

// AITimeoutForProfile returns the HTTP timeout for the given profile, falling
// back to the cfg-wide AITimeout default when the profile's timeout is unset.
func (c *Config) AITimeoutForProfile(p AIProfile) time.Duration {
	if p.TimeoutSeconds > 0 {
		return time.Duration(p.TimeoutSeconds) * time.Second
	}
	return c.AITimeout()
}

// EvologAIMultiSplitHardMax is the upper bound for ai_evolog_multi_split_max (JSON + Settings UI).
// jj-tui's `jj evolog -n` limit is 128 rows; N-1 FAQ bases cannot exceed what evolog returns.
const EvologAIMultiSplitHardMax = 128

// DefaultEvologPostSplitDescribe is true when the evolog split modal should open with post-split AI describe enabled.
func (c *Config) DefaultEvologPostSplitDescribe() bool {
	return c != nil && c.AIEvologDescribeAfterSplitDefault != nil && *c.AIEvologDescribeAfterSplitDefault
}

// EvologAIFilePhaseEnabled is false when LLM-suggested file lists for jj split should be ignored.
func (c *Config) EvologAIFilePhaseEnabled() bool {
	if c == nil || c.AIEvologFileSplitEnabled == nil {
		return true
	}
	return *c.AIEvologFileSplitEnabled
}

// EvologAIHunkPhaseEnabled is false when LLM hunk_prefix_first_commit and hunk prompt excerpts are ignored.
func (c *Config) EvologAIHunkPhaseEnabled() bool {
	if c == nil || c.AIEvologHunkSplitEnabled == nil {
		return true
	}
	return *c.AIEvologHunkSplitEnabled
}

// EvologAIMultiSplitMaxCap returns the max number of bases in an AI multi-split plan (1..EvologAIMultiSplitHardMax).
// When unset in config, the hard max is used so the model is not truncated below evolog depth.
func (c *Config) EvologAIMultiSplitMaxCap() int {
	def := EvologAIMultiSplitHardMax
	if c == nil || c.AIEvologMultiSplitMax == nil {
		return def
	}
	v := *c.AIEvologMultiSplitMax
	if v < 1 {
		return 1
	}
	if v > EvologAIMultiSplitHardMax {
		return EvologAIMultiSplitHardMax
	}
	return v
}

// EvologAIMultiSplitStepwise is true when multi-split runs one FAQ step per user confirm with evolog reload between steps.
func (c *Config) EvologAIMultiSplitStepwise() bool {
	if c == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(c.AIEvologMultiSplitMode)) {
	case "stepwise":
		return true
	default:
		return false
	}
}
