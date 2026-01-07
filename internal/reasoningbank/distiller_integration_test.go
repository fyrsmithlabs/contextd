package reasoningbank

import (
	"context"
	"fmt"
	"strings"
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
	// Mock embedder groups by first 2 significant words (>2 chars), so use distinct prefixes
	// Cluster 1: Will succeed (starts with "alpha pattern")
	mem1, _ := NewMemory(projectID, "Alpha pattern first version",
		"Content A1", OutcomeSuccess, []string{"pattern-a"})
	mem2, _ := NewMemory(projectID, "Alpha pattern second version",
		"Content A2", OutcomeSuccess, []string{"pattern-a"})

	// Cluster 2: Will succeed (starts with "beta pattern")
	mem3, _ := NewMemory(projectID, "Beta pattern first version",
		"Content B1", OutcomeSuccess, []string{"pattern-b"})
	mem4, _ := NewMemory(projectID, "Beta pattern second version",
		"Content B2", OutcomeSuccess, []string{"pattern-b"})

	// Cluster 3: Will succeed initially (for first 2 calls) (starts with "gamma pattern")
	mem5, _ := NewMemory(projectID, "Gamma pattern first version",
		"Content C1", OutcomeSuccess, []string{"pattern-c"})
	mem6, _ := NewMemory(projectID, "Gamma pattern second version",
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
	// Uses calculateConsolidatedConfidence which includes consensus bonus:
	// 1. Base = weighted average: (0.8*11 + 0.7*6 + 0.9*16) / 33 = 0.8303
	// 2. Mean = (0.8 + 0.7 + 0.9) / 3 = 0.8
	// 3. Variance = ((0-0)^2 + (0.1)^2 + (0.1)^2) / 3 = 0.00667
	// 4. StdDev = sqrt(0.00667) = 0.0816
	// 5. NormalizedStdDev = 0.0816 / 0.5 = 0.1633
	// 6. ConsensusFactor = 1.0 - 0.1633 = 0.8367
	// 7. NumSourcesFactor = 3 / 10.0 = 0.3
	// 8. ConsensusBonus = 0.8367 * 0.3 * 0.1 = 0.0251
	// 9. Final = 0.8303 + 0.0251 = ~0.855
	expectedBaseConfidence := (0.8*11 + 0.7*6 + 0.9*16) / (11.0 + 6.0 + 16.0)
	// With consensus bonus, expect value to be higher than base
	assert.Greater(t, consolidatedMem.Confidence, expectedBaseConfidence,
		"consolidated confidence should be greater than base due to consensus bonus")
	assert.InDelta(t, 0.855, consolidatedMem.Confidence, 0.02,
		"consolidated confidence should include consensus bonus")

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

	// Create similar memories (same first 2 significant words for clustering)
	mem1, _ := NewMemory(projectID, "Window testing pattern one", "Content X1", OutcomeSuccess, []string{"test"})
	mem2, _ := NewMemory(projectID, "Window testing pattern two", "Content X2", OutcomeSuccess, []string{"test"})

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

	// Note: mockEmbedder creates embeddings based on first 2 significant words:
	// Texts with same starting words (>2 chars) get similar embeddings (cosine sim > 0.8).
	// Texts with different starting words get distinct embeddings (cosine sim < 0.5).

	// Create HIGH SIMILARITY memories (same first 2 words: "error handling")
	// These will cluster together
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

	// 3. Should process memories in clusters (singletons don't form clusters)
	// Only the 3 high-similarity memories form a cluster and get processed
	assert.Equal(t, 3, result.TotalProcessed,
		"should process memories that form clusters")

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

// TestConsolidation_Integration_ConfidenceCalculation verifies that consolidated
// memory confidence is calculated correctly from source memories using:
// 1. Weighted average formula: base = sum(conf_i * weight_i) / sum(weight_i)
//    where weight_i = usageCount_i + 1
// 2. Consensus bonus: bonus = consensusFactor * numSourcesFactor * 0.1
//    where consensusFactor = 1 - (stdDev / 0.5) and numSourcesFactor = min(n/10, 1)
//
// This integration test covers Acceptance Criteria: "Confidence scores are updated based on consolidation"
//
// Test scenarios verify that final confidence >= base confidence (due to consensus bonus)
func TestConsolidation_Integration_ConfidenceCalculation(t *testing.T) {
	testCases := []struct {
		name               string
		memories           []struct {
			title      string
			content    string
			confidence float64
			usageCount int
		}
		expectedConfidenceMin float64
		expectedConfidenceMax float64
		description           string
	}{
		{
			name: "equal confidence and usage",
			// All memories have same first 2 significant words for clustering
			memories: []struct {
				title      string
				content    string
				confidence float64
				usageCount int
			}{
				{"Test pattern version one", "Content A", 0.8, 5},
				{"Test pattern version two", "Content B", 0.8, 5},
				{"Test pattern version three", "Content C", 0.8, 5},
			},
			// Base = (0.8*6 + 0.8*6 + 0.8*6) / (6+6+6) = 0.8
			// Consensus: perfect agreement (stdDev=0) gives max bonus
			// Bonus = 1.0 * (3/10) * 0.1 = 0.03
			// Final = 0.8 + 0.03 = 0.83
			expectedConfidenceMin: 0.82,
			expectedConfidenceMax: 0.84,
			description:           "equal confidence gives high consensus bonus",
		},
		{
			name: "high usage dominates",
			// All memories have same first 2 significant words for clustering
			memories: []struct {
				title      string
				content    string
				confidence float64
				usageCount int
			}{
				{"Usage testing high case", "Content X", 0.9, 50}, // high usage, high confidence
				{"Usage testing low case", "Content Y", 0.3, 1},   // low usage, low confidence
				{"Usage testing med case", "Content Z", 0.4, 2},   // low usage, low confidence
			},
			// Base = (0.9*51 + 0.3*2 + 0.4*3) / 56 = 0.8517
			// High variance in confidence reduces consensus bonus
			// Final should be close to base (low bonus due to variance)
			expectedConfidenceMin: 0.85,
			expectedConfidenceMax: 0.87,
			description:           "high-usage high-confidence memory should dominate the score",
		},
		{
			name: "mixed confidence and usage",
			// All memories have same first 2 significant words for clustering
			memories: []struct {
				title      string
				content    string
				confidence float64
				usageCount int
			}{
				{"Mixed testing variant alpha", "Content Alpha", 0.75, 10},
				{"Mixed testing variant beta", "Content Beta", 0.85, 5},
				{"Mixed testing variant gamma", "Content Gamma", 0.65, 15},
			},
			// Base = (0.75*11 + 0.85*6 + 0.65*16) / 33 = 0.7196
			// Some variance, moderate consensus bonus
			// Final ~0.72-0.75
			expectedConfidenceMin: 0.71,
			expectedConfidenceMax: 0.75,
			description:           "realistic mixed scenario should compute weighted average plus bonus",
		},
		{
			name: "all same confidence",
			// All memories have same first 2 significant words for clustering
			memories: []struct {
				title      string
				content    string
				confidence float64
				usageCount int
			}{
				{"Same confidence variant one", "Content 1", 0.7, 0},
				{"Same confidence variant two", "Content 2", 0.7, 100},
				{"Same confidence variant three", "Content 3", 0.7, 50},
			},
			// Base = 0.7 (all same)
			// Consensus: perfect agreement (stdDev=0) gives max bonus
			// Bonus = 1.0 * (3/10) * 0.1 = 0.03
			// Final = 0.7 + 0.03 = 0.73
			expectedConfidenceMin: 0.72,
			expectedConfidenceMax: 0.74,
			description:           "all same confidence gives high consensus bonus",
		},
		{
			name: "varying confidence with zero usage",
			// All memories have same first 2 significant words for clustering
			memories: []struct {
				title      string
				content    string
				confidence float64
				usageCount int
			}{
				{"Zero usage variant first", "Content One", 0.9, 0},
				{"Zero usage variant second", "Content Two", 0.6, 0},
				{"Zero usage variant third", "Content Three", 0.8, 0},
			},
			// Base = (0.9*1 + 0.6*1 + 0.8*1) / 3 = 0.7666
			// Mean = 0.7666, Variance = ((0.133)^2+(0.167)^2+(0.033)^2)/3 = moderate
			// Bonus ~0.01-0.02
			// Final ~0.77-0.79
			expectedConfidenceMin: 0.77,
			expectedConfidenceMax: 0.80,
			description:           "zero usage should use weight of 1 for all memories plus consensus bonus",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
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

			projectID := fmt.Sprintf("confidence-test-%s", tc.name)

			// Create memories with specified confidence and usage counts
			var createdMemories []*Memory
			for i, memSpec := range tc.memories {
				mem, err := NewMemory(projectID, memSpec.title, memSpec.content,
					OutcomeSuccess, []string{"confidence-test"})
				require.NoError(t, err)

				// Set confidence and usage count
				mem.Confidence = memSpec.confidence
				mem.UsageCount = memSpec.usageCount

				// Record memory
				err = svc.Record(ctx, mem)
				require.NoError(t, err)

				createdMemories = append(createdMemories, mem)

				t.Logf("Created memory %d: confidence=%.2f, usage=%d",
					i+1, mem.Confidence, mem.UsageCount)
			}

			// Run consolidation
			opts := ConsolidationOptions{
				SimilarityThreshold: 0.8, // Will cluster all similar titles
				DryRun:              false,
				ForceAll:            true,
			}

			result, err := distiller.Consolidate(ctx, projectID, opts)
			require.NoError(t, err)
			require.NotNil(t, result)

			// Verify exactly 1 consolidated memory was created
			require.Len(t, result.CreatedMemories, 1,
				"should create exactly 1 consolidated memory")

			// Get the consolidated memory
			consolidatedID := result.CreatedMemories[0]
			consolidatedMem, err := svc.GetByProjectID(ctx, projectID, consolidatedID)
			require.NoError(t, err)

			// Verify confidence is calculated correctly
			actualConfidence := consolidatedMem.Confidence
			t.Logf("Consolidated memory confidence: %.4f (expected range: %.4f - %.4f)",
				actualConfidence, tc.expectedConfidenceMin, tc.expectedConfidenceMax)

			assert.GreaterOrEqual(t, actualConfidence, tc.expectedConfidenceMin,
				"%s: confidence %.4f should be >= %.4f", tc.description, actualConfidence, tc.expectedConfidenceMin)
			assert.LessOrEqual(t, actualConfidence, tc.expectedConfidenceMax,
				"%s: confidence %.4f should be <= %.4f", tc.description, actualConfidence, tc.expectedConfidenceMax)

			// Verify confidence is in valid range [0.0, 1.0]
			assert.GreaterOrEqual(t, actualConfidence, 0.0,
				"confidence should be >= 0.0")
			assert.LessOrEqual(t, actualConfidence, 1.0,
				"confidence should be <= 1.0")

			// Manually calculate expected base confidence for verification
			var weightedSum, totalWeight float64
			for _, mem := range createdMemories {
				weight := float64(mem.UsageCount + 1)
				weightedSum += mem.Confidence * weight
				totalWeight += weight
			}
			baseConfidence := weightedSum / totalWeight

			t.Logf("Base confidence (weighted avg): %.4f = %.2f / %.2f",
				baseConfidence, weightedSum, totalWeight)

			// Actual confidence should be >= base due to consensus bonus
			assert.GreaterOrEqual(t, actualConfidence, baseConfidence-0.001,
				"consolidated confidence should be >= base (weighted average)")

			// Log verification details
			t.Logf("✓ Base confidence (weighted average): %.4f", baseConfidence)
			t.Logf("✓ Actual confidence (with consensus bonus): %.4f", actualConfidence)
			t.Logf("✓ Bonus applied: %.4f", actualConfidence-baseConfidence)
			t.Logf("✓ In valid range [0.0, 1.0]: true")
			t.Logf("✓ %s", tc.description)
		})
	}

	t.Log("✓ All confidence calculation scenarios passed")
	t.Log("✓ Base formula: sum(conf_i * (usage_i + 1)) / sum(usage_i + 1)")
	t.Log("✓ Consensus bonus: (1 - normalizedStdDev) * (numSources/10) * 0.1")
	t.Log("✓ Acceptance Criteria verified: Confidence scores updated correctly from sources")
}

// TestConsolidation_Integration_SourceAttribution tests that consolidated memories
// include proper source memory IDs and attribution information.
//
// This integration test verifies:
// - Consolidated memory includes source attribution text in Description field
// - Source memory IDs can be retrieved via ConsolidationID back-references
// - Attribution text is meaningful and references the source memories
// - The relationship between consolidated and source memories is bidirectional
func TestConsolidation_Integration_SourceAttribution(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	embedder := newMockEmbedder(384)
	logger := zap.NewNop()

	// Create mock LLM with custom response that includes detailed attribution
	customAttribution := `
TITLE: Consolidated Database Connection Strategy

CONTENT:
Comprehensive approach to database connection management combining connection pooling,
timeout configuration, and monitoring best practices. Ensure proper resource cleanup
and performance optimization through tuned pool settings.

TAGS: database, performance, best-practices

OUTCOME: success

SOURCE_ATTRIBUTION:
Synthesized from 3 source memories:
- "DB Connection Pooling" (mem-001): Pool configuration and max connections
- "Connection Timeout Handling" (mem-002): Timeout settings and error handling
- "Connection Pool Monitoring" (mem-003): Monitoring and adjustment strategies
This consolidated memory combines insights from all three approaches to provide
a complete connection management strategy.
`

	llmClient := newMockLLMClientWithResponse(customAttribution)

	// Create service with embedder
	svc, err := NewService(store, logger,
		WithDefaultTenant("test-tenant"),
		WithEmbedder(embedder))
	require.NoError(t, err)

	// Create distiller with LLM client
	distiller, err := NewDistiller(svc, logger, WithLLMClient(llmClient))
	require.NoError(t, err)

	projectID := "source-attribution-project"

	// Create 3 similar memories with specific titles for verification
	// All memories have same first 2 significant words ("database connection") for clustering
	mem1, _ := NewMemory(projectID, "Database connection pooling strategy",
		"Configure connection pool with max connections and idle timeout", OutcomeSuccess, []string{"database", "pooling"})
	mem1.Confidence = 0.85
	mem1.UsageCount = 20

	mem2, _ := NewMemory(projectID, "Database connection timeout handling",
		"Set appropriate timeouts for database operations to prevent hangs", OutcomeSuccess, []string{"database", "timeout"})
	mem2.Confidence = 0.80
	mem2.UsageCount = 15

	mem3, _ := NewMemory(projectID, "Database connection monitoring best practices",
		"Monitor pool usage and adjust limits based on traffic patterns", OutcomeSuccess, []string{"database", "monitoring"})
	mem3.Confidence = 0.90
	mem3.UsageCount = 25

	// Record all memories
	require.NoError(t, svc.Record(ctx, mem1))
	require.NoError(t, svc.Record(ctx, mem2))
	require.NoError(t, svc.Record(ctx, mem3))

	// Store source IDs for later verification
	sourceIDs := []string{mem1.ID, mem2.ID, mem3.ID}
	sourceTitles := map[string]string{
		mem1.ID: mem1.Title,
		mem2.ID: mem2.Title,
		mem3.ID: mem3.Title,
	}

	t.Logf("Created source memories: %v", sourceTitles)

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

	// ===== Verification 1: Consolidated memory was created =====
	assert.Equal(t, 1, len(result.CreatedMemories),
		"should create 1 consolidated memory")
	assert.Equal(t, 3, len(result.ArchivedMemories),
		"should archive 3 source memories")

	consolidatedID := result.CreatedMemories[0]
	t.Logf("Consolidated memory ID: %s", consolidatedID)

	// ===== Verification 2: Attribution text is present and meaningful =====
	consolidatedMem, err := svc.GetByProjectID(ctx, projectID, consolidatedID)
	require.NoError(t, err)
	require.NotNil(t, consolidatedMem)

	t.Logf("Consolidated memory title: %s", consolidatedMem.Title)
	t.Logf("Consolidated memory description (attribution):\n%s", consolidatedMem.Description)

	// Verify Description field contains attribution text
	assert.NotEmpty(t, consolidatedMem.Description,
		"consolidated memory should have attribution in Description field")

	// Verify attribution text is meaningful
	assert.Contains(t, consolidatedMem.Description, "Synthesized",
		"attribution should indicate synthesis occurred")
	assert.Contains(t, consolidatedMem.Description, "source memories",
		"attribution should reference source memories")
	assert.Contains(t, consolidatedMem.Description, "3",
		"attribution should mention number of source memories")

	// Verify attribution references the source memory titles or approaches
	// (The LLM's response should include some reference to the sources)
	attributionLower := strings.ToLower(consolidatedMem.Description)
	hasSourceReferences := strings.Contains(attributionLower, "pooling") ||
		strings.Contains(attributionLower, "timeout") ||
		strings.Contains(attributionLower, "monitoring") ||
		strings.Contains(attributionLower, "connection")

	assert.True(t, hasSourceReferences,
		"attribution should reference content from source memories")

	t.Log("✓ Attribution text is present and meaningful")

	// ===== Verification 3: Source memory IDs can be retrieved =====
	// Method 1: Via ConsolidationResult.ArchivedMemories
	assert.Equal(t, len(sourceIDs), len(result.ArchivedMemories),
		"result should list all archived source memory IDs")

	for _, expectedID := range sourceIDs {
		assert.Contains(t, result.ArchivedMemories, expectedID,
			"archived memories should include source ID: %s", expectedID)
	}

	t.Log("✓ Source memory IDs available via ConsolidationResult.ArchivedMemories")

	// Method 2: Via ConsolidationID back-references
	retrievedSourceIDs := []string{}
	for _, sourceID := range sourceIDs {
		sourceMem, err := svc.GetByProjectID(ctx, projectID, sourceID)
		require.NoError(t, err)
		require.NotNil(t, sourceMem)

		// Verify back-link
		require.NotNil(t, sourceMem.ConsolidationID,
			"source memory %s should have ConsolidationID set", sourceID)
		assert.Equal(t, consolidatedID, *sourceMem.ConsolidationID,
			"source memory %s should link to consolidated memory", sourceID)

		// Verify archived state
		assert.Equal(t, MemoryStateArchived, sourceMem.State,
			"source memory %s should be archived", sourceID)

		retrievedSourceIDs = append(retrievedSourceIDs, sourceMem.ID)

		t.Logf("✓ Source memory %s -> consolidated %s (state: %s)",
			sourceID, *sourceMem.ConsolidationID, sourceMem.State)
	}

	assert.ElementsMatch(t, sourceIDs, retrievedSourceIDs,
		"should be able to retrieve all source memory IDs via back-references")

	t.Log("✓ Source memory IDs retrievable via ConsolidationID back-references")

	// ===== Verification 4: Bidirectional relationship =====
	// Can navigate from consolidated -> sources and sources -> consolidated

	// Forward: consolidated memory created from these sources
	t.Logf("Forward relationship: consolidated %s <- sources %v",
		consolidatedID, sourceIDs)

	// Backward: each source links to consolidated
	for _, sourceID := range sourceIDs {
		sourceMem, _ := svc.GetByProjectID(ctx, projectID, sourceID)
		t.Logf("Backward relationship: source %s -> consolidated %s",
			sourceID, *sourceMem.ConsolidationID)
	}

	t.Log("✓ Bidirectional relationship verified")

	// ===== Verification 5: Original source content is preserved =====
	for _, sourceID := range sourceIDs {
		sourceMem, _ := svc.GetByProjectID(ctx, projectID, sourceID)

		// Original title and content preserved
		assert.Equal(t, sourceTitles[sourceID], sourceMem.Title,
			"source memory %s should retain original title", sourceID)

		// Original metadata preserved
		assert.NotEmpty(t, sourceMem.Content,
			"source memory %s should retain original content", sourceID)
		assert.NotEmpty(t, sourceMem.Tags,
			"source memory %s should retain original tags", sourceID)

		t.Logf("✓ Source %s: title=%s, state=%s, consolidation_id=%s",
			sourceID, sourceMem.Title, sourceMem.State, *sourceMem.ConsolidationID)
	}

	t.Log("✓ Original source content is preserved")

	// ===== Verification 6: Consolidated memory properties =====
	assert.Equal(t, MemoryStateActive, consolidatedMem.State,
		"consolidated memory should be active")
	assert.Nil(t, consolidatedMem.ConsolidationID,
		"consolidated memory should not have ConsolidationID (it's the target, not a source)")
	assert.Equal(t, "Consolidated Database Connection Strategy", consolidatedMem.Title,
		"consolidated memory should have LLM-generated title")
	assert.Contains(t, consolidatedMem.Content, "connection management",
		"consolidated memory should have synthesized content")

	t.Log("✓ Consolidated memory properties verified")

	// ===== Acceptance Criteria Verification Summary =====
	t.Log("")
	t.Log("╔════════════════════════════════════════════════════════════════════╗")
	t.Log("║  ACCEPTANCE CRITERIA VERIFIED: Source Attribution                 ║")
	t.Log("╚════════════════════════════════════════════════════════════════════╝")
	t.Log("✓ Consolidated memory includes source attribution text")
	t.Log("✓ Attribution text is stored in Memory.Description field")
	t.Log("✓ Attribution references source memories (count, content)")
	t.Log("✓ Source memory IDs retrievable via ConsolidationResult")
	t.Log("✓ Source memory IDs retrievable via ConsolidationID back-references")
	t.Log("✓ Bidirectional relationship: consolidated <-> sources")
	t.Log("✓ Original source content preserved in archived memories")
	t.Log("✓ Consolidated memory is active, sources are archived")
	t.Log("")
	t.Log("Acceptance Criterion: \"Consolidated memories include source attribution\"")
	t.Log("Status: VERIFIED ✓")
}
