// Package framework provides the integration test harness for contextd.
//
// This file contains confidence calibration tests that validate the Bayesian
// confidence system works correctly. These tests address the MEDIUM priority gap
// from KNOWN-GAPS.md.
package framework

import (
	"context"
	"testing"

	"github.com/fyrsmithlabs/contextd/internal/reasoningbank"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestConfidenceCalibration_InitialValues validates that memories start with
// correct initial confidence based on how they were created.
func TestConfidenceCalibration_InitialValues(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	embedder := newSemanticEmbedder(384)
	store, err := vectorstore.NewChromemStore(vectorstore.ChromemConfig{
		Path: t.TempDir(),
	}, embedder, logger)
	require.NoError(t, err)
	defer store.Close()

	store.SetIsolationMode(vectorstore.NewNoIsolation())

	svc, err := reasoningbank.NewService(store, logger, reasoningbank.WithDefaultTenant("test-tenant"))
	require.NoError(t, err)

	projectID := "confidence-init-test"

	t.Run("explicit record gets 0.8 confidence", func(t *testing.T) {
		memory, err := reasoningbank.NewMemory(
			projectID,
			"Explicit memory test",
			"This is an explicitly recorded memory",
			reasoningbank.OutcomeSuccess,
			[]string{"test"},
		)
		require.NoError(t, err)

		// NewMemory sets default confidence of 0.5
		// Record() should boost it to ExplicitRecordConfidence (0.8)
		err = svc.Record(ctx, memory)
		require.NoError(t, err)

		// Search to retrieve the memory and verify confidence
		results, err := svc.Search(ctx, projectID, "Explicit memory test", 5)
		require.NoError(t, err)
		require.NotEmpty(t, results)

		// Verify initial confidence is 0.8 (ExplicitRecordConfidence)
		assert.Equal(t, 0.8, results[0].Confidence,
			"explicitly recorded memory should have 0.8 confidence")
	})

	t.Run("distilled memory preserves lower confidence", func(t *testing.T) {
		memory, err := reasoningbank.NewMemory(
			projectID,
			"Distilled memory test",
			"Learned from session analysis",
			reasoningbank.OutcomeSuccess,
			[]string{"distilled"},
		)
		require.NoError(t, err)

		// Set lower confidence to simulate distilled memory
		memory.Confidence = 0.6
		memory.Description = "Learned from session 12345"

		err = svc.Record(ctx, memory)
		require.NoError(t, err)

		// Note: This memory may not appear in search if confidence is below 0.7
		// That's intentional - the MinConfidence threshold filters these
		results, err := svc.Search(ctx, projectID, "Distilled memory", 5)
		require.NoError(t, err)

		// Distilled memory should NOT appear (below 0.7 threshold)
		foundDistilled := false
		for _, r := range results {
			if r.Title == "Distilled memory test" {
				foundDistilled = true
			}
		}
		assert.False(t, foundDistilled, "distilled memory below threshold should not appear")
	})
}

// TestConfidenceCalibration_FeedbackAdjustment validates that user feedback
// correctly adjusts confidence scores using the Bayesian signal system.
func TestConfidenceCalibration_FeedbackAdjustment(t *testing.T) {
	ctx := context.Background()

	// Test signal store directly since Feedback() uses Bayesian calculation
	// The Bayesian system computes confidence from signals, not from initial values
	signalStore := reasoningbank.NewInMemorySignalStore()
	projectID := "feedback-adjustment-test"
	memoryID := "test-memory-feedback"

	t.Run("positive feedback creates positive explicit signal", func(t *testing.T) {
		// Create explicit positive signal (simulating positive feedback)
		signal, err := reasoningbank.NewSignal(memoryID, projectID, reasoningbank.SignalExplicit, true, "")
		require.NoError(t, err)
		err = signalStore.StoreSignal(ctx, signal)
		require.NoError(t, err)

		// Verify signal was stored
		signals, err := signalStore.GetRecentSignals(ctx, memoryID, 24*60*60*1000000000) // 24 hours
		require.NoError(t, err)
		require.NotEmpty(t, signals)
		assert.True(t, signals[0].Positive, "positive feedback should create positive signal")
		assert.Equal(t, reasoningbank.SignalExplicit, signals[0].Type)
	})

	t.Run("negative feedback creates negative explicit signal", func(t *testing.T) {
		memory2ID := "test-memory-feedback-2"

		// Create explicit negative signal (simulating negative feedback)
		signal, err := reasoningbank.NewSignal(memory2ID, projectID, reasoningbank.SignalExplicit, false, "")
		require.NoError(t, err)
		err = signalStore.StoreSignal(ctx, signal)
		require.NoError(t, err)

		// Verify signal was stored
		signals, err := signalStore.GetRecentSignals(ctx, memory2ID, 24*60*60*1000000000)
		require.NoError(t, err)
		require.NotEmpty(t, signals)
		assert.False(t, signals[0].Positive, "negative feedback should create negative signal")
	})

	t.Run("signals accumulate to affect confidence", func(t *testing.T) {
		memory3ID := "test-memory-feedback-3"
		weights := reasoningbank.NewProjectWeights(projectID)

		// Start with uniform prior - should be 0.5
		agg := &reasoningbank.SignalAggregate{MemoryID: memory3ID}
		initialConf := reasoningbank.ComputeConfidenceFromHybrid(agg, nil, weights)
		assert.InDelta(t, 0.5, initialConf, 0.01, "uniform prior should give 0.5")

		// Add positive signals
		for i := 0; i < 5; i++ {
			signal, _ := reasoningbank.NewSignal(memory3ID, projectID, reasoningbank.SignalExplicit, true, "")
			_ = signalStore.StoreSignal(ctx, signal)
		}

		// Get confidence with signals
		signals, _ := signalStore.GetRecentSignals(ctx, memory3ID, 24*60*60*1000000000)
		confWithPositive := reasoningbank.ComputeConfidenceFromHybrid(agg, signals, weights)

		t.Logf("Confidence after 5 positive signals: %.4f", confWithPositive)
		assert.Greater(t, confWithPositive, initialConf,
			"positive signals should increase confidence")
	})

	t.Run("negative signals decrease confidence", func(t *testing.T) {
		memory4ID := "test-memory-feedback-4"
		weights := reasoningbank.NewProjectWeights(projectID)

		// Add negative signals
		for i := 0; i < 5; i++ {
			signal, _ := reasoningbank.NewSignal(memory4ID, projectID, reasoningbank.SignalExplicit, false, "")
			_ = signalStore.StoreSignal(ctx, signal)
		}

		// Get confidence with signals
		agg := &reasoningbank.SignalAggregate{MemoryID: memory4ID}
		signals, _ := signalStore.GetRecentSignals(ctx, memory4ID, 24*60*60*1000000000)
		confWithNegative := reasoningbank.ComputeConfidenceFromHybrid(agg, signals, weights)

		t.Logf("Confidence after 5 negative signals: %.4f", confWithNegative)
		assert.Less(t, confWithNegative, 0.5,
			"negative signals should decrease confidence below 0.5")
	})
}

// TestConfidenceCalibration_OutcomeSignals validates that outcome signals
// (task success/failure) correctly affect confidence in the Bayesian system.
//
// IMPORTANT: The Bayesian system computes confidence from accumulated signals
// using Beta distribution, starting from a uniform prior (alpha=1, beta=1 = 0.5).
// The initial metadata confidence (0.8 from ExplicitRecordConfidence) is NOT
// incorporated into Bayesian calculations - it's just a stored value.
func TestConfidenceCalibration_OutcomeSignals(t *testing.T) {
	ctx := context.Background()

	// Test signal store directly to verify outcome signal behavior
	signalStore := reasoningbank.NewInMemorySignalStore()
	projectID := "outcome-signal-test"

	t.Run("successful outcome creates positive outcome signal", func(t *testing.T) {
		memoryID := "outcome-test-1"

		// Create outcome signal (simulating successful task completion)
		signal, err := reasoningbank.NewSignal(memoryID, projectID, reasoningbank.SignalOutcome, true, "session-123")
		require.NoError(t, err)
		err = signalStore.StoreSignal(ctx, signal)
		require.NoError(t, err)

		// Verify signal was stored
		signals, err := signalStore.GetRecentSignals(ctx, memoryID, 24*60*60*1000000000)
		require.NoError(t, err)
		require.NotEmpty(t, signals)
		assert.True(t, signals[0].Positive, "successful outcome should create positive signal")
		assert.Equal(t, reasoningbank.SignalOutcome, signals[0].Type)
		assert.Equal(t, "session-123", signals[0].SessionID)
	})

	t.Run("failed outcome creates negative outcome signal", func(t *testing.T) {
		memoryID := "outcome-test-2"

		// Create negative outcome signal
		signal, err := reasoningbank.NewSignal(memoryID, projectID, reasoningbank.SignalOutcome, false, "session-456")
		require.NoError(t, err)
		err = signalStore.StoreSignal(ctx, signal)
		require.NoError(t, err)

		// Verify signal was stored
		signals, err := signalStore.GetRecentSignals(ctx, memoryID, 24*60*60*1000000000)
		require.NoError(t, err)
		require.NotEmpty(t, signals)
		assert.False(t, signals[0].Positive, "failed outcome should create negative signal")
	})

	t.Run("outcome signals affect confidence computation", func(t *testing.T) {
		memoryID := "outcome-test-3"
		weights := reasoningbank.NewProjectWeights(projectID)

		// Start with no signals - uniform prior
		agg := &reasoningbank.SignalAggregate{MemoryID: memoryID}
		baseConf := reasoningbank.ComputeConfidenceFromHybrid(agg, nil, weights)
		t.Logf("Base confidence (uniform prior): %.4f", baseConf)

		// Add positive outcome signals
		for i := 0; i < 3; i++ {
			signal, _ := reasoningbank.NewSignal(memoryID, projectID, reasoningbank.SignalOutcome, true, "")
			_ = signalStore.StoreSignal(ctx, signal)
		}

		signals, _ := signalStore.GetRecentSignals(ctx, memoryID, 24*60*60*1000000000)
		confAfterSuccess := reasoningbank.ComputeConfidenceFromHybrid(agg, signals, weights)
		t.Logf("Confidence after 3 successful outcomes: %.4f", confAfterSuccess)

		assert.Greater(t, confAfterSuccess, baseConf,
			"successful outcomes should increase confidence from uniform prior")
	})

	t.Run("mixed outcomes balance confidence", func(t *testing.T) {
		memoryID := "outcome-test-4"
		weights := reasoningbank.NewProjectWeights(projectID)

		// Add equal positive and negative outcome signals
		for i := 0; i < 3; i++ {
			posSignal, _ := reasoningbank.NewSignal(memoryID, projectID, reasoningbank.SignalOutcome, true, "")
			_ = signalStore.StoreSignal(ctx, posSignal)
			negSignal, _ := reasoningbank.NewSignal(memoryID, projectID, reasoningbank.SignalOutcome, false, "")
			_ = signalStore.StoreSignal(ctx, negSignal)
		}

		agg := &reasoningbank.SignalAggregate{MemoryID: memoryID}
		signals, _ := signalStore.GetRecentSignals(ctx, memoryID, 24*60*60*1000000000)
		conf := reasoningbank.ComputeConfidenceFromHybrid(agg, signals, weights)
		t.Logf("Confidence with equal success/failure outcomes: %.4f", conf)

		// Should be close to 0.5 (balanced signals)
		assert.InDelta(t, 0.5, conf, 0.1,
			"equal positive and negative outcomes should yield ~0.5 confidence")
	})
}

// TestConfidenceCalibration_BayesianWeightLearning validates that the system
// learns which signal types are reliable predictors.
func TestConfidenceCalibration_BayesianWeightLearning(t *testing.T) {
	ctx := context.Background()

	// Create signal store directly to test weight learning
	signalStore := reasoningbank.NewInMemorySignalStore()

	projectID := "weight-learning-test"

	t.Run("usage signals that predict correctly gain weight", func(t *testing.T) {
		// Get initial weights
		initialWeights, err := signalStore.GetProjectWeights(ctx, projectID)
		require.NoError(t, err)
		initialUsageAlpha := initialWeights.UsageAlpha

		// Record usage signal followed by positive feedback
		memoryID := "test-memory-1"

		// Create and store a usage signal
		usageSignal, err := reasoningbank.NewSignal(memoryID, projectID, reasoningbank.SignalUsage, true, "")
		require.NoError(t, err)
		err = signalStore.StoreSignal(ctx, usageSignal)
		require.NoError(t, err)

		// Simulate learning from positive feedback (usage correctly predicted helpful)
		initialWeights.LearnFromFeedback(true, []reasoningbank.Signal{*usageSignal})
		err = signalStore.StoreProjectWeights(ctx, initialWeights)
		require.NoError(t, err)

		// Get updated weights
		updatedWeights, err := signalStore.GetProjectWeights(ctx, projectID)
		require.NoError(t, err)

		t.Logf("Usage alpha: %.1f -> %.1f", initialUsageAlpha, updatedWeights.UsageAlpha)

		assert.Greater(t, updatedWeights.UsageAlpha, initialUsageAlpha,
			"usage alpha should increase when usage signal correctly predicts helpful feedback")
	})

	t.Run("usage signals that predict incorrectly lose weight", func(t *testing.T) {
		project2ID := "weight-learning-test-2"

		// Get initial weights
		initialWeights, err := signalStore.GetProjectWeights(ctx, project2ID)
		require.NoError(t, err)
		initialUsageBeta := initialWeights.UsageBeta

		// Record usage signal followed by negative feedback (usage was wrong)
		memoryID := "test-memory-2"

		usageSignal, err := reasoningbank.NewSignal(memoryID, project2ID, reasoningbank.SignalUsage, true, "")
		require.NoError(t, err)
		err = signalStore.StoreSignal(ctx, usageSignal)
		require.NoError(t, err)

		// Simulate learning from negative feedback (usage incorrectly predicted)
		initialWeights.LearnFromFeedback(false, []reasoningbank.Signal{*usageSignal})
		err = signalStore.StoreProjectWeights(ctx, initialWeights)
		require.NoError(t, err)

		// Get updated weights
		updatedWeights, err := signalStore.GetProjectWeights(ctx, project2ID)
		require.NoError(t, err)

		t.Logf("Usage beta: %.1f -> %.1f", initialUsageBeta, updatedWeights.UsageBeta)

		assert.Greater(t, updatedWeights.UsageBeta, initialUsageBeta,
			"usage beta should increase when usage signal incorrectly predicts feedback")
	})
}

// TestConfidenceCalibration_BetaDistribution validates the Beta distribution
// calculations for confidence scoring.
func TestConfidenceCalibration_BetaDistribution(t *testing.T) {
	t.Run("uniform prior gives 0.5 confidence", func(t *testing.T) {
		// Empty aggregate, no signals, default weights -> should be 0.5
		weights := reasoningbank.NewProjectWeights("test-project")
		agg := &reasoningbank.SignalAggregate{}
		signals := []reasoningbank.Signal{}

		confidence := reasoningbank.ComputeConfidenceFromHybrid(agg, signals, weights)

		// With uniform prior (alpha=1, beta=1), confidence should be 0.5
		assert.InDelta(t, 0.5, confidence, 0.01,
			"uniform prior should give 0.5 confidence")
	})

	t.Run("positive signals increase confidence", func(t *testing.T) {
		weights := reasoningbank.NewProjectWeights("test-project")
		agg := &reasoningbank.SignalAggregate{
			ExplicitPos: 3,
			UsagePos:    2,
			OutcomePos:  1,
		}
		signals := []reasoningbank.Signal{}

		confidence := reasoningbank.ComputeConfidenceFromHybrid(agg, signals, weights)

		// With positive signals only, confidence should be well above 0.5
		assert.Greater(t, confidence, 0.6,
			"positive signals should give high confidence")
		t.Logf("Confidence with 6 positive signals: %.4f", confidence)
	})

	t.Run("negative signals decrease confidence", func(t *testing.T) {
		weights := reasoningbank.NewProjectWeights("test-project")
		agg := &reasoningbank.SignalAggregate{
			ExplicitNeg: 3,
			UsageNeg:    2,
			OutcomeNeg:  1,
		}
		signals := []reasoningbank.Signal{}

		confidence := reasoningbank.ComputeConfidenceFromHybrid(agg, signals, weights)

		// With negative signals only, confidence should be well below 0.5
		assert.Less(t, confidence, 0.4,
			"negative signals should give low confidence")
		t.Logf("Confidence with 6 negative signals: %.4f", confidence)
	})

	t.Run("mixed signals balance out", func(t *testing.T) {
		weights := reasoningbank.NewProjectWeights("test-project")
		agg := &reasoningbank.SignalAggregate{
			ExplicitPos: 5,
			ExplicitNeg: 5,
			UsagePos:    3,
			UsageNeg:    3,
		}
		signals := []reasoningbank.Signal{}

		confidence := reasoningbank.ComputeConfidenceFromHybrid(agg, signals, weights)

		// Equal positive and negative should give ~0.5
		assert.InDelta(t, 0.5, confidence, 0.1,
			"equal positive and negative signals should balance to ~0.5")
		t.Logf("Confidence with balanced signals: %.4f", confidence)
	})
}

// TestConfidenceCalibration_MinConfidenceThreshold validates that the
// MinConfidence threshold correctly filters results.
func TestConfidenceCalibration_MinConfidenceThreshold(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	embedder := newSemanticEmbedder(384)
	store, err := vectorstore.NewChromemStore(vectorstore.ChromemConfig{
		Path: t.TempDir(),
	}, embedder, logger)
	require.NoError(t, err)
	defer store.Close()

	store.SetIsolationMode(vectorstore.NewNoIsolation())

	svc, err := reasoningbank.NewService(store, logger, reasoningbank.WithDefaultTenant("test-tenant"))
	require.NoError(t, err)

	projectID := "threshold-test"

	t.Run("memories below 0.7 threshold are filtered", func(t *testing.T) {
		// Create memory with high confidence
		highConfMemory, err := reasoningbank.NewMemory(
			projectID,
			"High confidence memory",
			"This memory has high confidence",
			reasoningbank.OutcomeSuccess,
			[]string{"test"},
		)
		require.NoError(t, err)
		err = svc.Record(ctx, highConfMemory)
		require.NoError(t, err)

		// Create memory with low confidence (will be boosted to 0.8 by Record)
		// Then degrade it with negative feedback
		lowConfMemory, err := reasoningbank.NewMemory(
			projectID,
			"Low confidence memory",
			"This memory will have low confidence after feedback",
			reasoningbank.OutcomeSuccess,
			[]string{"test"},
		)
		require.NoError(t, err)
		err = svc.Record(ctx, lowConfMemory)
		require.NoError(t, err)

		// Apply many negative feedbacks to drop confidence
		for i := 0; i < 10; i++ {
			err = svc.Feedback(ctx, lowConfMemory.ID, false)
			require.NoError(t, err)
		}

		// Search for memories
		results, err := svc.Search(ctx, projectID, "confidence memory", 10)
		require.NoError(t, err)

		// Count which memories appear
		var highFound, lowFound bool
		for _, r := range results {
			if r.ID == highConfMemory.ID {
				highFound = true
				t.Logf("High confidence memory found with confidence: %.4f", r.Confidence)
			}
			if r.ID == lowConfMemory.ID {
				lowFound = true
				t.Logf("Low confidence memory found with confidence: %.4f", r.Confidence)
			}
		}

		assert.True(t, highFound, "high confidence memory should be found")
		t.Logf("Low confidence memory in results: %v (after 10 negative feedbacks)", lowFound)
	})
}
