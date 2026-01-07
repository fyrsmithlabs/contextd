package reasoningbank

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// NOTE: Manual MCP trigger tests are in internal/mcp/handlers/memory_test.go
// to avoid import cycles (handlers imports reasoningbank).
//
// This file tests the automatic scheduler trigger, which lives in the
// reasoningbank package and doesn't have import cycle issues.

// TestMemoryConsolidation_AutomaticSchedulerTrigger verifies that the
// scheduler can automatically trigger consolidation on an interval.
//
// This test verifies the automatic trigger path:
// 1. Scheduler starts with configured interval
// 2. Scheduler automatically triggers consolidation on interval
// 3. Consolidation runs without manual intervention
// 4. Scheduler continues running after consolidation
func TestMemoryConsolidation_AutomaticSchedulerTrigger(t *testing.T) {
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

	projectID := "scheduler-trigger-project"

	// Create similar memories that should be consolidated
	mem1, _ := NewMemory(projectID, "Caching strategy for API responses",
		"Cache GET requests with TTL based on data volatility", OutcomeSuccess, []string{"caching"})
	mem2, _ := NewMemory(projectID, "Caching best practices for APIs",
		"Use cache headers to reduce server load", OutcomeSuccess, []string{"caching"})
	mem3, _ := NewMemory(projectID, "Caching implementation for API layer",
		"Implement Redis caching for frequently accessed data", OutcomeSuccess, []string{"caching"})

	// Record memories
	require.NoError(t, svc.Record(ctx, mem1))
	require.NoError(t, svc.Record(ctx, mem2))
	require.NoError(t, svc.Record(ctx, mem3))

	// Create scheduler with short interval for testing
	scheduler, err := NewConsolidationScheduler(
		distiller,
		logger,
		WithInterval(50*time.Millisecond), // Short interval for fast test
		WithProjectIDs([]string{projectID}),
		WithConsolidationOptions(ConsolidationOptions{
			SimilarityThreshold: 0.8,
			DryRun:              false,
			ForceAll:            true, // Bypass consolidation window for test
			MaxClustersPerRun:   0,
		}),
	)
	require.NoError(t, err)

	// Start the scheduler (automatic trigger)
	err = scheduler.Start()
	require.NoError(t, err, "scheduler should start successfully")
	assert.True(t, scheduler.running, "scheduler should be running")

	t.Log("✓ Scheduler started - waiting for automatic consolidation trigger...")

	// Wait for at least one consolidation run
	time.Sleep(100 * time.Millisecond)

	// Stop the scheduler
	err = scheduler.Stop()
	require.NoError(t, err, "scheduler should stop successfully")
	assert.False(t, scheduler.running, "scheduler should be stopped")

	// Give goroutine time to finish
	time.Sleep(20 * time.Millisecond)

	// Verify that consolidation was triggered automatically
	// The scheduler calls ConsolidateAll -> Consolidate -> FindSimilarClusters -> ListMemories
	// ListMemories uses SearchInCollection, so we can verify it was called
	assert.True(t, store.searchCalled, "scheduler should have triggered consolidation")

	// Verify the LLM was called (indicates consolidation actually ran, not just skipped)
	assert.Greater(t, llmClient.CallCount(), 0, "LLM should be called when consolidation runs")

	t.Logf("✓ Automatic scheduler trigger successful:")
	t.Logf("  Consolidation triggered: %v", store.searchCalled)
	t.Logf("  LLM calls made: %d", llmClient.CallCount())
	t.Logf("  Scheduler lifecycle: start → run → stop")
}

// TestMemoryConsolidation_SchedulerDryRun verifies that dry run mode
// works correctly with the scheduler.
func TestMemoryConsolidation_SchedulerDryRun(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()
	store := newMockStore()
	embedder := newMockEmbedder(384)
	llmClient := newMockLLMClient()

	svc, err := NewService(store, logger,
		WithDefaultTenant("test-tenant"),
		WithEmbedder(embedder))
	require.NoError(t, err)

	distiller, err := NewDistiller(svc, logger, WithLLMClient(llmClient))
	require.NoError(t, err)

	projectID := "dry-run-auto"

	// Create memories
	mem1, _ := NewMemory(projectID, "Memory A", "Content A", OutcomeSuccess, []string{"test"})
	mem2, _ := NewMemory(projectID, "Memory B", "Content B", OutcomeSuccess, []string{"test"})
	require.NoError(t, svc.Record(ctx, mem1))
	require.NoError(t, svc.Record(ctx, mem2))

	// Automatic trigger with dry run
	scheduler, err := NewConsolidationScheduler(
		distiller,
		logger,
		WithInterval(30*time.Millisecond),
		WithProjectIDs([]string{projectID}),
		WithConsolidationOptions(ConsolidationOptions{
			DryRun:   true,
			ForceAll: true,
		}),
	)
	require.NoError(t, err)

	err = scheduler.Start()
	require.NoError(t, err)

	time.Sleep(60 * time.Millisecond)

	err = scheduler.Stop()
	require.NoError(t, err)

	time.Sleep(20 * time.Millisecond)

	// Dry run mode should still search for clusters but not call LLM
	assert.True(t, store.searchCalled, "scheduler should search even in dry run")
	assert.Equal(t, 0, llmClient.CallCount(), "dry run should not call LLM")
	t.Log("✓ Automatic trigger dry run: search called, no LLM calls")
}
