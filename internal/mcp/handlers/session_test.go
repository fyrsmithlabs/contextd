// internal/mcp/handlers/session_test.go
package handlers

import (
	"encoding/json"
	"testing"
)

func TestSessionStartInput(t *testing.T) {
	input := `{"project_id": "test-project", "session_id": "sess-123"}`
	var req SessionStartInput
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if req.ProjectID != "test-project" {
		t.Errorf("expected project_id=test-project, got %s", req.ProjectID)
	}
	if req.SessionID != "sess-123" {
		t.Errorf("expected session_id=sess-123, got %s", req.SessionID)
	}
}

func TestSessionEndInput(t *testing.T) {
	input := `{
		"project_id": "test-project",
		"session_id": "sess-123",
		"task": "Implement feature X",
		"approach": "TDD with mocks",
		"outcome": "success",
		"tags": ["go", "tdd"]
	}`
	var req SessionEndInput
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if req.Task != "Implement feature X" {
		t.Errorf("unexpected task: %s", req.Task)
	}
	if req.Outcome != "success" {
		t.Errorf("unexpected outcome: %s", req.Outcome)
	}
	if len(req.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(req.Tags))
	}
}

func TestContextThresholdInput(t *testing.T) {
	input := `{"project_id": "test-project", "session_id": "sess-123", "percent": 75}`
	var req ContextThresholdInput
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if req.Percent != 75 {
		t.Errorf("expected percent=75, got %d", req.Percent)
	}
}
