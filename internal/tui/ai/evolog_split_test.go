package ai

import (
	"strings"
	"testing"

	"github.com/madicen/jj-tui/internal/integrations/jj"
)

func TestParseEvologSplitJSON(t *testing.T) {
	entries := []jj.EvologEntry{{CommitID: "a"}, {CommitID: "b"}, {CommitID: "c"}}
	res, err := parseEvologSplitJSON(`{"recommended_index": 2, "rationale": "Groups API work", "confidence": "medium"}`, 2, entries, 99)
	if err != nil {
		t.Fatal(err)
	}
	if res.NoSplit || res.PickIndex != 2 {
		t.Fatalf("noSplit=%v pick=%d", res.NoSplit, res.PickIndex)
	}
	if res.Rationale == "" || !strings.Contains(res.Rationale, "Groups API") || !strings.Contains(res.Rationale, "medium") {
		t.Fatalf("rationale=%q", res.Rationale)
	}
}

func TestParseEvologSplitJSONFenced(t *testing.T) {
	raw := "```json\n{\"recommended_index\": 1, \"rationale\": \"x\"}\n```"
	entries := []jj.EvologEntry{{CommitID: "a"}, {CommitID: "b"}, {CommitID: "c"}}
	res, err := parseEvologSplitJSON(raw, 2, entries, 99)
	if err != nil {
		t.Fatal(err)
	}
	if res.PickIndex != 1 {
		t.Fatalf("pick=%d", res.PickIndex)
	}
}

func TestParseEvologSplitJSONOutOfRange(t *testing.T) {
	entries := []jj.EvologEntry{{CommitID: "a"}, {CommitID: "b"}, {CommitID: "c"}}
	_, err := parseEvologSplitJSON(`{"recommended_index": 9, "rationale": "nope"}`, 2, entries, 99)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseEvologSplitJSONNoSplitFlag(t *testing.T) {
	entries := []jj.EvologEntry{{CommitID: "a"}, {CommitID: "b"}}
	res, err := parseEvologSplitJSON(`{"no_split": true, "recommended_index": 0, "rationale": "one change", "confidence": "high"}`, 1, entries, 99)
	if err != nil {
		t.Fatal(err)
	}
	if !res.NoSplit || res.PickIndex != 0 {
		t.Fatalf("NoSplit=%v PickIndex=%d", res.NoSplit, res.PickIndex)
	}
	if !strings.Contains(res.Rationale, "one change") {
		t.Fatalf("rationale=%q", res.Rationale)
	}
}

func TestParseEvologSplitJSONNoSplitIndexZero(t *testing.T) {
	entries := []jj.EvologEntry{{CommitID: "a"}, {CommitID: "b"}}
	res, err := parseEvologSplitJSON(`{"no_split": false, "recommended_index": 0, "rationale": "skip"}`, 1, entries, 99)
	if err != nil {
		t.Fatal(err)
	}
	if !res.NoSplit {
		t.Fatal("expected no split from recommended_index 0")
	}
}

func TestParseEvologSplitJSONMultiBaseCommitIDs(t *testing.T) {
	entries := []jj.EvologEntry{
		{CommitID: "full11111111111111111111111111111111"},
		{CommitID: "full22222222222222222222222222222222"},
		{CommitID: "full33333333333333333333333333333333"},
	}
	raw := `{"recommended_index": 1, "rationale": "ok", "split_base_commit_ids": ["full33333333333333333333333333333333", "full22222222222222222222222222222222"]}`
	res, err := parseEvologSplitJSON(raw, 2, entries, 99)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.MultiSplitBaseCommitIDs) != 2 {
		t.Fatalf("multi=%v", res.MultiSplitBaseCommitIDs)
	}
	if res.MultiSplitBaseCommitIDs[0] != entries[2].CommitID || res.MultiSplitBaseCommitIDs[1] != entries[1].CommitID {
		t.Fatalf("order or ids wrong: %#v", res.MultiSplitBaseCommitIDs)
	}
}

func TestParseEvologSplitJSONMultiMaxParseTruncates(t *testing.T) {
	entries := []jj.EvologEntry{
		{CommitID: "full11111111111111111111111111111111"},
		{CommitID: "full22222222222222222222222222222222"},
		{CommitID: "full33333333333333333333333333333333"},
	}
	raw := `{"recommended_index": 1, "rationale": "ok", "split_base_commit_ids": ["full33333333333333333333333333333333", "full22222222222222222222222222222222"]}`
	res, err := parseEvologSplitJSON(raw, 2, entries, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.MultiSplitBaseCommitIDs) != 1 || res.MultiSplitBaseCommitIDs[0] != entries[2].CommitID {
		t.Fatalf("multi=%v", res.MultiSplitBaseCommitIDs)
	}
}

func TestParseEvologSplitJSONUnknownCommitID(t *testing.T) {
	entries := []jj.EvologEntry{{CommitID: "aaa"}, {CommitID: "bbb"}, {CommitID: "ccc"}}
	_, err := parseEvologSplitJSON(`{"recommended_index": 1, "rationale": "x", "split_base_commit_ids": ["not-in-table"]}`, 2, entries, 99)
	if err == nil || !strings.Contains(err.Error(), "unknown split_base_commit_id") {
		t.Fatalf("err=%v", err)
	}
}

func TestNormalizeEvologCommitIDPrefix(t *testing.T) {
	entries := []jj.EvologEntry{{CommitID: "deadbeef11111111111111111111111111"}}
	got := normalizeEvologCommitID("deadbeef", entries)
	if got != entries[0].CommitID {
		t.Fatalf("got=%q", got)
	}
}

func TestEvologSplitMaxMultiIDsToParse(t *testing.T) {
	if evologSplitMaxMultiIDsToParse(1) != 1 {
		t.Fatal("1 row")
	}
	if evologSplitMaxMultiIDsToParse(5) != 4 {
		t.Fatalf("5 rows: %d", evologSplitMaxMultiIDsToParse(5))
	}
}

func TestNormalizeEvologCommitIDAmbiguousPrefix(t *testing.T) {
	entries := []jj.EvologEntry{
		{CommitID: "deadbeef11111111111111111111111111"},
		{CommitID: "deadbeef22222222222222222222222222"},
	}
	if normalizeEvologCommitID("deadbeef", entries) != "" {
		t.Fatal("expected empty for ambiguous prefix")
	}
}

func TestTrimEvologUserPromptUnderCapUnchanged(t *testing.T) {
	s := strings.Repeat("a", 100)
	got := TrimEvologUserPrompt(s)
	if got != s {
		t.Fatalf("expected unchanged short string")
	}
}

func TestTrimEvologUserPromptTruncatesRunes(t *testing.T) {
	// evologSplitMaxPromptRunes is 14_000 — build something clearly over cap in runes
	s := strings.Repeat("é", evologSplitMaxPromptRunes+50) // 2 UTF-8 bytes per rune
	got := TrimEvologUserPrompt(s)
	if !strings.Contains(got, "…(truncated for size)") {
		t.Fatal("expected truncation marker")
	}
	if len([]rune(got)) > evologSplitMaxPromptRunes+30 { // small slack for trailer runes
		t.Fatalf("expected capped rune count, got %d", len([]rune(got)))
	}
}
