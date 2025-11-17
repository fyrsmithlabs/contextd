package secrets

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// RedactOptions configures the redaction operation.
type RedactOptions struct {
	ProjectPath string // Path to project (for .gitleaks.toml)
	UserPath    string // Path to user allowlist.toml
}

// RedactResult contains redacted content and audit information.
type RedactResult struct {
	Content string   // Redacted content with markers
	Audit   AuditLog // Audit trail of redactions
}

// Redact detects and redacts secrets from content using Gitleaks SDK.
// Returns redacted content with [REDACTED:rule-id:preview] markers and audit log.
//
// The function:
//  1. Loads allowlists (project and/or user)
//  2. Detects secrets using Gitleaks SDK
//  3. Replaces secrets with redaction markers
//  4. Returns audit log with timing and redaction details
//
// Redaction markers preserve semantic context for embeddings while hiding actual secrets.
func Redact(content string, opts RedactOptions) (RedactResult, error) {
	startTime := time.Now()

	// Load allowlists (missing files are silently ignored)
	allowlist, err := LoadAllowlists(opts.ProjectPath, opts.UserPath)
	if err != nil {
		return RedactResult{}, fmt.Errorf("loading allowlists: %w", err)
	}

	// Detect secrets
	findings, err := Detect(content, allowlist)
	if err != nil {
		return RedactResult{}, fmt.Errorf("detecting secrets: %w", err)
	}

	// Build audit log
	audit := buildAuditLog(findings, time.Since(startTime))

	// If no secrets found, return original content
	if len(findings) == 0 {
		return RedactResult{
			Content: content,
			Audit:   audit,
		}, nil
	}

	// Redact secrets
	redacted := replaceFindings(content, findings)

	return RedactResult{
		Content: redacted,
		Audit:   audit,
	}, nil
}

// replaceFindings replaces secrets with redaction markers.
// Works backwards through findings to preserve string indices.
//
// Thread Safety: This function is NOT safe for concurrent calls with the same
// findings slice. The internal sort modifies the slice in place. However, normal
// usage is safe because Redact() creates a fresh findings slice on each call.
func replaceFindings(content string, findings []Finding) string {
	// Sort findings by position (reverse order to preserve indices)
	sorted := make([]Finding, len(findings))
	copy(sorted, findings)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Line != sorted[j].Line {
			return sorted[i].Line > sorted[j].Line
		}
		return sorted[i].StartCol > sorted[j].StartCol
	})

	lines := strings.Split(content, "\n")

	for _, finding := range sorted {
		if finding.Line < 1 || finding.Line > len(lines) {
			continue // Skip invalid line numbers
		}

		line := lines[finding.Line-1]

		// Generate marker
		preview := extractPreview(finding.Match, 4)
		marker := fmt.Sprintf("[REDACTED:%s:%s]", finding.RuleID, preview)

		// Replace secret with marker
		if finding.StartCol >= 0 && finding.EndCol <= len(line) {
			before := line[:finding.StartCol]
			after := line[finding.EndCol:]
			lines[finding.Line-1] = before + marker + after
		}
	}

	return strings.Join(lines, "\n")
}

// extractPreview returns the first N characters of a string as a preview.
func extractPreview(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// buildAuditLog constructs an audit log from findings and timing information.
func buildAuditLog(findings []Finding, processingTime time.Duration) AuditLog {
	redactions := make([]Redaction, 0, len(findings))
	ruleCounts := make(map[string]int)
	uniqueRules := make(map[string]struct{})

	for _, f := range findings {
		redactions = append(redactions, Redaction{
			RuleID:      f.RuleID,
			RuleDesc:    f.RuleDesc,
			LineNumber:  f.Line,
			Column:      f.StartCol,
			OriginalLen: len(f.Match),
			Preview:     extractPreview(f.Match, 4),
		})

		ruleCounts[f.RuleID]++
		uniqueRules[f.RuleID] = struct{}{}
	}

	return AuditLog{
		Timestamp:  time.Now(),
		Redactions: redactions,
		Summary: Summary{
			TotalSecrets:     len(findings),
			UniqueRules:      len(uniqueRules),
			RuleCounts:       ruleCounts,
			ProcessingTimeMs: processingTime.Milliseconds(),
		},
	}
}
