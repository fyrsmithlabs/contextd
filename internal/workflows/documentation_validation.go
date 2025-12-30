package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Type definitions moved to types.go

// ValidateDocumentationActivity validates that plugin docs match code changes.
// TODO: Implement Claude API integration
func ValidateDocumentationActivity(ctx context.Context, input DocumentationValidationInput) (*DocumentationValidationResult, error) {
	// Stub implementation - will be guided by tests
	return &DocumentationValidationResult{
		Valid:   true,
		Summary: "TODO: Agent validation not yet implemented",
	}, nil
}

// parseValidationResponse parses agent JSON output into structured result.
func parseValidationResponse(agentOutput string) (*DocumentationValidationResult, error) {
	// Extract JSON from markdown code blocks if present
	jsonPattern := regexp.MustCompile("(?s)```json\\s*(.+?)\\s*```")
	matches := jsonPattern.FindStringSubmatch(agentOutput)
	
	var jsonStr string
	if len(matches) > 1 {
		jsonStr = matches[1]
	} else {
		jsonStr = agentOutput
	}

	var result DocumentationValidationResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse agent response: %w", err)
	}

	return &result, nil
}

// sanitizeMarkdown escapes potentially dangerous markdown/HTML in user-generated content
// to prevent XSS attacks in GitHub PR comments.
func sanitizeMarkdown(s string) string {
	// Escape HTML entities first
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	
	// Escape markdown link syntax to prevent javascript: URLs
	s = strings.ReplaceAll(s, "[", `\[`)
	s = strings.ReplaceAll(s, "]", `\]`)
	
	// Escape backticks to prevent code injection
	s = strings.ReplaceAll(s, "`", "\\`")
	
	return s
}

// buildValidationComment creates a PR comment from validation results.
func buildValidationComment(result *DocumentationValidationResult) string {
	var b strings.Builder

	if result.Valid {
		b.WriteString("## ✅ Documentation Validation Passed\n\n")
		b.WriteString(sanitizeMarkdown(result.Summary))
		b.WriteString("\n\n")
		return b.String()
	}

	b.WriteString("## ⚠️ Documentation Validation Issues Found\n\n")
	b.WriteString(sanitizeMarkdown(result.Summary))
	b.WriteString("\n\n")

	// Critical issues
	if len(result.CriticalIssues) > 0 {
		b.WriteString("### 🔴 Critical Issues (MUST FIX)\n\n")
		for i, issue := range result.CriticalIssues {
			location := issue.File
			if issue.Line > 0 {
				location = fmt.Sprintf("%s:%d", sanitizeMarkdown(issue.File), issue.Line)
			}
			b.WriteString(fmt.Sprintf("%d. **%s**\n", i+1, location))
			b.WriteString(fmt.Sprintf("   - Issue: %s\n", sanitizeMarkdown(issue.Issue)))
			if issue.Current != "" {
				b.WriteString(fmt.Sprintf("   - Current: `%s`\n", sanitizeMarkdown(issue.Current)))
			}
			if issue.ShouldBe != "" {
				b.WriteString(fmt.Sprintf("   - Should be: `%s`\n", sanitizeMarkdown(issue.ShouldBe)))
			}
			b.WriteString(fmt.Sprintf("   - Fix: %s\n", sanitizeMarkdown(issue.Fix)))
			if issue.Impact != "" {
				b.WriteString(fmt.Sprintf("   - Impact: %s\n", sanitizeMarkdown(issue.Impact)))
			}
			b.WriteString("\n")
		}
	}

	// High priority issues
	if len(result.HighIssues) > 0 {
		b.WriteString("### 🟠 High Priority Issues (SHOULD FIX)\n\n")
		for i, issue := range result.HighIssues {
			b.WriteString(fmt.Sprintf("%d. **%s**: %s\n", i+1, sanitizeMarkdown(issue.File), issue.Issue))
			b.WriteString(fmt.Sprintf("   - Fix: %s\n\n", sanitizeMarkdown(issue.Fix)))
		}
	}

	// Medium and low issues (collapsed)
	if len(result.MediumIssues) > 0 || len(result.LowIssues) > 0 {
		b.WriteString("<details>\n")
		b.WriteString("<summary>Medium and Low Priority Issues (optional improvements)</summary>\n\n")

		if len(result.MediumIssues) > 0 {
			b.WriteString("#### Medium Priority\n\n")
			for i, issue := range result.MediumIssues {
				b.WriteString(fmt.Sprintf("%d. %s: %s\n", i+1, sanitizeMarkdown(issue.File), issue.Issue))
			}
			b.WriteString("\n")
		}

		if len(result.LowIssues) > 0 {
			b.WriteString("#### Low Priority\n\n")
			for i, issue := range result.LowIssues {
				b.WriteString(fmt.Sprintf("%d. %s: %s\n", i+1, sanitizeMarkdown(issue.File), issue.Issue))
			}
			b.WriteString("\n")
		}

		b.WriteString("</details>\n\n")
	}

	b.WriteString("---\n")
	b.WriteString("🤖 Validation performed by documentation-validator agent\n")

	return b.String()
}
