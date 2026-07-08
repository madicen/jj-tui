package config

import (
	"encoding/json"
	"reflect"
	"testing"
)

// goldenConfigJSON mirrors the on-disk schema of today's config files (the repo's
// own .jj-tui.json is a live sample) with sanitized placeholder values. It covers
// every JSON key so the round-trip test fails loudly if the embedded sub-struct
// refactor (P2.6) ever renames or drops a key.
const goldenConfigJSON = `{
  "github_token": "ghp_placeholder",
  "github_token_source": "gh_cli",
  "github_auth_method": "token",
  "github_show_merged": true,
  "github_show_closed": false,
  "github_only_mine": true,
  "github_pr_limit": 25,
  "github_refresh_interval": 600,
  "github_issues_excluded_statuses": "closed",
  "ticket_provider": "codecks",
  "jira_url": "https://example.atlassian.net",
  "jira_user": "user@example.com",
  "jira_token": "jira_placeholder",
  "jira_project": "APP",
  "jira_project_filter": "APP,TEAM",
  "jira_issue_type": "Task",
  "jira_jql": "sprint in openSprints()",
  "jira_excluded_statuses": "Won't Fix, Will Not Do",
  "codecks_subdomain": "example",
  "codecks_token": "codecks_placeholder",
  "codecks_project": "Personal Projects",
  "codecks_excluded_statuses": "done",
  "ticket_auto_in_progress": true,
  "branch_limit": 20,
  "sanitize_bookmark_names": true,
  "branches_show_all_remotes": false,
  "graph_revset": "trunk() | ancestors(@)",
  "graph_show_everyones_commits": false,
  "confirm_destructive": true,
  "external_file_editor": "cursor",
  "external_file_editor_custom": "cursor -g {path}",
  "theme_primary": "#9529be",
  "theme_secondary": "#50FA7B",
  "theme_muted": "#6272A4",
  "ai_enabled": true,
  "ai_base_url": "http://127.0.0.1:11434/v1",
  "ai_model": "qwen2.5-coder:7b",
  "ai_timeout_seconds": 150,
  "ai_provider": "ollama",
  "ai_api_key": "ai_placeholder",
  "ai_profiles": [
    {
      "name": "default",
      "provider": "ollama",
      "base_url": "http://127.0.0.1:11434/v1",
      "model": "qwen2.5-coder:7b",
      "timeout_seconds": 150
    }
  ],
  "ai_active_profile": "default",
  "ai_evolog_describe_after_split_default": true,
  "ai_evolog_file_split_enabled": true,
  "ai_evolog_hunk_split_enabled": true,
  "ai_evolog_multi_split_max": 128,
  "ai_evolog_multi_split_mode": "stepwise"
}`

// TestConfigGoldenRoundTrip loads a full-schema config, re-marshals it, and
// verifies (1) every original JSON key survives the embedded-sub-struct layout
// and (2) the config is semantically identical after a load→save→load cycle.
// This guards P2.6's "old config files load byte-compatibly" acceptance.
func TestConfigGoldenRoundTrip(t *testing.T) {
	var original map[string]json.RawMessage
	if err := json.Unmarshal([]byte(goldenConfigJSON), &original); err != nil {
		t.Fatalf("golden fixture is not valid JSON: %v", err)
	}

	var cfg Config
	if err := json.Unmarshal([]byte(goldenConfigJSON), &cfg); err != nil {
		t.Fatalf("failed to unmarshal golden config into Config: %v", err)
	}

	out, err := json.Marshal(&cfg)
	if err != nil {
		t.Fatalf("failed to marshal Config: %v", err)
	}

	var round map[string]json.RawMessage
	if err := json.Unmarshal(out, &round); err != nil {
		t.Fatalf("re-marshaled config is not valid JSON: %v", err)
	}

	// Every original key must still be present with an equal value (order-agnostic).
	for key, want := range original {
		got, ok := round[key]
		if !ok {
			t.Errorf("key %q was dropped by the sub-struct refactor", key)
			continue
		}
		if !jsonEqual(t, want, got) {
			t.Errorf("key %q changed value: want %s, got %s", key, want, got)
		}
	}
	// And no unexpected new keys should appear (the internal loadedFrom is json:"-").
	for key := range round {
		if _, ok := original[key]; !ok {
			t.Errorf("unexpected new key %q in serialized config", key)
		}
	}

	// Full semantic round-trip: load → marshal → unmarshal must be deep-equal.
	var reloaded Config
	if err := json.Unmarshal(out, &reloaded); err != nil {
		t.Fatalf("failed to reload serialized config: %v", err)
	}
	if !reflect.DeepEqual(cfg, reloaded) {
		t.Fatalf("config not stable across round-trip:\n first: %+v\nsecond: %+v", cfg, reloaded)
	}
}

// TestConfirmDestructiveDefaultsOnForOldConfigs verifies that a config file written before the
// confirm_destructive key existed (i.e. without the key) still defaults to prompting, and that
// re-marshaling such a config does not inject the key (omitempty keeps old files byte-compatible).
func TestConfirmDestructiveDefaultsOnForOldConfigs(t *testing.T) {
	const oldSchema = `{"github_token":"x","graph_revset":"trunk()"}`

	var cfg Config
	if err := json.Unmarshal([]byte(oldSchema), &cfg); err != nil {
		t.Fatalf("failed to unmarshal old-schema config: %v", err)
	}
	if cfg.ConfirmDestructive != nil {
		t.Errorf("expected ConfirmDestructive to be nil for a file without the key, got %v", *cfg.ConfirmDestructive)
	}
	if !cfg.ConfirmDestructiveOps() {
		t.Error("ConfirmDestructiveOps() must default to true when the key is absent")
	}

	out, err := json.Marshal(&cfg)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}
	var round map[string]json.RawMessage
	if err := json.Unmarshal(out, &round); err != nil {
		t.Fatalf("re-marshaled config is not valid JSON: %v", err)
	}
	if _, ok := round["confirm_destructive"]; ok {
		t.Error("confirm_destructive must not be emitted for a config that never set it (breaks byte-compat)")
	}

	// And an explicit false disables the prompt.
	off := false
	cfg.ConfirmDestructive = &off
	if cfg.ConfirmDestructiveOps() {
		t.Error("ConfirmDestructiveOps() must return false when the toggle is explicitly off")
	}
}

func jsonEqual(t *testing.T, a, b json.RawMessage) bool {
	t.Helper()
	var av, bv any
	if err := json.Unmarshal(a, &av); err != nil {
		return false
	}
	if err := json.Unmarshal(b, &bv); err != nil {
		return false
	}
	return reflect.DeepEqual(av, bv)
}
