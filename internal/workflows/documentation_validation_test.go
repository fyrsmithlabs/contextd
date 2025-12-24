package workflows

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestParseValidationResponse tests parsing of agent JSON responses.
func TestParseValidationResponse(t *testing.T) {
	tests := []struct {
		name          string
		agentOutput   string
		expectValid   bool
		expectError   bool
		criticalCount int
		highCount     int
	}{
		{
			name: "valid documentation with no issues",
			agentOutput: `{
				"valid": true,
				"summary": "All documentation is accurate and up-to-date",
				"critical_issues": [],
				"high_issues": [],
				"medium_issues": [],
				"low_issues": []
			}`,
			expectValid:   true,
			expectError:   false,
			criticalCount: 0,
			highCount:     0,
		},
		{
			name: "documentation with critical issues",
			agentOutput: `{
				"valid": false,
				"summary": "Found critical documentation errors",
				"critical_issues": [
					{
						"file": ".claude-plugin/skills/checkpoint-workflow/SKILL.md",
						"line": 52,
						"severity": "critical",
						"issue": "Code example uses wrong parameter",
						"current": "checkpoint_save(session_id)",
						"should_be": "checkpoint_save(session_id, tenant_id, project_path, ...)",
						"fix": "Update code example to include all required parameters",
						"impact": "Users will get errors if they copy this example"
					}
				],
				"high_issues": [],
				"medium_issues": [],
				"low_issues": []
			}`,
			expectValid:   false,
			expectError:   false,
			criticalCount: 1,
			highCount:     0,
		},
		{
			name: "json wrapped in markdown code blocks",
			agentOutput: "```json\n" + `{
				"valid": false,
				"summary": "Found issues",
				"critical_issues": [],
				"high_issues": [
					{
						"file": "test.md",
						"issue": "Missing documentation",
						"fix": "Add docs"
					}
				],
				"medium_issues": [],
				"low_issues": []
			}` + "\n```",
			expectValid:   false,
			expectError:   false,
			criticalCount: 0,
			highCount:     1,
		},
		{
			name:        "invalid json",
			agentOutput: "this is not json",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseValidationResponse(tt.agentOutput)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectValid, result.Valid)
			assert.Len(t, result.CriticalIssues, tt.criticalCount)
			assert.Len(t, result.HighIssues, tt.highCount)
		})
	}
}

// TestBuildValidationComment tests PR comment generation.
func TestBuildValidationComment(t *testing.T) {
	t.Run("valid documentation", func(t *testing.T) {
		result := &DocumentationValidationResult{
			Valid:   true,
			Summary: "All checks passed",
		}

		comment := buildValidationComment(result)

		assert.Contains(t, comment, "✅ Documentation Validation Passed")
		assert.Contains(t, comment, "All checks passed")
	})

	t.Run("critical issues", func(t *testing.T) {
		result := &DocumentationValidationResult{
			Valid:   false,
			Summary: "Found critical issues",
			CriticalIssues: []ValidationIssue{
				{
					File:     ".claude-plugin/skills/test/SKILL.md",
					Line:     42,
					Issue:    "Wrong parameter type",
					Current:  "string",
					ShouldBe: "integer",
					Fix:      "Change type to integer",
					Impact:   "Users will get type errors",
				},
			},
		}

		comment := buildValidationComment(result)

		assert.Contains(t, comment, "⚠️ Documentation Validation Issues Found")
		assert.Contains(t, comment, "🔴 Critical Issues (MUST FIX)")
		assert.Contains(t, comment, ".claude-plugin/skills/test/SKILL.md:42")
		assert.Contains(t, comment, "Wrong parameter type")
		assert.Contains(t, comment, "Current: `string`")
		assert.Contains(t, comment, "Should be: `integer`")
	})

	t.Run("mixed severity issues", func(t *testing.T) {
		result := &DocumentationValidationResult{
			Valid:   false,
			Summary: "Found issues across multiple severity levels",
			CriticalIssues: []ValidationIssue{
				{File: "file1.md", Issue: "Critical issue", Fix: "Fix it"},
			},
			HighIssues: []ValidationIssue{
				{File: "file2.md", Issue: "High issue", Fix: "Fix it"},
			},
			MediumIssues: []ValidationIssue{
				{File: "file3.md", Issue: "Medium issue", Fix: "Fix it"},
			},
			LowIssues: []ValidationIssue{
				{File: "file4.md", Issue: "Low issue", Fix: "Fix it"},
			},
		}

		comment := buildValidationComment(result)

		assert.Contains(t, comment, "🔴 Critical Issues")
		assert.Contains(t, comment, "🟠 High Priority Issues")
		assert.Contains(t, comment, "<details>")
		assert.Contains(t, comment, "Medium and Low Priority Issues")
		assert.Contains(t, comment, "documentation-validator agent")
	})
}

// TestBuildValidationPrompt tests prompt construction.
func TestBuildValidationPrompt(t *testing.T) {
	t.Run("prompt includes all context", func(t *testing.T) {
		// This would require mocking GitHub client
		// For now, just verify the structure exists
		t.Skip("requires GitHub client mocking")
	})
}

// TestSanitizeMarkdown tests XSS prevention in PR comments.
func TestSanitizeMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "escapes HTML script tags",
			input:    `<script>alert("xss")</script>`,
			expected: `&lt;script&gt;alert(&quot;xss&quot;)&lt;/script&gt;`,
		},
		{
			name:     "escapes markdown link syntax",
			input:    `[Click me](javascript:alert(1))`,
			expected: `\[Click me\](javascript:alert(1))`,
		},
		{
			name:     "escapes backticks",
			input:    "code with `backticks` here",
			expected: "code with \\`backticks\\` here",
		},
		{
			name:     "escapes ampersands",
			input:    "A & B",
			expected: "A &amp; B",
		},
		{
			name:     "escapes single quotes",
			input:    "It's a test",
			expected: "It&#39;s a test",
		},
		{
			name:     "escapes angle brackets",
			input:    "if x < y then y > x",
			expected: "if x &lt; y then y &gt; x",
		},
		{
			name:     "escapes double quotes",
			input:    `He said "hello"`,
			expected: `He said &quot;hello&quot;`,
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "handles normal text unchanged",
			input:    "Normal text without special chars",
			expected: "Normal text without special chars",
		},
		{
			name:     "prevents XSS via img tag",
			input:    `<img src=x onerror=alert(1)>`,
			expected: `&lt;img src=x onerror=alert(1)&gt;`,
		},
		{
			name:     "escapes multiple markdown links",
			input:    `[Link1](url1) and [Link2](url2)`,
			expected: `\[Link1\](url1) and \[Link2\](url2)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeMarkdown(tt.input)
			assert.Equal(t, tt.expected, got, "sanitized output should match expected")
		})
	}
}
