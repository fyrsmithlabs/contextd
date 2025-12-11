// Package framework provides the integration test framework for contextd.
package framework

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTestMetrics_Creation(t *testing.T) {
	t.Run("creates metrics instance successfully", func(t *testing.T) {
		metrics, err := NewTestMetrics()
		require.NoError(t, err)
		require.NotNil(t, metrics)
	})
}

func TestTestMetrics_MemorySearchTracking(t *testing.T) {
	t.Run("tracks memory search hits and misses", func(t *testing.T) {
		metrics, err := NewTestMetrics()
		require.NoError(t, err)

		ctx := context.Background()

		// Record some hits
		metrics.RecordMemorySearch(ctx, 10*time.Millisecond, true, false)
		metrics.RecordMemorySearch(ctx, 15*time.Millisecond, true, false)
		metrics.RecordMemorySearch(ctx, 20*time.Millisecond, true, false)

		// Record a miss
		metrics.RecordMemorySearch(ctx, 5*time.Millisecond, false, false)

		stats := metrics.GetStats()
		assert.Equal(t, int64(3), stats.MemoryHitCount, "should have 3 hits")
		assert.Equal(t, int64(1), stats.MemoryMissCount, "should have 1 miss")
		assert.Equal(t, 0.75, stats.MemoryHitRate, "hit rate should be 75%")
	})

	t.Run("tracks cross-developer searches", func(t *testing.T) {
		metrics, err := NewTestMetrics()
		require.NoError(t, err)

		ctx := context.Background()

		// Record regular searches
		metrics.RecordMemorySearch(ctx, 10*time.Millisecond, true, false)
		metrics.RecordMemorySearch(ctx, 10*time.Millisecond, true, false)

		// Record cross-developer searches
		metrics.RecordMemorySearch(ctx, 10*time.Millisecond, true, true)
		metrics.RecordMemorySearch(ctx, 10*time.Millisecond, true, true)

		stats := metrics.GetStats()
		assert.Equal(t, int64(4), stats.TotalSearchCount, "should have 4 total searches")
		assert.Equal(t, int64(2), stats.CrossDevSearchCount, "should have 2 cross-dev searches")
		assert.Equal(t, 0.5, stats.CrossDevSearchRate, "cross-dev rate should be 50%")
	})
}

func TestTestMetrics_CheckpointTracking(t *testing.T) {
	t.Run("tracks checkpoint success and failure", func(t *testing.T) {
		metrics, err := NewTestMetrics()
		require.NoError(t, err)

		ctx := context.Background()

		// Record successful loads
		metrics.RecordCheckpointLoad(ctx, 50*time.Millisecond, true)
		metrics.RecordCheckpointLoad(ctx, 60*time.Millisecond, true)
		metrics.RecordCheckpointLoad(ctx, 70*time.Millisecond, true)

		// Record a failure
		metrics.RecordCheckpointLoad(ctx, 100*time.Millisecond, false)

		stats := metrics.GetStats()
		assert.Equal(t, int64(3), stats.CheckpointSuccessCount, "should have 3 successes")
		assert.Equal(t, int64(1), stats.CheckpointFailureCount, "should have 1 failure")
		assert.Equal(t, 0.75, stats.CheckpointSuccessRate, "success rate should be 75%")
	})
}

func TestTestMetrics_SpanCreation(t *testing.T) {
	t.Run("creates suite span", func(t *testing.T) {
		metrics, err := NewTestMetrics()
		require.NoError(t, err)

		ctx := context.Background()
		newCtx, span := metrics.StartSuiteSpan(ctx, "policy_compliance")
		defer span.End()

		assert.NotNil(t, span, "span should not be nil")
		// Note: Without a configured OTEL exporter, IsRecording() may return false
		// The important thing is that the span was created without error
		assert.NotEqual(t, ctx, newCtx, "context should be updated with span")
	})

	t.Run("creates test span", func(t *testing.T) {
		metrics, err := NewTestMetrics()
		require.NoError(t, err)

		ctx := context.Background()
		newCtx, span := metrics.StartTestSpan(ctx, "tdd_policy_enforcement")
		defer span.End()

		assert.NotNil(t, span, "span should not be nil")
		// Note: Without a configured OTEL exporter, IsRecording() may return false
		assert.NotEqual(t, ctx, newCtx, "context should be updated with span")
	})

	t.Run("creates phase span", func(t *testing.T) {
		metrics, err := NewTestMetrics()
		require.NoError(t, err)

		ctx := context.Background()
		_, span := metrics.StartPhaseSpan(ctx, "setup")
		defer span.End()

		assert.NotNil(t, span, "span should not be nil")
	})
}

func TestTestMetrics_TestPassFail(t *testing.T) {
	t.Run("records test pass", func(t *testing.T) {
		metrics, err := NewTestMetrics()
		require.NoError(t, err)

		ctx := context.Background()
		// This doesn't panic - the counter is incremented internally
		metrics.RecordTestPass(ctx, "policy_compliance", "tdd_enforcement")
	})

	t.Run("records test fail", func(t *testing.T) {
		metrics, err := NewTestMetrics()
		require.NoError(t, err)

		ctx := context.Background()
		// This doesn't panic - the counter is incremented internally
		metrics.RecordTestFail(ctx, "policy_compliance", "secrets_test")
	})
}

func TestTestMetrics_SuiteDuration(t *testing.T) {
	t.Run("records suite duration", func(t *testing.T) {
		metrics, err := NewTestMetrics()
		require.NoError(t, err)

		ctx := context.Background()
		// This doesn't panic - the histogram is recorded internally
		metrics.RecordSuiteDuration(ctx, "policy_compliance", 5*time.Second)
	})
}

func TestTestMetrics_Reset(t *testing.T) {
	t.Run("resets all counters", func(t *testing.T) {
		metrics, err := NewTestMetrics()
		require.NoError(t, err)

		ctx := context.Background()

		// Add some data
		metrics.RecordMemorySearch(ctx, 10*time.Millisecond, true, true)
		metrics.RecordCheckpointLoad(ctx, 50*time.Millisecond, true)

		// Verify data exists
		stats := metrics.GetStats()
		assert.Equal(t, int64(1), stats.MemoryHitCount)
		assert.Equal(t, int64(1), stats.CheckpointSuccessCount)

		// Reset
		metrics.Reset()

		// Verify data is cleared
		stats = metrics.GetStats()
		assert.Equal(t, int64(0), stats.MemoryHitCount)
		assert.Equal(t, int64(0), stats.MemoryMissCount)
		assert.Equal(t, int64(0), stats.CheckpointSuccessCount)
		assert.Equal(t, int64(0), stats.CheckpointFailureCount)
		assert.Equal(t, int64(0), stats.CrossDevSearchCount)
		assert.Equal(t, int64(0), stats.TotalSearchCount)
	})
}

func TestTestMetrics_EdgeCases(t *testing.T) {
	t.Run("handles zero divisions gracefully", func(t *testing.T) {
		metrics, err := NewTestMetrics()
		require.NoError(t, err)

		stats := metrics.GetStats()

		// All rates should be 0 when no data
		assert.Equal(t, float64(0), stats.MemoryHitRate)
		assert.Equal(t, float64(0), stats.CheckpointSuccessRate)
		assert.Equal(t, float64(0), stats.CrossDevSearchRate)
	})

	t.Run("handles 100% hit rate", func(t *testing.T) {
		metrics, err := NewTestMetrics()
		require.NoError(t, err)

		ctx := context.Background()

		// All hits, no misses
		metrics.RecordMemorySearch(ctx, 10*time.Millisecond, true, false)
		metrics.RecordMemorySearch(ctx, 10*time.Millisecond, true, false)

		stats := metrics.GetStats()
		assert.Equal(t, float64(1), stats.MemoryHitRate, "100% hit rate")
	})

	t.Run("handles 0% hit rate", func(t *testing.T) {
		metrics, err := NewTestMetrics()
		require.NoError(t, err)

		ctx := context.Background()

		// All misses
		metrics.RecordMemorySearch(ctx, 10*time.Millisecond, false, false)
		metrics.RecordMemorySearch(ctx, 10*time.Millisecond, false, false)

		stats := metrics.GetStats()
		assert.Equal(t, float64(0), stats.MemoryHitRate, "0% hit rate")
	})
}
