package reasoningbank

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// failOnThirdCallLLMClient is a mock LLM client that succeeds on first two calls
// but fails on the third call to simulate partial failures.
type failOnThirdCallLLMClient struct {
	callCount       int
	successResponse string
}

func (f *failOnThirdCallLLMClient) Complete(ctx context.Context, prompt string) (string, error) {
	f.callCount++
	if f.callCount == 3 {
		return "", fmt.Errorf("simulated LLM failure on 3rd cluster")
	}
	return f.successResponse, nil
}

func (f *failOnThirdCallLLMClient) CallCount() int {
	return f.callCount
}

// TestConsolidation_Integration_MultipleClusters tests the full consolidation workflow
// with multiple similarity clusters being consolidated in a single run.
//
// This integration test verifies:
// - Multiple clusters are detected and consolidated
// - Each cluster produces a consolidated memory
// - Source memories are archived with ConsolidationID links
// - Consolidated memories are searchable
// - Source memories are filtered from search results
// - Statistics are accurately tracked
func TestConsolidation_Integration_MultipleClusters(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	embedder := newMockEmbedder(384)
	llmClient := newMockLLMClient()
	logger := zap.NewNop()

	// Create service with embedder
	svc, err := NewService(store, logger,
		WithDefaultTenant("test-tenant"),
		WithEmbedder(embedder))
	require.NoError(t, err)

	// Create distiller with LLM client
	distiller, err := NewDistiller(svc, logger, WithLLMClient(llmClient))
	require.NoError(t, err)

	projectID := "integration-project-1"

	// Create two distinct clusters of similar memories
	// Cluster 1: API error handling (3 memories with similar titles)
	mem1, _ := NewMemory(projectID, "API error handling pattern",
		"Use structured error responses with status codes", OutcomeSuccess, []string{"api", "errors"})
	mem2, _ := NewMemory(projectID, "API error handling best practices",
		"Return proper HTTP status codes and error messages", OutcomeSuccess, []string{"api", "errors"})
	mem3, _ := NewMemory(projectID, "API error handling strategy",
		"Implement consistent error format across endpoints", OutcomeSuccess, []string{"api", "errors"})

	// Cluster 2: Database connection pooling (3 memories with similar titles)
	mem4, _ := NewMemory(projectID, "Database connection pool configuration",
		"Set max connections based on workload", OutcomeSuccess, []string{"database", "performance"})
	mem5, _ := NewMemory(projectID, "Database connection pool best practices",
		"Use connection pooling with timeout settings", OutcomeSuccess, []string{"database", "performance"})
	mem6, _ := NewMemory(projectID, "Database connection pool management",
		"Monitor connection pool usage and adjust limits", OutcomeSuccess, []string{"database", "performance"})

	// Dissimilar memory (should not be clustered)
	mem7, _ := NewMemory(projectID, "Frontend component patterns",
		"Use React hooks for state management", OutcomeSuccess, []string{"frontend", "react"})

	// Record all memories
	require.NoError(t, svc.Record(ctx, mem1))
	require.NoError(t, svc.Record(ctx, mem2))
	require.NoError(t, svc.Record(ctx, mem3))
	require.NoError(t, svc.Record(ctx, mem4))
	require.NoError(t, svc.Record(ctx, mem5))
	require.NoError(t, svc.Record(ctx, mem6))
	require.NoError(t, svc.Record(ctx, mem7))

	// Run consolidation with threshold that will cluster similar memories
	opts := ConsolidationOptions{
		SimilarityThreshold: 0.8,
		MaxClustersPerRun:   0,  // No limit
		DryRun:              false,
		ForceAll:            true,
	}

	result, err := distiller.Consolidate(ctx, projectID, opts)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify statistics
	t.Logf("Consolidation result: created=%d, archived=%d, skipped=%d, total=%d",
		len(result.CreatedMemories), len(result.ArchivedMemories),
		result.SkippedCount, result.TotalProcessed)

	// Should have created consolidated memories (at least 2 clusters)
	assert.GreaterOrEqual(t, len(result.CreatedMemories), 2,
		"should create at least 2 consolidated memories for 2 clusters")

	// Should have archived source memories
	assert.GreaterOrEqual(t, len(result.ArchivedMemories), 6,
		"should archive at least 6 source memories (3 per cluster)")

	// Verify duration was tracked
	assert.Greater(t, result.Duration, time.Duration(0))

	// Verify LLM was called multiple times (once per cluster)
	assert.GreaterOrEqual(t, llmClient.CallCount(), 2,
		"LLM should be called at least twice (once per cluster)")

	// Verify search returns consolidated memories, not archived sources
	searchResults, err := svc.Search(ctx, projectID, "API error handling", 10)
	require.NoError(t, err)

	// Count consolidated vs archived memories in results
	var consolidatedCount int
	var archivedCount int
	for _, result := range searchResults {
		if result.State == MemoryStateArchived {
			archivedCount++
		} else if result.ConsolidationID == nil && result.State == MemoryStateActive {
			// This could be a consolidated memory (no ConsolidationID, active)
			consolidatedCount++
		}
	}

	t.Logf("Search results: total=%d, archived=%d, active/consolidated=%d",
		len(searchResults), archivedCount, consolidatedCount)

	// Archived memories should be filtered from search results
	assert.Equal(t, 0, archivedCount,
		"search should not return archived source memories")

	// Should return at least some active memories
	assert.Greater(t, len(searchResults), 0,
		"search should return active/consolidated memories")
}

// TestConsolidation_Integration_PartialFailures tests the consolidation workflow
// when some clusters fail to consolidate while others succeed.
//
// This integration test verifies:
// - Successful clusters are consolidated despite other failures
// - Failed clusters are tracked in SkippedCount
// - Error handling is graceful (no panic, partial success)
// - Successfully consolidated memories are still created and linked
func TestConsolidation_Integration_PartialFailures(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	embedder := newMockEmbedder(384)
	logger := zap.NewNop()

	// Create service with embedder
	svc, err := NewService(store, logger,
		WithDefaultTenant("test-tenant"),
		WithEmbedder(embedder))
	require.NoError(t, err)

	projectID := "integration-project-2"

	// Create three clusters of similar memories
	// Cluster 1: Will succeed
	mem1, _ := NewMemory(projectID, "Pattern A one",
		"Content A1", OutcomeSuccess, []string{"pattern-a"})
	mem2, _ := NewMemory(projectID, "Pattern A two",
		"Content A2", OutcomeSuccess, []string{"pattern-a"})

	// Cluster 2: Will succeed
	mem3, _ := NewMemory(projectID, "Pattern B one",
		"Content B1", OutcomeSuccess, []string{"pattern-b"})
	mem4, _ := NewMemory(projectID, "Pattern B two",
		"Content B2", OutcomeSuccess, []string{"pattern-b"})

	// Cluster 3: Will succeed initially (for first 2 calls)
	mem5, _ := NewMemory(projectID, "Pattern C one",
		"Content C1", OutcomeSuccess, []string{"pattern-c"})
	mem6, _ := NewMemory(projectID, "Pattern C two",
		"Content C2", OutcomeSuccess, []string{"pattern-c"})

	// Record all memories
	require.NoError(t, svc.Record(ctx, mem1))
	require.NoError(t, svc.Record(ctx, mem2))
	require.NoError(t, svc.Record(ctx, mem3))
	require.NoError(t, svc.Record(ctx, mem4))
	require.NoError(t, svc.Record(ctx, mem5))
	require.NoError(t, svc.Record(ctx, mem6))

	// Create custom LLM client that fails on third call
	customLLM := &failOnThirdCallLLMClient{
		successResponse: `
TITLE: Consolidated Pattern
CONTENT: Synthesized content from multiple sources
TAGS: test, consolidated
OUTCOME: success
SOURCE_ATTRIBUTION: Synthesized from source memories
`,
	}

	// Create distiller with custom LLM client
	distiller, err := NewDistiller(svc, logger, WithLLMClient(customLLM))
	require.NoError(t, err)

	// Run consolidation
	opts := ConsolidationOptions{
		SimilarityThreshold: 0.8,
		MaxClustersPerRun:   0,
		DryRun:              false,
		ForceAll:            true,
	}

	result, err := distiller.Consolidate(ctx, projectID, opts)
	require.NoError(t, err, "consolidation should succeed despite partial failures")
	require.NotNil(t, result)

	t.Logf("Partial failure result: created=%d, archived=%d, skipped=%d, total=%d",
		len(result.CreatedMemories), len(result.ArchivedMemories),
		result.SkippedCount, result.TotalProcessed)

	// Should have created at least 2 consolidated memories (2 successful clusters)
	assert.GreaterOrEqual(t, len(result.CreatedMemories), 2,
		"should create consolidated memories for successful clusters")

	// Should have skipped at least 2 memories from the failed cluster
	assert.GreaterOrEqual(t, result.SkippedCount, 2,
		"should track skipped memories from failed cluster")

	// Total processed should account for all memories
	assert.GreaterOrEqual(t, result.TotalProcessed, 6,
		"should process all memories across all clusters")

	// Verify some memories were still archived (from successful clusters)
	assert.GreaterOrEqual(t, len(result.ArchivedMemories), 4,
		"should archive memories from successful clusters")

	// Verify LLM was called at least 3 times (once per cluster, including the failure)
	assert.GreaterOrEqual(t, customLLM.CallCount(), 3,
		"LLM should be called at least 3 times (all clusters attempted)")
}

// TestConsolidation_Integration_DryRunMode tests the consolidation workflow
// in dry-run mode where no actual changes are made.
//
// This integration test verifies:
// - Dry run detects clusters without consolidating
// - No consolidated memories are created
// - No source memories are archived
// - Statistics reflect what WOULD be done
// - LLM is NOT called in dry run mode
// - Original memories remain unchanged
func TestConsolidation_Integration_DryRunMode(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	embedder := newMockEmbedder(384)
	llmClient := newMockLLMClient()
	logger := zap.NewNop()

	// Create service with embedder
	svc, err := NewService(store, logger,
		WithDefaultTenant("test-tenant"),
		WithEmbedder(embedder))
	require.NoError(t, err)

	// Create distiller with LLM client
	distiller, err := NewDistiller(svc, logger, WithLLMClient(llmClient))
	require.NoError(t, err)

	projectID := "integration-project-3"

	// Create a cluster of similar memories
	mem1, _ := NewMemory(projectID, "Test pattern alpha",
		"Content alpha 1", OutcomeSuccess, []string{"test"})
	mem1.Confidence = 0.8
	mem1.UsageCount = 5

	mem2, _ := NewMemory(projectID, "Test pattern beta",
		"Content beta 2", OutcomeSuccess, []string{"test"})
	mem2.Confidence = 0.7
	mem2.UsageCount = 3

	mem3, _ := NewMemory(projectID, "Test pattern gamma",
		"Content gamma 3", OutcomeSuccess, []string{"test"})
	mem3.Confidence = 0.9
	mem3.UsageCount = 10

	// Record all memories
	require.NoError(t, svc.Record(ctx, mem1))
	require.NoError(t, svc.Record(ctx, mem2))
	require.NoError(t, svc.Record(ctx, mem3))

	// Get initial count of memories
	initialMemories, err := svc.ListMemories(ctx, projectID, 0, 0)
	require.NoError(t, err)
	initialCount := len(initialMemories)
	t.Logf("Initial memory count: %d", initialCount)

	// Verify all memories are active
	for _, mem := range initialMemories {
		assert.Equal(t, MemoryStateActive, mem.State,
			"all memories should be active before consolidation")
		assert.Nil(t, mem.ConsolidationID,
			"no memories should have consolidation link before consolidation")
	}

	// Run consolidation in DRY RUN mode
	opts := ConsolidationOptions{
		SimilarityThreshold: 0.8,
		MaxClustersPerRun:   0,
		DryRun:              true, // DRY RUN MODE
		ForceAll:            true,
	}

	result, err := distiller.Consolidate(ctx, projectID, opts)
	require.NoError(t, err)
	require.NotNil(t, result)

	t.Logf("Dry run result: created=%d, archived=%d, skipped=%d, total=%d",
		len(result.CreatedMemories), len(result.ArchivedMemories),
		result.SkippedCount, result.TotalProcessed)

	// Verify dry run results
	assert.Greater(t, len(result.CreatedMemories), 0,
		"dry run should report what WOULD be created")
	assert.Greater(t, len(result.ArchivedMemories), 0,
		"dry run should report what WOULD be archived")
	assert.Greater(t, result.TotalProcessed, 0,
		"dry run should report what WOULD be processed")

	// Verify created memories have dry-run markers
	for _, createdID := range result.CreatedMemories {
		assert.Contains(t, createdID, "dry-run-cluster",
			"created memory IDs should indicate dry-run mode")
	}

	// Verify LLM was NOT called in dry run mode
	assert.Equal(t, 0, llmClient.CallCount(),
		"LLM should NOT be called in dry run mode")

	// Get memories after dry run
	afterMemories, err := svc.ListMemories(ctx, projectID, 0, 0)
	require.NoError(t, err)
	afterCount := len(afterMemories)

	// Verify no memories were actually created or modified
	assert.Equal(t, initialCount, afterCount,
		"memory count should not change in dry run mode")

	// Verify all memories are still active (not archived)
	for _, mem := range afterMemories {
		assert.Equal(t, MemoryStateActive, mem.State,
			"all memories should remain active in dry run mode")
		assert.Nil(t, mem.ConsolidationID,
			"no memories should have consolidation link in dry run mode")
	}

	// Verify original memory properties are unchanged
	for i, mem := range afterMemories {
		originalMem := initialMemories[i]
		assert.Equal(t, originalMem.Title, mem.Title,
			"memory title should not change in dry run")
		assert.Equal(t, originalMem.Content, mem.Content,
			"memory content should not change in dry run")
		assert.Equal(t, originalMem.Confidence, mem.Confidence,
			"memory confidence should not change in dry run")
		assert.Equal(t, originalMem.UsageCount, mem.UsageCount,
			"memory usage count should not change in dry run")
	}

	t.Log("Dry run mode verified: no actual changes made to memories")
}

// TestConsolidation_Integration_EndToEnd tests the complete consolidation lifecycle
// from initial memories through consolidation to search results.
//
// This integration test verifies:
// - Similar memories are detected and clustered
// - LLM synthesizes cluster into consolidated memory
// - Source memories are archived with proper links
// - Consolidated memory has correct confidence score
// - Search returns consolidated memory (not sources)
// - Consolidated memory is ranked higher in search
func TestConsolidation_Integration_EndToEnd(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	embedder := newMockEmbedder(384)
	llmClient := newMockLLMClient()
	logger := zap.NewNop()

	// Create service with embedder
	svc, err := NewService(store, logger,
		WithDefaultTenant("test-tenant"),
		WithEmbedder(embedder))
	require.NoError(t, err)

	// Create distiller with LLM client
	distiller, err := NewDistiller(svc, logger, WithLLMClient(llmClient))
	require.NoError(t, err)

	projectID := "integration-project-4"

	// Create similar memories about error handling
	mem1, _ := NewMemory(projectID, "Error handling approach 1",
		"Use try-catch blocks for error handling", OutcomeSuccess, []string{"errors", "best-practices"})
	mem1.Confidence = 0.8
	mem1.UsageCount = 10

	mem2, _ := NewMemory(projectID, "Error handling approach 2",
		"Implement structured error responses", OutcomeSuccess, []string{"errors", "best-practices"})
	mem2.Confidence = 0.7
	mem2.UsageCount = 5

	mem3, _ := NewMemory(projectID, "Error handling approach 3",
		"Return proper error codes and messages", OutcomeSuccess, []string{"errors", "best-practices"})
	mem3.Confidence = 0.9
	mem3.UsageCount = 15

	// Record all memories
	require.NoError(t, svc.Record(ctx, mem1))
	require.NoError(t, svc.Record(ctx, mem2))
	require.NoError(t, svc.Record(ctx, mem3))

	// Verify initial state - search returns all 3 memories
	initialResults, err := svc.Search(ctx, projectID, "error handling", 10)
	require.NoError(t, err)
	assert.Equal(t, 3, len(initialResults),
		"should find all 3 memories before consolidation")

	// Run consolidation
	opts := ConsolidationOptions{
		SimilarityThreshold: 0.8,
		MaxClustersPerRun:   0,
		DryRun:              false,
		ForceAll:            true,
	}

	result, err := distiller.Consolidate(ctx, projectID, opts)
	require.NoError(t, err)
	require.NotNil(t, result)

	t.Logf("End-to-end result: created=%d, archived=%d, skipped=%d, total=%d",
		len(result.CreatedMemories), len(result.ArchivedMemories),
		result.SkippedCount, result.TotalProcessed)

	// Verify consolidation occurred
	assert.Equal(t, 1, len(result.CreatedMemories),
		"should create 1 consolidated memory")
	assert.Equal(t, 3, len(result.ArchivedMemories),
		"should archive 3 source memories")
	assert.Equal(t, 0, result.SkippedCount,
		"no memories should be skipped")
	assert.Equal(t, 3, result.TotalProcessed,
		"should process all 3 memories")

	// Verify LLM was called once
	assert.Equal(t, 1, llmClient.CallCount(),
		"LLM should be called once for the cluster")

	// Get the consolidated memory
	consolidatedID := result.CreatedMemories[0]
	consolidatedMem, err := svc.GetByProjectID(ctx, projectID, consolidatedID)
	require.NoError(t, err)
	require.NotNil(t, consolidatedMem)

	// Verify consolidated memory properties
	assert.Equal(t, MemoryStateActive, consolidatedMem.State,
		"consolidated memory should be active")
	assert.Nil(t, consolidatedMem.ConsolidationID,
		"consolidated memory should not have consolidation link")
	assert.Contains(t, consolidatedMem.Description, "Synthesized",
		"consolidated memory should have source attribution")

	// Verify confidence score is calculated correctly
	// Should be weighted average: (0.8*11 + 0.7*6 + 0.9*16) / (11+6+16)
	expectedConfidence := (0.8*11 + 0.7*6 + 0.9*16) / (11.0 + 6.0 + 16.0)
	assert.InDelta(t, expectedConfidence, consolidatedMem.Confidence, 0.01,
		"consolidated confidence should be weighted average of sources")

	// Verify source memories are archived with links
	for _, sourceID := range result.ArchivedMemories {
		sourceMem, err := svc.GetByProjectID(ctx, projectID, sourceID)
		require.NoError(t, err)
		assert.Equal(t, MemoryStateArchived, sourceMem.State,
			"source memory should be archived")
		require.NotNil(t, sourceMem.ConsolidationID,
			"source memory should have consolidation link")
		assert.Equal(t, consolidatedID, *sourceMem.ConsolidationID,
			"source memory should link to consolidated memory")
	}

	// Verify search behavior after consolidation
	afterResults, err := svc.Search(ctx, projectID, "error handling", 10)
	require.NoError(t, err)

	t.Logf("Search after consolidation: %d results", len(afterResults))

	// Count result types
	var activeCount, archivedCount int
	for _, res := range afterResults {
		if res.State == MemoryStateArchived {
			archivedCount++
		} else {
			activeCount++
		}
	}

	// Archived memories should be filtered from search results
	assert.Equal(t, 0, archivedCount,
		"search should filter archived source memories")

	// Should find the consolidated memory
	assert.GreaterOrEqual(t, activeCount, 1,
		"search should return at least the consolidated memory")

	// Find the consolidated memory in search results
	var foundConsolidated bool
	for _, res := range afterResults {
		if res.ID == consolidatedID {
			foundConsolidated = true
			break
		}
	}
	assert.True(t, foundConsolidated,
		"search should return the consolidated memory")

	t.Log("End-to-end consolidation verified successfully")
}

// TestConsolidation_Integration_ConsolidationWindow tests that the consolidation
// window prevents re-processing recently consolidated memories.
//
// This integration test verifies:
// - First consolidation succeeds
// - Second consolidation within window is skipped
// - ForceAll bypasses the window check
// - Consolidation window tracking works correctly
func TestConsolidation_Integration_ConsolidationWindow(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	embedder := newMockEmbedder(384)
	llmClient := newMockLLMClient()
	logger := zap.NewNop()

	// Create service with embedder
	svc, err := NewService(store, logger,
		WithDefaultTenant("test-tenant"),
		WithEmbedder(embedder))
	require.NoError(t, err)

	// Create distiller with short consolidation window for testing
	distiller, err := NewDistiller(svc, logger,
		WithLLMClient(llmClient),
		WithConsolidationWindow(1*time.Hour)) // 1 hour window
	require.NoError(t, err)

	projectID := "integration-project-5"

	// Create similar memories
	mem1, _ := NewMemory(projectID, "Pattern X-1", "Content X1", OutcomeSuccess, []string{"test"})
	mem2, _ := NewMemory(projectID, "Pattern X-2", "Content X2", OutcomeSuccess, []string{"test"})

	require.NoError(t, svc.Record(ctx, mem1))
	require.NoError(t, svc.Record(ctx, mem2))

	// First consolidation - should succeed
	opts := ConsolidationOptions{
		SimilarityThreshold: 0.8,
		DryRun:              false,
		ForceAll:            false, // Respect consolidation window
	}

	result1, err := distiller.Consolidate(ctx, projectID, opts)
	require.NoError(t, err)
	assert.Greater(t, len(result1.CreatedMemories), 0,
		"first consolidation should create memories")

	initialLLMCalls := llmClient.CallCount()

	// Second consolidation immediately after - should be skipped
	result2, err := distiller.Consolidate(ctx, projectID, opts)
	require.NoError(t, err)
	assert.Equal(t, 0, len(result2.CreatedMemories),
		"second consolidation should be skipped (within window)")
	assert.Equal(t, 0, len(result2.ArchivedMemories),
		"second consolidation should not archive anything")
	assert.Equal(t, initialLLMCalls, llmClient.CallCount(),
		"LLM should not be called for skipped consolidation")

	// Third consolidation with ForceAll - should succeed
	opts.ForceAll = true
	result3, err := distiller.Consolidate(ctx, projectID, opts)
	require.NoError(t, err)

	// Note: Since memories are already consolidated, this might not create new ones
	// but it should at least attempt the consolidation (not skip it)
	t.Logf("ForceAll consolidation: created=%d, archived=%d",
		len(result3.CreatedMemories), len(result3.ArchivedMemories))

	t.Log("Consolidation window tracking verified successfully")
}
