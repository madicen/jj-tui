package jj

import (
	"context"
	"fmt"
	"strings"
)

// GetCurrentOperationID returns the current operation ID
func (s *Service) GetCurrentOperationID(ctx context.Context) (string, error) {
	out, err := s.runJJOutput(ctx, "op", "log", "--no-graph", "--limit", "1", "-T", "id")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// Undo undoes the last jj operation and returns the operation ID that was undone (for Redo)
func (s *Service) Undo(ctx context.Context) (string, error) {
	// Capture current op ID before undoing so we can restore to it if needed
	opID, err := s.GetCurrentOperationID(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get current op id: %w", err)
	}
	return opID, s.runJJ(ctx, "undo")
}

// Redo restores to a specific operation ID
func (s *Service) Redo(ctx context.Context, opID string) error {
	if opID == "" {
		return fmt.Errorf("no operation ID provided for redo")
	}
	return s.runJJ(ctx, "op", "restore", opID)
}
