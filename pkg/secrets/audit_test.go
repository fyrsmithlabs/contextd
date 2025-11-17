package secrets

import (
	"encoding/json"
	"testing"
	"time"
)

func TestAuditLog_JSON(t *testing.T) {
	log := AuditLog{
		Timestamp: time.Date(2025, 1, 15, 10, 30, 45, 0, time.UTC),
		FilePath:  "/home/user/project/config.yaml",
		Redactions: []Redaction{
			{
				RuleID:      "github-pat",
				RuleDesc:    "GitHub Personal Access Token",
				LineNumber:  12,
				Column:      15,
				OriginalLen: 93,
				Preview:     "ghp_",
			},
		},
		Summary: Summary{
			TotalSecrets:     1,
			UniqueRules:      1,
			RuleCounts:       map[string]int{"github-pat": 1},
			ProcessingTimeMs: 3,
		},
	}

	// Test JSON() method
	jsonStr := log.JSON()
	if jsonStr == "" {
		t.Error("JSON() returned empty string")
	}

	// Verify it's valid JSON
	var decoded AuditLog
	if err := json.Unmarshal([]byte(jsonStr), &decoded); err != nil {
		t.Errorf("JSON() produced invalid JSON: %v", err)
	}

	// Test PrettyJSON() method
	prettyJSON := log.PrettyJSON()
	if prettyJSON == "" {
		t.Error("PrettyJSON() returned empty string")
	}

	// Verify pretty JSON is also valid
	var decodedPretty AuditLog
	if err := json.Unmarshal([]byte(prettyJSON), &decodedPretty); err != nil {
		t.Errorf("PrettyJSON() produced invalid JSON: %v", err)
	}
}

func TestAuditLog_HasRedactions(t *testing.T) {
	tests := []struct {
		name string
		log  AuditLog
		want bool
	}{
		{
			name: "has redactions",
			log: AuditLog{
				Redactions: []Redaction{
					{RuleID: "test", LineNumber: 1},
				},
			},
			want: true,
		},
		{
			name: "no redactions - empty slice",
			log: AuditLog{
				Redactions: []Redaction{},
			},
			want: false,
		},
		{
			name: "no redactions - nil slice",
			log:  AuditLog{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.log.HasRedactions(); got != tt.want {
				t.Errorf("HasRedactions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuditLog_Structure(t *testing.T) {
	// Test that all fields serialize correctly
	log := AuditLog{
		Timestamp:  time.Now(),
		FilePath:   "/test/path",
		Redactions: []Redaction{},
		Summary: Summary{
			TotalSecrets: 0,
			UniqueRules:  0,
			RuleCounts:   map[string]int{},
		},
	}

	jsonBytes, err := json.Marshal(log)
	if err != nil {
		t.Fatalf("Failed to marshal AuditLog: %v", err)
	}

	var decoded AuditLog
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal AuditLog: %v", err)
	}

	// Verify timestamp preserved
	if !decoded.Timestamp.Equal(log.Timestamp) {
		t.Error("Timestamp not preserved in JSON round-trip")
	}

	// Verify file path preserved
	if decoded.FilePath != log.FilePath {
		t.Errorf("FilePath = %q, want %q", decoded.FilePath, log.FilePath)
	}
}

func TestRedaction_NoSecretValue(t *testing.T) {
	// Ensure Redaction struct never stores actual secret value
	r := Redaction{
		RuleID:      "test-rule",
		RuleDesc:    "Test Secret",
		LineNumber:  1,
		Column:      10,
		OriginalLen: 50,
		Preview:     "test",
	}

	jsonBytes, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("Failed to marshal Redaction: %v", err)
	}

	jsonStr := string(jsonBytes)
	// Verify only safe fields are in JSON
	expectedFields := []string{"rule_id", "rule_desc", "line_number", "column", "original_len", "preview"}
	for _, field := range expectedFields {
		if !contains(jsonStr, field) {
			t.Errorf("JSON missing expected field: %s", field)
		}
	}

	// Verify no "value" or "secret" field exists
	if contains(jsonStr, "\"value\"") || contains(jsonStr, "\"secret\"") {
		t.Error("JSON contains forbidden field (value/secret) - potential secret leakage")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && len(s) >= len(substr) && stringContains(s, substr)
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
