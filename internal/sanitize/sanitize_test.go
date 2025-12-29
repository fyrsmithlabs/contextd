package sanitize

import (
	"strings"
	"testing"
)

func TestIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple lowercase",
			input:    "myproject",
			expected: "myproject",
		},
		{
			name:     "uppercase conversion",
			input:    "MyProject",
			expected: "myproject",
		},
		{
			name:     "dots to underscores",
			input:    "github.com",
			expected: "github_com",
		},
		{
			name:     "slashes to underscores",
			input:    "user/repo",
			expected: "user_repo",
		},
		{
			name:     "github remote URL pattern",
			input:    "github.com/dahendel",
			expected: "github_com_dahendel",
		},
		{
			name:     "special characters",
			input:    "my-project!@#$%",
			expected: "my_project",
		},
		{
			name:     "multiple underscores collapsed",
			input:    "foo___bar",
			expected: "foo_bar",
		},
		{
			name:     "leading/trailing underscores trimmed",
			input:    "_foo_bar_",
			expected: "foo_bar",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "default",
		},
		{
			name:     "only invalid chars",
			input:    "!!!",
			expected: "default",
		},
		{
			name:     "numbers preserved",
			input:    "project123",
			expected: "project123",
		},
		{
			name:     "underscores preserved",
			input:    "my_project",
			expected: "my_project",
		},
		{
			name:     "spaces to underscores",
			input:    "my project",
			expected: "my_project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Identifier(tt.input)
			if result != tt.expected {
				t.Errorf("Identifier(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIdentifier_LengthLimit(t *testing.T) {
	// Test that long identifiers are truncated with hash
	longInput := strings.Repeat("a", 100)
	result := Identifier(longInput)

	if len(result) > MaxIdentifierLength {
		t.Errorf("Identifier should be <= %d chars, got %d", MaxIdentifierLength, len(result))
	}

	// Should end with hash suffix pattern _XXXXXXXX
	if !strings.Contains(result, "_") {
		t.Error("Truncated identifier should contain hash suffix")
	}
}

func TestIdentifier_LengthLimit_Uniqueness(t *testing.T) {
	// Different long inputs should produce different outputs
	input1 := strings.Repeat("a", 100)
	input2 := strings.Repeat("a", 99) + "b"

	result1 := Identifier(input1)
	result2 := Identifier(input2)

	if result1 == result2 {
		t.Error("Different inputs should produce different hashed outputs")
	}
}

func TestIdentifier_ExactlyMaxLength(t *testing.T) {
	// Input exactly at max length should not be truncated
	input := strings.Repeat("a", MaxIdentifierLength)
	result := Identifier(input)

	if result != input {
		t.Errorf("Input at max length should not be modified, got %q", result)
	}
}

func TestCollectionName(t *testing.T) {
	tests := []struct {
		name     string
		tenant   string
		project  string
		suffix   string
		expected string
	}{
		{
			name:     "simple components",
			tenant:   "user",
			project:  "project",
			suffix:   "codebase",
			expected: "user_project_codebase",
		},
		{
			name:     "github tenant",
			tenant:   "github.com/dahendel",
			project:  "contextd",
			suffix:   "codebase",
			expected: "github_com_dahendel_contextd_codebase",
		},
		{
			name:     "no suffix",
			tenant:   "user",
			project:  "project",
			suffix:   "",
			expected: "user_project",
		},
		{
			name:     "sanitization applied",
			tenant:   "My-Tenant!",
			project:  "My Project",
			suffix:   "memories",
			expected: "my_tenant_my_project_memories",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CollectionName(tt.tenant, tt.project, tt.suffix)
			if result != tt.expected {
				t.Errorf("CollectionName(%q, %q, %q) = %q, want %q",
					tt.tenant, tt.project, tt.suffix, result, tt.expected)
			}
		})
	}
}

func TestCollectionName_LengthLimit(t *testing.T) {
	// Very long tenant + project should still produce valid collection name
	longTenant := strings.Repeat("a", 50)
	longProject := strings.Repeat("b", 50)

	result := CollectionName(longTenant, longProject, "codebase")

	if len(result) > MaxIdentifierLength {
		t.Errorf("CollectionName should be <= %d chars, got %d", MaxIdentifierLength, len(result))
	}
}

func TestCollectionName_ValidChars(t *testing.T) {
	// Result should only contain valid chars
	result := CollectionName("github.com/user", "my-project!", "test")

	for _, r := range result {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_') {
			t.Errorf("CollectionName contains invalid char %q in %q", string(r), result)
		}
	}
}
