package reasoningbank

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignalType_Constants(t *testing.T) {
	// Test that signal type constants are defined
	assert.Equal(t, SignalType("explicit"), SignalExplicit)
	assert.Equal(t, SignalType("usage"), SignalUsage)
	assert.Equal(t, SignalType("outcome"), SignalOutcome)
}

func TestNewSignal(t *testing.T) {
	tests := []struct {
		name       string
		memoryID   string
		projectID  string
		signalType SignalType
		positive   bool
		sessionID  string
		wantErr    bool
	}{
		{
			name:       "valid explicit positive signal",
			memoryID:   "mem_abc123",
			projectID:  "proj_123",
			signalType: SignalExplicit,
			positive:   true,
			sessionID:  "sess_xyz",
			wantErr:    false,
		},
		{
			name:       "valid usage negative signal",
			memoryID:   "mem_def456",
			projectID:  "proj_123",
			signalType: SignalUsage,
			positive:   false,
			sessionID:  "",
			wantErr:    false,
		},
		{
			name:       "valid outcome signal",
			memoryID:   "mem_ghi789",
			projectID:  "proj_456",
			signalType: SignalOutcome,
			positive:   true,
			sessionID:  "sess_abc",
			wantErr:    false,
		},
		{
			name:       "empty memory ID",
			memoryID:   "",
			projectID:  "proj_123",
			signalType: SignalExplicit,
			positive:   true,
			wantErr:    true,
		},
		{
			name:       "empty project ID",
			memoryID:   "mem_abc123",
			projectID:  "",
			signalType: SignalExplicit,
			positive:   true,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signal, err := NewSignal(tt.memoryID, tt.projectID, tt.signalType, tt.positive, tt.sessionID)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, signal.ID)
			assert.Equal(t, tt.memoryID, signal.MemoryID)
			assert.Equal(t, tt.projectID, signal.ProjectID)
			assert.Equal(t, tt.signalType, signal.Type)
			assert.Equal(t, tt.positive, signal.Positive)
			assert.Equal(t, tt.sessionID, signal.SessionID)
			assert.WithinDuration(t, time.Now(), signal.Timestamp, time.Second)
		})
	}
}

func TestSignalAggregate_InitialState(t *testing.T) {
	agg := NewSignalAggregate("mem_abc123", "proj_123")

	assert.Equal(t, "mem_abc123", agg.MemoryID)
	assert.Equal(t, "proj_123", agg.ProjectID)
	assert.Equal(t, 0, agg.ExplicitPos)
	assert.Equal(t, 0, agg.ExplicitNeg)
	assert.Equal(t, 0, agg.UsagePos)
	assert.Equal(t, 0, agg.UsageNeg)
	assert.Equal(t, 0, agg.OutcomePos)
	assert.Equal(t, 0, agg.OutcomeNeg)
}

func TestSignalAggregate_AddSignal(t *testing.T) {
	agg := NewSignalAggregate("mem_abc123", "proj_123")

	// Add explicit positive
	agg.AddSignal(SignalExplicit, true)
	assert.Equal(t, 1, agg.ExplicitPos)
	assert.Equal(t, 0, agg.ExplicitNeg)

	// Add explicit negative
	agg.AddSignal(SignalExplicit, false)
	assert.Equal(t, 1, agg.ExplicitPos)
	assert.Equal(t, 1, agg.ExplicitNeg)

	// Add usage signals
	agg.AddSignal(SignalUsage, true)
	agg.AddSignal(SignalUsage, true)
	assert.Equal(t, 2, agg.UsagePos)
	assert.Equal(t, 0, agg.UsageNeg)

	// Add outcome signals
	agg.AddSignal(SignalOutcome, false)
	assert.Equal(t, 0, agg.OutcomePos)
	assert.Equal(t, 1, agg.OutcomeNeg)
}

func TestProjectWeights_InitialPriors(t *testing.T) {
	weights := NewProjectWeights("proj_123")

	assert.Equal(t, "proj_123", weights.ProjectID)

	// Initial priors as specified in DESIGN.md:
	// Explicit 7:3 (70%), Usage/Outcome 5:5 (50%)
	assert.Equal(t, 7.0, weights.ExplicitAlpha)
	assert.Equal(t, 3.0, weights.ExplicitBeta)
	assert.Equal(t, 5.0, weights.UsageAlpha)
	assert.Equal(t, 5.0, weights.UsageBeta)
	assert.Equal(t, 5.0, weights.OutcomeAlpha)
	assert.Equal(t, 5.0, weights.OutcomeBeta)
}

func TestProjectWeights_ComputeWeights(t *testing.T) {
	weights := NewProjectWeights("proj_123")

	// Initial state: explicit=70%, usage=50%, outcome=50%
	// Raw weights: 0.7, 0.5, 0.5
	// Total: 1.7
	// Normalized: 0.7/1.7 ≈ 0.41, 0.5/1.7 ≈ 0.29, 0.5/1.7 ≈ 0.29

	explicit, usage, outcome := weights.ComputeWeights()

	// Sum should be 1.0
	assert.InDelta(t, 1.0, explicit+usage+outcome, 0.001)

	// Explicit should have highest initial weight
	assert.Greater(t, explicit, usage)
	assert.Greater(t, explicit, outcome)

	// Usage and outcome should be equal initially
	assert.InDelta(t, usage, outcome, 0.001)
}

func TestProjectWeights_WeightFor(t *testing.T) {
	weights := NewProjectWeights("proj_123")

	explicit, usage, outcome := weights.ComputeWeights()

	assert.Equal(t, explicit, weights.WeightFor(SignalExplicit))
	assert.Equal(t, usage, weights.WeightFor(SignalUsage))
	assert.Equal(t, outcome, weights.WeightFor(SignalOutcome))
}

func TestProjectWeights_LearnFromFeedback(t *testing.T) {
	weights := NewProjectWeights("proj_123")

	initialUsageAlpha := weights.UsageAlpha
	initialUsageBeta := weights.UsageBeta

	// Simulate: usage signal predicted positive, explicit feedback was helpful
	// This means usage correctly predicted - should increase UsageAlpha
	recentSignals := []Signal{
		{Type: SignalUsage, Positive: true},
	}

	weights.LearnFromFeedback(true, recentSignals)

	// UsageAlpha should have increased (usage correctly predicted helpful)
	assert.Equal(t, initialUsageAlpha+1, weights.UsageAlpha)
	assert.Equal(t, initialUsageBeta, weights.UsageBeta)
}

func TestProjectWeights_LearnFromFeedback_IncorrectPrediction(t *testing.T) {
	weights := NewProjectWeights("proj_123")

	initialUsageAlpha := weights.UsageAlpha
	initialUsageBeta := weights.UsageBeta

	// Simulate: usage signal predicted positive, but feedback was unhelpful
	// This means usage incorrectly predicted - should increase UsageBeta
	recentSignals := []Signal{
		{Type: SignalUsage, Positive: true},
	}

	weights.LearnFromFeedback(false, recentSignals) // not helpful

	// UsageBeta should have increased (usage incorrectly predicted)
	assert.Equal(t, initialUsageAlpha, weights.UsageAlpha)
	assert.Equal(t, initialUsageBeta+1, weights.UsageBeta)
}

func TestMemoryConfidence_InitialState(t *testing.T) {
	mc := NewMemoryConfidence("mem_abc123")

	assert.Equal(t, "mem_abc123", mc.MemoryID)
	assert.Equal(t, 1.0, mc.Alpha) // Starts at 1
	assert.Equal(t, 1.0, mc.Beta)  // Starts at 1
	assert.Equal(t, 0.5, mc.Score()) // 1/(1+1) = 0.5
}

func TestMemoryConfidence_Update(t *testing.T) {
	mc := NewMemoryConfidence("mem_abc123")
	weights := NewProjectWeights("proj_123")

	// Get explicit weight
	explicitWeight := weights.WeightFor(SignalExplicit)

	// Add positive explicit signal
	signal := Signal{Type: SignalExplicit, Positive: true}
	mc.Update(signal, weights)

	// Alpha should have increased by explicit weight
	assert.InDelta(t, 1.0+explicitWeight, mc.Alpha, 0.001)
	assert.Equal(t, 1.0, mc.Beta)

	// Confidence should be higher than 0.5 now
	assert.Greater(t, mc.Score(), 0.5)
}

func TestMemoryConfidence_Update_Negative(t *testing.T) {
	mc := NewMemoryConfidence("mem_abc123")
	weights := NewProjectWeights("proj_123")

	// Get explicit weight
	explicitWeight := weights.WeightFor(SignalExplicit)

	// Add negative explicit signal
	signal := Signal{Type: SignalExplicit, Positive: false}
	mc.Update(signal, weights)

	// Beta should have increased by explicit weight
	assert.Equal(t, 1.0, mc.Alpha)
	assert.InDelta(t, 1.0+explicitWeight, mc.Beta, 0.001)

	// Confidence should be lower than 0.5 now
	assert.Less(t, mc.Score(), 0.5)
}
