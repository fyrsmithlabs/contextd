package secrets

import "time"

// Result contains the scrubbing result.
type Result struct {
	// Original is the original input content
	Original string `json:"-"`

	// Scrubbed is the content with secrets redacted
	Scrubbed string `json:"scrubbed"`

	// Findings contains the detected secrets (without actual values)
	Findings []Finding `json:"findings,omitempty"`

	// Duration is how long scrubbing took
	Duration time.Duration `json:"duration"`

	// TotalFindings is the count of secrets found
	TotalFindings int `json:"total_findings"`

	// ByRule maps rule IDs to finding counts
	ByRule map[string]int `json:"by_rule,omitempty"`
}

// Finding represents a detected secret.
type Finding struct {
	// RuleID identifies which rule matched
	RuleID string `json:"rule_id"`

	// Description explains what was found
	Description string `json:"description"`

	// Severity indicates the importance
	Severity string `json:"severity"`

	// StartIndex is the start position in original content
	StartIndex int `json:"start_index"`

	// EndIndex is the end position in original content
	EndIndex int `json:"end_index"`

	// Line is the line number (1-indexed)
	Line int `json:"line,omitempty"`

	// Match is NOT included to avoid leaking the secret
}

// HasFindings returns true if any secrets were found.
func (r *Result) HasFindings() bool {
	return r.TotalFindings > 0
}

// FindingsBySeverity returns findings filtered by severity.
func (r *Result) FindingsBySeverity(severity string) []Finding {
	var filtered []Finding
	for _, f := range r.Findings {
		if f.Severity == severity {
			filtered = append(filtered, f)
		}
	}
	return filtered
}

// RuleIDs returns the unique rule IDs that matched.
func (r *Result) RuleIDs() []string {
	ids := make([]string, 0, len(r.ByRule))
	for id := range r.ByRule {
		ids = append(ids, id)
	}
	return ids
}

// Summary returns a brief summary of findings.
func (r *Result) Summary() string {
	if !r.HasFindings() {
		return "no secrets detected"
	}

	high := len(r.FindingsBySeverity("high"))
	medium := len(r.FindingsBySeverity("medium"))
	low := len(r.FindingsBySeverity("low"))

	if high > 0 {
		return "secrets redacted (high severity)"
	}
	if medium > 0 {
		return "secrets redacted (medium severity)"
	}
	if low > 0 {
		return "secrets redacted (low severity)"
	}
	return "secrets redacted"
}
