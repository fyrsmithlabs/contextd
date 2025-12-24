package reasoningbank

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Signal-related errors.
var (
	ErrEmptyMemoryID = errors.New("memory ID cannot be empty")
)

// SignalType identifies the source of a confidence signal.
type SignalType string

const (
	// SignalExplicit is from memory_feedback tool - user rates helpful/unhelpful.
	SignalExplicit SignalType = "explicit"

	// SignalUsage is from memory_search tool - memory retrieved in search results.
	SignalUsage SignalType = "usage"

	// SignalOutcome is from memory_outcome tool - agent reports task success/failure.
	SignalOutcome SignalType = "outcome"
)

// Signal represents a single confidence event.
//
// Signals are recorded when:
// - User provides explicit feedback (memory_feedback) → SignalExplicit
// - Memory is retrieved in search results (memory_search) → SignalUsage
// - Agent reports task outcome (memory_outcome) → SignalOutcome
type Signal struct {
	// ID is the unique signal identifier.
	ID string `json:"id"`

	// MemoryID is the memory this signal relates to.
	MemoryID string `json:"memory_id"`

	// ProjectID is the project context for this signal.
	ProjectID string `json:"project_id"`

	// Type identifies the signal source.
	Type SignalType `json:"type"`

	// Positive indicates if this was a positive signal (helpful, success).
	Positive bool `json:"positive"`

	// SessionID is optional session context for correlation.
	SessionID string `json:"session_id,omitempty"`

	// Timestamp is when this signal was recorded.
	Timestamp time.Time `json:"timestamp"`
}

// NewSignal creates a new Signal with generated ID and current timestamp.
func NewSignal(memoryID, projectID string, signalType SignalType, positive bool, sessionID string) (*Signal, error) {
	if memoryID == "" {
		return nil, ErrEmptyMemoryID
	}
	if projectID == "" {
		return nil, ErrEmptyProjectID
	}

	return &Signal{
		ID:        uuid.New().String(),
		MemoryID:  memoryID,
		ProjectID: projectID,
		Type:      signalType,
		Positive:  positive,
		SessionID: sessionID,
		Timestamp: time.Now(),
	}, nil
}

// SignalAggregate stores rolled-up signal counts for data older than 30 days.
//
// Instead of storing individual events forever, old signals are aggregated
// into counts per signal type per memory. This provides storage efficiency
// while preserving the statistical information needed for confidence calculation.
type SignalAggregate struct {
	// MemoryID is the memory this aggregate belongs to.
	MemoryID string `json:"memory_id"`

	// ProjectID is the project context.
	ProjectID string `json:"project_id"`

	// ExplicitPos is the count of positive explicit signals.
	ExplicitPos int `json:"explicit_pos"`

	// ExplicitNeg is the count of negative explicit signals.
	ExplicitNeg int `json:"explicit_neg"`

	// UsagePos is the count of positive usage signals.
	UsagePos int `json:"usage_pos"`

	// UsageNeg is the count of negative usage signals.
	UsageNeg int `json:"usage_neg"`

	// OutcomePos is the count of positive outcome signals.
	OutcomePos int `json:"outcome_pos"`

	// OutcomeNeg is the count of negative outcome signals.
	OutcomeNeg int `json:"outcome_neg"`

	// LastRollup is when signals were last rolled up into this aggregate.
	LastRollup time.Time `json:"last_rollup"`
}

// NewSignalAggregate creates a new SignalAggregate with zero counts.
func NewSignalAggregate(memoryID, projectID string) *SignalAggregate {
	return &SignalAggregate{
		MemoryID:  memoryID,
		ProjectID: projectID,
	}
}

// AddSignal increments the appropriate counter based on signal type and polarity.
func (agg *SignalAggregate) AddSignal(signalType SignalType, positive bool) {
	switch signalType {
	case SignalExplicit:
		if positive {
			agg.ExplicitPos++
		} else {
			agg.ExplicitNeg++
		}
	case SignalUsage:
		if positive {
			agg.UsagePos++
		} else {
			agg.UsageNeg++
		}
	case SignalOutcome:
		if positive {
			agg.OutcomePos++
		} else {
			agg.OutcomeNeg++
		}
	}
}

// ProjectWeights tracks learned signal weights per project using Beta distributions.
//
// Each signal type has alpha/beta parameters that form a Beta distribution.
// The mean of the distribution (alpha / (alpha + beta)) represents how well
// that signal type predicts memory usefulness.
//
// The system learns by observing which signals correctly predict explicit feedback:
// - If usage signals predict helpful feedback, UsageAlpha increases
// - If usage signals incorrectly predict, UsageBeta increases
//
// Initial priors (from DESIGN.md):
// - Explicit: 7:3 (70% weight) - trust user feedback highly
// - Usage: 5:5 (50% weight) - uncertain initially
// - Outcome: 5:5 (50% weight) - uncertain initially
type ProjectWeights struct {
	// ProjectID identifies which project these weights belong to.
	ProjectID string `json:"project_id"`

	// ExplicitAlpha is the success count for explicit signal predictions.
	ExplicitAlpha float64 `json:"explicit_alpha"`

	// ExplicitBeta is the failure count for explicit signal predictions.
	ExplicitBeta float64 `json:"explicit_beta"`

	// UsageAlpha is the success count for usage signal predictions.
	UsageAlpha float64 `json:"usage_alpha"`

	// UsageBeta is the failure count for usage signal predictions.
	UsageBeta float64 `json:"usage_beta"`

	// OutcomeAlpha is the success count for outcome signal predictions.
	OutcomeAlpha float64 `json:"outcome_alpha"`

	// OutcomeBeta is the failure count for outcome signal predictions.
	OutcomeBeta float64 `json:"outcome_beta"`
}

// NewProjectWeights creates a new ProjectWeights with initial priors.
//
// Initial priors from DESIGN.md:
// - Explicit 7:3 (70%) - trust user feedback
// - Usage/Outcome 5:5 (50%) - uncertain initially
func NewProjectWeights(projectID string) *ProjectWeights {
	return &ProjectWeights{
		ProjectID:     projectID,
		ExplicitAlpha: 7.0,
		ExplicitBeta:  3.0,
		UsageAlpha:    5.0,
		UsageBeta:     5.0,
		OutcomeAlpha:  5.0,
		OutcomeBeta:   5.0,
	}
}

// ComputeWeights returns normalized weights for each signal type.
//
// Uses Beta distribution mean: alpha / (alpha + beta)
// Then normalizes so all weights sum to 1.0.
func (pw *ProjectWeights) ComputeWeights() (explicit, usage, outcome float64) {
	// Beta distribution mean = alpha / (alpha + beta)
	rawExplicit := pw.ExplicitAlpha / (pw.ExplicitAlpha + pw.ExplicitBeta)
	rawUsage := pw.UsageAlpha / (pw.UsageAlpha + pw.UsageBeta)
	rawOutcome := pw.OutcomeAlpha / (pw.OutcomeAlpha + pw.OutcomeBeta)

	// Normalize to sum to 1.0
	total := rawExplicit + rawUsage + rawOutcome
	if total == 0 {
		// Avoid division by zero - return equal weights
		return 1.0 / 3.0, 1.0 / 3.0, 1.0 / 3.0
	}

	return rawExplicit / total, rawUsage / total, rawOutcome / total
}

// WeightFor returns the normalized weight for a specific signal type.
func (pw *ProjectWeights) WeightFor(signalType SignalType) float64 {
	explicit, usage, outcome := pw.ComputeWeights()

	switch signalType {
	case SignalExplicit:
		return explicit
	case SignalUsage:
		return usage
	case SignalOutcome:
		return outcome
	default:
		return 0
	}
}

// LearnFromFeedback updates weights based on whether signals correctly predicted feedback.
//
// When explicit feedback arrives (helpful or unhelpful), we check if other signals
// (usage, outcome) correctly predicted this feedback. If they did, their alpha
// increases. If they didn't, their beta increases.
//
// This allows the system to learn which signal types are reliable predictors
// of memory usefulness for this specific project.
func (pw *ProjectWeights) LearnFromFeedback(helpful bool, recentSignals []Signal) {
	// Check if usage signals predicted this feedback
	usagePredictedPositive := hasPositiveSignal(recentSignals, SignalUsage)
	if usagePredictedPositive {
		if usagePredictedPositive == helpful {
			pw.UsageAlpha++ // Usage correctly predicted
		} else {
			pw.UsageBeta++ // Usage incorrectly predicted
		}
	}

	// Check if outcome signals predicted this feedback
	outcomePredictedPositive := hasPositiveSignal(recentSignals, SignalOutcome)
	if outcomePredictedPositive {
		if outcomePredictedPositive == helpful {
			pw.OutcomeAlpha++ // Outcome correctly predicted
		} else {
			pw.OutcomeBeta++ // Outcome incorrectly predicted
		}
	}
}

// hasPositiveSignal checks if there's a positive signal of the given type.
func hasPositiveSignal(signals []Signal, signalType SignalType) bool {
	for _, s := range signals {
		if s.Type == signalType && s.Positive {
			return true
		}
	}
	return false
}

// MemoryConfidence tracks confidence for a single memory using Beta distribution.
//
// Each memory maintains its own alpha/beta counts which are updated by weighted signals.
// The confidence score is the Beta distribution mean: alpha / (alpha + beta).
type MemoryConfidence struct {
	// MemoryID identifies the memory.
	MemoryID string `json:"memory_id"`

	// Alpha represents positive evidence (starts at 1 for uniform prior).
	Alpha float64 `json:"alpha"`

	// Beta represents negative evidence (starts at 1 for uniform prior).
	Beta float64 `json:"beta"`
}

// NewMemoryConfidence creates a new MemoryConfidence with uniform prior (1:1 = 50%).
func NewMemoryConfidence(memoryID string) *MemoryConfidence {
	return &MemoryConfidence{
		MemoryID: memoryID,
		Alpha:    1.0,
		Beta:     1.0,
	}
}

// Score returns the confidence score: alpha / (alpha + beta).
//
// This is the mean of the Beta distribution, representing our best estimate
// of the memory's usefulness based on accumulated evidence.
func (mc *MemoryConfidence) Score() float64 {
	if mc.Alpha+mc.Beta == 0 {
		return 0.5 // Uniform prior
	}
	return mc.Alpha / (mc.Alpha + mc.Beta)
}

// Update adjusts confidence based on a new signal.
//
// The signal's contribution is weighted by the project's learned weights.
// Positive signals increase alpha, negative signals increase beta.
func (mc *MemoryConfidence) Update(signal Signal, weights *ProjectWeights) {
	w := weights.WeightFor(signal.Type)

	if signal.Positive {
		mc.Alpha += w
	} else {
		mc.Beta += w
	}
}
