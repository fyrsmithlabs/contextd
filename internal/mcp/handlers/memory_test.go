package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/reasoningbank"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockDistiller is a mock implementation of the Distiller for testing.
type mockDistiller struct {
	consolidateFunc func(ctx context.Context, projectID string, opts reasoningbank.ConsolidationOptions) (*reasoningbank.ConsolidationResult, error)
	callCount       int
	lastProjectID   string
	lastOpts        reasoningbank.ConsolidationOptions
}

func newMockDistiller() *mockDistiller {
	return &mockDistiller{
		consolidateFunc: func(ctx context.Context, projectID string, opts reasoningbank.ConsolidationOptions) (*reasoningbank.ConsolidationResult, error) {
			return &reasoningbank.ConsolidationResult{
				CreatedMemories:  []string{"mem-1", "mem-2"},
				ArchivedMemories: []string{"mem-3", "mem-4", "mem-5"},
				SkippedCount:     1,
				TotalProcessed:   6,
				Duration:         2 * time.Second,
			}, nil
		},
	}
}

func (m *mockDistiller) Consolidate(ctx context.Context, projectID string, opts reasoningbank.ConsolidationOptions) (*reasoningbank.ConsolidationResult, error) {
	m.callCount++
	m.lastProjectID = projectID
	m.lastOpts = opts
	return m.consolidateFunc(ctx, projectID, opts)
}

// Helper to create mock distillers with specific behaviors
func newMockDistillerWithError(err error) *mockDistiller {
	return &mockDistiller{
		consolidateFunc: func(ctx context.Context, projectID string, opts reasoningbank.ConsolidationOptions) (*reasoningbank.ConsolidationResult, error) {
			return nil, err
		},
	}
}

func newMockDistillerWithResult(result *reasoningbank.ConsolidationResult) *mockDistiller {
	return &mockDistiller{
		consolidateFunc: func(ctx context.Context, projectID string, opts reasoningbank.ConsolidationOptions) (*reasoningbank.ConsolidationResult, error) {
			return result, nil
		},
	}
}

func TestMemoryHandler_Consolidate_ValidInput(t *testing.T) {
	// Test successful consolidation with all parameters specified
	distiller := newMockDistiller()
	handler := NewMemoryHandler(distiller)

	input := MemoryConsolidateInput{
		ProjectID:           "project-123",
		SimilarityThreshold: 0.85,
		DryRun:              false,
		MaxClusters:         10,
	}

	inputJSON, err := json.Marshal(input)
	require.NoError(t, err)

	result, err := handler.Consolidate(context.Background(), inputJSON)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify result structure
	output, ok := result.(MemoryConsolidateOutput)
	require.True(t, ok)
	assert.Equal(t, []string{"mem-1", "mem-2"}, output.CreatedMemories)
	assert.Equal(t, []string{"mem-3", "mem-4", "mem-5"}, output.ArchivedMemories)
	assert.Equal(t, 1, output.SkippedCount)
	assert.Equal(t, 6, output.TotalProcessed)
	assert.Equal(t, 2.0, output.DurationSeconds)

	// Verify distiller was called with correct parameters
	assert.Equal(t, 1, distiller.callCount)
	assert.Equal(t, "project-123", distiller.lastProjectID)
	assert.Equal(t, 0.85, distiller.lastOpts.SimilarityThreshold)
	assert.False(t, distiller.lastOpts.DryRun)
	assert.Equal(t, 10, distiller.lastOpts.MaxClustersPerRun)
}

func TestMemoryHandler_Consolidate_DefaultThreshold(t *testing.T) {
	// Test that default threshold (0.8) is applied when not specified
	distiller := newMockDistiller()
	handler := NewMemoryHandler(distiller)

	input := MemoryConsolidateInput{
		ProjectID: "project-123",
		// SimilarityThreshold not specified (0 value)
	}

	inputJSON, err := json.Marshal(input)
	require.NoError(t, err)

	result, err := handler.Consolidate(context.Background(), inputJSON)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify default threshold was applied
	assert.Equal(t, 1, distiller.callCount)
	assert.Equal(t, 0.8, distiller.lastOpts.SimilarityThreshold)
}

func TestMemoryHandler_Consolidate_DryRunMode(t *testing.T) {
	// Test dry run mode is correctly passed through
	distiller := newMockDistiller()
	handler := NewMemoryHandler(distiller)

	input := MemoryConsolidateInput{
		ProjectID:           "project-123",
		SimilarityThreshold: 0.8,
		DryRun:              true,
	}

	inputJSON, err := json.Marshal(input)
	require.NoError(t, err)

	result, err := handler.Consolidate(context.Background(), inputJSON)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify dry run was enabled
	assert.Equal(t, 1, distiller.callCount)
	assert.True(t, distiller.lastOpts.DryRun)
}

func TestMemoryHandler_Consolidate_MaxClusters(t *testing.T) {
	// Test max clusters limit is correctly passed through
	distiller := newMockDistiller()
	handler := NewMemoryHandler(distiller)

	input := MemoryConsolidateInput{
		ProjectID:   "project-123",
		MaxClusters: 5,
	}

	inputJSON, err := json.Marshal(input)
	require.NoError(t, err)

	result, err := handler.Consolidate(context.Background(), inputJSON)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify max clusters was set
	assert.Equal(t, 1, distiller.callCount)
	assert.Equal(t, 5, distiller.lastOpts.MaxClustersPerRun)
}

func TestMemoryHandler_Consolidate_EmptyProjectID(t *testing.T) {
	// Test error when project_id is missing
	distiller := newMockDistiller()
	handler := NewMemoryHandler(distiller)

	input := MemoryConsolidateInput{
		ProjectID: "", // Empty project ID
	}

	inputJSON, err := json.Marshal(input)
	require.NoError(t, err)

	result, err := handler.Consolidate(context.Background(), inputJSON)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "project_id is required")

	// Verify distiller was NOT called
	assert.Equal(t, 0, distiller.callCount)
}

func TestMemoryHandler_Consolidate_InvalidJSON(t *testing.T) {
	// Test error when input JSON is malformed
	handler := NewMemoryHandler(newMockDistiller())

	invalidJSON := []byte(`{"project_id": invalid json}`)

	result, err := handler.Consolidate(context.Background(), invalidJSON)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid input")
}

func TestMemoryHandler_Consolidate_NilDistiller(t *testing.T) {
	// Test error when distiller is not configured
	handler := NewMemoryHandler(nil)

	input := MemoryConsolidateInput{
		ProjectID: "project-123",
	}

	inputJSON, err := json.Marshal(input)
	require.NoError(t, err)

	result, err := handler.Consolidate(context.Background(), inputJSON)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "distiller not configured")
}

func TestMemoryHandler_Consolidate_DistillerError(t *testing.T) {
	// Test error handling when distiller fails
	expectedErr := errors.New("consolidation failed: LLM timeout")
	distiller := newMockDistillerWithError(expectedErr)
	handler := NewMemoryHandler(distiller)

	input := MemoryConsolidateInput{
		ProjectID: "project-123",
	}

	inputJSON, err := json.Marshal(input)
	require.NoError(t, err)

	result, err := handler.Consolidate(context.Background(), inputJSON)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "consolidation failed")
	assert.Contains(t, err.Error(), "LLM timeout")
}

func TestMemoryHandler_Consolidate_EmptyResult(t *testing.T) {
	// Test handling of empty consolidation result (no clusters found)
	emptyResult := &reasoningbank.ConsolidationResult{
		CreatedMemories:  []string{},
		ArchivedMemories: []string{},
		SkippedCount:     0,
		TotalProcessed:   0,
		Duration:         100 * time.Millisecond,
	}
	distiller := newMockDistillerWithResult(emptyResult)
	handler := NewMemoryHandler(distiller)

	input := MemoryConsolidateInput{
		ProjectID: "project-123",
	}

	inputJSON, err := json.Marshal(input)
	require.NoError(t, err)

	result, err := handler.Consolidate(context.Background(), inputJSON)
	require.NoError(t, err)
	assert.NotNil(t, result)

	output, ok := result.(MemoryConsolidateOutput)
	require.True(t, ok)
	assert.Empty(t, output.CreatedMemories)
	assert.Empty(t, output.ArchivedMemories)
	assert.Equal(t, 0, output.SkippedCount)
	assert.Equal(t, 0, output.TotalProcessed)
	assert.Equal(t, 0.1, output.DurationSeconds)
}

func TestMemoryHandler_Consolidate_DurationConversion(t *testing.T) {
	// Test duration is correctly converted to seconds
	result := &reasoningbank.ConsolidationResult{
		CreatedMemories:  []string{"mem-1"},
		ArchivedMemories: []string{"mem-2", "mem-3"},
		SkippedCount:     0,
		TotalProcessed:   3,
		Duration:         3*time.Second + 500*time.Millisecond,
	}
	distiller := newMockDistillerWithResult(result)
	handler := NewMemoryHandler(distiller)

	input := MemoryConsolidateInput{
		ProjectID: "project-123",
	}

	inputJSON, err := json.Marshal(input)
	require.NoError(t, err)

	res, err := handler.Consolidate(context.Background(), inputJSON)
	require.NoError(t, err)

	output, ok := res.(MemoryConsolidateOutput)
	require.True(t, ok)
	assert.Equal(t, 3.5, output.DurationSeconds)
}

func TestMemoryHandler_Consolidate_ContextCancellation(t *testing.T) {
	// Test that context cancellation is respected
	distiller := &mockDistiller{
		consolidateFunc: func(ctx context.Context, projectID string, opts reasoningbank.ConsolidationOptions) (*reasoningbank.ConsolidationResult, error) {
			// Check if context is cancelled
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return &reasoningbank.ConsolidationResult{
					CreatedMemories:  []string{},
					ArchivedMemories: []string{},
					SkippedCount:     0,
					TotalProcessed:   0,
					Duration:         0,
				}, nil
			}
		},
	}
	handler := NewMemoryHandler(distiller)

	input := MemoryConsolidateInput{
		ProjectID: "project-123",
	}

	inputJSON, err := json.Marshal(input)
	require.NoError(t, err)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result, err := handler.Consolidate(ctx, inputJSON)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestMemoryHandler_Consolidate_AllParameters(t *testing.T) {
	// Test comprehensive scenario with all parameters
	distiller := newMockDistiller()
	handler := NewMemoryHandler(distiller)

	input := MemoryConsolidateInput{
		ProjectID:           "project-xyz",
		SimilarityThreshold: 0.75,
		DryRun:              true,
		MaxClusters:         20,
	}

	inputJSON, err := json.Marshal(input)
	require.NoError(t, err)

	result, err := handler.Consolidate(context.Background(), inputJSON)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify all parameters were passed correctly
	assert.Equal(t, 1, distiller.callCount)
	assert.Equal(t, "project-xyz", distiller.lastProjectID)
	assert.Equal(t, 0.75, distiller.lastOpts.SimilarityThreshold)
	assert.True(t, distiller.lastOpts.DryRun)
	assert.Equal(t, 20, distiller.lastOpts.MaxClustersPerRun)

	// Verify result format
	output, ok := result.(MemoryConsolidateOutput)
	require.True(t, ok)
	assert.NotNil(t, output.CreatedMemories)
	assert.NotNil(t, output.ArchivedMemories)
}

func TestNewMemoryHandler(t *testing.T) {
	// Test handler creation
	distiller := newMockDistiller()
	handler := NewMemoryHandler(distiller)

	assert.NotNil(t, handler)
	assert.Equal(t, distiller, handler.distiller)
}

func TestNewMemoryHandler_NilDistiller(t *testing.T) {
	// Test handler creation with nil distiller
	handler := NewMemoryHandler(nil)

	assert.NotNil(t, handler)
	assert.Nil(t, handler.distiller)
}
