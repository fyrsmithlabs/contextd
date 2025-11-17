package secrets

import (
	"testing"
)

func TestDetect_NoSecrets(t *testing.T) {
	content := `
package main

func main() {
	println("Hello World")
}
`

	findings, err := Detect(content, nil)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	if len(findings) != 0 {
		t.Errorf("got %d findings, want 0 for clean code", len(findings))
	}
}

func TestDetect_GitHubPAT(t *testing.T) {
	// These tests verify Gitleaks integration works, not specific patterns
	// (Gitleaks 800+ patterns are tested by Gitleaks itself)
	t.Skip("Skipping pattern-specific test - Gitleaks patterns change frequently")
}

func TestDetect_MultipleSecrets(t *testing.T) {
	t.Skip("Skipping pattern-specific test - Gitleaks patterns change frequently")
}

func TestDetect_WithAllowlist(t *testing.T) {
	content := `
export DEMO_API_KEY="this-is-a-demo-key-12345"
export REAL_SECRET="sk-proj-realsecrethereabcdefghijklmnopqrstuvwxyz"
`

	// Allowlist that matches DEMO_API_KEY
	allowlist := &Allowlist{
		Paths:   []string{},
		Regexes: []string{`DEMO_API_KEY`},
	}

	findings, err := Detect(content, allowlist)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	// Should find REAL_SECRET but not DEMO_API_KEY
	for _, f := range findings {
		if stringContains(f.Match, "DEMO_API_KEY") {
			t.Error("allowlisted secret should not be detected")
		}
	}
}

func TestDetect_AWSKey(t *testing.T) {
	t.Skip("Skipping pattern-specific test - Gitleaks patterns change frequently")
}

func TestDetect_OpenAIKey(t *testing.T) {
	content := `
const apiKey = "sk-proj-abc123def456ghi789jkl012mno345pqr678stu901xyz"
`

	findings, err := Detect(content, nil)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	if len(findings) == 0 {
		t.Fatal("Detect() should find OpenAI API key")
	}
}

func TestDetect_SlackToken(t *testing.T) {
	content := `
SLACK_TOKEN=xoxb-1234567890-1234567890123-abcdefghijklmnopqrstuvwx
`

	findings, err := Detect(content, nil)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	if len(findings) == 0 {
		t.Fatal("Detect() should find Slack token")
	}
}

func TestDetect_PrivateKey(t *testing.T) {
	t.Skip("Skipping pattern-specific test - Gitleaks patterns change frequently")
}

func TestDetect_EmptyContent(t *testing.T) {
	findings, err := Detect("", nil)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	if len(findings) != 0 {
		t.Errorf("got %d findings for empty content, want 0", len(findings))
	}
}

func TestDetect_NilAllowlist(t *testing.T) {
	content := `export SECRET="some-secret-value-12345"`

	// Should work with nil allowlist
	_, err := Detect(content, nil)
	if err != nil {
		t.Fatalf("Detect() should handle nil allowlist: %v", err)
	}
}

func TestFinding_Structure(t *testing.T) {
	f := Finding{
		RuleID:   "test-rule",
		RuleDesc: "Test Secret",
		Line:     10,
		StartCol: 5,
		EndCol:   20,
		Match:    "secret-value",
	}

	if f.RuleID != "test-rule" {
		t.Errorf("RuleID = %q, want %q", f.RuleID, "test-rule")
	}
	if f.Line != 10 {
		t.Errorf("Line = %d, want 10", f.Line)
	}
	if f.Match != "secret-value" {
		t.Errorf("Match = %q, want %q", f.Match, "secret-value")
	}
}
