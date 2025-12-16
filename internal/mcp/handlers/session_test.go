// internal/mcp/handlers/session_test.go
package handlers

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/checkpoint"
	"github.com/fyrsmithlabs/contextd/internal/compression"
	"github.com/fyrsmithlabs/contextd/internal/hooks"
	"github.com/fyrsmithlabs/contextd/internal/reasoningbank"
	"github.com/fyrsmithlabs/contextd/internal/remediation"
	"github.com/fyrsmithlabs/contextd/internal/repository"
	"github.com/fyrsmithlabs/contextd/internal/secrets"
	"github.com/fyrsmithlabs/contextd/internal/troubleshoot"
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

func TestSessionHandler_Start(t *testing.T) {
	// Create mock registry with checkpoint
	now := time.Now()
	mockReg := &mockRegistry{
		checkpoints: []checkpointRecord{
			{ID: "cp-1", Summary: "Previous work", CreatedAt: now},
		},
	}
	handler := NewSessionHandler(mockReg)

	input := json.RawMessage(`{"project_id": "test-project", "session_id": "sess-123"}`)
	result, err := handler.Start(context.Background(), input)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	output, ok := result.(*SessionStartOutput)
	if !ok {
		t.Fatalf("unexpected result type: %T", result)
	}

	// Should have checkpoint offer
	if output.Checkpoint == nil {
		t.Error("expected checkpoint to be offered")
	} else {
		if output.Checkpoint.ID != "cp-1" {
			t.Errorf("expected checkpoint ID cp-1, got %s", output.Checkpoint.ID)
		}
		if output.Checkpoint.Summary != "Previous work" {
			t.Errorf("expected checkpoint summary 'Previous work', got %s", output.Checkpoint.Summary)
		}
	}

	// Memories array should be initialized (even if empty due to nil service)
	if output.Memories == nil {
		t.Error("expected memories array to be initialized")
	}
}

func TestSessionHandler_Start_ValidationError(t *testing.T) {
	mockReg := &mockRegistry{}
	handler := NewSessionHandler(mockReg)

	// Missing project_id
	input := json.RawMessage(`{"session_id": "sess-123"}`)
	_, err := handler.Start(context.Background(), input)
	if err == nil {
		t.Error("expected validation error for missing project_id")
	}

	// Missing session_id
	input = json.RawMessage(`{"project_id": "test-project"}`)
	_, err = handler.Start(context.Background(), input)
	if err == nil {
		t.Error("expected validation error for missing session_id")
	}
}

// Mock types and implementations

type memoryRecord struct {
	ID         string
	Title      string
	Confidence float64
}

type checkpointRecord struct {
	ID        string
	Summary   string
	CreatedAt time.Time
}

// mockRegistry implements services.Registry for testing
type mockRegistry struct {
	memories    []memoryRecord
	checkpoints []checkpointRecord
	distiller   *mockDistiller
}

func (m *mockRegistry) Checkpoint() checkpoint.Service {
	if len(m.checkpoints) == 0 {
		return nil
	}
	return &mockCheckpointSvc{checkpoints: m.checkpoints}
}

func (m *mockRegistry) Remediation() remediation.Service { return nil }

func (m *mockRegistry) Memory() *reasoningbank.Service {
	// We can't instantiate reasoningbank.Service here, so return nil
	// The handler will need to work with this limitation
	return nil
}

func (m *mockRegistry) Repository() *repository.Service  { return nil }
func (m *mockRegistry) Troubleshoot() *troubleshoot.Service { return nil }

func (m *mockRegistry) Hooks() *hooks.HookManager {
	return hooks.NewHookManager(&hooks.Config{
		CheckpointThreshold: 70,
	})
}

func (m *mockRegistry) Distiller() *reasoningbank.Distiller {
	// Return a mock distiller that implements the interface
	// We have to return the concrete type but cast it appropriately
	// For now, return nil and the handler will handle it gracefully
	return nil
}

func (m *mockRegistry) Scrubber() secrets.Scrubber { return nil }

func (m *mockRegistry) Compression() *compression.Service { return nil }

// mockCheckpointSvc implements checkpoint.Service
type mockCheckpointSvc struct {
	checkpoints []checkpointRecord
}

func (m *mockCheckpointSvc) Save(ctx context.Context, req *checkpoint.SaveRequest) (*checkpoint.Checkpoint, error) {
	return &checkpoint.Checkpoint{
		ID:          "checkpoint-123",
		SessionID:   req.SessionID,
		TenantID:    req.TenantID,
		TeamID:      req.TeamID,
		ProjectID:   req.ProjectID,
		Summary:     req.Summary,
		AutoCreated: req.AutoCreated,
		CreatedAt:   time.Now(),
	}, nil
}

func (m *mockCheckpointSvc) List(ctx context.Context, req *checkpoint.ListRequest) ([]*checkpoint.Checkpoint, error) {
	result := make([]*checkpoint.Checkpoint, len(m.checkpoints))
	for i, cp := range m.checkpoints {
		result[i] = &checkpoint.Checkpoint{
			ID:        cp.ID,
			Summary:   cp.Summary,
			CreatedAt: cp.CreatedAt,
		}
	}
	return result, nil
}

func (m *mockCheckpointSvc) Resume(ctx context.Context, req *checkpoint.ResumeRequest) (*checkpoint.ResumeResponse, error) {
	return nil, nil
}

func (m *mockCheckpointSvc) Get(ctx context.Context, tenantID, teamID, projectID, checkpointID string) (*checkpoint.Checkpoint, error) {
	return nil, nil
}

func (m *mockCheckpointSvc) Delete(ctx context.Context, tenantID, teamID, projectID, checkpointID string) error {
	return nil
}

func (m *mockCheckpointSvc) Close() error {
	return nil
}

// mockMemorySvc is a wrapper to fake reasoningbank.Service
type mockMemorySvc struct {
	memories []memoryRecord
}

func (m *mockMemorySvc) Search(ctx context.Context, projectID, query string, limit int) ([]reasoningbank.Memory, error) {
	result := make([]reasoningbank.Memory, len(m.memories))
	for i, mem := range m.memories {
		result[i] = reasoningbank.Memory{
			ID:         mem.ID,
			Title:      mem.Title,
			Confidence: mem.Confidence,
		}
	}
	return result, nil
}

func TestSessionHandler_ContextThreshold(t *testing.T) {
	mockReg := &mockRegistry{}
	handler := NewSessionHandler(mockReg)

	input := json.RawMessage(`{
		"project_id": "test-project",
		"session_id": "sess-123",
		"percent": 75
	}`)

	result, err := handler.ContextThreshold(context.Background(), input)
	if err != nil {
		t.Fatalf("ContextThreshold failed: %v", err)
	}

	output, ok := result.(*ContextThresholdOutput)
	if !ok {
		t.Fatalf("unexpected result type: %T", result)
	}

	if output.Message == "" {
		t.Error("expected non-empty message")
	}
}

func TestSessionHandler_ContextThreshold_InvalidPercent(t *testing.T) {
	mockReg := &mockRegistry{}
	handler := NewSessionHandler(mockReg)

	input := json.RawMessage(`{"project_id": "test", "session_id": "sess", "percent": 150}`)
	_, err := handler.ContextThreshold(context.Background(), input)
	if err == nil {
		t.Error("expected validation error for percent > 100")
	}
}

func TestSessionHandler_End(t *testing.T) {
	mockReg := &mockRegistry{}
	handler := NewSessionHandler(mockReg)

	input := json.RawMessage(`{
		"project_id": "test-project",
		"session_id": "sess-123",
		"task": "Implement hook lifecycle",
		"approach": "TDD with Registry pattern",
		"outcome": "success",
		"tags": ["hooks", "lifecycle"]
	}`)

	result, err := handler.End(context.Background(), input)
	if err != nil {
		t.Fatalf("End failed: %v", err)
	}

	output, ok := result.(*SessionEndOutput)
	if !ok {
		t.Fatalf("unexpected result type: %T", result)
	}

	if output.Message == "" {
		t.Error("expected non-empty message")
	}
}

func TestSessionHandler_End_ValidationError(t *testing.T) {
	mockReg := &mockRegistry{}
	handler := NewSessionHandler(mockReg)

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "missing project_id",
			input: `{"session_id": "sess-123", "task": "test", "approach": "test", "outcome": "success", "tags": ["test"]}`,
		},
		{
			name:  "missing session_id",
			input: `{"project_id": "test-project", "task": "test", "approach": "test", "outcome": "success", "tags": ["test"]}`,
		},
		{
			name:  "missing task",
			input: `{"project_id": "test-project", "session_id": "sess-123", "approach": "test", "outcome": "success", "tags": ["test"]}`,
		},
		{
			name:  "missing approach",
			input: `{"project_id": "test-project", "session_id": "sess-123", "task": "test", "outcome": "success", "tags": ["test"]}`,
		},
		{
			name:  "missing outcome",
			input: `{"project_id": "test-project", "session_id": "sess-123", "task": "test", "approach": "test", "tags": ["test"]}`,
		},
		{
			name:  "invalid outcome",
			input: `{"project_id": "test-project", "session_id": "sess-123", "task": "test", "approach": "test", "outcome": "invalid", "tags": ["test"]}`,
		},
		{
			name:  "missing tags",
			input: `{"project_id": "test-project", "session_id": "sess-123", "task": "test", "approach": "test", "outcome": "success"}`,
		},
		{
			name:  "empty tags array",
			input: `{"project_id": "test-project", "session_id": "sess-123", "task": "test", "approach": "test", "outcome": "success", "tags": []}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := json.RawMessage(tt.input)
			_, err := handler.End(context.Background(), input)
			if err == nil {
				t.Error("expected validation error")
			}
		})
	}
}

// mockDistiller implements reasoningbank.Distiller for testing
type mockDistiller struct {
	distillCalled bool
	lastSummary   *reasoningbank.SessionSummary
}

func (m *mockDistiller) DistillSession(ctx context.Context, summary reasoningbank.SessionSummary) error {
	m.distillCalled = true
	m.lastSummary = &summary
	return nil
}
