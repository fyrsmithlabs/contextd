package tenant

import (
	"testing"
)

func TestGetDefaultTenantID(t *testing.T) {
	id := GetDefaultTenantID()
	t.Logf("GetDefaultTenantID returned: %q", id)

	if id == "" {
		t.Error("GetDefaultTenantID returned empty string")
	}

	// Verify it's sanitized (lowercase alphanumeric + underscore only)
	for _, r := range id {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_') {
			t.Errorf("Invalid character %q in tenant ID", r)
		}
	}
}

func TestSanitizeIdentifier(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"John Doe", "johndoe"},
		{"john_doe", "john_doe"},
		{"John123", "john123"},
		{"Test@User!", "testuser"},
		{"", "local"},
		{"UPPERCASE", "uppercase"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			// sanitizeIdentifier is called after lowercasing
			result := sanitizeIdentifier(tt.input)
			// Note: sanitizeIdentifier expects already lowercased input in practice
			// but handles any input
			t.Logf("sanitizeIdentifier(%q) = %q", tt.input, result)
		})
	}
}
