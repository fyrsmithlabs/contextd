package vectorstore

import (
	"context"
	"testing"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/secrets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCircuitBreaker_Allow(t *testing.T) {
	cb := NewCircuitBreaker(3, 1*time.Second)

	// Initially closed - should allow
	assert.True(t, cb.Allow())

	// Record 3 failures - circuit should open
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()

	// Circuit open - should not allow
	assert.False(t, cb.Allow())

	// After reset duration, should allow one test request (half-open)
	time.Sleep(1100 * time.Millisecond)
	assert.True(t, cb.Allow())

	// Half-open - subsequent requests blocked
	assert.False(t, cb.Allow())

	// Success resets circuit
	cb.RecordSuccess()
	assert.True(t, cb.Allow())
}

func TestCircuitBreaker_State(t *testing.T) {
	cb := NewCircuitBreaker(2, 1*time.Second)

	assert.Equal(t, "closed", cb.State())

	cb.RecordFailure()
	cb.RecordFailure()
	assert.Equal(t, "open", cb.State())

	time.Sleep(1100 * time.Millisecond)
	cb.Allow() // Transition to half-open
	assert.Equal(t, "half-open", cb.State())

	cb.RecordSuccess()
	assert.Equal(t, "closed", cb.State())
}

func TestSyncManager_TriggerSync(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	// Create mock embedder
	embedder := &MockEmbedder{
		embedding: make([]float32, 384),
	}

	// Create stores
	remoteCfg := ChromemConfig{
		Path:              t.TempDir(),
		DefaultCollection: "test_remote",
		VectorSize:        384,
	}
	remote, err := NewChromemStore(remoteCfg, embedder, logger)
	require.NoError(t, err)
	defer remote.Close()

	localCfg := ChromemConfig{
		Path:              t.TempDir(),
		DefaultCollection: "test_local",
		VectorSize:        384,
	}
	local, err := NewChromemStore(localCfg, embedder, logger)
	require.NoError(t, err)
	defer local.Close()

	// Create health monitor
	healthChecker := NewMockHealthChecker()
	healthChecker.SetHealthy(true)
	health := NewHealthMonitor(ctx, healthChecker, 30*time.Second, logger)
	defer health.Stop()

	// Create WAL
	scrubber := secrets.NoopScrubber{}
	wal, err := NewWAL(t.TempDir(), &scrubber, logger)
	require.NoError(t, err)

	// Create sync manager
	sm := NewSyncManager(ctx, wal, local, remote, health, logger)
	sm.Start()
	defer sm.Stop()

	// Trigger sync
	sm.TriggerSync()

	// Give sync time to process
	time.Sleep(100 * time.Millisecond)

	// Should not error
}

func TestSyncManager_Stop(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	// Create mock embedder
	embedder := &MockEmbedder{
		embedding: make([]float32, 384),
	}

	// Create stores
	remoteCfg := ChromemConfig{
		Path:              t.TempDir(),
		DefaultCollection: "test_remote",
		VectorSize:        384,
	}
	remote, err := NewChromemStore(remoteCfg, embedder, logger)
	require.NoError(t, err)
	defer remote.Close()

	localCfg := ChromemConfig{
		Path:              t.TempDir(),
		DefaultCollection: "test_local",
		VectorSize:        384,
	}
	local, err := NewChromemStore(localCfg, embedder, logger)
	require.NoError(t, err)
	defer local.Close()

	// Create health monitor
	healthChecker := NewMockHealthChecker()
	healthChecker.SetHealthy(true)
	health := NewHealthMonitor(ctx, healthChecker, 30*time.Second, logger)
	defer health.Stop()

	// Create WAL
	scrubber := secrets.NoopScrubber{}
	wal, err := NewWAL(t.TempDir(), &scrubber, logger)
	require.NoError(t, err)

	// Create sync manager
	sm := NewSyncManager(ctx, wal, local, remote, health, logger)
	sm.Start()

	// Stop should not error
	err = sm.Stop()
	assert.NoError(t, err)
}
