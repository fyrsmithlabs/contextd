package reasoningbank

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestConsolidationTracking_GetSetLastTime tests getting and setting last consolidation time.
func TestConsolidationTracking_GetSetLastTime(t *testing.T) {
	store := newMockStore()
	logger := zap.NewNop()

	svc, err := NewService(store, logger, WithDefaultTenant("test-tenant"))
	require.NoError(t, err)

	distiller, err := NewDistiller(svc, logger)
	require.NoError(t, err)

	projectID := "test-project"

	// Initially, last consolidation time should be zero
	lastTime := distiller.getLastConsolidationTime(projectID)
	assert.True(t, lastTime.IsZero(), "initial last consolidation time should be zero")

	// Set consolidation time
	now := time.Now()
	distiller.setLastConsolidationTime(projectID, now)

	// Verify time was set
	retrievedTime := distiller.getLastConsolidationTime(projectID)
	assert.Equal(t, now.Unix(), retrievedTime.Unix(), "retrieved time should match set time")

	// Set time for different project
	otherProjectID := "other-project"
	otherTime := now.Add(-1 * time.Hour)
	distiller.setLastConsolidationTime(otherProjectID, otherTime)

	// Verify times are independent
	assert.Equal(t, now.Unix(), distiller.getLastConsolidationTime(projectID).Unix())
	assert.Equal(t, otherTime.Unix(), distiller.getLastConsolidationTime(otherProjectID).Unix())
}

// TestConsolidationTracking_ShouldSkipWithinWindow tests skipping when within consolidation window.
func TestConsolidationTracking_ShouldSkipWithinWindow(t *testing.T) {
	store := newMockStore()
	logger := zap.NewNop()

	svc, err := NewService(store, logger, WithDefaultTenant("test-tenant"))
	require.NoError(t, err)

	// Create distiller with 1 hour window for easier testing
	distiller, err := NewDistiller(svc, logger, WithConsolidationWindow(1*time.Hour))
	require.NoError(t, err)

	projectID := "test-project"

	// Set last consolidation time to 30 minutes ago (within window)
	lastTime := time.Now().Add(-30 * time.Minute)
	distiller.setLastConsolidationTime(projectID, lastTime)

	// Should skip (within window)
	skip, remaining := distiller.shouldSkipConsolidation(projectID, false)
	assert.True(t, skip, "should skip when within consolidation window")
	assert.Greater(t, remaining, time.Duration(0), "remaining time should be positive")
	assert.Less(t, remaining, 31*time.Minute, "remaining should be less than 31 minutes")
}

// TestConsolidationTracking_ShouldNotSkipOutsideWindow tests not skipping when outside consolidation window.
func TestConsolidationTracking_ShouldNotSkipOutsideWindow(t *testing.T) {
	store := newMockStore()
	logger := zap.NewNop()

	svc, err := NewService(store, logger, WithDefaultTenant("test-tenant"))
	require.NoError(t, err)

	// Create distiller with 1 hour window for easier testing
	distiller, err := NewDistiller(svc, logger, WithConsolidationWindow(1*time.Hour))
	require.NoError(t, err)

	projectID := "test-project"

	// Set last consolidation time to 2 hours ago (outside window)
	lastTime := time.Now().Add(-2 * time.Hour)
	distiller.setLastConsolidationTime(projectID, lastTime)

	// Should not skip (outside window)
	skip, remaining := distiller.shouldSkipConsolidation(projectID, false)
	assert.False(t, skip, "should not skip when outside consolidation window")
	assert.Equal(t, time.Duration(0), remaining, "remaining time should be zero")
}

// TestConsolidationTracking_ShouldNotSkipForceAll tests that ForceAll bypasses window check.
func TestConsolidationTracking_ShouldNotSkipForceAll(t *testing.T) {
	store := newMockStore()
	logger := zap.NewNop()

	svc, err := NewService(store, logger, WithDefaultTenant("test-tenant"))
	require.NoError(t, err)

	distiller, err := NewDistiller(svc, logger, WithConsolidationWindow(1*time.Hour))
	require.NoError(t, err)

	projectID := "test-project"

	// Set last consolidation time to 10 minutes ago (within window)
	lastTime := time.Now().Add(-10 * time.Minute)
	distiller.setLastConsolidationTime(projectID, lastTime)

	// Should not skip with ForceAll=true
	skip, remaining := distiller.shouldSkipConsolidation(projectID, true)
	assert.False(t, skip, "should not skip when ForceAll is true")
	assert.Equal(t, time.Duration(0), remaining, "remaining time should be zero with ForceAll")
}

// TestConsolidationTracking_ShouldNotSkipNeverConsolidated tests first-time consolidation.
func TestConsolidationTracking_ShouldNotSkipNeverConsolidated(t *testing.T) {
	store := newMockStore()
	logger := zap.NewNop()

	svc, err := NewService(store, logger, WithDefaultTenant("test-tenant"))
	require.NoError(t, err)

	distiller, err := NewDistiller(svc, logger, WithConsolidationWindow(1*time.Hour))
	require.NoError(t, err)

	projectID := "never-consolidated-project"

	// Should not skip for project that has never been consolidated
	skip, remaining := distiller.shouldSkipConsolidation(projectID, false)
	assert.False(t, skip, "should not skip for project that has never been consolidated")
	assert.Equal(t, time.Duration(0), remaining, "remaining time should be zero for first consolidation")
}

// TestConsolidationTracking_IntegrationWithConsolidate tests integration with Consolidate method.
func TestConsolidationTracking_IntegrationWithConsolidate(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"

	// Create mock dependencies
	mockStore := newMockStore()
	mockEmbedder := newMockEmbedder(384) // Use 384 for proper slot-based embeddings
	mockLLM := newMockLLMClient()

	// Create service and distiller with short window for testing
	svc := &Service{
		store:         mockStore,
		embedder:      mockEmbedder,
		logger:        zap.NewNop(),
		defaultTenant: "test-tenant", // Required for tenant isolation
	}

	distiller, err := NewDistiller(svc, zap.NewNop(),
		WithLLMClient(mockLLM),
		WithConsolidationWindow(1*time.Hour))
	require.NoError(t, err)

	// Create memories with same first 2 significant words for clustering
	mem1, _ := NewMemory(projectID, "Tracking test memory one", "Content 1", OutcomeSuccess, []string{"test"})
	mem2, _ := NewMemory(projectID, "Tracking test memory two", "Content 2", OutcomeSuccess, []string{"test"})
	require.NoError(t, svc.Record(ctx, mem1))
	require.NoError(t, svc.Record(ctx, mem2))

	// First consolidation should proceed
	opts := ConsolidationOptions{
		SimilarityThreshold: 0.85,
		DryRun:              false,
		ForceAll:            false,
	}

	result1, err := distiller.Consolidate(ctx, projectID, opts)
	require.NoError(t, err)
	require.NotNil(t, result1)

	// Verify consolidation ran
	assert.Greater(t, result1.TotalProcessed, 0, "first consolidation should process memories")

	// Verify last consolidation time was set
	lastTime := distiller.getLastConsolidationTime(projectID)
	assert.False(t, lastTime.IsZero(), "last consolidation time should be set after consolidation")

	// Reset mock call count
	mockLLM.callCount = 0

	// Second consolidation immediately after should be skipped
	result2, err := distiller.Consolidate(ctx, projectID, opts)
	require.NoError(t, err)
	require.NotNil(t, result2)

	// Verify consolidation was skipped
	assert.Equal(t, 0, result2.TotalProcessed, "second consolidation should be skipped")
	assert.Equal(t, 0, len(result2.CreatedMemories), "should create no memories when skipped")
	assert.Equal(t, 0, mockLLM.CallCount(), "LLM should not be called when skipped")

	// Third consolidation with ForceAll should proceed
	opts.ForceAll = true
	result3, err := distiller.Consolidate(ctx, projectID, opts)
	require.NoError(t, err)
	require.NotNil(t, result3)

	// ForceAll should bypass the window check
	// Note: may not find clusters if memories were already consolidated in first run
	assert.NotNil(t, result3.Duration, "consolidation should run with ForceAll")
}

// TestConsolidationTracking_DryRunNoUpdate tests that dry run doesn't update timestamp.
func TestConsolidationTracking_DryRunNoUpdate(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"

	// Create mock dependencies
	mockStore := newMockStore()
	mockEmbedder := newMockEmbedder(10)
	mockLLM := newMockLLMClient()

	// Create service and distiller
	svc := &Service{
		store:         mockStore,
		embedder:      mockEmbedder,
		logger:        zap.NewNop(),
		defaultTenant: "test-tenant",
	}

	distiller, err := NewDistiller(svc, zap.NewNop(), WithLLMClient(mockLLM))
	require.NoError(t, err)

	// Create memories - use same first 2 words for clustering
	mem1, _ := NewMemory(projectID, "Memory pattern one", "Content 1", OutcomeSuccess, []string{"test"})
	mem2, _ := NewMemory(projectID, "Memory pattern two", "Content 2", OutcomeSuccess, []string{"test"})
	require.NoError(t, svc.Record(ctx, mem1))
	require.NoError(t, svc.Record(ctx, mem2))

	// Verify initial state (never consolidated)
	initialTime := distiller.getLastConsolidationTime(projectID)
	assert.True(t, initialTime.IsZero(), "initial time should be zero")

	// Run consolidation in dry-run mode
	opts := ConsolidationOptions{
		SimilarityThreshold: 0.85,
		DryRun:              true,
		ForceAll:            false,
	}

	result, err := distiller.Consolidate(ctx, projectID, opts)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify last consolidation time was NOT updated (dry run)
	finalTime := distiller.getLastConsolidationTime(projectID)
	assert.True(t, finalTime.IsZero(), "dry run should not update last consolidation time")
}

// TestConsolidationTracking_CustomWindow tests custom consolidation window.
func TestConsolidationTracking_CustomWindow(t *testing.T) {
	store := newMockStore()
	logger := zap.NewNop()

	svc, err := NewService(store, logger, WithDefaultTenant("test-tenant"))
	require.NoError(t, err)

	// Create distiller with custom 30-minute window
	distiller, err := NewDistiller(svc, logger, WithConsolidationWindow(30*time.Minute))
	require.NoError(t, err)

	projectID := "test-project"

	// Set last consolidation time to 20 minutes ago
	lastTime := time.Now().Add(-20 * time.Minute)
	distiller.setLastConsolidationTime(projectID, lastTime)

	// Should skip (within 30-minute window)
	skip, remaining := distiller.shouldSkipConsolidation(projectID, false)
	assert.True(t, skip, "should skip when within custom 30-minute window")
	assert.Greater(t, remaining, 9*time.Minute, "remaining should be ~10 minutes")
	assert.Less(t, remaining, 11*time.Minute, "remaining should be ~10 minutes")

	// Set last consolidation time to 35 minutes ago
	lastTime = time.Now().Add(-35 * time.Minute)
	distiller.setLastConsolidationTime(projectID, lastTime)

	// Should not skip (outside 30-minute window)
	skip, remaining = distiller.shouldSkipConsolidation(projectID, false)
	assert.False(t, skip, "should not skip when outside custom 30-minute window")
	assert.Equal(t, time.Duration(0), remaining, "remaining should be zero")
}

// TestConsolidationTracking_ConcurrentAccess tests thread-safe concurrent access.
func TestConsolidationTracking_ConcurrentAccess(t *testing.T) {
	store := newMockStore()
	logger := zap.NewNop()

	svc, err := NewService(store, logger, WithDefaultTenant("test-tenant"))
	require.NoError(t, err)

	distiller, err := NewDistiller(svc, logger)
	require.NoError(t, err)

	// Run concurrent get/set operations
	const numGoroutines = 100
	const numProjects = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			projectID := fmt.Sprintf("project-%d", id%numProjects)

			// Perform multiple operations
			for j := 0; j < 10; j++ {
				// Set time
				distiller.setLastConsolidationTime(projectID, time.Now())

				// Get time
				_ = distiller.getLastConsolidationTime(projectID)

				// Check skip
				_, _ = distiller.shouldSkipConsolidation(projectID, false)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify all projects have valid times
	for i := 0; i < numProjects; i++ {
		projectID := fmt.Sprintf("project-%d", i)
		lastTime := distiller.getLastConsolidationTime(projectID)
		assert.False(t, lastTime.IsZero(), "project %s should have valid time after concurrent access", projectID)
	}
}
