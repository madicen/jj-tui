package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestConfigMerge tests that config merging works correctly
func TestConfigMerge(t *testing.T) {
	t.Run("MergeOverridesNonEmpty", func(t *testing.T) {
		dest := &Config{
			GitHubToken:      "original-token",
			TicketProvider:   "jira",
			JiraURL:          "https://original.atlassian.net",
			CodecksSubdomain: "",
		}
		
		source := &Config{
			GitHubToken:      "",  // Empty - should not override
			TicketProvider:   "codecks",  // Non-empty - should override
			JiraURL:          "",  // Empty - should not override
			CodecksSubdomain: "newteam",  // Non-empty - should override
		}
		
		mergeConfig(dest, source)
		
		if dest.GitHubToken != "original-token" {
			t.Errorf("GitHubToken should not be overwritten by empty value, got %s", dest.GitHubToken)
		}
		if dest.TicketProvider != "codecks" {
			t.Errorf("TicketProvider should be overwritten, got %s", dest.TicketProvider)
		}
		if dest.JiraURL != "https://original.atlassian.net" {
			t.Errorf("JiraURL should not be overwritten by empty value, got %s", dest.JiraURL)
		}
		if dest.CodecksSubdomain != "newteam" {
			t.Errorf("CodecksSubdomain should be overwritten, got %s", dest.CodecksSubdomain)
		}
	})
	
	t.Run("MergeWithNilSource", func(t *testing.T) {
		dest := &Config{
			GitHubToken: "token",
		}
		
		mergeConfig(dest, nil)
		
		if dest.GitHubToken != "token" {
			t.Error("Merging nil should not modify dest")
		}
	})
}

// TestConfigHasMethods tests the Has* methods
func TestConfigHasMethods(t *testing.T) {
	t.Run("HasGitHub", func(t *testing.T) {
		cfg := &Config{}
		if cfg.HasGitHub() {
			t.Error("HasGitHub should return false for empty config")
		}
		
		cfg.GitHubToken = "token"
		if !cfg.HasGitHub() {
			t.Error("HasGitHub should return true when token is set")
		}
	})
	
	t.Run("HasJira", func(t *testing.T) {
		cfg := &Config{}
		if cfg.HasJira() {
			t.Error("HasJira should return false for empty config")
		}
		
		cfg.JiraURL = "https://test.atlassian.net"
		if cfg.HasJira() {
			t.Error("HasJira should return false with only URL")
		}
		
		cfg.JiraUser = "user@example.com"
		cfg.JiraToken = "token"
		if !cfg.HasJira() {
			t.Error("HasJira should return true when all fields are set")
		}
	})
	
	t.Run("HasCodecks", func(t *testing.T) {
		cfg := &Config{}
		if cfg.HasCodecks() {
			t.Error("HasCodecks should return false for empty config")
		}
		
		cfg.CodecksSubdomain = "team"
		if cfg.HasCodecks() {
			t.Error("HasCodecks should return false with only subdomain")
		}
		
		cfg.CodecksToken = "token"
		if !cfg.HasCodecks() {
			t.Error("HasCodecks should return true when subdomain and token are set")
		}
	})
}

// TestGetTicketProvider tests the ticket provider detection
func TestGetTicketProvider(t *testing.T) {
	t.Run("ExplicitProvider", func(t *testing.T) {
		cfg := &Config{
			TicketProvider: "jira",
		}
		if cfg.GetTicketProvider() != "jira" {
			t.Error("Should return explicit provider")
		}
	})
	
	t.Run("AutoDetectCodecks", func(t *testing.T) {
		cfg := &Config{
			CodecksSubdomain: "team",
			CodecksToken:     "token",
		}
		if cfg.GetTicketProvider() != "codecks" {
			t.Error("Should auto-detect Codecks when configured")
		}
	})
	
	t.Run("AutoDetectJira", func(t *testing.T) {
		cfg := &Config{
			JiraURL:   "https://test.atlassian.net",
			JiraUser:  "user@example.com",
			JiraToken: "token",
		}
		if cfg.GetTicketProvider() != "jira" {
			t.Error("Should auto-detect Jira when configured")
		}
	})
	
	t.Run("CodecksPreferredOverJira", func(t *testing.T) {
		cfg := &Config{
			CodecksSubdomain: "team",
			CodecksToken:     "token",
			JiraURL:          "https://test.atlassian.net",
			JiraUser:         "user@example.com",
			JiraToken:        "token",
		}
		if cfg.GetTicketProvider() != "codecks" {
			t.Error("Should prefer Codecks when both are configured")
		}
	})
	
	t.Run("EmptyWhenNothingConfigured", func(t *testing.T) {
		cfg := &Config{}
		if cfg.GetTicketProvider() != "" {
			t.Error("Should return empty string when nothing configured")
		}
	})
}

// TestLocalConfigPath tests local config file handling
func TestLocalConfigPath(t *testing.T) {
	path := localConfigPath()
	if path != LocalConfigFileName {
		t.Errorf("localConfigPath() = %s, want %s", path, LocalConfigFileName)
	}
}

// TestHasLocalConfig tests local config detection
func TestHasLocalConfig(t *testing.T) {
	// Create a temp directory and change to it
	tempDir, err := os.MkdirTemp("", "jj-tui-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Save current directory
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	
	// Change to temp directory
	_ = os.Chdir(tempDir)
	
	// Initially should not have local config
	if HasLocalConfig() {
		t.Error("HasLocalConfig should return false when no local config exists")
	}
	
	// Create a local config file
	configPath := filepath.Join(tempDir, LocalConfigFileName)
	if err := os.WriteFile(configPath, []byte(`{"ticket_provider": "codecks"}`), 0600); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}
	
	// Now should have local config
	if !HasLocalConfig() {
		t.Error("HasLocalConfig should return true when local config exists")
	}
}

// TestConfigSaveAndLoad tests round-trip save/load
func TestConfigSaveAndLoad(t *testing.T) {
	// Create a temp directory
	tempDir, err := os.MkdirTemp("", "jj-tui-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	configPath := filepath.Join(tempDir, "config.json")
	
	// Create config
	original := &Config{
		GitHubToken:      "test-token",
		TicketProvider:   "codecks",
		CodecksSubdomain: "myteam",
		CodecksToken:     "codecks-token",
		CodecksProject:   "My Project",
	}
	
	// Save to specific path
	if err := original.SaveTo(configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}
	
	// Load from file
	loaded, err := loadFromFile(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	
	// Verify values
	if loaded.GitHubToken != original.GitHubToken {
		t.Errorf("GitHubToken mismatch: got %s, want %s", loaded.GitHubToken, original.GitHubToken)
	}
	if loaded.TicketProvider != original.TicketProvider {
		t.Errorf("TicketProvider mismatch: got %s, want %s", loaded.TicketProvider, original.TicketProvider)
	}
	if loaded.CodecksSubdomain != original.CodecksSubdomain {
		t.Errorf("CodecksSubdomain mismatch: got %s, want %s", loaded.CodecksSubdomain, original.CodecksSubdomain)
	}
	if loaded.CodecksProject != original.CodecksProject {
		t.Errorf("CodecksProject mismatch: got %s, want %s", loaded.CodecksProject, original.CodecksProject)
	}
}

// TestLoadFromNonExistentFile tests loading from a file that doesn't exist
func TestLoadFromNonExistentFile(t *testing.T) {
	cfg, err := loadFromFile("/nonexistent/path/config.json")
	if err != nil {
		t.Errorf("loadFromFile should not error for nonexistent file, got: %v", err)
	}
	if cfg != nil {
		t.Error("loadFromFile should return nil for nonexistent file")
	}
}

func TestAIModelResolved(t *testing.T) {
	t.Run("explicitModel", func(t *testing.T) {
		cfg := &Config{AIModel: "custom", AIProvider: "gemini"}
		if got := cfg.AIModelResolved(); got != "custom" {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("geminiDefault", func(t *testing.T) {
		cfg := &Config{AIProvider: "gemini"}
		if got := cfg.AIModelResolved(); got != "gemini-2.5-flash" {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("openaiDefault", func(t *testing.T) {
		cfg := &Config{}
		if got := cfg.AIModelResolved(); got != "gpt-4o-mini" {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("ollamaDefault", func(t *testing.T) {
		cfg := &Config{AIProvider: "ollama"}
		if got := cfg.AIModelResolved(); got != OllamaDefaultModel {
			t.Fatalf("got %q want %q", got, OllamaDefaultModel)
		}
	})
}

func TestIsOllamaOpenAICompatibleBaseURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{"loopbackV1", "http://127.0.0.1:11434/v1", true},
		{"loopbackV1Slash", "http://127.0.0.1:11434/v1/", true},
		{"localhostV1", "http://localhost:11434/v1", true},
		{"httpsLoopback", "https://127.0.0.1:11434/v1", true},
		{"noPath", "http://127.0.0.1:11434", false},
		{"lanHost", "http://192.168.0.1:11434/v1", false},
		{"wrongPort", "http://127.0.0.1:9999/v1", false},
		{"empty", "", false},
		{"openai", "https://api.openai.com/v1", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsOllamaOpenAICompatibleBaseURL(tc.url); got != tc.want {
				t.Fatalf("IsOllamaOpenAICompatibleBaseURL(%q) = %v want %v", tc.url, got, tc.want)
			}
		})
	}
}

func TestAISupportsGenerationCredentials(t *testing.T) {
	t.Run("keyInConfig", func(t *testing.T) {
		cfg := &Config{AIAPIKey: "sk-test"}
		if !cfg.AISupportsGenerationCredentials() {
			t.Fatal("expected true with API key")
		}
	})
	t.Run("ollamaProviderNoKey", func(t *testing.T) {
		cfg := &Config{AIProvider: "ollama"}
		if !cfg.AISupportsGenerationCredentials() {
			t.Fatal("expected true for ollama without key")
		}
	})
	t.Run("openaiDefaultNoKey", func(t *testing.T) {
		cfg := &Config{}
		if cfg.AISupportsGenerationCredentials() {
			t.Fatal("expected false with default OpenAI URL and no key")
		}
	})
	t.Run("localOllamaURLNoKey", func(t *testing.T) {
		cfg := &Config{AIBaseURL: "http://127.0.0.1:11434/v1"}
		if !cfg.AISupportsGenerationCredentials() {
			t.Fatal("expected true with local Ollama base and no key")
		}
	})
}

func TestResolveOpenAICompatibleBearerKey(t *testing.T) {
	t.Run("usesConfigKey", func(t *testing.T) {
		cfg := &Config{AIAPIKey: "real"}
		k, err := cfg.ResolveOpenAICompatibleBearerKey()
		if err != nil || k != "real" {
			t.Fatalf("got %q %v", k, err)
		}
	})
	t.Run("ollamaPlaceholder", func(t *testing.T) {
		cfg := &Config{AIProvider: "ollama"}
		k, err := cfg.ResolveOpenAICompatibleBearerKey()
		if err != nil || k != OllamaOpenAICompatiblePlaceholderKey {
			t.Fatalf("got %q %v", k, err)
		}
	})
	t.Run("localURLPlaceholder", func(t *testing.T) {
		cfg := &Config{AIBaseURL: "http://localhost:11434/v1"}
		k, err := cfg.ResolveOpenAICompatibleBearerKey()
		if err != nil || k != OllamaOpenAICompatiblePlaceholderKey {
			t.Fatalf("got %q %v", k, err)
		}
	})
	t.Run("openaiPublicMissing", func(t *testing.T) {
		cfg := &Config{AIBaseURL: ""}
		_, err := cfg.ResolveOpenAICompatibleBearerKey()
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

// TestAIProfileMigration ensures legacy flat AI configs are wrapped into a
// single "default" profile on Load.
func TestAIProfileMigration(t *testing.T) {
	t.Run("legacyFlatWrappedAsDefault", func(t *testing.T) {
		cfg := &Config{
			AIProvider: "ollama",
			AIBaseURL:  "http://127.0.0.1:11434/v1",
			AIModel:    "qwen2.5:1.5b",
			AIAPIKey:   "ignored",
		}
		cfg.normalizeAIProfiles()
		if len(cfg.AIProfiles) != 1 {
			t.Fatalf("expected 1 synthesised profile, got %d", len(cfg.AIProfiles))
		}
		if cfg.AIProfiles[0].Name != DefaultAIProfileName {
			t.Fatalf("name: got %q", cfg.AIProfiles[0].Name)
		}
		if cfg.AIProfiles[0].Provider != "ollama" {
			t.Fatalf("provider: got %q", cfg.AIProfiles[0].Provider)
		}
		if cfg.AIProfiles[0].Model != "qwen2.5:1.5b" {
			t.Fatalf("model: got %q", cfg.AIProfiles[0].Model)
		}
		if cfg.AIActiveProfile != DefaultAIProfileName {
			t.Fatalf("active: got %q", cfg.AIActiveProfile)
		}
	})

	t.Run("multiProfileActiveMirrorsToFlat", func(t *testing.T) {
		cfg := &Config{
			AIProfiles: []AIProfile{
				{Name: "openai", Provider: "openai_compatible", Model: "gpt-4o-mini", APIKey: "sk-a"},
				{Name: "local", Provider: "ollama", Model: "qwen2.5:1.5b", BaseURL: "http://127.0.0.1:11434/v1"},
			},
			AIActiveProfile: "local",
		}
		cfg.normalizeAIProfiles()
		if cfg.AIProvider != "ollama" {
			t.Fatalf("flat provider: got %q", cfg.AIProvider)
		}
		if cfg.AIModel != "qwen2.5:1.5b" {
			t.Fatalf("flat model: got %q", cfg.AIModel)
		}
		if cfg.AIBaseURL != "http://127.0.0.1:11434/v1" {
			t.Fatalf("flat base url: got %q", cfg.AIBaseURL)
		}
	})

	t.Run("invalidActiveFallsBackToFirst", func(t *testing.T) {
		cfg := &Config{
			AIProfiles: []AIProfile{
				{Name: "openai", Provider: "openai_compatible", Model: "gpt-4o-mini"},
				{Name: "local", Provider: "ollama", Model: "qwen2.5:1.5b"},
			},
			AIActiveProfile: "does-not-exist",
		}
		cfg.normalizeAIProfiles()
		if cfg.AIActiveProfile != "openai" {
			t.Fatalf("expected fallback to first; got %q", cfg.AIActiveProfile)
		}
	})
}

// TestAIProfileCRUD covers add/update/delete/cycle helpers.
func TestAIProfileCRUD(t *testing.T) {
	cfg := &Config{
		AIProfiles: []AIProfile{
			{Name: "a", Provider: "openai_compatible", Model: "gpt-4o-mini"},
		},
		AIActiveProfile: "a",
	}
	cfg.normalizeAIProfiles()

	if err := cfg.AddAIProfile(AIProfile{Name: "b", Provider: "ollama", Model: "qwen2.5:1.5b"}); err != nil {
		t.Fatalf("AddAIProfile: %v", err)
	}
	if err := cfg.AddAIProfile(AIProfile{Name: "a"}); err == nil {
		t.Fatal("expected error adding duplicate profile name")
	}
	if err := cfg.AddAIProfile(AIProfile{Name: ""}); err == nil {
		t.Fatal("expected error adding profile with empty name")
	}
	if err := cfg.SetActiveAIProfile("b"); err != nil {
		t.Fatalf("SetActiveAIProfile: %v", err)
	}
	if cfg.AIProvider != "ollama" {
		t.Fatalf("flat provider after set-active: %q", cfg.AIProvider)
	}

	cfg.CycleActiveAIProfile(1)
	if cfg.AIActiveProfile != "a" {
		t.Fatalf("cycle wrap: got %q", cfg.AIActiveProfile)
	}
	cfg.CycleActiveAIProfile(-1)
	if cfg.AIActiveProfile != "b" {
		t.Fatalf("cycle back: got %q", cfg.AIActiveProfile)
	}

	if err := cfg.UpdateAIProfile("b", AIProfile{Name: "b2", Provider: "ollama", Model: "llama3.2"}); err != nil {
		t.Fatalf("UpdateAIProfile rename: %v", err)
	}
	if cfg.AIActiveProfile != "b2" {
		t.Fatalf("active updated to renamed; got %q", cfg.AIActiveProfile)
	}
	if cfg.AIModel != "llama3.2" {
		t.Fatalf("flat model after edit: %q", cfg.AIModel)
	}
	if err := cfg.UpdateAIProfile("b2", AIProfile{Name: "a"}); err == nil {
		t.Fatal("expected rename collision error")
	}

	if err := cfg.DeleteAIProfile("b2"); err != nil {
		t.Fatalf("DeleteAIProfile: %v", err)
	}
	if cfg.AIActiveProfile != "a" {
		t.Fatalf("active after delete-active: %q", cfg.AIActiveProfile)
	}
	if err := cfg.DeleteAIProfile("a"); err == nil {
		t.Fatal("expected error deleting last profile")
	}
}

// TestAIProfileSaveLoadRoundTrip checks that ai_profiles + ai_active_profile
// survive a Save → loadFromFile → normalizeAIProfiles round-trip and the flat
// AI* fields are re-mirrored.
func TestAIProfileSaveLoadRoundTrip(t *testing.T) {
	dir, err := os.MkdirTemp("", "jj-tui-aiprofile-*")
	if err != nil {
		t.Fatalf("tempdir: %v", err)
	}
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "config.json")
	original := &Config{
		AIProfiles: []AIProfile{
			{Name: "fast", Provider: "openai_compatible", Model: "gpt-4o-mini", APIKey: "sk-fast"},
			{Name: "smart", Provider: "openai_compatible", Model: "gpt-4o", APIKey: "sk-smart", TimeoutSeconds: 180},
		},
		AIActiveProfile: "smart",
	}
	if err := original.SaveTo(path); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}
	loaded, err := loadFromFile(path)
	if err != nil {
		t.Fatalf("loadFromFile: %v", err)
	}
	loaded.normalizeAIProfiles()
	if len(loaded.AIProfiles) != 2 {
		t.Fatalf("profile count: %d", len(loaded.AIProfiles))
	}
	if loaded.AIActiveProfile != "smart" {
		t.Fatalf("active: %q", loaded.AIActiveProfile)
	}
	if loaded.AIModel != "gpt-4o" {
		t.Fatalf("flat model mirror: %q", loaded.AIModel)
	}
	if loaded.AITimeoutSeconds == nil || *loaded.AITimeoutSeconds != 180 {
		t.Fatalf("flat timeout mirror: %+v", loaded.AITimeoutSeconds)
	}
}

func TestGitHubTokenForAPI(t *testing.T) {
	t.Run("savedIgnoresEnv", func(t *testing.T) {
		t.Setenv("GITHUB_TOKEN", "env-token")
		cfg := &Config{GitHubToken: "cfg-token", GitHubTokenSource: GitHubTokenSourceSaved}
		cfg.loadedFrom = "/home/u/.config/jj-tui/config.json"
		tok, src := GitHubTokenForAPI(cfg)
		if tok != "cfg-token" {
			t.Fatalf("token: got %q", tok)
		}
		want := "config:/home/u/.config/jj-tui/config.json"
		if src != want {
			t.Fatalf("source: got %q want %q", src, want)
		}
	})
	t.Run("envIgnoresSaved", func(t *testing.T) {
		t.Setenv("GITHUB_TOKEN", "env-only")
		cfg := &Config{GitHubToken: "on-disk", GitHubTokenSource: GitHubTokenSourceEnv}
		tok, src := GitHubTokenForAPI(cfg)
		if tok != "env-only" || src != "env:GITHUB_TOKEN" {
			t.Fatalf("got %q %q", tok, src)
		}
	})
	t.Run("legacyInfersEnvWhenUnset", func(t *testing.T) {
		t.Setenv("GITHUB_TOKEN", "legacy-env")
		cfg := &Config{}
		if cfg.GitHubTokenSourceOrDefault() != GitHubTokenSourceEnv {
			t.Fatalf("default source: %q", cfg.GitHubTokenSourceOrDefault())
		}
		tok, src := GitHubTokenForAPI(cfg)
		if tok != "legacy-env" || src != "env:GITHUB_TOKEN" {
			t.Fatalf("got %q %q", tok, src)
		}
	})
	t.Run("nilCfgSavedEmpty", func(t *testing.T) {
		t.Setenv("GITHUB_TOKEN", "x")
		tok, src := GitHubTokenForAPI(nil)
		if tok != "" || src != "" {
			t.Fatalf("nil cfg uses saved with no token; got %q %q", tok, src)
		}
	})
	t.Run("noGhWhenPathEmpty", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("PATH=/nonexistent is not reliable for hiding gh on Windows")
		}
		t.Setenv("GITHUB_TOKEN", "")
		t.Setenv("PATH", "/nonexistent")
		cfg := &Config{GitHubTokenSource: GitHubTokenSourceGhCLI}
		tok, src := GitHubTokenForAPI(cfg)
		if tok != "" || src != "" {
			t.Fatalf("expected no gh token; got %q %q", tok, src)
		}
	})
}

