package autonomous_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fyrsmithlabs/contextd/internal/workflows/autonomous"
)

// TestBaseAgent_Execute tests basic agent execution.
func TestBaseAgent_Execute(t *testing.T) {
	ctx := context.Background()

	agent := autonomous.NewBaseAgent("test-agent", "You are a test agent")
	require.NotNil(t, agent)
	assert.Equal(t, "test-agent", agent.Name())

	input := autonomous.AgentInput{
		Task: "Analyze this test requirement",
		Context: map[string]interface{}{
			"requirement": "User needs authentication",
		},
		MCPClient:      &autonomous.MockMCPClient{},
		CollectionName: "test-collection",
		ProjectPath:    "/path/to/project",
		IssueNumber:    1,
	}

	output, err := agent.Execute(ctx, input)
	require.NoError(t, err)
	assert.NotNil(t, output.Result)
	assert.NotNil(t, output.Metrics)
}

// TestBaseAgent_WithMemorySearch tests agent using ReasoningBank.
func TestBaseAgent_WithMemorySearch(t *testing.T) {
	ctx := context.Background()

	mockMCP := &autonomous.MockMCPClient{
		Memories: []autonomous.Memory{
			{
				ID:      "mem-1",
				Title:   "JWT authentication pattern",
				Content: "Use JWT tokens for stateless auth",
				Outcome: "success",
				Tags:    []string{"auth", "jwt"},
			},
		},
	}

	agent := autonomous.NewBaseAgent("test-agent", "You are a test agent")
	input := autonomous.AgentInput{
		Task:           "How should we implement authentication?",
		Context:        map[string]interface{}{},
		MCPClient:      mockMCP,
		CollectionName: "test-collection",
		ProjectPath:    "/path/to/project",
		IssueNumber:    1,
	}

	output, err := agent.Execute(ctx, input)
	require.NoError(t, err)
	assert.NotNil(t, output.Result)

	// Agent should have searched memories
	assert.Greater(t, output.Metrics.MemoriesUsed, 0, "Agent should search ReasoningBank")
}

// TestBaseAgent_WithCodebaseSearch tests agent using semantic code search.
func TestBaseAgent_WithCodebaseSearch(t *testing.T) {
	ctx := context.Background()

	mockMCP := &autonomous.MockMCPClient{
		SearchResults: []autonomous.SearchResult{
			{
				FilePath: "internal/auth/middleware.go",
				Content:  "func AuthMiddleware() { ... }",
				Score:    0.95,
			},
		},
	}

	agent := autonomous.NewBaseAgent("test-agent", "You are a test agent")
	input := autonomous.AgentInput{
		Task:           "Find existing authentication code",
		Context:        map[string]interface{}{},
		MCPClient:      mockMCP,
		CollectionName: "test-collection",
		ProjectPath:    "/path/to/project",
		IssueNumber:    1,
	}

	output, err := agent.Execute(ctx, input)
	require.NoError(t, err)
	assert.NotNil(t, output.Result)
}

// TestBaseAgent_RecordsLearning tests agent recording new patterns.
func TestBaseAgent_RecordsLearning(t *testing.T) {
	ctx := context.Background()

	mockMCP := &autonomous.MockMCPClient{}

	agent := autonomous.NewBaseAgent("test-agent", "You are a test agent")
	input := autonomous.AgentInput{
		Task:           "Implement a new pattern",
		Context:        map[string]interface{}{},
		MCPClient:      mockMCP,
		CollectionName: "test-collection",
		ProjectPath:    "/path/to/project",
		IssueNumber:    1,
	}

	output, err := agent.Execute(ctx, input)
	require.NoError(t, err)
	assert.NotNil(t, output.Result)

	// Agent should record successful patterns
	assert.Greater(t, output.Metrics.MemoriesAdded, 0, "Agent should record new patterns")
	assert.Greater(t, len(mockMCP.RecordedMemories), 0, "Memories should be recorded in MCP")
}
