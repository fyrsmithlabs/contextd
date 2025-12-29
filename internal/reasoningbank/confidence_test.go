package reasoningbank

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfidenceCalculator_ComputeFromSignals(t *testing.T) {
	// Test that confidence is computed correctly from signals
	// Using the formula: alpha / (alpha + beta)
	// Where alpha/beta are updated by weighted signals

	weights := NewProjectWeights("proj_123")
	mc := NewMemoryConfidence("mem_abc123")

	// Initial confidence should be 0.5 (1:1 prior)
	assert.InDelta(t, 0.5, mc.Score(), 0.001)

	// Add a positive explicit signal (highest weight initially)
	signal := Signal{Type: SignalExplicit, Positive: true}
	mc.Update(signal, weights)

	// Confidence should increase
	assert.Greater(t, mc.Score(), 0.5)
}

func TestConfidenceCalculator_MultipleSignals(t *testing.T) {
	// Test confidence evolution with multiple signals of different types

	weights := NewProjectWeights("proj_123")
	mc := NewMemoryConfidence("mem_abc123")

	// Add positive signals of each type
	mc.Update(Signal{Type: SignalExplicit, Positive: true}, weights)
	mc.Update(Signal{Type: SignalUsage, Positive: true}, weights)
	mc.Update(Signal{Type: SignalOutcome, Positive: true}, weights)

	// Confidence should be significantly higher than initial
	assert.Greater(t, mc.Score(), 0.6)

	// Add some negative signals
	mc.Update(Signal{Type: SignalExplicit, Positive: false}, weights)
	mc.Update(Signal{Type: SignalOutcome, Positive: false}, weights)

	// Confidence should decrease but still be above 0.5 due to net positive
	score := mc.Score()
	assert.Greater(t, score, 0.4)
}

func TestConfidenceCalculator_WeightEvolution(t *testing.T) {
	// Test that weights evolve based on feedback accuracy

	weights := NewProjectWeights("proj_123")

	initialUsageWeight := weights.WeightFor(SignalUsage)

	// Simulate: usage signal correctly predicted helpful feedback 5 times
	for i := 0; i < 5; i++ {
		recentSignals := []Signal{{Type: SignalUsage, Positive: true}}
		weights.LearnFromFeedback(true, recentSignals)
	}

	// Usage weight should have increased
	newUsageWeight := weights.WeightFor(SignalUsage)
	assert.Greater(t, newUsageWeight, initialUsageWeight)
}

func TestConfidenceCalculator_OutcomeWeightIncreases(t *testing.T) {
	// Test that outcome signal weight increases when it correctly predicts

	weights := NewProjectWeights("proj_123")

	initialOutcomeWeight := weights.WeightFor(SignalOutcome)

	// Simulate: outcome signals correctly predicted helpful feedback
	for i := 0; i < 10; i++ {
		recentSignals := []Signal{{Type: SignalOutcome, Positive: true}}
		weights.LearnFromFeedback(true, recentSignals)
	}

	newOutcomeWeight := weights.WeightFor(SignalOutcome)
	assert.Greater(t, newOutcomeWeight, initialOutcomeWeight)
}

func TestConfidenceCalculator_ComputeFromAggregatesAndRecent(t *testing.T) {
	// Test computing confidence from both aggregates (old signals) and recent signals

	weights := NewProjectWeights("proj_123")

	// Create aggregate with historical data
	agg := NewSignalAggregate("mem_abc123", "proj_123")
	agg.ExplicitPos = 5
	agg.ExplicitNeg = 1
	agg.UsagePos = 10
	agg.UsageNeg = 2
	agg.OutcomePos = 3
	agg.OutcomeNeg = 1

	// Create recent signals
	recentSignals := []Signal{
		{Type: SignalExplicit, Positive: true},
		{Type: SignalUsage, Positive: true},
		{Type: SignalOutcome, Positive: false},
	}

	// Compute confidence using the hybrid approach
	confidence := ComputeConfidenceFromHybrid(agg, recentSignals, weights)

	// Should be above 0.5 given more positive than negative signals
	assert.Greater(t, confidence, 0.5)
	// But not too high given some negative signals
	assert.Less(t, confidence, 0.9)
}

func TestSignalStore_PersistAndRetrieve(t *testing.T) {
	// Test signal storage interface
	// This tests the interface that will be implemented with vectorstore

	ctx := context.Background()
	store := NewInMemorySignalStore()

	signal := &Signal{
		ID:        "sig_123",
		MemoryID:  "mem_abc123",
		ProjectID: "proj_123",
		Type:      SignalExplicit,
		Positive:  true,
		SessionID: "sess_xyz",
		Timestamp: time.Now(),
	}

	// Store signal
	err := store.StoreSignal(ctx, signal)
	require.NoError(t, err)

	// Retrieve recent signals
	signals, err := store.GetRecentSignals(ctx, "mem_abc123", 24*time.Hour)
	require.NoError(t, err)
	assert.Len(t, signals, 1)
	assert.Equal(t, "sig_123", signals[0].ID)
}

func TestSignalStore_GetAggregate(t *testing.T) {
	// Test aggregate retrieval

	ctx := context.Background()
	store := NewInMemorySignalStore()

	// Store an aggregate
	agg := NewSignalAggregate("mem_abc123", "proj_123")
	agg.ExplicitPos = 5
	agg.ExplicitNeg = 2

	err := store.StoreAggregate(ctx, agg)
	require.NoError(t, err)

	// Retrieve aggregate
	retrieved, err := store.GetAggregate(ctx, "mem_abc123")
	require.NoError(t, err)
	assert.Equal(t, 5, retrieved.ExplicitPos)
	assert.Equal(t, 2, retrieved.ExplicitNeg)
}

func TestSignalStore_GetProjectWeights(t *testing.T) {
	// Test project weights retrieval

	ctx := context.Background()
	store := NewInMemorySignalStore()

	// Store weights
	weights := NewProjectWeights("proj_123")
	weights.UsageAlpha = 8.0 // Modified from default

	err := store.StoreProjectWeights(ctx, weights)
	require.NoError(t, err)

	// Retrieve weights
	retrieved, err := store.GetProjectWeights(ctx, "proj_123")
	require.NoError(t, err)
	assert.Equal(t, 8.0, retrieved.UsageAlpha)
}

func TestSignalStore_RollupOldSignals(t *testing.T) {
	// Test rolling up signals older than 30 days

	ctx := context.Background()
	store := NewInMemorySignalStore()

	// Create signals: some old, some new
	oldTime := time.Now().Add(-31 * 24 * time.Hour)
	newTime := time.Now()

	oldSignal := &Signal{
		ID:        "sig_old",
		MemoryID:  "mem_abc123",
		ProjectID: "proj_123",
		Type:      SignalExplicit,
		Positive:  true,
		Timestamp: oldTime,
	}

	newSignal := &Signal{
		ID:        "sig_new",
		MemoryID:  "mem_abc123",
		ProjectID: "proj_123",
		Type:      SignalExplicit,
		Positive:  true,
		Timestamp: newTime,
	}

	_ = store.StoreSignal(ctx, oldSignal)
	_ = store.StoreSignal(ctx, newSignal)

	// Initialize aggregate
	agg := NewSignalAggregate("mem_abc123", "proj_123")
	store.StoreAggregate(ctx, agg)

	// Rollup old signals
	err := store.RollupOldSignals(ctx, "mem_abc123", 30*24*time.Hour)
	require.NoError(t, err)

	// Get aggregate - should have old signal counted
	updated, err := store.GetAggregate(ctx, "mem_abc123")
	require.NoError(t, err)
	assert.Equal(t, 1, updated.ExplicitPos)

	// Recent signals should only include new signal
	recent, err := store.GetRecentSignals(ctx, "mem_abc123", 30*24*time.Hour)
	require.NoError(t, err)
	assert.Len(t, recent, 1)
	assert.Equal(t, "sig_new", recent[0].ID)
}
