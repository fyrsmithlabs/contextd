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

func isValidChromemName(s string) bool {
	if len(s) < 1 || len(s) > MaxIdentifierLength {
		return false
	}
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}
	return true
}

// TestName_RealFailingPath reproduces the original bug: a remediation collection
// name built from the real long repo path exceeded 64 chars and hard-failed in
// chromem. After the fix the assembled name must be a valid <=64 char name.
func TestName_RealFailingPath(t *testing.T) {
	// Exact shape that previously failed: remediations_project_<tenant>_<sanitized full path>
	const failing = "remediations_project_contextd_mnt_c_users_dusti_projects_fyrsmithlabs_contextd"
	if len(failing) <= MaxIdentifierLength {
		t.Fatalf("test precondition: input should exceed %d chars, got %d", MaxIdentifierLength, len(failing))
	}

	got := Name(failing)
	if !isValidChromemName(got) {
		t.Errorf("Name(%q) = %q is not a valid chromem name (len=%d)", failing, got, len(got))
	}
}

func TestName_LongPathIsValid(t *testing.T) {
	long := "remediations_project_org_" + strings.Repeat("a", 100)
	got := Name(long)
	if !isValidChromemName(got) {
		t.Errorf("Name produced invalid/too-long name: %q (len=%d)", got, len(got))
	}
}

func TestName_DistinctLongPathsDistinctNames(t *testing.T) {
	base := "remediations_project_contextd_mnt_c_users_dusti_projects_fyrsmithlabs_"
	a := Name(base + "projecta_contextd")
	b := Name(base + "projectb_contextd")

	if a == b {
		t.Errorf("distinct long inputs collided to the same name: %q", a)
	}
	if !isValidChromemName(a) || !isValidChromemName(b) {
		t.Errorf("names not valid: a=%q b=%q", a, b)
	}
}

func TestName_ShortPathUnchanged(t *testing.T) {
	// A normal, already-valid short name must pass through unchanged so existing
	// collections remain addressable (no regression in addressing existing data).
	cases := []string{
		"remediations_project_contextd_simple_ctl",
		"my_project_memories",
		"remediations_org_contextd",
	}
	for _, c := range cases {
		if got := Name(c); got != c {
			t.Errorf("Name(%q) changed valid short name to %q", c, got)
		}
	}
}

func TestName_Deterministic(t *testing.T) {
	long := "remediations_project_" + strings.Repeat("x", 80)
	if Name(long) != Name(long) {
		t.Error("Name is not deterministic for the same input")
	}
}

func TestName_InvalidCharsReplaced(t *testing.T) {
	got := Name("Remediations/Project:Contextd!")
	if !isValidChromemName(got) {
		t.Errorf("Name did not produce valid name: %q", got)
	}
}

func TestName_EmptyReturnsDefault(t *testing.T) {
	if got := Name(""); got != DefaultIdentifier {
		t.Errorf("Name(\"\") = %q, want %q", got, DefaultIdentifier)
	}
}
