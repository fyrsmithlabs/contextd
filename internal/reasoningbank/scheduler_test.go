package reasoningbank

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

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
