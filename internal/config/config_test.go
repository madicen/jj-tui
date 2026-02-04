package config

import (
	"os"
	"path/filepath"
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
	defer os.Chdir(origDir)
	
	// Change to temp directory
	os.Chdir(tempDir)
	
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

