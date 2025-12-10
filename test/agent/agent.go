package agent

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ContextdClient defines the interface for interacting with contextd.
// This allows mocking for unit tests.
type ContextdClient interface {
	// Memory operations
	MemoryRecord(ctx context.Context, projectID, title, content, outcome string, tags []string) (string, float64, error)
	MemorySearch(ctx context.Context, projectID, query string, limit int) ([]MemoryResult, error)
	MemoryFeedback(ctx context.Context, memoryID string, helpful bool) (float64, error)
	MemoryOutcome(ctx context.Context, memoryID string, succeeded bool, sessionID string) (float64, error)

	// For observing state
	GetMemory(ctx context.Context, memoryID string) (*MemoryResult, error)
}

// MemoryResult represents a memory returned from contextd.
type MemoryResult struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Content    string   `json:"content"`
	Outcome    string   `json:"outcome"`
	Confidence float64  `json:"confidence"`
	Tags       []string `json:"tags"`
}

// LLMClient defines the interface for LLM interactions.
// Allows swapping Claude for other models or mocks.
type LLMClient interface {
	Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error)
	GenerateStructured(ctx context.Context, systemPrompt, userPrompt string, schema interface{}) (interface{}, error)
}

// Agent is a synthetic user agent for testing contextd.
type Agent struct {
	client    ContextdClient
	llm       LLMClient
	persona   Persona
	projectID string
	sessionID string
	logger    *zap.Logger

	// State
	history  []Turn
	feedback []FeedbackEvent
	outcomes []OutcomeEvent

	// Tracking
	memoriesRecorded  []string
	memoriesRetrieved []MemoryResult
	confidenceHistory map[string][]float64
}

// Config configures an Agent.
type Config struct {
	Client    ContextdClient
	LLM       LLMClient
	Persona   Persona
	ProjectID string
	Logger    *zap.Logger
}

// New creates a new test agent.
func New(cfg Config) (*Agent, error) {
	if cfg.Client == nil {
		return nil, fmt.Errorf("contextd client is required")
	}
	if cfg.Persona.Name == "" {
		return nil, fmt.Errorf("persona name is required")
	}
	if cfg.ProjectID == "" {
		return nil, fmt.Errorf("project ID is required")
	}

	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Agent{
		client:            cfg.Client,
		llm:               cfg.LLM,
		persona:           cfg.Persona,
		projectID:         cfg.ProjectID,
		sessionID:         uuid.New().String(),
		logger:            logger,
		history:           make([]Turn, 0),
		feedback:          make([]FeedbackEvent, 0),
		outcomes:          make([]OutcomeEvent, 0),
		memoriesRecorded:  make([]string, 0),
		memoriesRetrieved: make([]MemoryResult, 0),
		confidenceHistory: make(map[string][]float64),
	}, nil
}

// RecordMemory records a new memory and tracks confidence.
func (a *Agent) RecordMemory(ctx context.Context, title, content, outcome string, tags []string) (string, error) {
	memoryID, confidence, err := a.client.MemoryRecord(ctx, a.projectID, title, content, outcome, tags)
	if err != nil {
		return "", fmt.Errorf("recording memory: %w", err)
	}

	a.memoriesRecorded = append(a.memoriesRecorded, memoryID)
	a.confidenceHistory[memoryID] = []float64{confidence}

	a.logger.Info("recorded memory",
		zap.String("memory_id", memoryID),
		zap.String("title", title),
		zap.Float64("confidence", confidence))

	return memoryID, nil
}

// SearchMemories searches for relevant memories.
func (a *Agent) SearchMemories(ctx context.Context, query string, limit int) ([]MemoryResult, error) {
	results, err := a.client.MemorySearch(ctx, a.projectID, query, limit)
	if err != nil {
		return nil, fmt.Errorf("searching memories: %w", err)
	}

	a.memoriesRetrieved = append(a.memoriesRetrieved, results...)

	a.logger.Info("searched memories",
		zap.String("query", query),
		zap.Int("results", len(results)))

	return results, nil
}

// GiveFeedback provides feedback on a memory.
func (a *Agent) GiveFeedback(ctx context.Context, memoryID string, helpful bool, reasoning string) (float64, error) {
	newConfidence, err := a.client.MemoryFeedback(ctx, memoryID, helpful)
	if err != nil {
		return 0, fmt.Errorf("giving feedback: %w", err)
	}

	event := FeedbackEvent{
		Timestamp: time.Now(),
		MemoryID:  memoryID,
		Helpful:   helpful,
		Reasoning: reasoning,
	}
	a.feedback = append(a.feedback, event)

	// Track confidence history
	if _, ok := a.confidenceHistory[memoryID]; !ok {
		a.confidenceHistory[memoryID] = []float64{}
	}
	a.confidenceHistory[memoryID] = append(a.confidenceHistory[memoryID], newConfidence)

	a.logger.Info("gave feedback",
		zap.String("memory_id", memoryID),
		zap.Bool("helpful", helpful),
		zap.Float64("new_confidence", newConfidence))

	return newConfidence, nil
}

// ReportOutcome reports a task outcome for a memory.
func (a *Agent) ReportOutcome(ctx context.Context, memoryID string, succeeded bool, taskDesc string) (float64, error) {
	newConfidence, err := a.client.MemoryOutcome(ctx, memoryID, succeeded, a.sessionID)
	if err != nil {
		return 0, fmt.Errorf("reporting outcome: %w", err)
	}

	event := OutcomeEvent{
		Timestamp: time.Now(),
		MemoryID:  memoryID,
		Succeeded: succeeded,
		SessionID: a.sessionID,
		TaskDesc:  taskDesc,
	}
	a.outcomes = append(a.outcomes, event)

	// Track confidence history
	if _, ok := a.confidenceHistory[memoryID]; !ok {
		a.confidenceHistory[memoryID] = []float64{}
	}
	a.confidenceHistory[memoryID] = append(a.confidenceHistory[memoryID], newConfidence)

	a.logger.Info("reported outcome",
		zap.String("memory_id", memoryID),
		zap.Bool("succeeded", succeeded),
		zap.Float64("new_confidence", newConfidence))

	return newConfidence, nil
}

// ShouldGiveFeedback decides if feedback should be given based on persona.
func (a *Agent) ShouldGiveFeedback() bool {
	switch a.persona.FeedbackStyle {
	case "generous":
		return rand.Float64() < 0.8 // 80% chance
	case "critical":
		return rand.Float64() < 0.9 // 90% chance, but often negative
	case "realistic":
		return rand.Float64() < 0.3 // 30% chance (most users don't rate)
	case "random":
		return rand.Float64() < 0.5
	default:
		return rand.Float64() < 0.5
	}
}

// GenerateFeedbackDecision decides if a memory was helpful based on persona.
func (a *Agent) GenerateFeedbackDecision(memory MemoryResult) bool {
	switch a.persona.FeedbackStyle {
	case "generous":
		return rand.Float64() < 0.85 // 85% positive
	case "critical":
		return rand.Float64() < 0.4 // 40% positive
	case "realistic":
		return rand.Float64() < 0.7 // 70% positive
	case "random":
		return rand.Float64() < 0.5
	default:
		return rand.Float64() < 0.7
	}
}

// GenerateOutcome decides if a task succeeded based on persona's success rate.
func (a *Agent) GenerateOutcome() bool {
	rate := a.persona.SuccessRate
	if rate <= 0 || rate > 1 {
		rate = 0.7 // Default 70% success
	}
	return rand.Float64() < rate
}

// GetSession returns the current session state.
func (a *Agent) GetSession() *Session {
	return &Session{
		ID:        a.sessionID,
		Persona:   a.persona,
		ProjectID: a.projectID,
		StartTime: time.Now(), // Would track actual start
		Turns:     a.history,
		Feedback:  a.feedback,
		Outcomes:  a.outcomes,
		Metrics:   a.computeMetrics(),
	}
}

func (a *Agent) computeMetrics() SessionMetrics {
	positiveFeedback := 0
	for _, f := range a.feedback {
		if f.Helpful {
			positiveFeedback++
		}
	}

	successfulOutcomes := 0
	for _, o := range a.outcomes {
		if o.Succeeded {
			successfulOutcomes++
		}
	}

	// Calculate average confidence delta
	var totalDelta float64
	var count int
	for _, history := range a.confidenceHistory {
		if len(history) >= 2 {
			delta := history[len(history)-1] - history[0]
			totalDelta += delta
			count++
		}
	}
	avgDelta := 0.0
	if count > 0 {
		avgDelta = totalDelta / float64(count)
	}

	return SessionMetrics{
		MemoriesRecorded:   len(a.memoriesRecorded),
		MemoriesRetrieved:  len(a.memoriesRetrieved),
		FeedbackGiven:      len(a.feedback),
		PositiveFeedback:   positiveFeedback,
		OutcomesRecorded:   len(a.outcomes),
		SuccessfulOutcomes: successfulOutcomes,
		AvgConfidenceDelta: avgDelta,
	}
}

// GetConfidenceHistory returns confidence tracking for a memory.
func (a *Agent) GetConfidenceHistory(memoryID string) []float64 {
	return a.confidenceHistory[memoryID]
}
