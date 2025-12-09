package orchestrator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockMCPClient mocks MCP tool calls
type MockMCPClient struct {
	mock.Mock
}

func (m *MockMCPClient) CallTool(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
	called := m.Called(ctx, name, args)
	return called.Get(0), called.Error(1)
}

func TestContextdRecorder_RecordLearning(t *testing.T) {
	mcpClient := &MockMCPClient{}

	// Expect memory_record call
	mcpClient.On("CallTool", mock.Anything, "memory_record", mock.MatchedBy(func(args map[string]interface{}) bool {
		content, ok := args["content"].(string)
		return ok && len(content) > 0
	})).Return(map[string]interface{}{"id": "mem-123"}, nil)

	recorder := NewContextdRecorder(mcpClient)

	err := recorder.RecordLearning(context.Background(), "Test learning content", []string{"test", "learning"})

	require.NoError(t, err)
	mcpClient.AssertExpectations(t)
}

func TestContextdRecorder_RecordViolation(t *testing.T) {
	mcpClient := &MockMCPClient{}

	// Expect memory_record call with violation details
	mcpClient.On("CallTool", mock.Anything, "memory_record", mock.MatchedBy(func(args map[string]interface{}) bool {
		content, ok := args["content"].(string)
		return ok && len(content) > 0
	})).Return(map[string]interface{}{"id": "mem-456"}, nil)

	recorder := NewContextdRecorder(mcpClient)

	violation := Violation{
		Type:        ViolationTDDNotFollowed,
		Phase:       PhaseImplement,
		Description: "Implementation without tests",
		Severity:    SeverityError,
	}

	err := recorder.RecordViolation(context.Background(), violation)

	require.NoError(t, err)
	mcpClient.AssertExpectations(t)
}

func TestContextdRecorder_SearchMemory(t *testing.T) {
	mcpClient := &MockMCPClient{}

	// Expect memory_search call
	mcpClient.On("CallTool", mock.Anything, "memory_search", mock.MatchedBy(func(args map[string]interface{}) bool {
		query, ok := args["query"].(string)
		return ok && query != ""
	})).Return([]map[string]interface{}{
		{"id": "mem-001", "content": "Previous learning", "score": 0.85},
	}, nil)

	recorder := NewContextdRecorder(mcpClient)

	results, err := recorder.SearchMemory(context.Background(), "test query", 5)

	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "mem-001", results[0].ID)
	mcpClient.AssertExpectations(t)
}

func TestContextdRecorder_RecordRemediation(t *testing.T) {
	mcpClient := &MockMCPClient{}

	// Expect remediation_record call
	mcpClient.On("CallTool", mock.Anything, "remediation_record", mock.MatchedBy(func(args map[string]interface{}) bool {
		errorPattern, ok := args["error_pattern"].(string)
		solution, ok2 := args["solution"].(string)
		return ok && ok2 && errorPattern != "" && solution != ""
	})).Return(map[string]interface{}{"id": "rem-789"}, nil)

	recorder := NewContextdRecorder(mcpClient)

	err := recorder.RecordRemediation(context.Background(), "error: undefined variable", "Declare the variable before use")

	require.NoError(t, err)
	mcpClient.AssertExpectations(t)
}

func TestContextdRecorder_SearchRemediation(t *testing.T) {
	mcpClient := &MockMCPClient{}

	// Expect remediation_search call
	mcpClient.On("CallTool", mock.Anything, "remediation_search", mock.MatchedBy(func(args map[string]interface{}) bool {
		query, ok := args["query"].(string)
		return ok && query != ""
	})).Return([]map[string]interface{}{
		{
			"id":            "rem-001",
			"error_pattern": "undefined variable",
			"solution":      "Declare variable first",
			"score":         0.92,
		},
	}, nil)

	recorder := NewContextdRecorder(mcpClient)

	results, err := recorder.SearchRemediation(context.Background(), "variable not defined")

	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Contains(t, results[0].ErrorPattern, "undefined")
	mcpClient.AssertExpectations(t)
}

func TestContextdRecorder_SaveCheckpoint(t *testing.T) {
	mcpClient := &MockMCPClient{}

	// Expect checkpoint_save call
	mcpClient.On("CallTool", mock.Anything, "checkpoint_save", mock.MatchedBy(func(args map[string]interface{}) bool {
		name, ok := args["name"].(string)
		return ok && name != ""
	})).Return(map[string]interface{}{"id": "chk-123", "name": "test-checkpoint"}, nil)

	recorder := NewContextdRecorder(mcpClient)

	checkpointID, err := recorder.SaveCheckpoint(context.Background(), "test-checkpoint", map[string]interface{}{
		"phase": "implement",
	})

	require.NoError(t, err)
	assert.Equal(t, "chk-123", checkpointID)
	mcpClient.AssertExpectations(t)
}

func TestContextdRecorder_ResumeCheckpoint(t *testing.T) {
	mcpClient := &MockMCPClient{}

	// Expect checkpoint_resume call
	mcpClient.On("CallTool", mock.Anything, "checkpoint_resume", mock.MatchedBy(func(args map[string]interface{}) bool {
		id, ok := args["id"].(string)
		return ok && id != ""
	})).Return(map[string]interface{}{
		"id":   "chk-123",
		"name": "test-checkpoint",
		"data": map[string]interface{}{"phase": "implement"},
	}, nil)

	recorder := NewContextdRecorder(mcpClient)

	checkpoint, err := recorder.ResumeCheckpoint(context.Background(), "chk-123")

	require.NoError(t, err)
	assert.Equal(t, "chk-123", checkpoint.ID)
	assert.Equal(t, "test-checkpoint", checkpoint.Name)
	mcpClient.AssertExpectations(t)
}

func TestContextdRecorder_ProvideFeedback(t *testing.T) {
	mcpClient := &MockMCPClient{}

	// Expect memory_feedback call
	mcpClient.On("CallTool", mock.Anything, "memory_feedback", mock.MatchedBy(func(args map[string]interface{}) bool {
		id, ok1 := args["id"].(string)
		_, ok2 := args["helpful"].(bool)
		return ok1 && ok2 && id != ""
	})).Return(map[string]interface{}{"success": true}, nil)

	recorder := NewContextdRecorder(mcpClient)

	err := recorder.ProvideFeedback(context.Background(), "mem-123", true)

	require.NoError(t, err)
	mcpClient.AssertExpectations(t)
}

func TestIntegration_OrchestratorWithContextd(t *testing.T) {
	mcpClient := &MockMCPClient{}

	// Setup expectations for a full workflow
	mcpClient.On("CallTool", mock.Anything, "memory_search", mock.Anything).Return([]map[string]interface{}{}, nil)
	mcpClient.On("CallTool", mock.Anything, "memory_record", mock.Anything).Return(map[string]interface{}{"id": "mem-new"}, nil)
	mcpClient.On("CallTool", mock.Anything, "checkpoint_save", mock.Anything).Return(map[string]interface{}{"id": "chk-new"}, nil)

	recorder := NewContextdRecorder(mcpClient)
	client := &MockClaudeClient{}

	executor := NewExecutor(client, recorder)

	// Register all phase handlers with mock behavior
	for _, phase := range AllPhases() {
		handler := NewMockPhaseHandler(phase)
		handler.On("Execute", mock.Anything, mock.Anything).Return(&PhaseResult{
			Phase:  phase,
			Status: StatusCompleted,
			Output: "Phase completed successfully",
			Artifacts: []Artifact{
				{Type: ArtifactTypeTestFile, Path: "test.go"},
			},
		}, nil)
		executor.RegisterHandler(handler)
	}

	config := TaskConfig{
		ID:             "integration-test",
		Description:    "Integration test task",
		RecordToMemory: true,
		EnforceTDD:     false, // Disable for this test
	}

	ctx := context.Background()
	state, err := executor.Execute(ctx, config)

	require.NoError(t, err)
	assert.Equal(t, StatusCompleted, state.Status)
}
