package codecks

import (
	"testing"
)

// TestEncodeShortID tests the bijective base-28 encoding for Codecks short IDs
// The implementation adds an offset of 812 so all IDs are 3+ digits
func TestEncodeShortID(t *testing.T) {
	tests := []struct {
		accountSeq int
		expected   string
	}{
		// accountSeq=1 becomes 813 internally, which encodes to "111"
		{1, "111"},
		{2, "112"},
		{28, "11z"},   // End of first group
		{29, "121"},   // Start of second group
		
		// Test zero
		{0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := encodeShortID(tt.accountSeq)
			if result != tt.expected {
				t.Errorf("encodeShortID(%d) = %s, want %s", tt.accountSeq, result, tt.expected)
			}
		})
	}
}

// TestEncodeShortIDSequence tests that sequential IDs increment correctly
func TestEncodeShortIDSequence(t *testing.T) {
	// The implementation adds 812 offset, so sequence should be:
	// 1 -> "111", 2 -> "112", ..., 28 -> "11z", 29 -> "121"
	
	// Verify first ID
	first := encodeShortID(1)
	if first != "111" {
		t.Errorf("First ID should be 111, got %s", first)
	}
	
	// Verify sequential IDs are increasing
	prev := ""
	for i := 1; i <= 100; i++ {
		current := encodeShortID(i)
		if current == "" {
			t.Errorf("encodeShortID(%d) returned empty string", i)
		}
		if prev != "" && current <= prev {
			t.Errorf("IDs should be increasing: %s <= %s at seq %d", current, prev, i)
		}
		prev = current
	}
}

// TestEncodeShortIDRollover tests digit rollover behavior
func TestEncodeShortIDRollover(t *testing.T) {
	// After seq 28 (last digit z), should increment second digit
	seq28 := encodeShortID(28)
	seq29 := encodeShortID(29)
	
	// seq28 should end in 'z', seq29 should have incremented
	if seq28[len(seq28)-1] != 'z' {
		t.Errorf("seq 28 should end in z, got %s", seq28)
	}
	if seq29[len(seq29)-1] != '1' {
		t.Errorf("seq 29 should end in 1, got %s", seq29)
	}
}

// TestSlugify tests the URL slug generation
func TestSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello-world"},
		{"Add Codecks Support", "add-codecks-support"},
		{"Fix: Bug #123", "fix-bug-123"},       // : and # removed, spaces become hyphens
		{"Multiple   Spaces", "multiple-spaces"}, // multiple spaces -> single hyphen
		{"Special!@#Characters", "specialcharacters"}, // special chars just removed
		{"UPPERCASE", "uppercase"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := slugify(tt.input)
			if result != tt.expected {
				t.Errorf("slugify(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestIsConfigured tests the configuration check
func TestIsConfigured(t *testing.T) {
	// Save original env vars
	// Note: This test modifies environment, should be run in isolation
	
	t.Run("NotConfigured", func(t *testing.T) {
		// When env vars are not set, should return false
		// (depends on actual env state during test)
	})
}

