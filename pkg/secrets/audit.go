package secrets

import (
	"encoding/json"
	"time"
)

// AuditLog contains the audit trail of secret redactions.
// It includes timing information and detailed redaction metadata.
type AuditLog struct {
	Timestamp  time.Time   `json:"timestamp"`
	FilePath   string      `json:"file_path,omitempty"`
	Redactions []Redaction `json:"redactions"`
	Summary    Summary     `json:"summary"`
}

// Redaction represents a single secret that was redacted.
// It never stores the actual secret value, only metadata for auditing.
type Redaction struct {
	RuleID      string `json:"rule_id"`      // e.g., "github-pat"
	RuleDesc    string `json:"rule_desc"`    // e.g., "GitHub Personal Access Token"
	LineNumber  int    `json:"line_number"`  // Line where secret was found
	Column      int    `json:"column"`       // Column where secret starts
	OriginalLen int    `json:"original_len"` // Length of redacted secret (not value)
	Preview     string `json:"preview"`      // First 4 chars only
}

// Summary provides aggregate statistics about redactions.
type Summary struct {
	TotalSecrets     int            `json:"total_secrets"`      // Total number of secrets found
	UniqueRules      int            `json:"unique_rules"`       // Number of unique rule IDs
	RuleCounts       map[string]int `json:"rule_counts"`        // Count per rule ID
	ProcessingTimeMs int64          `json:"processing_time_ms"` // Time taken in milliseconds
}

// JSON returns the audit log as a compact JSON string.
func (a *AuditLog) JSON() string {
	data, err := json.Marshal(a)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// PrettyJSON returns the audit log as a human-readable JSON string.
func (a *AuditLog) PrettyJSON() string {
	data, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(data)
}

// HasRedactions returns true if any secrets were redacted.
func (a *AuditLog) HasRedactions() bool {
	return len(a.Redactions) > 0
}
