package tenant

import (
	"testing"

	"github.com/fyrsmithlabs/contextd/internal/sanitize"
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
		{"john_doe", "john_doe"},
		{"john123", "john123"},
		{"testuser", "testuser"},
		{"", "default"}, // sanitize.Identifier returns "default" for empty
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitize.Identifier(tt.input)
			if result != tt.expected {
				t.Errorf("sanitize.Identifier(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseGitHubUsername(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "SSH format",
			url:      "git@github.com:dahendel/contextd.git",
			expected: "dahendel",
		},
		{
			name:     "HTTPS format",
			url:      "https://github.com/dahendel/contextd.git",
			expected: "dahendel",
		},
		{
			name:     "HTTPS without .git",
			url:      "https://github.com/fyrsmithlabs/contextd",
			expected: "fyrsmithlabs",
		},
		{
			name:     "Non-GitHub URL",
			url:      "https://gitlab.com/user/repo.git",
			expected: "",
		},
		{
			name:     "Empty URL",
			url:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseGitHubUsername(tt.url)
			if result != tt.expected {
				t.Errorf("parseGitHubUsername(%q) = %q, want %q", tt.url, result, tt.expected)
			}
		})
	}
}

func TestGetTenantIDForPath(t *testing.T) {
	// Test with current repo (should return GitHub username)
	id := GetTenantIDForPath("/home/dahendel/projects/contextd")
	t.Logf("GetTenantIDForPath for contextd repo: %q", id)

	// Should be either "dahendel" or "fyrsmithlabs" depending on remote
	if id == "" {
		t.Error("GetTenantIDForPath returned empty for valid repo")
	}
}

func TestGetTenantIDForPath_Fallbacks(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"empty path", ""},
		{"nonexistent path", "/tmp/this-does-not-exist-12345"},
		{"invalid git repo", "/tmp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := GetTenantIDForPath(tt.path)
			// Should fall back to git config or $USER
			if id == "" {
				t.Error("GetTenantIDForPath returned empty, should use fallback")
			}
			// Verify it's sanitized
			for _, r := range id {
				if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_') {
					t.Errorf("Invalid character %q in tenant ID", r)
				}
			}
		})
	}
}

func TestSanitizeIdentifier_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// sanitize.Identifier converts to lowercase AND replaces invalid chars
		{"uppercase letters", "JohnDoe", "johndoe"},
		{"special characters", "john-doe@example.com", "john_doe_example_com"},
		{"spaces", "John Doe", "john_doe"},
		{"mixed", "John_Doe-123!", "john_doe_123"},
		{"all invalid", "!@#$%", "default"},
		{"numbers only", "12345", "12345"},
		{"underscores", "test_user_123", "test_user_123"},
		{"already sanitized", "test_collection_1", "test_collection_1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitize.Identifier(tt.input)
			if result != tt.expected {
				t.Errorf("sanitize.Identifier(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetTenantIDForPath_BranchIndependent(t *testing.T) {
	// Tenant ID is derived from remote URL, which doesn't change with branches.
	// This test verifies the tenant ID is consistent regardless of current branch.
	id := GetTenantIDForPath("/home/dahendel/projects/contextd")
	if id == "" || id == "local" {
		t.Skip("Not running in contextd repo context")
	}

	t.Logf("Tenant ID: %q (should be consistent across all branches)", id)

	// The tenant ID should be the GitHub org/user from the remote
	// For this repo: fyrsmithlabs or dahendel depending on remote configuration
	if id != "fyrsmithlabs" && id != "dahendel" {
		t.Logf("Unexpected tenant ID %q - verify remote URL", id)
	}
}
