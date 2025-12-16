package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

// MockContextdClient is a mock implementation of ContextdClient for testing.
// It simulates the Bayesian confidence system behavior.
type MockContextdClient struct {
	mu sync.RWMutex

	// Storage
	memories map[string]*mockMemory

	// Signal tracking for Bayesian simulation
	signals map[string][]mockSignal

	// Configuration
	initialConfidence float64
	confidenceStep    float64
}

type mockMemory struct {
	ID         string
	ProjectID  string
	Title      string
	Content    string
	Outcome    string
	Confidence float64
	Tags       []string
}

type mockSignal struct {
	Type     string // "explicit", "usage", "outcome"
	Positive bool
}

// NewMockContextdClient creates a new mock client.
func NewMockContextdClient() *MockContextdClient {
	return &MockContextdClient{
		memories:          make(map[string]*mockMemory),
		signals:           make(map[string][]mockSignal),
		initialConfidence: 0.5,  // Bayesian prior
		confidenceStep:    0.05, // Per-signal adjustment
	}
}

// MemoryRecord creates a new memory.
func (m *MockContextdClient) MemoryRecord(ctx context.Context, projectID, title, content, outcome string, tags []string) (string, float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := uuid.New().String()
	confidence := 0.8 // Explicit record confidence (like the real implementation)

	mem := &mockMemory{
		ID:         id,
		ProjectID:  projectID,
		Title:      title,
		Content:    content,
		Outcome:    outcome,
		Confidence: confidence,
		Tags:       tags,
	}

	m.memories[id] = mem
	m.signals[id] = []mockSignal{}

	return id, confidence, nil
}

// MemorySearch returns memories matching the query.
func (m *MockContextdClient) MemorySearch(ctx context.Context, projectID, query string, limit int) ([]MemoryResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make([]MemoryResult, 0)

	for _, mem := range m.memories {
		if mem.ProjectID == projectID && mem.Confidence >= 0.7 {
			results = append(results, MemoryResult{
				ID:         mem.ID,
				Title:      mem.Title,
				Content:    mem.Content,
				Outcome:    mem.Outcome,
				Confidence: mem.Confidence,
				Tags:       mem.Tags,
			})

			// Record usage signal
			m.signals[mem.ID] = append(m.signals[mem.ID], mockSignal{
				Type:     "usage",
				Positive: true,
			})

			if len(results) >= limit {
				break
			}
		}
	}

	return results, nil
}

// MemoryFeedback provides feedback on a memory.
func (m *MockContextdClient) MemoryFeedback(ctx context.Context, memoryID string, helpful bool) (float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	mem, ok := m.memories[memoryID]
	if !ok {
		return 0, fmt.Errorf("memory not found: %s", memoryID)
	}

	// Record explicit signal
	m.signals[memoryID] = append(m.signals[memoryID], mockSignal{
		Type:     "explicit",
		Positive: helpful,
	})

	// Simulate Bayesian confidence update
	newConfidence := m.computeConfidence(memoryID)
	mem.Confidence = newConfidence

	return newConfidence, nil
}

// MemoryOutcome reports a task outcome.
func (m *MockContextdClient) MemoryOutcome(ctx context.Context, memoryID string, succeeded bool, sessionID string) (float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	mem, ok := m.memories[memoryID]
	if !ok {
		return 0, fmt.Errorf("memory not found: %s", memoryID)
	}

	// Record outcome signal
	m.signals[memoryID] = append(m.signals[memoryID], mockSignal{
		Type:     "outcome",
		Positive: succeeded,
	})

	// Simulate Bayesian confidence update
	newConfidence := m.computeConfidence(memoryID)
	mem.Confidence = newConfidence

	return newConfidence, nil
}

// GetMemory retrieves a memory by ID.
func (m *MockContextdClient) GetMemory(ctx context.Context, memoryID string) (*MemoryResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mem, ok := m.memories[memoryID]
	if !ok {
		return nil, fmt.Errorf("memory not found: %s", memoryID)
	}

	return &MemoryResult{
		ID:         mem.ID,
		Title:      mem.Title,
		Content:    mem.Content,
		Outcome:    mem.Outcome,
		Confidence: mem.Confidence,
		Tags:       mem.Tags,
	}, nil
}

// computeConfidence simulates the Bayesian confidence calculation.
// This is a simplified version that mimics the real behavior.
func (m *MockContextdClient) computeConfidence(memoryID string) float64 {
	signals := m.signals[memoryID]
	if len(signals) == 0 {
		return m.initialConfidence
	}

	// Count positive and negative signals with type weighting
	// Weights: explicit (0.5), usage (0.2), outcome (0.3)
	weights := map[string]float64{
		"explicit": 0.5,
		"usage":    0.2,
		"outcome":  0.3,
	}

	var alpha, beta float64 = 1.0, 1.0 // Uniform prior

	for _, sig := range signals {
		weight := weights[sig.Type]
		if sig.Positive {
			alpha += weight
		} else {
			beta += weight
		}
	}

	// Beta distribution mean: alpha / (alpha + beta)
	confidence := alpha / (alpha + beta)

	// Clamp to valid range
	if confidence < 0.0 {
		confidence = 0.0
	}
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// Reset clears all state (useful between tests).
func (m *MockContextdClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.memories = make(map[string]*mockMemory)
	m.signals = make(map[string][]mockSignal)
}

// GetSignalCount returns the number of signals for a memory.
func (m *MockContextdClient) GetSignalCount(memoryID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.signals[memoryID])
}
