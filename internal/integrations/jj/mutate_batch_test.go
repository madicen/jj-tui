package jj_test

import (
	"context"
	"testing"

	"github.com/madicen/jj-tui/internal/integrations/jj"
	"github.com/madicen/jj-tui/internal/mock"
)

func TestAbandonCommitsBatch_singleJJCall(t *testing.T) {
	fake := &mock.FakeRunner{}
	svc := jj.NewServiceWithRunner("/fake", fake)
	ctx := context.Background()

	if err := svc.AbandonCommitsBatch(ctx, []string{"aaa", "bbb", "ccc"}); err != nil {
		t.Fatalf("AbandonCommitsBatch: %v", err)
	}
	if len(fake.Calls) != 1 {
		t.Fatalf("expected 1 jj call, got %d: %+v", len(fake.Calls), fake.Calls)
	}
	args := fake.Calls[0].Args
	if len(args) != 2 || args[0] != "abandon" || args[1] != "aaa | bbb | ccc" {
		t.Fatalf("unexpected abandon args: %v", args)
	}
}

func TestRebaseCommitsBatch_multipleRFlags(t *testing.T) {
	fake := &mock.FakeRunner{}
	svc := jj.NewServiceWithRunner("/fake", fake)
	ctx := context.Background()

	if err := svc.RebaseCommitsBatch(ctx, []string{"src1", "src2"}, "dest"); err != nil {
		t.Fatalf("RebaseCommitsBatch: %v", err)
	}
	if len(fake.Calls) != 1 {
		t.Fatalf("expected 1 jj call, got %d", len(fake.Calls))
	}
	want := []string{"rebase", "-d", "dest", "-r", "src1", "-r", "src2"}
	got := fake.Calls[0].Args
	if len(got) != len(want) {
		t.Fatalf("args len = %d, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("arg[%d] = %q, want %q (full: %v)", i, got[i], want[i], got)
		}
	}
}

func TestAbandonCommitsBatch_empty(t *testing.T) {
	svc := jj.NewServiceWithRunner("/fake", &mock.FakeRunner{})
	if err := svc.AbandonCommitsBatch(context.Background(), nil); err == nil {
		t.Fatal("expected error for empty batch")
	}
}
