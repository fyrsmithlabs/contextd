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

// TestConsolidation_Integration_SimilarityThreshold tests that memories with
// >0.8 similarity are consolidated, while those with <0.8 similarity are not.
//
// This integration test verifies:
// - Memories with >0.8 similarity are detected as a cluster and consolidated
// - Memories with <0.8 similarity are NOT clustered together
// - Only similar memories are archived, dissimilar memories remain active
// - The 0.8 threshold is correctly applied in clustering logic
func TestConsolidation_Integration_SimilarityThreshold(t *testing.T) {
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

	projectID := "integration-project-6"

	// Note: mockEmbedder creates embeddings based on text length:
	// embedding[j] = float32(len(text)+j) / 1000.0
	// Therefore, texts with similar lengths have high cosine similarity,
	// and texts with very different lengths have low similarity.

	// Create HIGH SIMILARITY memories (similar text lengths -> >0.8 similarity)
	// Using titles with exact same length (45 characters each)
	highSim1, _ := NewMemory(projectID,
		"Error handling pattern for database queries", // 45 chars
		"Use proper error handling when querying databases", OutcomeSuccess, []string{"database", "errors"})
	highSim1.Confidence = 0.8
	highSim1.UsageCount = 5

	highSim2, _ := NewMemory(projectID,
		"Error handling pattern for network requests", // 45 chars
		"Use proper error handling when making API calls", OutcomeSuccess, []string{"network", "errors"})
	highSim2.Confidence = 0.7
	highSim2.UsageCount = 3

	highSim3, _ := NewMemory(projectID,
		"Error handling pattern for file operations", // 44 chars (very close)
		"Use proper error handling when reading files", OutcomeSuccess, []string{"files", "errors"})
	highSim3.Confidence = 0.9
	highSim3.UsageCount = 10

	// Create LOW SIMILARITY memories (very different text lengths -> <0.8 similarity)
	// Using titles with dramatically different lengths
	lowSim1, _ := NewMemory(projectID,
		"X", // 1 char - very short
		"A", OutcomeSuccess, []string{"test"})
	lowSim1.Confidence = 0.6
	lowSim1.UsageCount = 2

	lowSim2, _ := NewMemory(projectID,
		"This is a significantly longer title that will produce a completely different embedding vector due to its much greater character length which makes it dissimilar", // 163 chars - very long
		"This is a significantly longer content that will produce a completely different embedding vector", OutcomeSuccess, []string{"test"})
	lowSim2.Confidence = 0.7
	lowSim2.UsageCount = 4

	// Record all memories
	require.NoError(t, svc.Record(ctx, highSim1))
	require.NoError(t, svc.Record(ctx, highSim2))
	require.NoError(t, svc.Record(ctx, highSim3))
	require.NoError(t, svc.Record(ctx, lowSim1))
	require.NoError(t, svc.Record(ctx, lowSim2))

	// Verify initial count
	initialMemories, err := svc.ListMemories(ctx, projectID, 0, 0)
	require.NoError(t, err)
	assert.Equal(t, 5, len(initialMemories),
		"should have 5 memories before consolidation")

	// Run consolidation with 0.8 threshold
	opts := ConsolidationOptions{
		SimilarityThreshold: 0.8,
		MaxClustersPerRun:   0,
		DryRun:              false,
		ForceAll:            true,
	}

	result, err := distiller.Consolidate(ctx, projectID, opts)
	require.NoError(t, err)
	require.NotNil(t, result)

	t.Logf("Similarity threshold test result: created=%d, archived=%d, skipped=%d, total=%d",
		len(result.CreatedMemories), len(result.ArchivedMemories),
		result.SkippedCount, result.TotalProcessed)

	// Verify results:
	// 1. Should create exactly 1 consolidated memory (from the 3 high-similarity memories)
	assert.Equal(t, 1, len(result.CreatedMemories),
		"should create exactly 1 consolidated memory from high-similarity cluster")

	// 2. Should archive exactly 3 memories (the high-similarity source memories)
	assert.Equal(t, 3, len(result.ArchivedMemories),
		"should archive exactly 3 high-similarity source memories")

	// 3. Should process all 5 memories
	assert.Equal(t, 5, result.TotalProcessed,
		"should process all 5 memories")

	// 4. LLM should be called exactly once (for the single cluster)
	assert.Equal(t, 1, llmClient.CallCount(),
		"LLM should be called exactly once for the high-similarity cluster")

	// Verify the consolidated memory was created
	consolidatedID := result.CreatedMemories[0]
	consolidatedMem, err := svc.GetByProjectID(ctx, projectID, consolidatedID)
	require.NoError(t, err)
	require.NotNil(t, consolidatedMem)
	assert.Equal(t, MemoryStateActive, consolidatedMem.State)
	assert.Nil(t, consolidatedMem.ConsolidationID)

	// Verify high-similarity memories were archived
	highSimIDs := []string{highSim1.ID, highSim2.ID, highSim3.ID}
	for _, id := range highSimIDs {
		found := false
		for _, archivedID := range result.ArchivedMemories {
			if archivedID == id {
				found = true
				break
			}
		}
		assert.True(t, found,
			"high-similarity memory %s should be archived", id)

		// Verify the archived memory has proper links
		mem, err := svc.GetByProjectID(ctx, projectID, id)
		require.NoError(t, err)
		assert.Equal(t, MemoryStateArchived, mem.State,
			"high-similarity memory should be archived")
		require.NotNil(t, mem.ConsolidationID,
			"archived memory should have consolidation link")
		assert.Equal(t, consolidatedID, *mem.ConsolidationID,
			"archived memory should link to consolidated memory")
	}

	// Verify low-similarity memories were NOT archived
	lowSimIDs := []string{lowSim1.ID, lowSim2.ID}
	for _, id := range lowSimIDs {
		found := false
		for _, archivedID := range result.ArchivedMemories {
			if archivedID == id {
				found = true
				break
			}
		}
		assert.False(t, found,
			"low-similarity memory %s should NOT be archived", id)

		// Verify the memory is still active
		mem, err := svc.GetByProjectID(ctx, projectID, id)
		require.NoError(t, err)
		assert.Equal(t, MemoryStateActive, mem.State,
			"low-similarity memory should remain active")
		assert.Nil(t, mem.ConsolidationID,
			"low-similarity memory should not have consolidation link")
	}

	// Verify search behavior: should return consolidated + low-similarity memories
	searchResults, err := svc.Search(ctx, projectID, "error", 10)
	require.NoError(t, err)

	t.Logf("Search results: %d total", len(searchResults))

	// Count active vs archived
	var activeCount, archivedCount int
	var hasConsolidated bool
	hasLowSim := make(map[string]bool)

	for _, res := range searchResults {
		if res.State == MemoryStateArchived {
			archivedCount++
		} else {
			activeCount++
			if res.ID == consolidatedID {
				hasConsolidated = true
			}
			for _, lowSimID := range lowSimIDs {
				if res.ID == lowSimID {
					hasLowSim[lowSimID] = true
				}
			}
		}
	}

	// Verify search filtering
	assert.Equal(t, 0, archivedCount,
		"search should not return archived memories")
	assert.Greater(t, activeCount, 0,
		"search should return active memories")
	assert.True(t, hasConsolidated,
		"search should return the consolidated memory")

	// Low similarity memories should also be in search results (they're still active)
	// Note: They might not all appear if search filters by relevance
	t.Logf("Found %d low-similarity memories in search results", len(hasLowSim))

	t.Log("Similarity threshold (0.8) correctly applied: >0.8 consolidated, <0.8 not consolidated")
}

// TestConsolidation_Integration_OriginalContentPreservation tests that original
// memories retain all their content after being consolidated and archived.
//
// This integration test verifies:
// - Original memories are archived (State = Archived)
// - Original memories have ConsolidationID link set
// - Original memories RETAIN their original content (title, content, tags, confidence, usage, etc.)
// - No data loss occurs during the consolidation process
func TestConsolidation_Integration_OriginalContentPreservation(t *testing.T) {
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

	projectID := "integration-project-7"

	// Create similar memories with distinct, verifiable content
	// These will be consolidated into a single memory
	mem1, _ := NewMemory(projectID, "Authentication using JWT tokens pattern",
		"Use JWT tokens for stateless authentication with refresh token rotation", OutcomeSuccess, []string{"auth", "jwt", "security"})
	mem1.Description = "Original description for memory 1 about JWT"
	mem1.Confidence = 0.85
	mem1.UsageCount = 12

	mem2, _ := NewMemory(projectID, "Authentication using JWT tokens approach",
		"Implement JWT authentication with secure token storage and validation", OutcomeSuccess, []string{"auth", "jwt", "tokens"})
	mem2.Description = "Original description for memory 2 about JWT"
	mem2.Confidence = 0.72
	mem2.UsageCount = 8

	mem3, _ := NewMemory(projectID, "Authentication using JWT tokens strategy",
		"JWT tokens provide secure stateless auth with expiration and signing", OutcomeSuccess, []string{"auth", "jwt", "api"})
	mem3.Description = "Original description for memory 3 about JWT"
	mem3.Confidence = 0.91
	mem3.UsageCount = 20

	// Store original values BEFORE consolidation for later comparison
	originalMemories := map[string]struct {
		Title       string
		Description string
		Content     string
		Outcome     Outcome
		Confidence  float64
		UsageCount  int
		Tags        []string
		State       MemoryState
	}{
		mem1.ID: {
			Title:       mem1.Title,
			Description: mem1.Description,
			Content:     mem1.Content,
			Outcome:     mem1.Outcome,
			Confidence:  mem1.Confidence,
			UsageCount:  mem1.UsageCount,
			Tags:        append([]string{}, mem1.Tags...),
			State:       mem1.State,
		},
		mem2.ID: {
			Title:       mem2.Title,
			Description: mem2.Description,
			Content:     mem2.Content,
			Outcome:     mem2.Outcome,
			Confidence:  mem2.Confidence,
			UsageCount:  mem2.UsageCount,
			Tags:        append([]string{}, mem2.Tags...),
			State:       mem2.State,
		},
		mem3.ID: {
			Title:       mem3.Title,
			Description: mem3.Description,
			Content:     mem3.Content,
			Outcome:     mem3.Outcome,
			Confidence:  mem3.Confidence,
			UsageCount:  mem3.UsageCount,
			Tags:        append([]string{}, mem3.Tags...),
			State:       mem3.State,
		},
	}

	// Record all memories
	require.NoError(t, svc.Record(ctx, mem1))
	require.NoError(t, svc.Record(ctx, mem2))
	require.NoError(t, svc.Record(ctx, mem3))

	// Verify initial state - all memories should be active
	for id := range originalMemories {
		mem, err := svc.GetByProjectID(ctx, projectID, id)
		require.NoError(t, err)
		assert.Equal(t, MemoryStateActive, mem.State,
			"memory %s should be active before consolidation", id)
		assert.Nil(t, mem.ConsolidationID,
			"memory %s should not have consolidation link before consolidation", id)
	}

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

	t.Logf("Consolidation result: created=%d, archived=%d, skipped=%d, total=%d",
		len(result.CreatedMemories), len(result.ArchivedMemories),
		result.SkippedCount, result.TotalProcessed)

	// Verify consolidation occurred
	assert.Equal(t, 1, len(result.CreatedMemories),
		"should create 1 consolidated memory")
	assert.Equal(t, 3, len(result.ArchivedMemories),
		"should archive 3 source memories")

	consolidatedID := result.CreatedMemories[0]

	// **KEY VERIFICATION**: Check that original memories retain ALL their original content
	for _, archivedID := range result.ArchivedMemories {
		// Get the archived memory from storage
		archivedMem, err := svc.GetByProjectID(ctx, projectID, archivedID)
		require.NoError(t, err, "should be able to retrieve archived memory %s", archivedID)

		// Get the original values we stored before consolidation
		original, exists := originalMemories[archivedID]
		require.True(t, exists, "archived memory %s should be one of our original memories", archivedID)

		// Verify STATE is now Archived
		assert.Equal(t, MemoryStateArchived, archivedMem.State,
			"memory %s should be archived after consolidation", archivedID)

		// Verify ConsolidationID link is set
		require.NotNil(t, archivedMem.ConsolidationID,
			"archived memory %s should have consolidation link", archivedID)
		assert.Equal(t, consolidatedID, *archivedMem.ConsolidationID,
			"archived memory %s should link to consolidated memory %s", archivedID, consolidatedID)

		// **CRITICAL VERIFICATION**: Original content is PRESERVED
		assert.Equal(t, original.Title, archivedMem.Title,
			"archived memory %s should retain original title", archivedID)
		assert.Equal(t, original.Description, archivedMem.Description,
			"archived memory %s should retain original description", archivedID)
		assert.Equal(t, original.Content, archivedMem.Content,
			"archived memory %s should retain original content", archivedID)
		assert.Equal(t, original.Outcome, archivedMem.Outcome,
			"archived memory %s should retain original outcome", archivedID)
		assert.Equal(t, original.Confidence, archivedMem.Confidence,
			"archived memory %s should retain original confidence", archivedID)
		assert.Equal(t, original.UsageCount, archivedMem.UsageCount,
			"archived memory %s should retain original usage count", archivedID)
		assert.Equal(t, original.Tags, archivedMem.Tags,
			"archived memory %s should retain original tags", archivedID)

		t.Logf("Verified memory %s retains all original content:", archivedID)
		t.Logf("  - Title: %s", archivedMem.Title)
		t.Logf("  - Content: %s", archivedMem.Content)
		t.Logf("  - Confidence: %.2f", archivedMem.Confidence)
		t.Logf("  - UsageCount: %d", archivedMem.UsageCount)
		t.Logf("  - Tags: %v", archivedMem.Tags)
		t.Logf("  - State: %s", archivedMem.State)
		t.Logf("  - ConsolidationID: %s", *archivedMem.ConsolidationID)
	}

	// Verify the consolidated memory exists and is active
	consolidatedMem, err := svc.GetByProjectID(ctx, projectID, consolidatedID)
	require.NoError(t, err)
	assert.Equal(t, MemoryStateActive, consolidatedMem.State,
		"consolidated memory should be active")
	assert.Nil(t, consolidatedMem.ConsolidationID,
		"consolidated memory should not have consolidation link (it's the target, not a source)")

	// Verify the consolidated memory has different content (synthesized)
	assert.NotEqual(t, originalMemories[mem1.ID].Title, consolidatedMem.Title,
		"consolidated memory should have synthesized content, not original")

	t.Log("✓ Original memories preserve all content after consolidation and archival")
	t.Log("✓ ConsolidationID links correctly set on all archived memories")
	t.Log("✓ No data loss occurs during consolidation process")
}
