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
		{"john_doe", "john_doe"},
		{"john123", "john123"},
		{"testuser", "testuser"},
		{"", "local"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeIdentifier(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeIdentifier(%q) = %q, want %q", tt.input, result, tt.expected)
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
