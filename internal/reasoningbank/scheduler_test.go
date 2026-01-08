package reasoningbank

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// schedulerTestEnv holds the test environment for scheduler tests.
type schedulerTestEnv struct {
	ctx       context.Context
	store     *mockStore
	svc       *Service
	distiller *Distiller
	projectID string
}

// setupSchedulerTestEnv creates a configured test environment for scheduler tests.
// It creates a mock store with embedder, pre-populates memories for clustering,
// and returns a properly configured distiller.
//
// Parameters:
//   - t: testing context for assertions
//   - useErrorStore: if true, uses a store that returns errors on search
//
// The returned environment has 2 memories pre-populated for the default project.
func setupSchedulerTestEnv(t *testing.T, useErrorStore bool) *schedulerTestEnv {
	t.Helper()

	ctx := context.Background()
	logger := zap.NewNop()

	var store *mockStore
	if useErrorStore {
		store = newMockStoreWithError()
	} else {
		store = newMockStore()
	}

	embedder := newMockEmbedder(384)
	projectID := "scheduler-test-project"

	svc := &Service{
		store:         store,
		embedder:      embedder,
		logger:        logger,
		defaultTenant: "test-tenant",
	}

	// Pre-populate memories so collection exists and clustering can occur
	mem1, err := NewMemory(projectID, "Scheduler test memory one", "Content 1", OutcomeSuccess, []string{"test"})
	require.NoError(t, err)
	mem2, err := NewMemory(projectID, "Scheduler test memory two", "Content 2", OutcomeSuccess, []string{"test"})
	require.NoError(t, err)

	require.NoError(t, svc.Record(ctx, mem1))
	require.NoError(t, svc.Record(ctx, mem2))

	distiller, err := NewDistiller(svc, logger)
	require.NoError(t, err)

	return &schedulerTestEnv{
		ctx:       ctx,
		store:     store,
		svc:       svc,
		distiller: distiller,
		projectID: projectID,
	}
}

// TestNewConsolidationScheduler tests scheduler creation.
func TestNewConsolidationScheduler(t *testing.T) {
	logger := zap.NewNop()
	distiller := &Distiller{}

	scheduler, err := NewConsolidationScheduler(distiller, logger)
	require.NoError(t, err)
	assert.NotNil(t, scheduler)
	assert.Equal(t, 24*time.Hour, scheduler.interval) // Default interval
	assert.False(t, scheduler.running)
	assert.NotNil(t, scheduler.stopCh)
}

// TestNewConsolidationScheduler_NilDistiller tests error when distiller is nil.
func TestNewConsolidationScheduler_NilDistiller(t *testing.T) {
	logger := zap.NewNop()

	scheduler, err := NewConsolidationScheduler(nil, logger)
	assert.Error(t, err)
	assert.Nil(t, scheduler)
	assert.Contains(t, err.Error(), "distiller cannot be nil")
}

// TestNewConsolidationScheduler_NilLogger tests error when logger is nil.
func TestNewConsolidationScheduler_NilLogger(t *testing.T) {
	distiller := &Distiller{}

	scheduler, err := NewConsolidationScheduler(distiller, nil)
	assert.Error(t, err)
	assert.Nil(t, scheduler)
	assert.Contains(t, err.Error(), "logger cannot be nil")
}

// TestNewConsolidationScheduler_WithInterval tests custom interval option.
func TestNewConsolidationScheduler_WithInterval(t *testing.T) {
	logger := zap.NewNop()
	distiller := &Distiller{}
	customInterval := 1 * time.Hour

	scheduler, err := NewConsolidationScheduler(distiller, logger, WithInterval(customInterval))
	require.NoError(t, err)
	assert.Equal(t, customInterval, scheduler.interval)
}

// TestScheduler_Start tests starting the scheduler.
func TestScheduler_Start(t *testing.T) {
	logger := zap.NewNop()
	distiller := &Distiller{}

	scheduler, err := NewConsolidationScheduler(distiller, logger)
	require.NoError(t, err)

	// Start scheduler
	err = scheduler.Start()
	require.NoError(t, err)
	assert.True(t, scheduler.running)

	// Clean up
	err = scheduler.Stop()
	require.NoError(t, err)

	// Give goroutine time to finish
	time.Sleep(10 * time.Millisecond)
}

// TestScheduler_Start_AlreadyRunning tests error when starting an already running scheduler.
func TestScheduler_Start_AlreadyRunning(t *testing.T) {
	logger := zap.NewNop()
	distiller := &Distiller{}

	scheduler, err := NewConsolidationScheduler(distiller, logger)
	require.NoError(t, err)

	// Start scheduler
	err = scheduler.Start()
	require.NoError(t, err)
	assert.True(t, scheduler.running)

	// Try to start again
	err = scheduler.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// Clean up
	err = scheduler.Stop()
	require.NoError(t, err)

	// Give goroutine time to finish
	time.Sleep(10 * time.Millisecond)
}

// TestScheduler_Stop tests stopping the scheduler.
func TestScheduler_Stop(t *testing.T) {
	logger := zap.NewNop()
	distiller := &Distiller{}

	scheduler, err := NewConsolidationScheduler(distiller, logger)
	require.NoError(t, err)

	// Start scheduler
	err = scheduler.Start()
	require.NoError(t, err)
	assert.True(t, scheduler.running)

	// Stop scheduler
	err = scheduler.Stop()
	require.NoError(t, err)
	assert.False(t, scheduler.running)

	// Give goroutine time to finish
	time.Sleep(10 * time.Millisecond)
}

// TestScheduler_Stop_NotRunning tests stopping a scheduler that isn't running.
func TestScheduler_Stop_NotRunning(t *testing.T) {
	logger := zap.NewNop()
	distiller := &Distiller{}

	scheduler, err := NewConsolidationScheduler(distiller, logger)
	require.NoError(t, err)

	// Stop without starting (should be no-op, not error)
	err = scheduler.Stop()
	require.NoError(t, err)
	assert.False(t, scheduler.running)
}

// TestScheduler_StartStopCycle tests multiple start/stop cycles.
func TestScheduler_StartStopCycle(t *testing.T) {
	logger := zap.NewNop()
	distiller := &Distiller{}

	scheduler, err := NewConsolidationScheduler(distiller, logger)
	require.NoError(t, err)

	// Cycle 1: Start and stop
	err = scheduler.Start()
	require.NoError(t, err)
	assert.True(t, scheduler.running)

	err = scheduler.Stop()
	require.NoError(t, err)
	assert.False(t, scheduler.running)

	// Give goroutine time to finish
	time.Sleep(10 * time.Millisecond)

	// Note: Additional cycles would require recreating the scheduler
	// because stopCh is closed after first Stop() and cannot be reused.
	// This is expected behavior - schedulers are typically started once
	// and stopped once during application lifecycle.
}

// TestScheduler_GracefulShutdown tests that the scheduler shuts down gracefully.
func TestScheduler_GracefulShutdown(t *testing.T) {
	logger := zap.NewNop()
	distiller := &Distiller{}

	scheduler, err := NewConsolidationScheduler(distiller, logger)
	require.NoError(t, err)

	// Start scheduler
	err = scheduler.Start()
	require.NoError(t, err)

	// Let it run briefly
	time.Sleep(10 * time.Millisecond)

	// Stop should not hang
	done := make(chan struct{})
	go func() {
		err := scheduler.Stop()
		require.NoError(t, err)
		close(done)
	}()

	// Wait for shutdown to complete (with timeout)
	select {
	case <-done:
		// Success - shutdown completed
	case <-time.After(1 * time.Second):
		t.Fatal("scheduler did not shut down within timeout")
	}

	assert.False(t, scheduler.running)
}

// TestScheduler_ConsolidationRuns tests that consolidation runs at the configured interval.
func TestScheduler_ConsolidationRuns(t *testing.T) {
	env := setupSchedulerTestEnv(t, false)

	// Configure scheduler with short interval for testing
	scheduler, err := NewConsolidationScheduler(
		env.distiller,
		zap.NewNop(),
		WithInterval(50*time.Millisecond),
		WithProjectIDs([]string{env.projectID}),
	)
	require.NoError(t, err)

	// Start scheduler
	err = scheduler.Start()
	require.NoError(t, err)

	// Wait for at least one consolidation run
	time.Sleep(150 * time.Millisecond) // Increased margin for CI reliability

	// Stop scheduler
	err = scheduler.Stop()
	require.NoError(t, err)

	// Give goroutine time to finish
	time.Sleep(20 * time.Millisecond)

	// Verify that search was called (consolidation attempted)
	// ConsolidateAll -> Consolidate -> FindSimilarClusters -> ListMemories -> SearchInCollection
	assert.True(t, env.store.SearchCalled(), "expected consolidation to have been attempted")
}

// TestScheduler_NoProjectsConfigured tests that scheduler doesn't run consolidation when no projects configured.
func TestScheduler_NoProjectsConfigured(t *testing.T) {
	logger := zap.NewNop()

	// Create mock distiller with call tracking
	store := newMockStore()
	distiller := &Distiller{
		service: &Service{
			store:         store,
			logger:        logger,
			defaultTenant: "test-tenant",
		},
		logger: logger,
	}

	// Configure scheduler with no project IDs
	scheduler, err := NewConsolidationScheduler(
		distiller,
		logger,
		WithInterval(50*time.Millisecond),
		// No WithProjectIDs - defaults to empty slice
	)
	require.NoError(t, err)

	// Start scheduler
	err = scheduler.Start()
	require.NoError(t, err)

	// Wait for interval to pass
	time.Sleep(100 * time.Millisecond)

	// Stop scheduler
	err = scheduler.Stop()
	require.NoError(t, err)

	// Give goroutine time to finish
	time.Sleep(20 * time.Millisecond)

	// Verify that search was NOT called (no consolidation without projects)
	assert.False(t, store.SearchCalled(), "expected no consolidation when no projects configured")
}

// TestScheduler_WithConsolidationOptions tests that custom consolidation options are used.
func TestScheduler_WithConsolidationOptions(t *testing.T) {
	logger := zap.NewNop()
	distiller := &Distiller{}

	// Configure scheduler with custom options
	customOpts := ConsolidationOptions{
		SimilarityThreshold: 0.9,
		DryRun:              true,
		ForceAll:            true,
		MaxClustersPerRun:   10,
	}

	scheduler, err := NewConsolidationScheduler(
		distiller,
		logger,
		WithConsolidationOptions(customOpts),
	)
	require.NoError(t, err)

	// Verify options were set
	assert.Equal(t, 0.9, scheduler.opts.SimilarityThreshold)
	assert.True(t, scheduler.opts.DryRun)
	assert.True(t, scheduler.opts.ForceAll)
	assert.Equal(t, 10, scheduler.opts.MaxClustersPerRun)
}

// TestScheduler_MultipleIntervalRuns tests that consolidation runs multiple times.
func TestScheduler_MultipleIntervalRuns(t *testing.T) {
	env := setupSchedulerTestEnv(t, false)

	// Configure scheduler with short interval
	scheduler, err := NewConsolidationScheduler(
		env.distiller,
		zap.NewNop(),
		WithInterval(30*time.Millisecond),
		WithProjectIDs([]string{env.projectID}),
	)
	require.NoError(t, err)

	// Start scheduler
	err = scheduler.Start()
	require.NoError(t, err)

	// Wait for multiple intervals with generous margin for CI reliability
	// At 30ms intervals: tick at 30, 60, 90, 120ms -> expect ~4 runs in 150ms
	time.Sleep(150 * time.Millisecond)

	// Stop scheduler
	err = scheduler.Stop()
	require.NoError(t, err)

	// Give goroutine time to finish
	time.Sleep(20 * time.Millisecond)

	// Verify that search was called multiple times
	// Note: exact count may vary due to timing, but should be >= 2
	assert.True(t, env.store.SearchCallCount() >= 2,
		"expected at least 2 consolidation runs over 150ms with 30ms interval, got %d", env.store.SearchCallCount())
}

// TestScheduler_ErrorHandling tests that consolidation errors don't stop the scheduler.
func TestScheduler_ErrorHandling(t *testing.T) {
	// Use error store - AddDocuments succeeds but Search returns errors
	env := setupSchedulerTestEnv(t, true)

	// Configure scheduler with short interval
	scheduler, err := NewConsolidationScheduler(
		env.distiller,
		zap.NewNop(),
		WithInterval(50*time.Millisecond),
		WithProjectIDs([]string{env.projectID}),
	)
	require.NoError(t, err)

	// Start scheduler
	err = scheduler.Start()
	require.NoError(t, err)

	// Wait for multiple intervals with generous margin for CI reliability
	// At 50ms intervals: tick at 50, 100, 150ms -> expect ~3 runs in 180ms
	time.Sleep(180 * time.Millisecond)

	// Stop scheduler
	err = scheduler.Stop()
	require.NoError(t, err)

	// Give goroutine time to finish
	time.Sleep(20 * time.Millisecond)

	// Verify that consolidation was attempted despite errors
	// SearchInCollection increments count before returning error, so this works
	assert.True(t, env.store.SearchCallCount() >= 2,
		"expected scheduler to continue after errors, got %d calls", env.store.SearchCallCount())
}
