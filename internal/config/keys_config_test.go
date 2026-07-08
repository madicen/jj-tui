package config

import (
	"encoding/json"
	"testing"
)

// TestKeysConfigRoundTrip verifies the optional "keys" override map survives a
// load→save→load cycle (P5.1).
func TestKeysConfigRoundTrip(t *testing.T) {
	const withKeys = `{"github_token":"x","keys":{"graph.abandon":"x","branches.push":"p"}}`

	var cfg Config
	if err := json.Unmarshal([]byte(withKeys), &cfg); err != nil {
		t.Fatalf("failed to unmarshal config with keys: %v", err)
	}
	if cfg.Keys["graph.abandon"] != "x" || cfg.Keys["branches.push"] != "p" {
		t.Fatalf("keys not parsed: %v", cfg.Keys)
	}

	out, err := json.Marshal(&cfg)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var round map[string]json.RawMessage
	if err := json.Unmarshal(out, &round); err != nil {
		t.Fatalf("re-marshaled config invalid: %v", err)
	}
	if _, ok := round["keys"]; !ok {
		t.Error("keys map should be emitted after round-trip")
	}
}

// TestKeysAbsentStaysAbsent verifies old config files (no "keys") load with a
// nil map and don't gain a "keys" field on re-marshal (byte-compat).
func TestKeysAbsentStaysAbsent(t *testing.T) {
	const old = `{"github_token":"x"}`
	var cfg Config
	if err := json.Unmarshal([]byte(old), &cfg); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if cfg.Keys != nil {
		t.Errorf("expected nil Keys for a config without the field, got %v", cfg.Keys)
	}
	out, err := json.Marshal(&cfg)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var round map[string]json.RawMessage
	if err := json.Unmarshal(out, &round); err != nil {
		t.Fatalf("re-marshaled config invalid: %v", err)
	}
	if _, ok := round["keys"]; ok {
		t.Error("keys must not be emitted for a config that never set it")
	}
}

// TestKeysMergePerKey verifies local config rebinds merge onto the global map
// per key rather than replacing it wholesale (P5.1: global + per-repo).
func TestKeysMergePerKey(t *testing.T) {
	global := &Config{KeysConfig: KeysConfig{Keys: map[string]string{
		"graph.abandon": "x",
		"graph.squash":  "S",
	}}}
	local := &Config{KeysConfig: KeysConfig{Keys: map[string]string{
		"graph.squash": "z", // override just this one
		"prs.merge":    "g",
	}}}

	mergeConfig(global, local)

	if global.Keys["graph.abandon"] != "x" {
		t.Error("global-only key should survive merge")
	}
	if global.Keys["graph.squash"] != "z" {
		t.Error("local should override the shared key")
	}
	if global.Keys["prs.merge"] != "g" {
		t.Error("local-only key should be added")
	}
}
