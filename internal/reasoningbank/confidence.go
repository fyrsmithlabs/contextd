package reasoningbank

import (
	"context"
	"sync"
	"time"
)

// ComputeConfidenceFromHybrid calculates confidence using both historical
// aggregates and recent signals.
//
// This is the core Bayesian confidence calculation that combines:
// - Historical data from aggregates (signals older than 30 days, rolled up)
// - Recent signals (last 30 days, stored individually)
// - Learned project weights for each signal type
//
// The formula uses Beta distribution: confidence = alpha / (alpha + beta)
func ComputeConfidenceFromHybrid(agg *SignalAggregate, recentSignals []Signal, weights *ProjectWeights) float64 {
	// Get normalized weights
	explicitW, usageW, outcomeW := weights.ComputeWeights()

	// Start from uniform prior
	alpha := 1.0
	beta := 1.0

	// Add aggregate contributions (historical data)
	if agg != nil {
		alpha += float64(agg.ExplicitPos) * explicitW
		alpha += float64(agg.UsagePos) * usageW
		alpha += float64(agg.OutcomePos) * outcomeW

		beta += float64(agg.ExplicitNeg) * explicitW
		beta += float64(agg.UsageNeg) * usageW
		beta += float64(agg.OutcomeNeg) * outcomeW
	}

	// Add recent signal contributions
	for _, sig := range recentSignals {
		w := weights.WeightFor(sig.Type)
		if sig.Positive {
			alpha += w
		} else {
			beta += w
		}
	}

	// Beta distribution mean
	if alpha+beta == 0 {
		return 0.5
	}
	return alpha / (alpha + beta)
}

// SignalStore defines the interface for signal persistence.
//
// Implementations can use vectorstore, SQL database, or in-memory storage.
// The interface supports the hybrid storage model:
// - Individual signals (last 30 days)
// - Rolled-up aggregates (older than 30 days)
// - Per-project weight learning
type SignalStore interface {
	// StoreSignal persists a new signal event.
	StoreSignal(ctx context.Context, signal *Signal) error

	// GetRecentSignals retrieves signals within the given duration.
	GetRecentSignals(ctx context.Context, memoryID string, duration time.Duration) ([]Signal, error)

	// StoreAggregate persists an aggregate for a memory.
	StoreAggregate(ctx context.Context, agg *SignalAggregate) error

	// GetAggregate retrieves the aggregate for a memory.
	GetAggregate(ctx context.Context, memoryID string) (*SignalAggregate, error)

	// StoreProjectWeights persists project weights.
	StoreProjectWeights(ctx context.Context, weights *ProjectWeights) error

	// GetProjectWeights retrieves weights for a project.
	// Returns default weights if none exist.
	GetProjectWeights(ctx context.Context, projectID string) (*ProjectWeights, error)

	// RollupOldSignals moves signals older than the cutoff into aggregates.
	// This is called by a background worker daily.
	RollupOldSignals(ctx context.Context, memoryID string, cutoff time.Duration) error
}

// InMemorySignalStore is an in-memory implementation of SignalStore for testing.
type InMemorySignalStore struct {
	mu         sync.RWMutex
	signals    map[string][]Signal      // memoryID -> signals
	aggregates map[string]*SignalAggregate
	weights    map[string]*ProjectWeights
}

// NewInMemorySignalStore creates a new in-memory signal store.
func NewInMemorySignalStore() *InMemorySignalStore {
	return &InMemorySignalStore{
		signals:    make(map[string][]Signal),
		aggregates: make(map[string]*SignalAggregate),
		weights:    make(map[string]*ProjectWeights),
	}
}

// StoreSignal adds a signal to the store.
func (s *InMemorySignalStore) StoreSignal(ctx context.Context, signal *Signal) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.signals[signal.MemoryID] == nil {
		s.signals[signal.MemoryID] = []Signal{}
	}
	s.signals[signal.MemoryID] = append(s.signals[signal.MemoryID], *signal)
	return nil
}

// GetRecentSignals retrieves signals newer than the cutoff.
func (s *InMemorySignalStore) GetRecentSignals(ctx context.Context, memoryID string, duration time.Duration) ([]Signal, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cutoff := time.Now().Add(-duration)
	result := []Signal{}

	for _, sig := range s.signals[memoryID] {
		if sig.Timestamp.After(cutoff) {
			result = append(result, sig)
		}
	}

	return result, nil
}

// StoreAggregate saves an aggregate.
func (s *InMemorySignalStore) StoreAggregate(ctx context.Context, agg *SignalAggregate) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.aggregates[agg.MemoryID] = agg
	return nil
}

// GetAggregate retrieves an aggregate, returning empty aggregate if not found.
func (s *InMemorySignalStore) GetAggregate(ctx context.Context, memoryID string) (*SignalAggregate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if agg, ok := s.aggregates[memoryID]; ok {
		return agg, nil
	}
	// Return empty aggregate
	return &SignalAggregate{MemoryID: memoryID}, nil
}

// StoreProjectWeights saves project weights.
func (s *InMemorySignalStore) StoreProjectWeights(ctx context.Context, weights *ProjectWeights) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.weights[weights.ProjectID] = weights
	return nil
}

// GetProjectWeights retrieves project weights, returning defaults if not found.
func (s *InMemorySignalStore) GetProjectWeights(ctx context.Context, projectID string) (*ProjectWeights, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if w, ok := s.weights[projectID]; ok {
		return w, nil
	}
	// Return default weights
	return NewProjectWeights(projectID), nil
}

// RollupOldSignals moves old signals into aggregates.
func (s *InMemorySignalStore) RollupOldSignals(ctx context.Context, memoryID string, cutoff time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoffTime := time.Now().Add(-cutoff)

	// Get or create aggregate
	agg := s.aggregates[memoryID]
	if agg == nil {
		agg = &SignalAggregate{MemoryID: memoryID}
	}

	// Separate old and new signals
	oldSignals := []Signal{}
	newSignals := []Signal{}

	for _, sig := range s.signals[memoryID] {
		if sig.Timestamp.Before(cutoffTime) {
			oldSignals = append(oldSignals, sig)
		} else {
			newSignals = append(newSignals, sig)
		}
	}

	// Aggregate old signals
	for _, sig := range oldSignals {
		agg.AddSignal(sig.Type, sig.Positive)
	}
	agg.LastRollup = time.Now()

	// Update storage
	s.aggregates[memoryID] = agg
	s.signals[memoryID] = newSignals

	return nil
}

// ConfidenceCalculator provides methods for computing and updating memory confidence.
type ConfidenceCalculator struct {
	store SignalStore
}

// NewConfidenceCalculator creates a new confidence calculator.
func NewConfidenceCalculator(store SignalStore) *ConfidenceCalculator {
	return &ConfidenceCalculator{store: store}
}

// ComputeConfidence calculates the current confidence for a memory.
func (c *ConfidenceCalculator) ComputeConfidence(ctx context.Context, memoryID, projectID string) (float64, error) {
	// Get project weights
	weights, err := c.store.GetProjectWeights(ctx, projectID)
	if err != nil {
		return 0, err
	}

	// Get aggregate
	agg, err := c.store.GetAggregate(ctx, memoryID)
	if err != nil {
		return 0, err
	}

	// Get recent signals (30 days)
	recentSignals, err := c.store.GetRecentSignals(ctx, memoryID, 30*24*time.Hour)
	if err != nil {
		return 0, err
	}

	return ComputeConfidenceFromHybrid(agg, recentSignals, weights), nil
}

// RecordSignal stores a new signal and updates confidence.
func (c *ConfidenceCalculator) RecordSignal(ctx context.Context, signal *Signal) error {
	return c.store.StoreSignal(ctx, signal)
}

// LearnFromFeedback updates project weights based on feedback accuracy.
func (c *ConfidenceCalculator) LearnFromFeedback(ctx context.Context, projectID, memoryID string, helpful bool) error {
	// Get recent signals for this memory
	recentSignals, err := c.store.GetRecentSignals(ctx, memoryID, 24*time.Hour)
	if err != nil {
		return err
	}

	// Get project weights
	weights, err := c.store.GetProjectWeights(ctx, projectID)
	if err != nil {
		return err
	}

	// Learn from feedback
	weights.LearnFromFeedback(helpful, recentSignals)

	// Save updated weights
	return c.store.StoreProjectWeights(ctx, weights)
}
